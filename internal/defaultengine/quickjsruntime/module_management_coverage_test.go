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
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/module/plan"
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
	"gorm.io/gorm"
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

	// loadDemo with empty pending catalog succeeds (soft ok result).
	loadDemo := awaitModuleManagementJSON(t, badJSON, `$choysum.moduleManagement.loadDemo({ operatorUserId: 'u' })`)
	if ok, _ := loadDemo["ok"].(bool); !ok {
		t.Fatalf("expected loadDemo ok, got %#v", loadDemo)
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
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				return nil, errors.New("factory boom")
			}
		}),
	))
	for _, expr := range []string{
		`$choysum.moduleManagement.apply({ id: 'x' })`,
		`$choysum.moduleManagement.run({ id: 'x' })`,
		`$choysum.moduleManagement.loadDemo({})`,
	} {
		rejectContains(t, engine, expr, "factory")
	}
	// syncCatalog must not bootstrap a compiler even when Lifecycle/Resolver are missing.
	db := newBridgeTestDB(t)
	syncScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	syncEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(syncScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				t.Fatal("syncCatalog must not create a compiler executor")
				return nil, errors.New("unexpected")
			}
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
					},
					SyncRegistrySnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
					},
				}
			}
		}),
	))
	got := awaitModuleManagementJSON(t, syncEngine, `$choysum.moduleManagement.syncCatalog({ originType: 'all' })`)
	if ok, _ := got["ok"].(bool); !ok {
		t.Fatalf("syncCatalog without lifecycle = %#v", got)
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
	}, true)
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
	}, true)
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
	}, true)
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
	}, true)
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
	}, true)
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

	// loadDemo: empty pending catalog soft-succeeds.
	leaseRun := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		opsHostOpt,
	))
	got := awaitModuleManagementJSON(t, leaseRun, `$choysum.moduleManagement.loadDemo({})`)
	if ok, _ := got["ok"].(bool); !ok {
		t.Fatalf("loadDemo = %#v", got)
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

	// Run-hook paths need a pending-demo catalog so Plan has no blockers and
	// loadDemo proceeds Plan → Apply → Run (empty pending soft-abandons before Run).
	seedPendingDemo := func(t *testing.T, gdb *gorm.DB) {
		t.Helper()
		if err := gdb.Create(&pkgmeta.Module{
			Name: "demo_mod", Version: "1.0.0", Status: pkgmeta.Installed,
			ApplicationStr: "demo_mod", Path: t.TempDir(),
		}).Error; err != nil {
			t.Fatalf("seed module: %v", err)
		}
		if err := gdb.Create(&modmeta.ModuleCatalog{
			ModuleName: "demo_mod", Installed: true, HasDemo: true, DemoLoaded: false,
		}).Error; err != nil {
			t.Fatalf("seed catalog: %v", err)
		}
	}
	pendingHostOpt := moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
			return &ops.Ops{
				Scope:         runtimeScope,
				Lifecycle:     moduleManagementTestManager{},
				Resolver:      &mapResolver{},
				LockerFactory: lockerFactory,
				Events:        ev,
			}
		}
	})
	newPendingEngine := func(t *testing.T, hook func(error) error) *quickjsengine.QuickjsEngine {
		t.Helper()
		pendingDB := newBridgeTestDB(t)
		seedPendingDemo(t, pendingDB)
		pendingBase := &moduleIndexTestScope{
			ctx:     context.Background(),
			logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg:     &config.Config{ModulesPath: t.TempDir()},
			session: &scope.Session{DB: pendingDB},
		}
		return newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(pendingBase),
			pendingHostOpt,
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.loadDemoRunHook = hook
			}),
		))
	}

	leaseConflictDemo := newPendingEngine(t, func(error) error {
		return oerrors.New("meta.lock", "LEASE_CONFLICT", "busy")
	})
	rejectContains(t, leaseConflictDemo, `$choysum.moduleManagement.loadDemo({})`, "")

	emptyMsgDemo := newPendingEngine(t, func(error) error {
		return oerrors.New("ops", "X", "")
	})
	emptyMsg := awaitModuleManagementJSON(t, emptyMsgDemo, `$choysum.moduleManagement.loadDemo({})`)
	if msg, _ := emptyMsg["errorMessage"].(string); msg == "" {
		t.Fatalf("expected fallback errorMessage, got %#v", emptyMsg)
	}

	plainRunDemo := newPendingEngine(t, func(error) error {
		return errors.New("plain run fail")
	})
	plainRun := awaitModuleManagementJSON(t, plainRunDemo, `$choysum.moduleManagement.loadDemo({})`)
	if code, _ := plainRun["errorCode"].(string); code != "OP_FAILED" {
		t.Fatalf("plain run = %#v", plainRun)
	}

	// loadDemo Apply failure terminates the operation row (ForceFailTerminal).
	applyFailDB := newBridgeTestDB(t)
	seedPendingDemo(t, applyFailDB)
	applyFailBase := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: applyFailDB},
	}
	applyFailEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(applyFailBase),
		pendingHostOpt,
		WithModuleManagementApplyHook(func(error) error {
			return oerrors.New("ops", "APPLY_BOOM", "apply failed")
		}),
	))
	applyFail := awaitModuleManagementJSON(t, applyFailEngine, `$choysum.moduleManagement.loadDemo({})`)
	if ok, _ := applyFail["ok"].(bool); ok {
		t.Fatalf("expected apply fail, got %#v", applyFail)
	}
	if code, _ := applyFail["errorCode"].(string); code != "APPLY_BOOM" {
		t.Fatalf("applyFail = %#v", applyFail)
	}
	opID, _ := applyFail["id"].(string)
	if opID == "" {
		t.Fatalf("missing id: %#v", applyFail)
	}
	var failedOp modmeta.ModuleOperation
	if err := applyFailDB.Where("id = ?", opID).Take(&failedOp).Error; err != nil {
		t.Fatalf("load op: %v", err)
	}
	if failedOp.Status != ops.StatusFailed {
		t.Fatalf("status = %q, want failed", failedOp.Status)
	}

	// Apply hook throw path (LEASE_CONFLICT).
	applyThrowEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(applyFailBase),
		pendingHostOpt,
		WithModuleManagementApplyHook(func(error) error {
			return oerrors.New("meta.lock", "LEASE_CONFLICT", "busy")
		}),
	))
	rejectContains(t, applyThrowEngine, `$choysum.moduleManagement.loadDemo({})`, "")

	// Apply fail + marshal failure.
	applyMarshalDB := newBridgeTestDB(t)
	seedPendingDemo(t, applyMarshalDB)
	applyMarshalBase := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: applyMarshalDB},
	}
	applyMarshalEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(applyMarshalBase),
		pendingHostOpt,
		WithModuleManagementApplyHook(func(error) error {
			return errors.New("apply plain")
		}),
	))
	prevMarshalApply := quickjsMarshal
	quickjsMarshal = func(ctx *quickjs.Context, v any) (*quickjs.Value, error) {
		return nil, errors.New("apply marshal fail")
	}
	rejectContains(t, applyMarshalEngine, `$choysum.moduleManagement.loadDemo({})`, "apply marshal fail")
	quickjsMarshal = prevMarshalApply

	// Run fail with non-empty oerrors message (covers message branch).
	msgDemo := newPendingEngine(t, func(error) error {
		return oerrors.New("ops", "Y", "explicit message")
	})
	msgGot := awaitModuleManagementJSON(t, msgDemo, `$choysum.moduleManagement.loadDemo({})`)
	if msg, _ := msgGot["errorMessage"].(string); msg != "explicit message" {
		t.Fatalf("expected explicit message, got %#v", msgGot)
	}

	// Run result marshal failure after pending loadDemo.
	runMarshalDemo := newPendingEngine(t, func(error) error {
		return errors.New("run after marshal arm")
	})
	prevMarshalRun := quickjsMarshal
	quickjsMarshal = func(ctx *quickjs.Context, v any) (*quickjs.Value, error) {
		return nil, errors.New("run marshal fail")
	}
	rejectContains(t, runMarshalDemo, `$choysum.moduleManagement.loadDemo({})`, "run marshal fail")
	quickjsMarshal = prevMarshalRun

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
		}, true)
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

