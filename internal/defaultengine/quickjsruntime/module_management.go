// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/server/reload"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type moduleOpParams struct {
	ModuleName     string `json:"moduleName"`
	WithDemo       bool   `json:"withDemo"`
	OperatorUserId string `json:"operatorUserId"`
	JobId          string `json:"jobId"`
	Action         string `json:"action"`
	BaseRevision   string `json:"baseRevision"`
}

type moduleOpResult struct {
	Ok           bool   `json:"ok"`
	ErrorDomain  string `json:"errorDomain,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type moduleIndexSyncParams struct {
	OriginType string `json:"originType"`
	Force      bool   `json:"force"`
}

type moduleIndexSyncResult struct {
	Ok         bool   `json:"ok"`
	OriginType string `json:"originType"`
	Total      int    `json:"total"`
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

type moduleIndexSyncRunner func(ctx context.Context, runtimeScope scope.Scope, originType string) (lifecycle.ModuleIndexSyncStats, error)

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

type moduleManagementConfig struct {
	lockerFactory          statepkg.LockerFactory
	moduleLifecycleFactory moduleLifecycleFactory
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

// WithModuleManagementLockerFactory injects the locker used by both module
// operations and local module-index sync inside the QuickJS runtime plugin.
func WithModuleManagementLockerFactory(factory statepkg.LockerFactory) ModuleManagementOption {
	if factory == nil {
		return moduleManagementOptionFunc(func(*moduleManagementConfig) {})
	}
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.lockerFactory = factory
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
		mmObj.Set("install", jse.Ctx.NewFunction(moduleOpAsyncFactory(jse, scopeProvider, cfg, "install")))
		mmObj.Set("uninstall", jse.Ctx.NewFunction(moduleOpAsyncFactory(jse, scopeProvider, cfg, "uninstall")))
		mmObj.Set("upgrade", jse.Ctx.NewFunction(moduleOpAsyncFactory(jse, scopeProvider, cfg, "upgrade")))
		mmObj.Set("reload", jse.Ctx.NewFunction(moduleReloadAsyncFactory(jse)))
		mmObj.Set("syncIndex", jse.Ctx.NewFunction(moduleIndexSyncAsyncFactory(jse, scopeProvider, cfg)))

		choysumObj.Set("moduleManagement", mmObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

// WithModuleManagement installs the module-management bridge for a fixed runtime scope.
func WithModuleManagement(runtimeScope scope.Scope, opts ...ModuleManagementOption) jsengine.JsEngineOption {
	return WithModuleManagementProvider(jsengine.StaticScopeProvider(runtimeScope), opts...)
}

func moduleOpAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, action string) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performModuleOp(ctx, jse, scopeProvider, cfg, action, args)
			if ret.IsError() {
				defer ret.Free()
				reject(ret)
			} else {
				defer ret.Free()
				resolve(ret)
			}
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

			val, err := ctx.Marshal(result)
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

func moduleIndexSyncAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performModuleIndexSync(ctx, jse, scopeProvider, cfg, args)
			if ret.IsError() {
				defer ret.Free()
				reject(ret)
			} else {
				defer ret.Free()
				resolve(ret)
			}
		})
	}
}

func newModuleLifecycleForModuleManagement(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, cfg moduleManagementConfig) lifecycle.Service {
	return cfg.moduleLifecycleFactory(runtimeScope, jsExecutor, cfg.lockerFactory)
}

func performModuleOp(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, action string, args []*quickjs.Value) *quickjs.Value {
	params, err := parseModuleOpParams(args)
	if err != nil {
		return ctx.ThrowError(err)
	}
	if params.ModuleName == "" {
		return ctx.ThrowError(status.Error(codes.InvalidArgument, "moduleName is required"))
	}
	execCtx := jse.ExecContext()
	if execCtx == nil {
		execCtx = context.Background()
	}
	runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)

	result := moduleOpResult{Ok: false}
	opScope := runtimeScope.WithContext(execCtx)
	if opScope == nil {
		opScope = runtimeScope
	}

	err = func() (opErr error) {
		compilerExecutor, err := jsexecutor.NewCompilerExecutor(opScope)
		if err != nil {
			return err
		}
		if err := compilerExecutor.Start(); err != nil {
			return err
		}
		defer func() {
			if stopErr := compilerExecutor.Stop(); opErr == nil && stopErr != nil {
				opErr = stopErr
			}
		}()

		moduleLifecycle := newModuleLifecycleForModuleManagement(opScope, compilerExecutor, cfg)
		switch action {
		case "install":
			return moduleLifecycle.Install(execCtx, lifecycle.InstallRequest{Name: params.ModuleName, WithDemo: params.WithDemo})
		case "uninstall":
			return moduleLifecycle.Uninstall(execCtx, lifecycle.UninstallRequest{Name: params.ModuleName})
		case "upgrade":
			return moduleLifecycle.Upgrade(execCtx, lifecycle.UpgradeRequest{Input: params.ModuleName, WithDemo: params.WithDemo})
		default:
			return status.Error(codes.InvalidArgument, "unknown action")
		}
	}()
	if err == nil {
		result.Ok = true
	}

	if err != nil {
		info := oerrors.GetErrorInfo(err)
		if info != nil && info.Domain == "meta.lock" && info.Code == "LEASE_CONFLICT" {
			return ctx.ThrowError(err)
		}
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded {
				return ctx.ThrowError(err)
			}
		}
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

	val, marshalErr := ctx.Marshal(result)
	if marshalErr != nil {
		return ctx.ThrowError(marshalErr)
	}
	return val
}

func performModuleIndexSync(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseModuleIndexSyncParams(args)
	if err != nil {
		return ctx.ThrowError(err)
	}
	originType := normalizeModuleIndexOriginType(params.OriginType)
	switch originType {
	case "local":
	case "registry":
		// implemented in syncModuleIndexRegistry below
	case "all":
		// sequentially sync registry and local in a single backend request
	default:
		return ctx.ThrowError(status.Error(codes.InvalidArgument, "originType must be one of: local, registry, all"))
	}

	execCtx := jse.ExecContext()
	if execCtx == nil {
		execCtx = context.Background()
	}
	runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)

	start := time.Now()
	result, err := runModuleIndexSync(execCtx, runtimeScope, originType, func(runCtx context.Context, txScope scope.Scope, target string) (lifecycle.ModuleIndexSyncStats, error) {
		switch target {
		case "local":
			return syncModuleIndexLocal(runCtx, txScope, cfg.lockerFactory)
		case "registry":
			return syncModuleIndexRegistry(runCtx, txScope, cfg.lockerFactory)
		default:
			return lifecycle.ModuleIndexSyncStats{}, status.Error(codes.InvalidArgument, "originType must be one of: local, registry, all")
		}
	})

	result.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		info := oerrors.GetErrorInfo(err)
		if info != nil && info.Domain == "meta.lock" && info.Code == "LEASE_CONFLICT" {
			return ctx.ThrowError(err)
		}
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded {
				return ctx.ThrowError(err)
			}
		}
		result.Error = err.Error()
	}

	val, marshalErr := ctx.Marshal(result)
	if marshalErr != nil {
		return ctx.ThrowError(marshalErr)
	}
	return val
}

func parseModuleOpParams(args []*quickjs.Value) (moduleOpParams, error) {
	params := moduleOpParams{}
	if len(args) == 0 || args[0] == nil || args[0].IsUndefined() || args[0].IsNull() {
		return params, nil
	}
	jsonStr := args[0].JSONStringify()
	if jsonStr == "" {
		return params, nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		return params, err
	}
	return params, nil
}

func parseModuleIndexSyncParams(args []*quickjs.Value) (moduleIndexSyncParams, error) {
	params := moduleIndexSyncParams{}
	if len(args) == 0 || args[0] == nil || args[0].IsUndefined() || args[0].IsNull() {
		return params, nil
	}
	jsonStr := args[0].JSONStringify()
	if jsonStr == "" {
		return params, nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		return params, err
	}
	return params, nil
}

func normalizeModuleIndexOriginType(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "all"
	}
	if value == "local" {
		return "local"
	}
	if value == "registry" {
		return "registry"
	}
	if value == "all" {
		return "all"
	}
	return ""
}

func runModuleIndexSync(ctx context.Context, runtimeScope scope.Scope, originType string, runner moduleIndexSyncRunner) (moduleIndexSyncResult, error) {
	result := moduleIndexSyncResult{Ok: false, OriginType: originType}
	if runtimeScope == nil {
		return result, status.Error(codes.Internal, "runtime scope is nil")
	}
	if runner == nil {
		return result, status.Error(codes.Internal, "module index sync runner is nil")
	}

	runOrigin := func(target string) (lifecycle.ModuleIndexSyncStats, error) {
		runnerScope := runtimeScope.WithContext(ctx)
		if runnerScope == nil {
			runnerScope = runtimeScope
		}
		runnerCtx := runnerScope.Context()
		if runnerCtx == nil {
			runnerCtx = ctx
		}
		return runner(runnerCtx, runnerScope, target)
	}

	if originType == "all" {
		partialErrors := make([]string, 0, 2)
		successfulOrigins := 0
		for _, target := range []string{"registry", "local"} {
			stats, syncErr := runOrigin(target)
			result.Total += stats.Total
			result.Success += stats.Success
			result.Failed += stats.Failed
			if syncErr != nil {
				if isCancellationOrDeadlineError(syncErr) {
					return result, syncErr
				}
				partialErrors = append(partialErrors, target+": "+syncErr.Error())
				continue
			}
			successfulOrigins++
		}

		if successfulOrigins == 0 {
			return result, status.Error(codes.Unavailable, "module index sync failed for all origins: "+strings.Join(partialErrors, "; "))
		}
		result.Ok = true
		if len(partialErrors) > 0 {
			result.Error = strings.Join(partialErrors, "; ")
		}
		return result, nil
	}

	stats, syncErr := runOrigin(originType)
	if syncErr != nil {
		return result, syncErr
	}
	result.Total = stats.Total
	result.Success = stats.Success
	result.Failed = stats.Failed
	result.Ok = true
	return result, nil
}

func isCancellationOrDeadlineError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded
}

func syncModuleIndexLocal(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (lifecycle.ModuleIndexSyncStats, error) {
	return lifecycle.SyncLocalModuleIndex(ctx, runtimeScope, lockerFactory)
}

func syncModuleIndexRegistry(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (lifecycle.ModuleIndexSyncStats, error) {
	return lifecycle.SyncRegistryModuleIndex(ctx, runtimeScope, lockerFactory)
}
