// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/server/reload"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type modulePlanParams struct {
	ModuleName     string `json:"moduleName"`
	Action         string `json:"action"`
	Origin         string `json:"origin"`
	OperatorUserId string `json:"operatorUserId"`
}

type moduleApplyParams struct {
	Id string `json:"id"`
}

type moduleRunParams struct {
	Id       string `json:"id"`
	WithDemo bool   `json:"withDemo"`
}

type moduleOpResult struct {
	Ok           bool   `json:"ok"`
	ErrorDomain  string `json:"errorDomain,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type moduleCatalogSyncParams struct {
	OriginType string `json:"originType"`
	Force      bool   `json:"force"`
}

type moduleCatalogSyncResult struct {
	Ok         bool   `json:"ok"`
	OriginType string `json:"originType"`
	Total      int    `json:"total"`
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

type moduleLoadDemoParams struct {
	OperatorUserId string `json:"operatorUserId"`
}

// ModuleManagementOption configures the QuickJS module-management runtime plugin.
type ModuleManagementOption interface {
	apply(*moduleManagementConfig)
}

type moduleManagementOptionFunc func(*moduleManagementConfig)

func (f moduleManagementOptionFunc) apply(cfg *moduleManagementConfig) {
	if f == nil {
		return
	}
	f(cfg)
}

type moduleLifecycleFactory func(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) lifecycle.Service

type opsHostFactory func(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, events bus.EventBus) *ops.Ops

type compilerExecutorFactory func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error)

type moduleManagementConfig struct {
	lockerFactory           statepkg.LockerFactory
	moduleLifecycleFactory  moduleLifecycleFactory
	opsHostFactory          opsHostFactory
	compilerExecutorFactory compilerExecutorFactory
	events                  bus.EventBus
	// Test-only loadDemo hooks (nil in production).
	loadDemoPlanHook func(ops.PlanResult) ops.PlanResult
	loadDemoRunHook  func(error) error
}

// Overridable for tests that force Marshal failures.
var quickjsMarshal = func(ctx *quickjs.Context, v any) (*quickjs.Value, error) {
	return ctx.Marshal(v)
}

func defaultCompilerExecutorFactory(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
	return jsexecutor.NewCompilerExecutor(runtimeScope)
}

// bridgeThrow builds an Error value for promise rejection without setting a
// context exception (ThrowError + IsError is unreliable across quickjs-go).
// When err carries ChoysumError info, attaches errorDomain/errorCode for TS callers.
func bridgeThrow(ctx *quickjs.Context, err error) *quickjs.Value {
	val := ctx.NewError(err)
	if info := oerrors.GetErrorInfo(err); info != nil {
		if info.Domain != "" {
			val.Set("errorDomain", ctx.NewString(info.Domain))
		}
		if info.Code != "" {
			val.Set("errorCode", ctx.NewString(info.Code))
		}
	}
	return val
}

func defaultModuleManagementLockerFactory(runtimeScope scope.Scope) statepkg.Locker {
	return lease.New(runtimeScope)
}

func defaultModuleLifecycleFactory(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) lifecycle.Service {
	return lifecycle.NewService(runtimeScope, jsExecutor, lifecycle.WithLockerFactory(lockerFactory))
}

func resolveModuleManagementConfig(opts ...ModuleManagementOption) moduleManagementConfig {
	cfg := moduleManagementConfig{
		lockerFactory:          defaultModuleManagementLockerFactory,
		moduleLifecycleFactory: defaultModuleLifecycleFactory,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(&cfg)
	}
	if cfg.lockerFactory == nil {
		cfg.lockerFactory = defaultModuleManagementLockerFactory
	}
	if cfg.moduleLifecycleFactory == nil {
		cfg.moduleLifecycleFactory = defaultModuleLifecycleFactory
	}
	return cfg
}

// WithModuleManagementLockerFactory injects the locker used by module ops and catalog sync.
func WithModuleManagementLockerFactory(factory statepkg.LockerFactory) ModuleManagementOption {
	if factory == nil {
		return moduleManagementOptionFunc(func(*moduleManagementConfig) {})
	}
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.lockerFactory = factory
	})
}

// WithModuleManagementEvents injects the EventBus used for operation tip Publish.
func WithModuleManagementEvents(events bus.EventBus) ModuleManagementOption {
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.events = events
	})
}

// WithModuleManagementProvider installs the module-management bridge for a scope provider.
func WithModuleManagementProvider(scopeProvider jsengine.ScopeProvider, opts ...ModuleManagementOption) jsengine.JsEngineOption {
	cfg := resolveModuleManagementConfig(opts...)
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()
		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		mmObj := jse.Ctx.Object()
		mmObj.Set("plan", jse.Ctx.NewFunction(moduleBridgeAsyncFactory(jse, scopeProvider, cfg, performModulePlan)))
		mmObj.Set("apply", jse.Ctx.NewFunction(moduleBridgeAsyncFactory(jse, scopeProvider, cfg, performModuleApply)))
		mmObj.Set("run", jse.Ctx.NewFunction(moduleBridgeAsyncFactory(jse, scopeProvider, cfg, performModuleRun)))
		mmObj.Set("reload", jse.Ctx.NewFunction(moduleReloadAsyncFactory(jse)))
		mmObj.Set("syncCatalog", jse.Ctx.NewFunction(moduleBridgeAsyncFactory(jse, scopeProvider, cfg, performModuleCatalogSync)))
		mmObj.Set("loadDemo", jse.Ctx.NewFunction(moduleBridgeAsyncFactory(jse, scopeProvider, cfg, performModuleLoadDemo)))

		choysumObj.Set("moduleManagement", mmObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

// WithModuleManagement installs the module-management bridge for a fixed runtime scope.
func WithModuleManagement(runtimeScope scope.Scope, opts ...ModuleManagementOption) jsengine.JsEngineOption {
	return WithModuleManagementProvider(jsengine.StaticScopeProvider(runtimeScope), opts...)
}

type moduleBridgePerformer func(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value

func moduleBridgeAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, perform moduleBridgePerformer) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		// perform runs synchronously inside NewPromise; pass args directly.
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := perform(ctx, jse, scopeProvider, cfg, args)
			if ret.IsError() {
				defer ret.Free()
				reject(ret)
				return
			}
			defer ret.Free()
			resolve(ret)
		})
	}
}

func moduleReloadAsyncFactory(jse *quickjsengine.QuickjsEngine) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			result := map[string]any{"triggered": true, "failed": false}
			go func() {
				time.Sleep(1 * time.Millisecond)
				_ = reload.Trigger()
			}()

			val, err := quickjsMarshal(ctx, result)
			if err != nil {
				errVal := ctx.ThrowError(err)
				defer errVal.Free()
				reject(errVal)
				return
			}
			defer val.Free()
			resolve(val)
		})
	}
}

type startedOpsHost struct {
	host     *ops.Ops
	execCtx  context.Context
	executor jsexecutor.JsExecutor
}

func (h startedOpsHost) Close() error {
	if h.executor == nil {
		return nil
	}
	return h.executor.Stop()
}

func startOpsHost(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig) (startedOpsHost, error) {
	execCtx := scope.ContextWithoutTransaction(jse.ExecContext())
	runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)
	if runtimeScope == nil {
		return startedOpsHost{}, status.Error(codes.Internal, "module management scope is unavailable")
	}
	opScope := runtimeScope.WithContext(execCtx)
	if opScope == nil {
		opScope = runtimeScope
	}

	events := cfg.events
	if !bus.IsUsable(events) {
		events = bus.EnsureHost(opScope)
	}

	var host *ops.Ops
	if cfg.opsHostFactory != nil {
		host = cfg.opsHostFactory(opScope, nil, cfg.lockerFactory, events)
	}
	if host == nil {
		host = &ops.Ops{}
	}
	if host.Scope == nil {
		host.Scope = opScope
	}
	if host.LockerFactory == nil {
		host.LockerFactory = cfg.lockerFactory
	}
	if host.Events == nil {
		host.Events = events
	}

	// Compiler executor is only required when filling default Lifecycle/Resolver.
	var compilerExecutor jsexecutor.JsExecutor
	if host.Lifecycle == nil || host.Resolver == nil {
		newExecutor := cfg.compilerExecutorFactory
		if newExecutor == nil {
			newExecutor = defaultCompilerExecutorFactory
		}
		executor, err := newExecutor(opScope)
		if err != nil {
			return startedOpsHost{}, err
		}
		if err := executor.Start(); err != nil {
			_ = executor.Stop()
			return startedOpsHost{}, err
		}
		// Stop the executor if Lifecycle/Resolver construction panics before return.
		handedOff := false
		defer func() {
			if !handedOff {
				_ = executor.Stop()
			}
		}()
		if host.Lifecycle == nil {
			host.Lifecycle = cfg.moduleLifecycleFactory(opScope, executor, cfg.lockerFactory)
		}
		if host.Resolver == nil {
			host.Resolver = lifecycle.NewModuleManager(opScope, executor, lifecycle.WithLockerFactory(cfg.lockerFactory))
		}
		compilerExecutor = executor
		handedOff = true
	}
	return startedOpsHost{host: host, execCtx: execCtx, executor: compilerExecutor}, nil
}

func performModulePlan(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseJSONArgs[modulePlanParams](args)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	if strings.TrimSpace(params.Action) == "" {
		return bridgeThrow(ctx, status.Error(codes.InvalidArgument, "action is required"))
	}
	started, err := startOpsHost(jse, scopeProvider, cfg)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	result, err := started.host.Plan(started.execCtx, ops.PlanRequest{
		ModuleName:     params.ModuleName,
		Action:         params.Action,
		Origin:         params.Origin,
		OperatorUserId: params.OperatorUserId,
	})
	if err != nil {
		return throwOrOpResult(ctx, err)
	}
	val, marshalErr := quickjsMarshal(ctx, planResultToMap(result))
	if marshalErr != nil {
		return bridgeThrow(ctx, marshalErr)
	}
	return val
}

func performModuleApply(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseJSONArgs[moduleApplyParams](args)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	if strings.TrimSpace(params.Id) == "" {
		return bridgeThrow(ctx, status.Error(codes.InvalidArgument, "id is required"))
	}
	started, err := startOpsHost(jse, scopeProvider, cfg)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	err = started.host.Apply(started.execCtx, ops.ApplyRequest{Id: params.Id})
	return marshalOpResult(ctx, err)
}

func performModuleRun(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseJSONArgs[moduleRunParams](args)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	if strings.TrimSpace(params.Id) == "" {
		return bridgeThrow(ctx, status.Error(codes.InvalidArgument, "id is required"))
	}
	started, err := startOpsHost(jse, scopeProvider, cfg)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	err = started.host.Run(started.execCtx, ops.RunRequest{Id: params.Id, WithDemo: params.WithDemo})
	return marshalOpResult(ctx, err)
}

func performModuleCatalogSync(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseJSONArgs[moduleCatalogSyncParams](args)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	originType := normalizeModuleCatalogOriginType(params.OriginType)
	if originType == "" {
		return bridgeThrow(ctx, status.Error(codes.InvalidArgument, "originType must be one of: local, registry, all"))
	}

	started, err := startOpsHost(jse, scopeProvider, cfg)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	start := time.Now()
	stats, syncErr := started.host.SyncSourceSnapshotsThenMaterialize(started.execCtx, originType)
	result := moduleCatalogSyncResult{
		Ok:         syncErr == nil,
		OriginType: originType,
		Total:      stats.Total,
		Success:    stats.Success,
		Failed:     stats.Failed,
		DurationMs: time.Since(start).Milliseconds(),
	}
	if syncErr != nil {
		if shouldThrowBridgeError(syncErr) {
			return bridgeThrow(ctx, syncErr)
		}
		result.Error = syncErr.Error()
	}
	val, marshalErr := quickjsMarshal(ctx, result)
	if marshalErr != nil {
		return bridgeThrow(ctx, marshalErr)
	}
	return val
}

func performModuleLoadDemo(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseJSONArgs[moduleLoadDemoParams](args)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	started, err := startOpsHost(jse, scopeProvider, cfg)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	planRes, err := started.host.Plan(started.execCtx, ops.PlanRequest{
		Action:         ops.ActionLoadDemo,
		OperatorUserId: params.OperatorUserId,
	})
	if err != nil {
		return throwOrOpResult(ctx, err)
	}
	if cfg.loadDemoPlanHook != nil {
		planRes = cfg.loadDemoPlanHook(planRes)
	}
	if strings.TrimSpace(planRes.Id) == "" {
		return marshalOpResult(ctx, status.Error(codes.Internal, "plan returned empty id"))
	}
	runErr := started.host.Run(started.execCtx, ops.RunRequest{Id: planRes.Id})
	if cfg.loadDemoRunHook != nil {
		runErr = cfg.loadDemoRunHook(runErr)
	}
	if runErr != nil && shouldThrowBridgeError(runErr) {
		return bridgeThrow(ctx, runErr)
	}
	payload := map[string]any{
		"ok": runErr == nil,
		"id": planRes.Id,
	}
	if runErr != nil {
		info := oerrors.GetErrorInfo(runErr)
		if info != nil {
			payload["errorDomain"] = info.Domain
			payload["errorCode"] = info.Code
			if info.Message != "" {
				payload["errorMessage"] = info.Message
			} else {
				payload["errorMessage"] = runErr.Error()
			}
		} else {
			payload["errorDomain"] = "MODULE_MANAGEMENT"
			payload["errorCode"] = "OP_FAILED"
			payload["errorMessage"] = runErr.Error()
		}
	}
	val, marshalErr := quickjsMarshal(ctx, payload)
	if marshalErr != nil {
		return bridgeThrow(ctx, marshalErr)
	}
	return val
}

func planResultToMap(result ops.PlanResult) map[string]any {
	order := make([]map[string]any, 0, len(result.ModuleOrder))
	for _, e := range result.ModuleOrder {
		order = append(order, map[string]any{
			"moduleName":     e.ModuleName,
			"reason":         e.Reason,
			"currentVersion": e.CurrentVersion,
			"targetVersion":  e.TargetVersion,
			"origin":         e.Origin,
		})
	}
	issues := func(list []ops.PlanIssue) []map[string]any {
		out := make([]map[string]any, 0, len(list))
		for _, issue := range list {
			out = append(out, map[string]any{
				"code":    issue.Code,
				"level":   issue.Level,
				"message": issue.Message,
			})
		}
		return out
	}
	return map[string]any{
		"id":           result.Id,
		"action":       result.Action,
		"moduleName":   result.ModuleName,
		"moduleOrder":  order,
		"affectedApps": result.AffectedApps,
		"risks":        issues(result.Risks),
		"blockers":     issues(result.Blockers),
		"baseRevision": result.BaseRevision,
	}
}

func marshalOpResult(ctx *quickjs.Context, err error) *quickjs.Value {
	result := moduleOpResult{Ok: err == nil}
	if err != nil {
		if shouldThrowBridgeError(err) {
			return bridgeThrow(ctx, err)
		}
		info := oerrors.GetErrorInfo(err)
		if info != nil {
			result.ErrorDomain = info.Domain
			result.ErrorCode = info.Code
			if info.Message != "" {
				result.ErrorMessage = info.Message
			} else {
				result.ErrorMessage = err.Error()
			}
		} else {
			result.ErrorDomain = "MODULE_MANAGEMENT"
			result.ErrorCode = "OP_FAILED"
			result.ErrorMessage = err.Error()
		}
	}
	val, marshalErr := quickjsMarshal(ctx, result)
	if marshalErr != nil {
		return bridgeThrow(ctx, marshalErr)
	}
	return val
}

func throwOrOpResult(ctx *quickjs.Context, err error) *quickjs.Value {
	if shouldThrowBridgeError(err) {
		return bridgeThrow(ctx, err)
	}
	return marshalOpResult(ctx, err)
}

func shouldThrowBridgeError(err error) bool {
	if err == nil {
		return false
	}
	info := oerrors.GetErrorInfo(err)
	if info != nil && info.Domain == "meta.lock" && info.Code == "LEASE_CONFLICT" {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if st, ok := status.FromError(err); ok {
		if st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded {
			return true
		}
	}
	return false
}

func parseJSONArgs[T any](args []*quickjs.Value) (T, error) {
	var params T
	if len(args) == 0 || args[0] == nil || args[0].IsUndefined() || args[0].IsNull() {
		return params, nil
	}
	jsonStr := args[0].JSONStringify()
	// Non-serializable values (functions/symbols) stringify to "undefined";
	// treat that and empty output as omitted args.
	if jsonStr == "" || jsonStr == "undefined" {
		return params, nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		return params, err
	}
	return params, nil
}

func normalizeModuleCatalogOriginType(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "", "all":
		return "all"
	case "local", "registry":
		return value
	default:
		return ""
	}
}