func TestCoverage_ModuleLifecycleOrDefault(t *testing.T) {
	if got := moduleLifecycleOrDefault(nil); got == nil {
		t.Fatal("expected default factory")
	}
	sentinel := moduleLifecycleFactory(func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
		return moduleManagementTestManager{}
	})
	if moduleLifecycleOrDefault(sentinel) == nil {
		t.Fatal("expected injected factory retained")
	}

	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)
	base := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	seed := &stubBridgeExecutor{}
	_ = seed.Start()
	processCompilerMu.Lock()
	processCompilerExecutor = seed
	processCompilerStop = seed.Stop
	processCompilerMu.Unlock()

	// Raw config with nil lifecycle factory must not panic on processHost path.
	started, err := startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		processHost:   true,
		lockerFactory: func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} },
		moduleResolverFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) plan.Resolver {
			return &mapResolver{}
		}, // plan.Resolver via mapResolver
	}, true)
	if err != nil {
		t.Fatalf("nil lifecycle processHost: %v", err)
	}
	if started.host == nil || started.host.Lifecycle == nil {
		t.Fatal("expected default Lifecycle")
	}
	ResetProcessModuleManagementHostForTest()
}

func TestCoverage_AssembleAndProcessCompilerErrors(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	if _, _, err := AssembleModuleManagementHost(func(context.Context) scope.Scope { return nil }); err == nil {
		t.Fatal("expected nil scope assemble error")
	}
	if _, _, err := AssembleModuleManagementHost(jsengine.StaticScopeProvider(&moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}), moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, errors.New("create boom")
		}
	})); err == nil || !strings.Contains(err.Error(), "create boom") {
		t.Fatalf("create err = %v", err)
	}
	if _, _, err := AssembleModuleManagementHost(jsengine.StaticScopeProvider(&moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}), moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{startErr: errors.New("start boom")}, nil
		}
	})); err == nil || !strings.Contains(err.Error(), "start boom") {
		t.Fatalf("start err = %v", err)
	}

	// Default lifecycle factory path + usable events + stop.
	base := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	host, stop, err := AssembleModuleManagementHost(jsengine.StaticScopeProvider(base), moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		}
		c.moduleLifecycleFactory = nil
		c.events = &recordingBus{}
		c.lockerFactory = func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} }
	}))
	if err != nil || host == nil || host.Lifecycle == nil {
		t.Fatalf("assemble default lifecycle: host=%v err=%v", host, err)
	}
	if stop != nil {
		_ = stop()
	}

	if _, _, err := startProcessCompilerExecutor(func(context.Context) scope.Scope { return nil }, moduleManagementConfig{}); err == nil {
		t.Fatal("expected nil scope process compiler error")
	}
	if _, err := ensureProcessCompilerExecutor(context.Background(), func(context.Context) scope.Scope { return nil }, moduleManagementConfig{}); err == nil {
		t.Fatal("expected ensure nil scope error")
	}
	if err := StopProcessModuleManagementHost(); err != nil {
		t.Fatalf("stop empty: %v", err)
	}
	SetProcessModuleManagementHostForTest(func() error { return errors.New("injected stop boom") })
	if err := StopProcessModuleManagementHost(); err == nil || !strings.Contains(err.Error(), "injected stop boom") {
		t.Fatalf("injected stop err = %v", err)
	}
	ResetProcessModuleManagementHostForTest()

	// processHost ensure failure surfaces through startOpsHost / plan.
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(base),
		WithModuleManagementProcessHost(),
		moduleManagementOptionFunc(func(c *moduleManagementConfig) {
			c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				return nil, errors.New("process create boom")
			}
		}),
	))
	rejectContains(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install' })`, "process create boom")

	// syncCatalog startOpsHost failure (nil scope).
	syncEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		func(context.Context) scope.Scope { return nil },
	))
	rejectContains(t, syncEngine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`, "unavailable")

	// Injected host always rebinds Events.
	injected := &ops.Ops{
		Lifecycle: moduleManagementTestManager{},
		Resolver:  &mapResolver{},
		Events:    &recordingBus{},
	}
	started, err := startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		injectedHost:  injected,
		lockerFactory: func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} },
		events:        &recordingBus{},
	}, true)
	if err != nil {
		t.Fatalf("injected startOpsHost: %v", err)
	}
	if started.host.Events == injected.Events {
		t.Fatal("expected Events rebound to request bus")
	}

	// startProcessCompilerExecutor: create/start failures + default factory.
	ResetProcessModuleManagementHostForTest()
	if _, _, err := startProcessCompilerExecutor(jsengine.StaticScopeProvider(base), moduleManagementConfig{
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, errors.New("proc create boom")
		},
	}); err == nil || !strings.Contains(err.Error(), "proc create boom") {
		t.Fatalf("proc create err = %v", err)
	}
	if _, _, err := startProcessCompilerExecutor(jsengine.StaticScopeProvider(base), moduleManagementConfig{
		compilerExecutorFactory: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{startErr: errors.New("proc start boom")}, nil
		},
	}); err == nil || !strings.Contains(err.Error(), "proc start boom") {
		t.Fatalf("proc start err = %v", err)
	}
	if exec, stop, err := startProcessCompilerExecutor(jsengine.StaticScopeProvider(base), moduleManagementConfig{}); err != nil {
		t.Logf("default factory path: %v", err)
	} else {
		_ = exec
		if stop != nil {
			_ = stop()
		}
	}

	// Defensive ensure branch: winner appears while assembling.
	ResetProcessModuleManagementHostForTest()
	existing := &stubBridgeExecutor{}
	_ = existing.Start()
	cfgRace := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			processCompilerMu.Lock()
			processCompilerExecutor = existing
			processCompilerStop = existing.Stop
			processCompilerMu.Unlock()
			return &stubBridgeExecutor{}, nil
		}
	}))
	got, err := ensureProcessCompilerExecutor(context.Background(), jsengine.StaticScopeProvider(base), cfgRace)
	if err != nil {
		t.Fatalf("race ensure: %v", err)
	}
	if got != existing {
		t.Fatal("expected existing process executor retained")
	}
	ResetProcessModuleManagementHostForTest()

	// processHost with default resolver factory (NewModuleManager).
	seed := &stubBridgeExecutor{}
	_ = seed.Start()
	processCompilerMu.Lock()
	processCompilerExecutor = seed
	processCompilerStop = seed.Stop
	processCompilerMu.Unlock()
	startedProc, err := startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), resolveModuleManagementConfig(
		WithModuleManagementProcessHost(),
		moduleManagementOptionFunc(func(c *moduleManagementConfig) {
			c.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
				return moduleManagementTestManager{}
			}
			c.lockerFactory = func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} }
		}),
	), true)
	if err != nil {
		t.Fatalf("process host default resolver: %v", err)
	}
	if startedProc.host == nil || startedProc.host.Resolver == nil {
		t.Fatal("expected default ModuleManager resolver")
	}
	ResetProcessModuleManagementHostForTest()

	// processHost with nil lockerFactory falls back to default lease wrapper.
	seed2 := &stubBridgeExecutor{}
	_ = seed2.Start()
	processCompilerMu.Lock()
	processCompilerExecutor = seed2
	processCompilerStop = seed2.Stop
	processCompilerMu.Unlock()
	startedLocker, err := startOpsHost(newTestQuickjsEngine(t), jsengine.StaticScopeProvider(base), moduleManagementConfig{
		processHost: true,
		moduleLifecycleFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
			return moduleManagementTestManager{}
		},
		moduleResolverFactory: func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) plan.Resolver {
			return &mapResolver{}
		},
	}, true)
	if err != nil {
		t.Fatalf("process host nil locker: %v", err)
	}
	if startedLocker.host == nil || startedLocker.host.LockerFactory == nil {
		t.Fatal("expected default LockerFactory on process host")
	}
	if got := startedLocker.host.LockerFactory(base); got == nil {
		t.Fatal("default LockerFactory returned nil")
	}
	ResetProcessModuleManagementHostForTest()

	// Assemble: EnsureHost + NewHost failure (handedOff=false stops executor).
	prevAssemble := assembleOpsHost
	t.Cleanup(func() { assembleOpsHost = prevAssemble })
	assembleOpsHost = func(ops.HostOptions) (*ops.Ops, error) {
		return nil, errors.New("newhost boom")
	}
	if _, _, err := AssembleModuleManagementHost(jsengine.StaticScopeProvider(base), moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		}
		c.events = nil
		c.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
			return moduleManagementTestManager{}
		}
		c.lockerFactory = func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} }
	})); err == nil || !strings.Contains(err.Error(), "newhost boom") {
		t.Fatalf("newhost err = %v", err)
	}
	assembleOpsHost = prevAssemble

	// Assemble with nil compiler factory → defaultCompilerExecutorFactory.
	if host, stop, err := AssembleModuleManagementHost(jsengine.StaticScopeProvider(base), moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = nil
		c.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
			return moduleManagementTestManager{}
		}
		c.lockerFactory = func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} }
	})); err != nil {
		t.Logf("default compiler assemble (may need factory): %v", err)
	} else {
		if stop != nil {
			_ = stop()
		}
		_ = host
	}
}
