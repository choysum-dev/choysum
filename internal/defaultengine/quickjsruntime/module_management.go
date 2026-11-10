// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/module/plan"
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

type moduleResolverFactory func(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) plan.Resolver

type opsHostFactory func(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, events bus.EventBus) *ops.Ops

type compilerExecutorFactory func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error)

type moduleManagementConfig struct {
	lockerFactory           statepkg.LockerFactory
	moduleLifecycleFactory  moduleLifecycleFactory
	moduleResolverFactory   moduleResolverFactory
	opsHostFactory          opsHostFactory
	compilerExecutorFactory compilerExecutorFactory
	events                  bus.EventBus
	// injectedHost is a process/engine-scoped Ops with Lifecycle/Resolver already set.
	// When present, startOpsHost only rebinds Scope/context and never Starts a compiler.
	injectedHost *ops.Ops
	// processHost enables lazy process-scoped assembly on first lifecycle call.
	processHost bool
	// Test-only loadDemo hooks (nil in production).
	loadDemoPlanHook func(ops.PlanResult) ops.PlanResult
	loadDemoRunHook  func(error) error
	// Test-only apply/run seams (nil in production). Used by unit tests and
	// e2e BackendEnv wiring (ModuleManagementE2ESeamsFromBackendEnv) to
	// reproduce former CHOYSUM_E2E_FORCE_* failure modes without product TS reads.
	applyHook         func(error) error
	runHook           func(error) error
	forceReloadFailed bool
}

// processCompilerExecutor is started once for production bridge wiring.
// Only the executor is process-scoped; Lifecycle/Resolver/Events are rebuilt per
// request against the caller's opScope so company/session binding stays correct.
// The compiler is created with scopeProvider(context.Background()) — the process
// ScopeProvider (not a request company/session) — so the first caller's execCtx
// does not pin the warm pool.
// Laziness avoids compiler-pool warmup recursing into host assembly.
var (
	processCompilerMu            sync.Mutex
	processCompilerExecutor      jsexecutor.JsExecutor
	processCompilerStop          func() error
	processCompilerAssembling    bool
	processCompilerStopRequested bool
	processCompilerWaiters       []chan struct{}
)

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

// WithModuleManagementHost injects a pre-assembled ops host (Lifecycle/Resolver ready).
// Callers own the host's compiler executor lifetime; bridge calls must not Start/Stop it.
func WithModuleManagementHost(host *ops.Ops) ModuleManagementOption {
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.injectedHost = host
	})
}

// WithModuleManagementProcessHost enables lazy process-scoped compiler assembly on
// the first plan/apply/run/loadDemo call (not at QuickJS engine create time).
func WithModuleManagementProcessHost() ModuleManagementOption {
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.processHost = true
	})
}

// WithModuleManagementApplyHook injects a test-only post-Apply error transform.
// Production callers must not set this; nil leaves Apply behavior unchanged.
func WithModuleManagementApplyHook(hook func(error) error) ModuleManagementOption {
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.applyHook = hook
	})
}

// WithModuleManagementRunHook injects a test-only post-Run error transform.
// Production callers must not set this; nil leaves Run behavior unchanged.
func WithModuleManagementRunHook(hook func(error) error) ModuleManagementOption {
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.runHook = hook
	})
}

// WithModuleManagementForceReloadFailed marks ReloadTriggered/ReloadFailed on
// the operation row after a successful Run (e2e reload-failed scenario).
func WithModuleManagementForceReloadFailed() ModuleManagementOption {
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.forceReloadFailed = true
	})
}

func wakeProcessCompilerWaitersLocked() {
	for _, ch := range processCompilerWaiters {
		close(ch)
	}
	processCompilerWaiters = nil
}

func moduleLifecycleOrDefault(factory moduleLifecycleFactory) moduleLifecycleFactory {
	if factory == nil {
		return defaultModuleLifecycleFactory
	}
	return factory
}

