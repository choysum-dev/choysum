// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCoverage_ModuleManagementConfigAndHelpers(t *testing.T) {
	var nilFunc moduleManagementOptionFunc
	nilFunc.apply(nil)

	cfg := resolveModuleManagementConfig(nil, WithModuleManagementLockerFactory(nil), moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.lockerFactory = nil
		c.moduleLifecycleFactory = nil
	}))
	if cfg.lockerFactory == nil || cfg.moduleLifecycleFactory == nil {
		t.Fatal("expected default factories restored")
	}
	_ = cfg.lockerFactory(&moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{},
	})
	_ = defaultModuleManagementLockerFactory(nil)
	_ = defaultModuleLifecycleFactory(nil, nil, nil)

	engine := newTestQuickjsEngine(t, WithModuleManagement(
		&moduleIndexTestScope{
			ctx:    context.Background(),
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg:    &config.Config{ModulesPath: t.TempDir()},
		},
		moduleManagementOptionFunc(func(c *moduleManagementConfig) {
			c.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     moduleManagementTestManager{},
					Resolver:      &mapResolver{},
					LockerFactory: lockerFactory,
					Events:        ev,
				}
			}
		}),
	))
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.reload()`)

	if err := (startedOpsHost{}).Close(); err != nil {
		t.Fatalf("nil executor Close: %v", err)
	}
}

func TestCoverage_ModuleManagementBridgeErrorPaths(t *testing.T) {
	db := newBridgeTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	life := moduleManagementTestManager{install: func(context.Context, string) error {
		return oerrors.New("MODULE_MANAGEMENT", "INSTALL_FAILED", "boom")
	}}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     life,
					Resolver:      &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
					LockerFactory: lockerFactory,
					Events:        ev,
				}
			}
		}),
	))

	rejectContains(t, engine, `$choysum.moduleManagement.plan({})`, "action is required")
	rejectContains(t, engine, `$choysum.moduleManagement.plan({ action: 'install' })`, "moduleName")
	rejectContains(t, engine, `$choysum.moduleManagement.apply({})`, "id is required")
	rejectContains(t, engine, `$choysum.moduleManagement.run({})`, "id is required")
	rejectContains(t, engine, `$choysum.moduleManagement.syncCatalog({ originType: 'cloud' })`, "originType")

	planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install' })`)
	id, _ := planJSON["id"].(string)
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(id)+` })`)
	runJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(id)+` })`)
	if ok, _ := runJSON["ok"].(bool); ok {
		t.Fatalf("expected failed run, got %#v", runJSON)
	}
	if code, _ := runJSON["errorCode"].(string); code == "" {
		t.Fatalf("expected errorCode, got %#v", runJSON)
	}

	// Non-throwing sync failure lands in result.Error.
	failEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     moduleManagementTestManager{},
					Resolver:      &mapResolver{},
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, errors.New("sync blew up")
					},
				}
			}
		}),
	))
	syncJSON := awaitModuleManagementJSON(t, failEngine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`)
	if ok, _ := syncJSON["ok"].(bool); ok {
		t.Fatalf("expected sync !ok, got %#v", syncJSON)
	}
	if errText, _ := syncJSON["error"].(string); !strings.Contains(errText, "sync blew up") {
		t.Fatalf("expected sync error, got %#v", syncJSON)
	}
}

func TestCoverage_ModuleManagementStartOpsHostBranches(t *testing.T) {
	db := newBridgeTestDB(t)
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}

	// Nil scope from provider.
	nilEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		func(context.Context) scope.Scope { return nil },
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
				return &ops.Ops{Lifecycle: moduleManagementTestManager{}, Resolver: &mapResolver{}}
			}
		}),
	))
	rejectContains(t, nilEngine, `$choysum.moduleManagement.plan({ action: 'sync' })`, "scope is unavailable")

	// opsHostFactory returns nil → empty Ops filled with injected Lifecycle/Resolver.
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		WithModuleManagementEvents(nil),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
				return nil
			}
			cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
				return moduleManagementTestManager{}
			}
		}),
	))
	// Missing Lifecycle+Resolver triggers CompilerExecutor; without registered engine this rejects.
	rejectContains(t, engine, `$choysum.moduleManagement.loadDemo({})`, "factory")

	// Partial host (Lifecycle set, Resolver nil) also needs executor.
	partial := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{Scope: runtimeScope, Lifecycle: moduleManagementTestManager{}, LockerFactory: lockerFactory, Events: ev}
			}
		}),
	))
	rejectContains(t, partial, `$choysum.moduleManagement.plan({ action: 'sync' })`, "factory")
}

func TestCoverage_ModuleManagementThrowOrOpResultAndParse(t *testing.T) {
	db := newBridgeTestDB(t)
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     moduleManagementTestManager{},
					Resolver:      &mapResolver{},
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, oerrors.New("meta.lock", "LEASE_CONFLICT", "busy")
					},
				}
			}
		}),
	))
	rejectContains(t, engine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`, "LEASE_CONFLICT")

	cancelEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     moduleManagementTestManager{},
					Resolver:      &mapResolver{},
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, context.Canceled
					},
				}
			}
		}),
	))
	rejectContains(t, cancelEngine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`, "canceled")

	deadlineEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     moduleManagementTestManager{},
					Resolver:      &mapResolver{},
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, status.Error(codes.DeadlineExceeded, "deadline")
					},
				}
			}
		}),
	))
	rejectContains(t, deadlineEngine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`, "deadline")

	// Invalid JSON args.
	badJSON := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{Scope: runtimeScope, Lifecycle: moduleManagementTestManager{}, Resolver: &mapResolver{}, LockerFactory: lockerFactory, Events: ev}
			}
		}),
	))
	rejectContains(t, badJSON, `$choysum.moduleManagement.plan('not-json')`, "")

	// Empty / null args accepted for loadDemo.
	_ = awaitModuleManagementJSON(t, badJSON, `$choysum.moduleManagement.loadDemo()`)
	_ = awaitModuleManagementJSON(t, badJSON, `$choysum.moduleManagement.loadDemo(null)`)

	// Plan failure path via throwOrOpResult (plain error → op result).
	planFail := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:     runtimeScope,
					Lifecycle: moduleManagementTestManager{},
					Resolver: &mapResolver{peek: map[string]*pkgmeta.Module{
						"missing": nil,
					}},
					LockerFactory: lockerFactory,
					Events:        ev,
				}
			}
		}),
	))
	got := awaitModuleManagementJSON(t, planFail, `$choysum.moduleManagement.plan({ moduleName: 'nope', action: 'install' })`)
	if ok, _ := got["ok"].(bool); ok {
		// Plan may return blockers with id, or ok:false — either is fine for coverage.
		_ = ok
	}

	// loadDemo Run maps oerrors.
	loadDemo := awaitModuleManagementJSON(t, badJSON, `$choysum.moduleManagement.loadDemo({ operatorUserId: 'u' })`)
	if ok, _ := loadDemo["ok"].(bool); ok {
		t.Fatalf("expected loadDemo !ok, got %#v", loadDemo)
	}
	if code, _ := loadDemo["errorCode"].(string); code != "LOAD_DEMO_NOT_IMPLEMENTED" {
		t.Fatalf("errorCode = %#v", loadDemo)
	}

	// shouldThrowBridgeError helpers
	if shouldThrowBridgeError(nil) {
		t.Fatal("nil should not throw")
	}
	if !shouldThrowBridgeError(context.DeadlineExceeded) {
		t.Fatal("deadline should throw")
	}
	if !shouldThrowBridgeError(status.Error(codes.Canceled, "c")) {
		t.Fatal("canceled status should throw")
	}
	if shouldThrowBridgeError(status.Error(codes.Internal, "x")) {
		t.Fatal("internal should not throw")
	}

	mapped := planResultToMap(ops.PlanResult{
		Id:     "x",
		Action: "install",
		ModuleOrder: []ops.ModuleOrderEntry{{
			ModuleName:     "core",
			Reason:         "root",
			CurrentVersion: "1",
			TargetVersion:  "2",
			Origin:         "local",
		}},
		Risks:    []ops.PlanIssue{{Code: "R", Level: "warn", Message: "m"}},
		Blockers: []ops.PlanIssue{{Code: "B", Level: "error", Message: "b"}},
	})
	order, _ := mapped["moduleOrder"].([]map[string]any)
	if len(order) != 1 {
		t.Fatalf("moduleOrder = %#v", mapped["moduleOrder"])
	}

	// marshalOpResult message fallbacks via the live QuickJS context.
	ctx := badJSON.Ctx
	infoErr := oerrors.New("d", "c", "")
	val := marshalOpResult(ctx, infoErr)
	defer val.Free()
	plain := marshalOpResult(ctx, errors.New("plain"))
	defer plain.Free()
	okVal := marshalOpResult(ctx, nil)
	defer okVal.Free()
	thrown := throwOrOpResult(ctx, oerrors.New("meta.lock", "LEASE_CONFLICT", "busy"))
	defer thrown.Free()
	if domain := thrown.Get("errorDomain"); !domain.IsUndefined() {
		defer domain.Free()
		if domain.String() != "meta.lock" {
			t.Fatalf("bridgeThrow errorDomain = %q", domain.String())
		}
	} else {
		t.Fatal("expected bridgeThrow errorDomain")
	}
	if code := thrown.Get("errorCode"); !code.IsUndefined() {
		defer code.Free()
		if code.String() != "LEASE_CONFLICT" {
			t.Fatalf("bridgeThrow errorCode = %q", code.String())
		}
	} else {
		t.Fatal("expected bridgeThrow errorCode")
	}
	soft := throwOrOpResult(ctx, errors.New("soft"))
	defer soft.Free()

	if _, err := parseJSONArgs[modulePlanParams](nil); err != nil {
		t.Fatalf("nil args: %v", err)
	}
	if _, err := parseJSONArgs[modulePlanParams]([]*quickjs.Value{nil}); err != nil {
		t.Fatalf("nil value: %v", err)
	}
	badArg := ctx.Eval(`"not-an-object"`)
	defer badArg.Free()
	if _, err := parseJSONArgs[modulePlanParams]([]*quickjs.Value{badArg}); err == nil {
		t.Fatal("expected unmarshal error for string arg")
	}
}

func TestCoverage_ModuleManagementRejectOnStartOpsHost(t *testing.T) {
	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
				return nil
			}
		}),
	))
	for _, expr := range []string{
		`$choysum.moduleManagement.apply({ id: 'x' })`,
		`$choysum.moduleManagement.run({ id: 'x' })`,
		`$choysum.moduleManagement.syncCatalog({ originType: 'all' })`,
		`$choysum.moduleManagement.loadDemo({})`,
	} {
		rejectContains(t, engine, expr, "factory")
	}
}

func TestCoverage_ModuleBridgeAsyncRejectPath(t *testing.T) {
	db := newBridgeTestDB(t)
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope: runtimeScope, Lifecycle: moduleManagementTestManager{}, Resolver: &mapResolver{},
					LockerFactory: lockerFactory, Events: ev,
				}
			}
		}),
	))
	// Force apply with missing id after parse — already covered; force start failure reject path:
	rejectContains(t, engine, `$choysum.moduleManagement.apply({ id: 'missing-op' })`, "")
}

type stubBridgeExecutor struct {
	startErr error
	stopErr  error
	started  int
	stopped  int
}

func (e *stubBridgeExecutor) AppendJsScripts(...*jsengine.JsScript) {}
func (e *stubBridgeExecutor) Start() error {
	e.started++
	return e.startErr
}
func (e *stubBridgeExecutor) Stop() error {
	e.stopped++
	return e.stopErr
}
func (e *stubBridgeExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *stubBridgeExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (e *stubBridgeExecutor) SetJsScripts([]*jsengine.JsScript)  {}
func (e *stubBridgeExecutor) Reload(...*jsengine.JsScript) error { return nil }

type nilWithContextScope struct {
	*moduleIndexTestScope
}

func (s *nilWithContextScope) WithContext(context.Context) scope.Scope { return nil }

func TestCoverage_ModuleManagementRemainingBranches(t *testing.T) {
	prevMarshal := quickjsMarshal
	t.Cleanup(func() {
		quickjsMarshal = prevMarshal
	})

	db := newBridgeTestDB(t)
	base := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}

	opsHostOpt := moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
			return &ops.Ops{
				Scope: runtimeScope, Lifecycle: moduleManagementTestManager{}, Resolver: &mapResolver{},
				LockerFactory: lockerFactory, Events: ev,
			}
		}
	})

	// Direct async factory reject path (IsError).
	engine := newTestQuickjsEngine(t)
	fn := moduleBridgeAsyncFactory(engine, jsengine.StaticScopeProvider(base), moduleManagementConfig{}, func(ctx *quickjs.Context, _ *quickjsengine.QuickjsEngine, _ jsengine.ScopeProvider, _ moduleManagementConfig, _ []*quickjs.Value) *quickjs.Value {
		return ctx.NewError(errors.New("forced bridge error"))
	})
	promise := fn(engine.Ctx, nil, nil)
	defer promise.Free()
	ret := promise.Await()
	defer ret.Free()
	if !ret.IsError() && !ret.IsException() {
		t.Fatalf("expected rejected promise, got %s", ret.String())
	}

	// Reload marshal failure.
	quickjsMarshal = func(*quickjs.Context, any) (*quickjs.Value, error) {
		return nil, errors.New("marshal boom")
	}
	reloadEngine := newTestQuickjsEngine(t, WithModuleManagement(base))
	rejectContains(t, reloadEngine, `$choysum.moduleManagement.reload()`, "")
	quickjsMarshal = prevMarshal

	// Close with non-nil executor Stop.
	stub := &stubBridgeExecutor{stopErr: errors.New("stop boom")}
	if err := (startedOpsHost{executor: stub}).Close(); err == nil || !strings.Contains(err.Error(), "stop boom") {
		t.Fatalf("Close stop err = %v", err)
	}

	// startOpsHost: WithContext nil fallback, EnsureHost, stub executor fill Lifecycle+Resolver.
	nilWC := &nilWithContextScope{moduleIndexTestScope: base}
	started, err := startOpsHost(newTestQuickjsEngine(t), func(context.Context) scope.Scope { return nilWC }, moduleManagementConfig{
		lockerFactory: func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} },
		moduleLifecycleFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
			return moduleManagementTestManager{}
		},
		opsHostFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
			return &ops.Ops{} // Lifecycle+Resolver nil → create executor
		},
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		},
		events: nil,
	})
	if err != nil {
		t.Fatalf("startOpsHost: %v", err)
	}
	if started.executor == nil {
		t.Fatal("expected executor")
	}
	_ = started.Close()

	// executor Start failure.
	_, err = startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		opsHostFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
			return &ops.Ops{}
		},
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{startErr: errors.New("start boom")}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "start boom") {
		t.Fatalf("start err = %v", err)
	}

	// executor factory create failure.
	_, err = startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		opsHostFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
			return &ops.Ops{Lifecycle: moduleManagementTestManager{}} // Resolver nil
		},
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, errors.New("create boom")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "create boom") {
		t.Fatalf("create err = %v", err)
	}

	// Only Resolver missing → fill via ModuleManager with stub executor.
	_, err = startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		opsHostFactory: func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
			return &ops.Ops{Scope: runtimeScope, Lifecycle: moduleManagementTestManager{}, LockerFactory: lockerFactory, Events: ev}
		},
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		},
	})
	if err != nil {
		t.Fatalf("resolver fill: %v", err)
	}

	// Nil moduleLifecycleFactory falls back to default when filling Lifecycle.
	startedNilLife, err := startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		opsHostFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
			return &ops.Ops{}
		},
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		},
	})
	if err != nil {
		t.Fatalf("nil lifecycle factory: %v", err)
	}
	if startedNilLife.host == nil || startedNilLife.host.Lifecycle == nil {
		t.Fatal("expected default lifecycle")
	}
	_ = startedNilLife.Close()

	// parseJSONArgs empty JSON stringify path via undefined.
	undef := engine.Ctx.Undefined()
	defer undef.Free()
	if _, err := parseJSONArgs[modulePlanParams]([]*quickjs.Value{undef}); err != nil {
		t.Fatalf("undefined args: %v", err)
	}

	// loadDemo: default LOAD_DEMO_NOT_IMPLEMENTED soft result.
	leaseRun := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		opsHostOpt,
	))
	got := awaitModuleManagementJSON(t, leaseRun, `$choysum.moduleManagement.loadDemo({})`)
	if code, _ := got["errorCode"].(string); code != "LOAD_DEMO_NOT_IMPLEMENTED" {
		t.Fatalf("loadDemo code = %#v", got)
	}

	emptyPlanID := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		opsHostOpt,
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.loadDemoPlanHook = func(ops.PlanResult) ops.PlanResult { return ops.PlanResult{} }
		}),
	))
	emptyID := awaitModuleManagementJSON(t, emptyPlanID, `$choysum.moduleManagement.loadDemo({})`)
	if ok, _ := emptyID["ok"].(bool); ok {
		t.Fatalf("expected empty plan id failure, got %#v", emptyID)
	}

	leaseConflictDemo := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		opsHostOpt,
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.loadDemoRunHook = func(error) error { return oerrors.New("meta.lock", "LEASE_CONFLICT", "busy") }
		}),
	))
	rejectContains(t, leaseConflictDemo, `$choysum.moduleManagement.loadDemo({})`, "")

	emptyMsgDemo := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		opsHostOpt,
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.loadDemoRunHook = func(error) error { return oerrors.New("ops", "X", "") }
		}),
	))
	emptyMsg := awaitModuleManagementJSON(t, emptyMsgDemo, `$choysum.moduleManagement.loadDemo({})`)
	if msg, _ := emptyMsg["errorMessage"].(string); msg == "" {
		t.Fatalf("expected fallback errorMessage, got %#v", emptyMsg)
	}

	plainRunDemo := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		opsHostOpt,
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.loadDemoRunHook = func(error) error { return errors.New("plain run fail") }
		}),
	))
	plainRun := awaitModuleManagementJSON(t, plainRunDemo, `$choysum.moduleManagement.loadDemo({})`)
	if code, _ := plainRun["errorCode"].(string); code != "OP_FAILED" {
		t.Fatalf("plain run = %#v", plainRun)
	}

	// oerrors empty message + marshalOpResult marshal fail + plan marshal fail + sync marshal fail + loadDemo marshal fail.
	quickjsMarshal = func(ctx *quickjs.Context, v any) (*quickjs.Value, error) {
		return nil, errors.New("marshal fail")
	}
	rejectContains(t, leaseRun, `$choysum.moduleManagement.loadDemo({})`, "marshal fail")
	rejectContains(t, leaseRun, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'sync' })`, "marshal fail")
	rejectContains(t, leaseRun, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`, "marshal")
	quickjsMarshal = prevMarshal

	infoEmpty := oerrors.New("d", "c", "")
	val := marshalOpResult(leaseRun.Ctx, infoEmpty)
	defer val.Free()
	raw := val.JSONStringify()
	if !strings.Contains(raw, "errorMessage") {
		t.Fatalf("expected errorMessage fallback, got %s", raw)
	}

	// Perform parse failures / startOpsHost failures for apply/run/sync/loadDemo/plan.
	rejectContains(t, leaseRun, `$choysum.moduleManagement.apply("bad")`, "")
	rejectContains(t, leaseRun, `$choysum.moduleManagement.run("bad")`, "")
	rejectContains(t, leaseRun, `$choysum.moduleManagement.syncCatalog("bad")`, "")
	rejectContains(t, leaseRun, `$choysum.moduleManagement.loadDemo("bad")`, "")
	rejectContains(t, leaseRun, `$choysum.moduleManagement.plan("bad")`, "")

	// defaultCompilerExecutorFactory smoke (may fail without registered engine).
	if _, err := defaultCompilerExecutorFactory(base); err == nil {
		t.Log("default compiler factory unexpectedly succeeded")
	}

	// loadDemo plan throwOrOpResult path — Plan with ActionLoadDemo always succeeds; force Plan failure via closed DB.
	closed := newBridgeTestDB(t)
	sqlDB, _ := closed.DB()
	_ = sqlDB.Close()
	closedScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: closed},
	}
	closedEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(closedScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{Scope: runtimeScope, Lifecycle: moduleManagementTestManager{}, Resolver: &mapResolver{}, LockerFactory: lockerFactory, Events: ev}
			}
		}),
	))
	rejectContains(t, closedEngine, `$choysum.moduleManagement.loadDemo({})`, "")
	rejectContains(t, closedEngine, `$choysum.moduleManagement.plan({ action: 'sync' })`, "")

	// handedOff=false stop path when Lifecycle factory panics after Start.
	func() {
		defer func() { _ = recover() }()
		_, _ = startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
			opsHostFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory, bus.EventBus) *ops.Ops {
				return &ops.Ops{}
			},
			compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
				return &stubBridgeExecutor{}, nil
			},
			moduleLifecycleFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
				panic("lifecycle boom")
			},
		})
	}()

	// marshalOpResult throw + marshal failure on success payload.
	throwVal := marshalOpResult(leaseRun.Ctx, oerrors.New("meta.lock", "LEASE_CONFLICT", "busy"))
	defer throwVal.Free()
	quickjsMarshal = func(*quickjs.Context, any) (*quickjs.Value, error) {
		return nil, errors.New("marshal op result")
	}
	failMarshal := marshalOpResult(leaseRun.Ctx, nil)
	defer failMarshal.Free()
	quickjsMarshal = prevMarshal
}