func ensureProcessCompilerExecutor(ctx context.Context, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig) (jsexecutor.JsExecutor, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		processCompilerMu.Lock()
		if processCompilerExecutor != nil {
			exec := processCompilerExecutor
			processCompilerMu.Unlock()
			return exec, nil
		}
		// Stop during assembly wakes waiters; exit promptly instead of re-parking
		// on a possibly hung factory.
		if processCompilerStopRequested {
			processCompilerMu.Unlock()
			return nil, status.Error(codes.Canceled, "module management compiler stopped")
		}
		if processCompilerAssembling {
			ch := make(chan struct{})
			processCompilerWaiters = append(processCompilerWaiters, ch)
			processCompilerMu.Unlock()
			select {
			case <-ch:
			case <-ctx.Done():
				processCompilerMu.Lock()
				for i, w := range processCompilerWaiters {
					if w == ch {
						processCompilerWaiters = append(processCompilerWaiters[:i], processCompilerWaiters[i+1:]...)
						break
					}
				}
				processCompilerMu.Unlock()
				return nil, ctx.Err()
			}
			continue
		}
		processCompilerAssembling = true
		processCompilerMu.Unlock()
		break
	}

	finished := false
	defer func() {
		if finished {
			return
		}
		// Panic during construction: release the assembling slot so waiters can retry.
		// Also clear stopRequested — a Stop that raced with this attempt must not
		// permanently block later ensure calls after reload.
		processCompilerMu.Lock()
		processCompilerAssembling = false
		processCompilerStopRequested = false
		wakeProcessCompilerWaitersLocked()
		processCompilerMu.Unlock()
	}()

	exec, stop, err := startProcessCompilerExecutor(scopeProvider, cfg)
	finished = true

	processCompilerMu.Lock()
	processCompilerAssembling = false
	if err != nil {
		// Do not cache failures or a raced Stop flag — later callers may retry.
		processCompilerStopRequested = false
		wakeProcessCompilerWaitersLocked()
		processCompilerMu.Unlock()
		return nil, err
	}
	if processCompilerStopRequested {
		processCompilerStopRequested = false
		wakeProcessCompilerWaitersLocked()
		processCompilerMu.Unlock()
		if stop != nil {
			_ = stop()
		}
		return nil, status.Error(codes.Canceled, "module management compiler stopped during assembly")
	}
	if processCompilerExecutor != nil {
		// Defensive: keep the first winner if one appeared.
		existing := processCompilerExecutor
		processCompilerMu.Unlock()
		if stop != nil {
			_ = stop()
		}
		return existing, nil
	}
	processCompilerExecutor = exec
	processCompilerStop = stop
	wakeProcessCompilerWaitersLocked()
	processCompilerMu.Unlock()
	return exec, nil
}

func startProcessCompilerExecutor(scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig) (jsexecutor.JsExecutor, func() error, error) {
	// Background: process ScopeProvider yields the process runtime scope, not a
	// request-bound company/session from the caller's execCtx.
	runtimeScope := jsengine.ResolveScope(scopeProvider, context.Background())
	if runtimeScope == nil {
		return nil, nil, status.Error(codes.Internal, "module management scope is unavailable")
	}
	newExecutor := cfg.compilerExecutorFactory
	if newExecutor == nil {
		newExecutor = defaultCompilerExecutorFactory
	}
	executor, err := newExecutor(runtimeScope)
	if err != nil {
		return nil, nil, err
	}
	if err := executor.Start(); err != nil {
		_ = executor.Stop()
		return nil, nil, err
	}
	return executor, executor.Stop, nil
}

// StopProcessModuleManagementHost stops the process-scoped compiler executor.
// Safe to call when none is running. Intended for server shutdown / reload.
// If assembly is in progress, the in-flight executor is discarded instead of stored.
func StopProcessModuleManagementHost() error {
	processCompilerMu.Lock()
	if processCompilerAssembling {
		processCompilerStopRequested = true
		// Unblock waiters parked on the in-flight assembly; they will observe
		// stopRequested / a cleared executor and retry or exit.
		wakeProcessCompilerWaitersLocked()
	}
	stop := processCompilerStop
	processCompilerExecutor = nil
	processCompilerStop = nil
	processCompilerMu.Unlock()
	if stop == nil {
		return nil
	}
	return stop()
}

// ResetProcessModuleManagementHostForTest clears the process-scoped compiler (tests only).
func ResetProcessModuleManagementHostForTest() {
	_ = StopProcessModuleManagementHost()
	processCompilerMu.Lock()
	processCompilerStopRequested = false
	processCompilerAssembling = false
	wakeProcessCompilerWaitersLocked()
	processCompilerMu.Unlock()
}

// SetProcessModuleManagementHostForTest installs a process-scoped compiler stop hook (tests only).
func SetProcessModuleManagementHostForTest(stop func() error) {
	processCompilerMu.Lock()
	defer processCompilerMu.Unlock()
	processCompilerStop = stop
	processCompilerAssembling = false
	processCompilerStopRequested = false
}

// Overridable for tests that force NewHost failures after the executor starts.
var assembleOpsHost = ops.NewHost