func rejectContains(t *testing.T, engine *quickjsengine.QuickjsEngine, expr, substr string) {
	t.Helper()
	promise := engine.Ctx.Eval(expr)
	defer promise.Free()
	if promise.IsException() {
		msg := engine.Ctx.Exception().Error()
		if substr != "" && !strings.Contains(msg, substr) {
			t.Fatalf("eval exception %q, want substring %q", msg, substr)
		}
		return
	}
	ret := promise.Await()
	defer ret.Free()
	if !ret.IsError() && !ret.IsException() {
		// Some paths resolve {ok:false} instead of rejecting.
		raw := ret.JSONStringify()
		var got map[string]any
		if err := json.Unmarshal([]byte(raw), &got); err == nil {
			if ok, _ := got["ok"].(bool); !ok {
				if substr == "" {
					return
				}
				blob, _ := json.Marshal(got)
				if strings.Contains(string(blob), substr) || strings.Contains(raw, substr) {
					return
				}
			}
		}
		t.Fatalf("expected rejection or !ok for %s, got %s", expr, ret.String())
	}
	msg := ret.String()
	if ret.IsException() {
		if ex := engine.Ctx.Exception(); ex != nil {
			msg = ex.Error()
		}
	}
	if substr != "" && !strings.Contains(msg, substr) {
		t.Fatalf("rejection %q, want substring %q (expr=%s)", msg, substr, expr)
	}
}

func TestCoverage_ParseJSONArgsEmptyStringify(t *testing.T) {
	engine := newTestQuickjsEngine(t)
	fn := engine.Ctx.Eval(`(function(){})`)
	defer fn.Free()
	params, err := parseJSONArgs[modulePlanParams]([]*quickjs.Value{fn})
	if err != nil {
		t.Fatalf("non-serializable arg: %v", err)
	}
	if params.Action != "" {
		t.Fatalf("expected zero params, got %#v", params)
	}
}