// AssembleModuleManagementHost builds a started ops host for process/engine wiring.
// The returned stop closes the owned compiler executor; pass nil-safe stop to defer.
func AssembleModuleManagementHost(scopeProvider jsengine.ScopeProvider, opts ...ModuleManagementOption) (host *ops.Ops, stop func() error, err error) {
	cfg := resolveModuleManagementConfig(opts...)
	runtimeScope := jsengine.ResolveScope(scopeProvider, context.Background())
	if runtimeScope == nil {
		return nil, nil, status.Error(codes.Internal, "module management scope is unavailable")
	}

	newExecutor := cfg.compilerExecutorFactory
	if newExecutor == nil {
		newExecutor = defaultCompilerExecutorFactory
	}
	executor, err := newExecutor(runtimeScope)
	if err != nil {
		return nil, nil, err
	}
	if err := executor.Start(); err != nil {
		_ = executor.Stop()
		return nil, nil, err
	}
	handedOff := false
	defer func() {
		if !handedOff {
			_ = executor.Stop()
		}
	}()

	events := cfg.events
	if !bus.IsUsable(events) {
		events = bus.EnsureHost(runtimeScope)
	}
	host, err = assembleOpsHost(ops.HostOptions{
		Scope:         runtimeScope,
		Executor:      executor,
		LockerFactory: cfg.lockerFactory,
		Events:        events,
		Lifecycle:     moduleLifecycleOrDefault(cfg.moduleLifecycleFactory)(runtimeScope, executor, cfg.lockerFactory),
		Resolver:      lifecycle.NewModuleManager(runtimeScope, executor, lifecycle.WithLockerFactory(cfg.lockerFactory)),
	})
	if err != nil {
		return nil, nil, err
	}
	handedOff = true
	return host, executor.Stop, nil
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

// startOpsHost rebinds request Scope/context onto an ops host.
// needLifecycle enables the test-only compiler bootstrap when nothing is injected;
// catalog sync always passes false so it never Starts a ModuleManager/compiler pool.
func startOpsHost(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, needLifecycle bool) (startedOpsHost, error) {
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

	if cfg.injectedHost != nil {
		host := *cfg.injectedHost
		host.Scope = opScope
		if host.LockerFactory == nil {
			host.LockerFactory = cfg.lockerFactory
		}
		// Always rebind Events to the current request bus.
		host.Events = events
		return startedOpsHost{host: &host, execCtx: execCtx, executor: nil}, nil
	}

	if cfg.processHost && needLifecycle {
		exec, err := ensureProcessCompilerExecutor(execCtx, scopeProvider, cfg)
		if err != nil {
			return startedOpsHost{}, err
		}
		// Match ops.NewHost: raw configs may omit lockerFactory (resolve fills it
		// for bridge wiring; keep a local fallback for direct startOpsHost calls).
		lockerFactory := cfg.lockerFactory
		if lockerFactory == nil {
			lockerFactory = defaultModuleManagementLockerFactory
		}
		resolverFactory := cfg.moduleResolverFactory
		if resolverFactory == nil {
			resolverFactory = func(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) plan.Resolver {
				return lifecycle.NewModuleManager(runtimeScope, jsExecutor, lifecycle.WithLockerFactory(lockerFactory))
			}
		}
		host := &ops.Ops{
			Scope:         opScope,
			Lifecycle:     moduleLifecycleOrDefault(cfg.moduleLifecycleFactory)(opScope, exec, lockerFactory),
			Resolver:      resolverFactory(opScope, exec, lockerFactory),
			LockerFactory: lockerFactory,
			Events:        events,
		}
		return startedOpsHost{host: host, execCtx: execCtx, executor: nil}, nil
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

	// Compiler executor is only required when filling default Lifecycle/Resolver
	// for plan/apply/run/loadDemo without an injected/process host (tests / fallback).
	var compilerExecutor jsexecutor.JsExecutor
	if needLifecycle && (host.Lifecycle == nil || host.Resolver == nil) {
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
			lifecycleFactory := cfg.moduleLifecycleFactory
			if lifecycleFactory == nil {
				lifecycleFactory = defaultModuleLifecycleFactory
			}
			host.Lifecycle = lifecycleFactory(opScope, executor, cfg.lockerFactory)
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
	started, err := startOpsHost(jse, scopeProvider, cfg, true)
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
	started, err := startOpsHost(jse, scopeProvider, cfg, true)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	err = started.host.Apply(started.execCtx, ops.ApplyRequest{Id: params.Id})
	if cfg.applyHook != nil {
		err = cfg.applyHook(err)
	}
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
	started, err := startOpsHost(jse, scopeProvider, cfg, true)
	if err != nil {
		return bridgeThrow(ctx, err)
	}
	defer func() { _ = started.Close() }()

	err = started.host.Run(started.execCtx, ops.RunRequest{Id: params.Id, WithDemo: params.WithDemo})
	if cfg.runHook != nil {
		hooked := cfg.runHook(err)
		// When Run already persisted success, flip the row to failed so C1 UI
		// (MetaModuleOperation Browse) observes the forced failure.
		if err == nil && hooked != nil {
			domain, code, message := "MODULE_MANAGEMENT", "OP_FAILED", hooked.Error()
			if info := oerrors.GetErrorInfo(hooked); info != nil {
				if info.Domain != "" {
					domain = info.Domain
				}
				if info.Code != "" {
					code = info.Code
				}
				if info.Message != "" {
					message = info.Message
				}
			}
			if markErr := started.host.ForceFailTerminal(context.WithoutCancel(started.execCtx), params.Id, domain, code, message); markErr != nil {
				return bridgeThrow(ctx, errors.Join(hooked, markErr))
			}
		}
		err = hooked
	}
	if err == nil && cfg.forceReloadFailed {
		if markErr := started.host.MarkReloadFailed(started.execCtx, params.Id); markErr != nil {
			return bridgeThrow(ctx, markErr)
		}
	}
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

	// Catalog sync only needs Scope/Locker/Events — never warm a compiler pool.
	started, err := startOpsHost(jse, scopeProvider, cfg, false)
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
	started, err := startOpsHost(jse, scopeProvider, cfg, true)
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
	// Soft no-op when Plan reports blockers (e.g. no pending demo): abandon draft and succeed.
	if len(planRes.Blockers) > 0 {
		if abandonErr := started.host.AbandonDraft(started.execCtx, planRes.Id, "plan blocked"); abandonErr != nil {
			if started.host != nil && started.host.Scope != nil {
				if lg := started.host.Scope.Logger(); lg != nil {
					lg.Warn("loadDemo abandon blocked draft failed", "op", planRes.Id, "error", abandonErr)
				}
			}
		}
		val, marshalErr := quickjsMarshal(ctx, map[string]any{
			"ok": true,
			"id": planRes.Id,
		})
		if marshalErr != nil {
			return bridgeThrow(ctx, marshalErr)
		}
		return val
	}
	// Match UI: Plan → Apply → Run.
	applyErr := started.host.Apply(started.execCtx, ops.ApplyRequest{Id: planRes.Id})
	if cfg.applyHook != nil {
		applyErr = cfg.applyHook(applyErr)
	}
	if applyErr != nil {
		domain, code, message := "MODULE_MANAGEMENT", "OP_FAILED", applyErr.Error()
		if info := oerrors.GetErrorInfo(applyErr); info != nil {
			if info.Domain != "" {
				domain = info.Domain
			}
			if info.Code != "" {
				code = info.Code
			}
			if info.Message != "" {
				message = info.Message
			}
		}
		// Draft or queued: terminate so retries do not leave a lingering open plan.
		// Apply may have already failed the row (stamp); ForceFailTerminal no-ops on failed.
		// Detach ctx so cancel/deadline from Apply does not block the terminal write.
		_ = started.host.ForceFailTerminal(context.WithoutCancel(started.execCtx), planRes.Id, domain, code, message)
		if shouldThrowBridgeError(applyErr) {
			return bridgeThrow(ctx, applyErr)
		}
		payload := map[string]any{
			"ok":           false,
			"id":           planRes.Id,
			"errorDomain":  domain,
			"errorCode":    code,
			"errorMessage": message,
		}
		val, marshalErr := quickjsMarshal(ctx, payload)
		if marshalErr != nil {
			return bridgeThrow(ctx, marshalErr)
		}
		return val
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
	mapEntry := func(e ops.ModuleOrderEntry) map[string]any {
		return map[string]any{
			"moduleName":     e.ModuleName,
			"application":    e.Application,
			"role":           e.Role,
			"effect":         e.Effect,
			"reason":         e.Reason,
			"currentVersion": e.CurrentVersion,
			"targetVersion":  e.TargetVersion,
			"origin":         e.Origin,
		}
	}
	order := make([]map[string]any, 0, len(result.ModuleOrder))
	for _, e := range result.ModuleOrder {
		order = append(order, mapEntry(e))
	}
	ensure := make([]map[string]any, 0, len(result.EnsureModules))
	for _, e := range result.EnsureModules {
		ensure = append(ensure, mapEntry(e))
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
		"id":            result.Id,
		"action":        result.Action,
		"moduleName":    result.ModuleName,
		"moduleOrder":   order,
		"ensureModules": ensure,
		"affectedApps":  result.AffectedApps,
		"risks":         issues(result.Risks),
		"blockers":      issues(result.Blockers),
		"baseRevision":  result.BaseRevision,
		"needsWebBuild": result.NeedsWebBuild,
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
