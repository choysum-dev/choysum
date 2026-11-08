// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type moduleIndexTestScope struct {
	ctx        context.Context
	logger     *slog.Logger
	cfg        *config.Config
	session    *scope.Session
	transactor scope.Transactor
}

func (e *moduleIndexTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *moduleIndexTestScope) Transactor() scope.Transactor {
	if e.transactor != nil {
		return e.transactor
	}
	return scopetest.NewPassthroughTransactor(e)
}
func (e *moduleIndexTestScope) Session() *scope.Session { return e.session }
func (e *moduleIndexTestScope) WithContext(ctx context.Context) scope.Scope {
	cp := *e
	cp.ctx = ctx
	return &cp
}
func (e *moduleIndexTestScope) Context() context.Context { return e.ctx }
func (e *moduleIndexTestScope) Logger() *slog.Logger     { return e.logger }
func (e *moduleIndexTestScope) Config() *config.Config   { return e.cfg }
func (e *moduleIndexTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

type moduleIndexCountingTransactor struct {
	inner         scope.Transactor
	requiredCalls int
}

func (t *moduleIndexCountingTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	if t.inner == nil {
		return scope.ErrTransactorUnavailable
	}
	return t.inner.Do(ctx, opts, fn)
}
func (t *moduleIndexCountingTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	t.requiredCalls++
	if t.inner == nil {
		return scope.ErrTransactorUnavailable
	}
	return t.inner.Required(ctx, fn)
}
func (t *moduleIndexCountingTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	if t.inner == nil {
		return scope.ErrTransactorUnavailable
	}
	return t.inner.RequiresNew(ctx, fn)
}
func (t *moduleIndexCountingTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	if t.inner == nil {
		return scope.ErrTransactorUnavailable
	}
	return t.inner.Nested(ctx, fn)
}

type moduleManagementTestManager struct {
	install func(context.Context, string) error
}

type moduleManagementContextTransaction struct{}

func (moduleManagementContextTransaction) Context() context.Context { return context.Background() }
func (moduleManagementContextTransaction) Session() *scope.Session  { return nil }
func (moduleManagementContextTransaction) Savepoint(string) error   { return nil }
func (moduleManagementContextTransaction) RollbackToSavepoint(string) error {
	return nil
}
func (moduleManagementContextTransaction) ReleaseSavepoint(string) error { return nil }

func (m moduleManagementTestManager) Install(ctx context.Context, req lifecycle.InstallRequest) error {
	if m.install != nil {
		return m.install(ctx, req.Name)
	}
	return nil
}
func (moduleManagementTestManager) Upgrade(context.Context, lifecycle.UpgradeRequest) error {
	return nil
}
func (moduleManagementTestManager) Uninstall(context.Context, lifecycle.UninstallRequest) error {
	return nil
}
func (moduleManagementTestManager) SchemaPlan(context.Context, string) (schema.SchemaPlan, error) {
	return schema.SchemaPlan{}, nil
}

type mapResolver struct {
	peek map[string]*pkgmeta.Module
}

func (r *mapResolver) Peek(_ context.Context, name string) (*pkgmeta.Module, error) {
	if r.peek == nil {
		return nil, nil
	}
	return r.peek[name], nil
}
func (r *mapResolver) Load(string) (*pkgmeta.Module, error) { return nil, nil }

type recordingBus struct {
	events []bus.Event
}

func (b *recordingBus) Publish(_ context.Context, ev bus.Event) error {
	b.events = append(b.events, ev)
	return nil
}
func (b *recordingBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return nopSub{}, nil
}

type nopSub struct{}

func (nopSub) Close() error { return nil }

type moduleIndexTestLocker struct {
	acquireErr error
	acquired   int
	renewed    int
	released   int
}

func (l *moduleIndexTestLocker) Acquire(context.Context, string, string, time.Duration) error {
	l.acquired++
	return l.acquireErr
}
func (l *moduleIndexTestLocker) Renew(context.Context, string, string, time.Duration) error {
	l.renewed++
	return nil
}
func (l *moduleIndexTestLocker) Release(context.Context, string, string) error {
	l.released++
	return nil
}

func newBridgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bridge.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&modmeta.ModuleOperation{},
		&modmeta.ModuleCatalog{},
		&modmeta.ModuleSourceSnapshot{},
		&pkgmeta.Module{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSanitizeModuleIndexError_PathErrorRedactsPath(t *testing.T) {
	modulesPath := "/tmp/choysum/modules"
	runtimeScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.Default(),
		cfg:    &config.Config{ModulesPath: modulesPath},
	}
	err := &os.PathError{Op: "open", Path: modulesPath + "/meta/manifest.json", Err: os.ErrNotExist}
	got := lifecycle.SanitizeModuleIndexError(runtimeScope, err)
	if strings.Contains(got, modulesPath) {
		t.Fatalf("expected redacted path, got %q", got)
	}
	if got != "open manifest.json" {
		t.Fatalf("expected basename only, got %q", got)
	}
}

func TestSanitizeModuleIndexError_RedactsModulesPathInMessage(t *testing.T) {
	modulesPath := "/tmp/choysum/modules"
	runtimeScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.Default(),
		cfg:    &config.Config{ModulesPath: modulesPath},
	}
	err := errors.New("failed to read " + modulesPath + "/meta/manifest.json")
	got := lifecycle.SanitizeModuleIndexError(runtimeScope, err)
	if strings.Contains(got, modulesPath) {
		t.Fatalf("expected redacted path, got %q", got)
	}
	if !strings.Contains(got, "<modulesPath>") {
		t.Fatalf("expected <modulesPath> placeholder, got %q", got)
	}
}

func TestSanitizeModuleIndexError_NilReturnsDefault(t *testing.T) {
	got := lifecycle.SanitizeModuleIndexError(nil, nil)
	if got != "package.json parsing failed" {
		t.Fatalf("expected default message, got %q", got)
	}
}

func TestNormalizeModuleCatalogOriginType(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "all"},
		{"all", "all"},
		{"registry", "registry"},
		{"  LOCAL ", "local"},
		{"remote", ""},
	}
	for _, tt := range tests {
		if got := normalizeModuleCatalogOriginType(tt.in); got != tt.want {
			t.Fatalf("normalizeModuleCatalogOriginType(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveModuleManagementConfigLockerFactory(t *testing.T) {
	sentinel := &moduleIndexTestLocker{}
	called := false
	cfg := resolveModuleManagementConfig(WithModuleManagementLockerFactory(func(scope.Scope) statepkg.Locker {
		return sentinel
	}))
	cfg.moduleLifecycleFactory = func(_ scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) lifecycle.Service {
		called = true
		if got := lockerFactory(nil); !reflect.DeepEqual(got, sentinel) {
			t.Fatalf("lockerFactory() = %#v, want %#v", got, sentinel)
		}
		return moduleManagementTestManager{}
	}
	_ = cfg.moduleLifecycleFactory(nil, nil, cfg.lockerFactory)
	if !called {
		t.Fatal("expected lifecycle factory call")
	}
}

func TestModuleManagementExportsPlanApplyRunNotInstall(t *testing.T) {
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(&moduleIndexTestScope{
			ctx:    context.Background(),
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg:    &config.Config{ModulesPath: t.TempDir()},
		}),
	))
	keys := evalString(t, engine, `(function() {
		const mm = $choysum.moduleManagement;
		return JSON.stringify({
			hasPlan: typeof mm.plan === 'function',
			hasApply: typeof mm.apply === 'function',
			hasRun: typeof mm.run === 'function',
			hasSyncCatalog: typeof mm.syncCatalog === 'function',
			hasLoadDemo: typeof mm.loadDemo === 'function',
			hasInstall: typeof mm.install === 'function',
			hasUninstall: typeof mm.uninstall === 'function',
			hasUpgrade: typeof mm.upgrade === 'function',
			hasSyncIndex: typeof mm.syncIndex === 'function',
		});
	})()`)
	var got map[string]bool
	if err := json.Unmarshal([]byte(keys), &got); err != nil {
		t.Fatalf("decode: %v (%s)", err, keys)
	}
	for _, key := range []string{"hasPlan", "hasApply", "hasRun", "hasSyncCatalog", "hasLoadDemo"} {
		if !got[key] {
			t.Fatalf("%s = false, want true (%s)", key, keys)
		}
	}
	for _, key := range []string{"hasInstall", "hasUninstall", "hasUpgrade", "hasSyncIndex"} {
		if got[key] {
			t.Fatalf("%s = true, want false (%s)", key, keys)
		}
	}
}

func TestModuleManagementPlanApplyRunPublishesTip(t *testing.T) {
	db := newBridgeTestDB(t)
	events := &recordingBus{}
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	var runCtx context.Context
	life := moduleManagementTestManager{install: func(ctx context.Context, name string) error {
		runCtx = ctx
		if name != "core" {
			return errors.New("unexpected module")
		}
		return nil
	}}
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		WithModuleManagementEvents(events),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
				return life
			}
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

	type ctxKey struct{}
	runtimeCtx := context.WithValue(context.Background(), ctxKey{}, "runtime")
	restore := engine.SwapExecContext(runtimeCtx)
	defer restore()

	planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local', operatorUserId: 'u1' })`)
	planID, _ := planJSON["id"].(string)
	if planID == "" {
		t.Fatalf("plan result = %#v", planJSON)
	}
	if order, _ := planJSON["moduleOrder"].([]any); len(order) == 0 {
		t.Fatalf("expected moduleOrder, got %#v", planJSON)
	}

	applyJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
	if ok, _ := applyJSON["ok"].(bool); !ok {
		t.Fatalf("apply = %#v", applyJSON)
	}

	runJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)
	if ok, _ := runJSON["ok"].(bool); !ok {
		t.Fatalf("run = %#v", runJSON)
	}
	if runCtx == nil || runCtx.Value(ctxKey{}) != "runtime" {
		t.Fatalf("run ctx = %#v", runCtx)
	}

	var row modmeta.ModuleOperation
	if err := db.Where("id = ?", planID).Take(&row).Error; err != nil {
		t.Fatalf("load operation: %v", err)
	}
	if row.Status != ops.StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", row.Status)
	}
	if len(events.events) == 0 {
		t.Fatal("expected tip Publish")
	}
	last := events.events[len(events.events)-1]
	if last.Topic != bus.TopicMetaModuleOpChanged {
		t.Fatalf("topic = %q", last.Topic)
	}
	if last.Payload["operationId"] != planID {
		t.Fatalf("payload = %#v", last.Payload)
	}
}

func TestModuleManagementApplyRunHooksForceFailureSeams(t *testing.T) {
	// Former CHOYSUM_E2E_FORCE_* product env seams now live as test-only bridge hooks.
	db := newBridgeTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	life := moduleManagementTestManager{install: func(context.Context, string) error { return nil }}
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}

	t.Run("lock-conflict via apply hook", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementApplyHook(func(error) error {
				return oerrors.New("meta.lock", "LEASE_CONFLICT", "lease is busy")
			}),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		// LEASE_CONFLICT is thrown (not ok:false) so the client surfaces a hard failure.
		rejectContains(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`, "LEASE_CONFLICT")
	})

	t.Run("result-failed via run hook", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementRunHook(func(error) error {
				return oerrors.New("MODULE_MANAGEMENT", "OP_FAILED", "E2E forced failure")
			}),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		applyJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
		if ok, _ := applyJSON["ok"].(bool); !ok {
			t.Fatalf("apply = %#v", applyJSON)
		}
		runJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)
		if ok, _ := runJSON["ok"].(bool); ok {
			t.Fatalf("run should fail, got %#v", runJSON)
		}
		if code, _ := runJSON["errorCode"].(string); code != "OP_FAILED" {
			t.Fatalf("errorCode = %#v, want OP_FAILED", runJSON)
		}
		var row modmeta.ModuleOperation
		if err := db.Where("id = ?", planID).Take(&row).Error; err != nil {
			t.Fatalf("load forced op: %v", err)
		}
		if row.Status != ops.StatusFailed || row.ErrorCode != "OP_FAILED" {
			t.Fatalf("forced op row = %#v", row)
		}
	})

	t.Run("result-failed run hook with plain error uses defaults", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementRunHook(func(error) error {
				return errors.New("plain forced failure")
			}),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
		runJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)
		if ok, _ := runJSON["ok"].(bool); ok {
			t.Fatalf("run should fail, got %#v", runJSON)
		}
	})

	t.Run("result-failed run hook with empty oerrors fields", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementRunHook(func(error) error {
				return oerrors.New("", "", "")
			}),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
		runJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)
		if ok, _ := runJSON["ok"].(bool); ok {
			t.Fatalf("run should fail, got %#v", runJSON)
		}
	})

	t.Run("reload-failed after successful run", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementForceReloadFailed(),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
		runJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)
		if ok, _ := runJSON["ok"].(bool); !ok {
			t.Fatalf("run = %#v", runJSON)
		}
		var row modmeta.ModuleOperation
		if err := db.Where("id = ?", planID).Take(&row).Error; err != nil {
			t.Fatalf("load reload-failed op: %v", err)
		}
		if !row.ReloadTriggered || !row.ReloadFailed {
			t.Fatalf("reload flags = triggered=%v failed=%v", row.ReloadTriggered, row.ReloadFailed)
		}
	})

	t.Run("result-failed ForceFailTerminal load error is joined", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementRunHook(func(error) error {
				if err := db.Where("1 = 1").Delete(&modmeta.ModuleOperation{}).Error; err != nil {
					t.Fatalf("delete ops: %v", err)
				}
				return oerrors.New("MODULE_MANAGEMENT", "OP_FAILED", "E2E forced failure")
			}),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
		rejectContains(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`, "OP_FAILED")
	})

	t.Run("reload-failed MarkReloadFailed error after row deleted", func(t *testing.T) {
		engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
			jsengine.StaticScopeProvider(baseScope),
			WithModuleManagementForceReloadFailed(),
			WithModuleManagementRunHook(func(err error) error {
				if err != nil {
					return err
				}
				if delErr := db.Where("1 = 1").Delete(&modmeta.ModuleOperation{}).Error; delErr != nil {
					t.Fatalf("delete ops: %v", delErr)
				}
				return nil
			}),
			moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
				cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
					return life
				}
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
		planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install', origin: 'local' })`)
		planID, _ := planJSON["id"].(string)
		_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
		rejectContains(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`, "not found")
	})
}

func TestModuleManagementPlanApplyRunSkipsOuterTransaction(t *testing.T) {
	db := newBridgeTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	var sawTx bool
	life := moduleManagementTestManager{install: func(ctx context.Context, _ string) error {
		_, sawTx = scope.TransactionFromContext(ctx)
		return nil
	}}
	counting := &moduleIndexCountingTransactor{}
	baseScope := &moduleIndexTestScope{
		ctx:        context.Background(),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:        &config.Config{ModulesPath: t.TempDir()},
		session:    &scope.Session{DB: db},
		transactor: counting,
	}
	counting.inner = scopetest.NewPassthroughTransactor(baseScope)

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
	runtimeCtx := scope.ContextWithTransaction(context.Background(), moduleManagementContextTransaction{})
	restore := engine.SwapExecContext(runtimeCtx)
	defer restore()

	planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install' })`)
	id, _ := planJSON["id"].(string)
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(id)+` })`)
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(id)+` })`)
	if sawTx {
		t.Fatal("run should strip outer transaction")
	}
	if counting.requiredCalls != 0 {
		t.Fatalf("transactor Required calls = %d, want 0", counting.requiredCalls)
	}
}

func TestModuleManagementSyncCatalogLeaseConflict(t *testing.T) {
	db := newBridgeTestDB(t)
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	locker := &moduleIndexTestLocker{acquireErr: statepkg.ErrLeaseBusy}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		WithModuleManagementLockerFactory(func(scope.Scope) statepkg.Locker {
			return locker
		}),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					Lifecycle:     moduleManagementTestManager{},
					Resolver:      &mapResolver{},
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(ctx context.Context, runtimeScope scope.Scope, factory statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						err := factory(runtimeScope).Acquire(ctx, "meta.module_source_snapshot.sync:local", "holder", time.Minute)
						if errors.Is(err, statepkg.ErrLeaseBusy) {
							return lifecycle.ModuleSourceSnapshotSyncStats{}, oerrors.New("meta.lock", "LEASE_CONFLICT", "lease is busy")
						}
						return lifecycle.ModuleSourceSnapshotSyncStats{}, err
					},
				}
			}
		}),
	))
	promise := engine.Ctx.Eval(`$choysum.moduleManagement.syncCatalog({ originType: 'local' })`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("syncCatalog eval exception: %v", engine.Ctx.Exception())
	}
	ret := promise.Await()
	defer ret.Free()
	if locker.acquired != 1 {
		t.Fatalf("locker Acquire calls = %d, want 1", locker.acquired)
	}
	if !ret.IsError() && !ret.IsException() {
		t.Fatalf("expected lease-conflict rejection, got %s", ret.String())
	}
	msg := ret.String()
	if ret.IsException() {
		msg = engine.Ctx.Exception().Error()
	}
	if !strings.Contains(msg, "LEASE_CONFLICT") {
		t.Fatalf("expected LEASE_CONFLICT, got %q", msg)
	}
}

func TestModuleManagementLoadDemoNotImplemented(t *testing.T) {
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
				}
			}
		}),
	))
	got := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.loadDemo({})`)
	if ok, _ := got["ok"].(bool); ok {
		t.Fatalf("expected not ok, got %#v", got)
	}
	if code, _ := got["errorCode"].(string); code != "LOAD_DEMO_NOT_IMPLEMENTED" {
		t.Fatalf("errorCode = %#v", got)
	}
}

func TestModuleManagementInjectedHostPlanApplyRunStartsAtMostOnce(t *testing.T) {
	db := newBridgeTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	life := moduleManagementTestManager{install: func(context.Context, string) error { return nil }}
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	counting := &stubBridgeExecutor{}
	if err := counting.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	host, err := ops.NewHost(ops.HostOptions{
		Scope:     baseScope,
		Executor:  counting,
		Lifecycle: life,
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &moduleIndexTestLocker{}
		},
		Events: bus.EnsureHost(baseScope),
	})
	if err != nil {
		t.Fatalf("NewHost: %v", err)
	}

	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		WithModuleManagementHost(host),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				t.Fatal("injected host must not create a compiler executor")
				return nil, errors.New("unexpected compiler create")
			}
		}),
	))

	planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install' })`)
	planID, _ := planJSON["id"].(string)
	if planID == "" {
		t.Fatalf("plan result = %#v", planJSON)
	}
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)

	if counting.started > 1 {
		t.Fatalf("executor Start calls = %d, want ≤ 1", counting.started)
	}
}

func TestModuleManagementSyncCatalogNeverStartsCompiler(t *testing.T) {
	db := newBridgeTestDB(t)
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	counting := &stubBridgeExecutor{}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				return counting, nil
			}
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				// Lifecycle/Resolver intentionally nil — sync must still skip compiler Start.
				return &ops.Ops{
					Scope:         runtimeScope,
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{Total: 1, Success: 1}, nil
					},
					SyncRegistrySnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
					},
				}
			}
		}),
	))

	got := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`)
	if ok, _ := got["ok"].(bool); !ok {
		t.Fatalf("syncCatalog = %#v", got)
	}
	if counting.started != 0 {
		t.Fatalf("executor Start calls = %d, want 0", counting.started)
	}
}

func TestAssembleModuleManagementHostStartsOnce(t *testing.T) {
	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	counting := &stubBridgeExecutor{}
	host, stop, err := AssembleModuleManagementHost(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				return counting, nil
			}
			cfg.moduleLifecycleFactory = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) lifecycle.Service {
				return moduleManagementTestManager{}
			}
			cfg.lockerFactory = func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} }
		}),
	)
	if err != nil {
		t.Fatalf("AssembleModuleManagementHost: %v", err)
	}
	t.Cleanup(func() {
		if stop != nil {
			_ = stop()
		}
	})
	if host == nil || host.Lifecycle == nil || host.Resolver == nil {
		t.Fatal("expected assembled host with Lifecycle and Resolver")
	}
	if counting.started != 1 {
		t.Fatalf("executor Start calls = %d, want 1", counting.started)
	}
}

func TestEnsureProcessCompilerExecutorStartsOnce(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	counting := &stubBridgeExecutor{}
	cfg := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return counting, nil
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)
	for i := 0; i < 3; i++ {
		exec, err := ensureProcessCompilerExecutor(context.Background(), provider, cfg)
		if err != nil {
			t.Fatalf("ensure[%d]: %v", i, err)
		}
		if exec == nil {
			t.Fatalf("ensure[%d]: nil executor", i)
		}
	}
	if counting.started != 1 {
		t.Fatalf("executor Start calls = %d, want 1", counting.started)
	}
}

func TestEnsureProcessCompilerExecutorConcurrentAndRetry(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	var (
		mu       sync.Mutex
		starts   int
		attempts int
	)
	entered := make(chan struct{})
	gate := make(chan struct{})
	cfg := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			mu.Lock()
			attempts++
			n := attempts
			mu.Unlock()
			if n == 1 {
				close(entered)
				<-gate
				return nil, errors.New("transient boom")
			}
			exec := &stubBridgeExecutor{}
			mu.Lock()
			starts++
			mu.Unlock()
			return exec, nil
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)

	errCh := make(chan error, 1)
	go func() {
		_, err := ensureProcessCompilerExecutor(context.Background(), provider, cfg)
		errCh <- err
	}()
	<-entered

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ensureProcessCompilerExecutor(context.Background(), provider, cfg)
		}()
	}
	// Waiters park on assembling; then release the failing first attempt.
	time.Sleep(20 * time.Millisecond)
	close(gate)
	firstErr := <-errCh
	if firstErr == nil || !strings.Contains(firstErr.Error(), "transient boom") {
		t.Fatalf("first err = %v", firstErr)
	}
	wg.Wait()

	exec, err := ensureProcessCompilerExecutor(context.Background(), provider, cfg)
	if err != nil || exec == nil {
		t.Fatalf("retry after transient: exec=%v err=%v", exec, err)
	}
	mu.Lock()
	gotStarts := starts
	mu.Unlock()
	if gotStarts != 1 {
		t.Fatalf("successful Start assemblies = %d, want 1", gotStarts)
	}
}

func TestEnsureProcessCompilerExecutorCancelWhileWaiting(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	cfg := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			close(entered)
			<-release
			return &stubBridgeExecutor{}, nil
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)

	assemblerDone := make(chan struct{})
	go func() {
		defer close(assemblerDone)
		_, _ = ensureProcessCompilerExecutor(context.Background(), provider, cfg)
	}()
	<-entered

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := ensureProcessCompilerExecutor(ctx, provider, cfg)
		errCh <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	err := <-errCh
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("wait cancel err = %v", err)
	}
	processCompilerMu.Lock()
	leaked := len(processCompilerWaiters)
	processCompilerMu.Unlock()
	if leaked != 0 {
		t.Fatalf("canceled waiter left %d entries in processCompilerWaiters", leaked)
	}
	close(release)
	<-assemblerDone // prevent late store into processCompilerExecutor after Reset
}

func TestEnsureProcessCompilerExecutorStopDuringAssembly(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	cfg := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			close(entered)
			<-release
			return &stubBridgeExecutor{}, nil
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)

	errCh := make(chan error, 1)
	go func() {
		_, err := ensureProcessCompilerExecutor(context.Background(), provider, cfg)
		errCh <- err
	}()
	<-entered

	// Park a waiter; Stop must wake it promptly (not wait for the hung factory).
	waiterDone := make(chan error, 1)
	go func() {
		_, err := ensureProcessCompilerExecutor(context.Background(), provider, cfg)
		waiterDone <- err
	}()
	time.Sleep(20 * time.Millisecond)

	if err := StopProcessModuleManagementHost(); err != nil {
		t.Fatalf("stop during assembly: %v", err)
	}
	select {
	case err := <-waiterDone:
		if err == nil || !strings.Contains(err.Error(), "stopped") {
			t.Fatalf("waiter after stop = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter blocked after Stop during assembly")
	}
	close(release)
	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "stopped during assembly") {
		t.Fatalf("assemble after stop = %v", err)
	}
}

func TestEnsureProcessCompilerExecutorNilContext(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	counting := &stubBridgeExecutor{}
	cfg := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return counting, nil
		}
	}))
	exec, err := ensureProcessCompilerExecutor(nil, jsengine.StaticScopeProvider(baseScope), cfg) //nolint:staticcheck // nil ctx is the coverage target
	if err != nil || exec == nil {
		t.Fatalf("nil ctx ensure: exec=%v err=%v", exec, err)
	}
	if counting.started != 1 {
		t.Fatalf("Start calls = %d, want 1", counting.started)
	}
}

func TestEnsureProcessCompilerExecutorStopThenFactoryErrorRecovers(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	cfgFail := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			close(entered)
			<-release
			return nil, errors.New("factory boom after stop")
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)

	errCh := make(chan error, 1)
	go func() {
		_, err := ensureProcessCompilerExecutor(context.Background(), provider, cfgFail)
		errCh <- err
	}()
	<-entered
	if err := StopProcessModuleManagementHost(); err != nil {
		t.Fatalf("stop during assembly: %v", err)
	}
	close(release)
	if err := <-errCh; err == nil || !strings.Contains(err.Error(), "factory boom after stop") {
		t.Fatalf("assemble after stop+error = %v", err)
	}

	// stopRequested must not stick after the failed attempt, or reload cannot recover.
	cfgOK := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		}
	}))
	exec, err := ensureProcessCompilerExecutor(context.Background(), provider, cfgOK)
	if err != nil || exec == nil {
		t.Fatalf("ensure after stop+factory error: exec=%v err=%v", exec, err)
	}
}

func TestEnsureProcessCompilerExecutorStopThenPanicRecovers(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	cfgPanic := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			close(entered)
			<-release
			panic("factory panic after stop")
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		_, _ = ensureProcessCompilerExecutor(context.Background(), provider, cfgPanic)
	}()
	<-entered
	if err := StopProcessModuleManagementHost(); err != nil {
		t.Fatalf("stop during assembly: %v", err)
	}
	close(release)
	<-done

	cfgOK := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		}
	}))
	exec, err := ensureProcessCompilerExecutor(context.Background(), provider, cfgOK)
	if err != nil || exec == nil {
		t.Fatalf("ensure after stop+panic: exec=%v err=%v", exec, err)
	}
}

func TestEnsureProcessCompilerExecutorPanicReleasesWaiters(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	cfg := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			panic("factory boom")
		}
	}))
	provider := jsengine.StaticScopeProvider(baseScope)

	func() {
		defer func() { _ = recover() }()
		_, _ = ensureProcessCompilerExecutor(context.Background(), provider, cfg)
	}()

	// After panic cleanup, a subsequent ensure with a healthy factory must proceed.
	cfgOK := resolveModuleManagementConfig(moduleManagementOptionFunc(func(c *moduleManagementConfig) {
		c.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &stubBridgeExecutor{}, nil
		}
	}))
	exec, err := ensureProcessCompilerExecutor(context.Background(), provider, cfgOK)
	if err != nil || exec == nil {
		t.Fatalf("after panic ensure: exec=%v err=%v", exec, err)
	}
}

func TestModuleManagementProcessHostPlanApplyRunAndSync(t *testing.T) {
	ResetProcessModuleManagementHostForTest()
	t.Cleanup(ResetProcessModuleManagementHostForTest)

	db := newBridgeTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	var sawScopes []scope.Scope
	life := moduleManagementTestManager{install: func(context.Context, string) error { return nil }}
	baseScope := &moduleIndexTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
	counting := &stubBridgeExecutor{}
	var factoryCalls int
	events := &recordingBus{}
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		WithModuleManagementProcessHost(),
		WithModuleManagementEvents(events),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				factoryCalls++
				return counting, nil
			}
			cfg.moduleLifecycleFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, _ statepkg.LockerFactory) lifecycle.Service {
				sawScopes = append(sawScopes, runtimeScope)
				return life
			}
			cfg.moduleResolverFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, _ statepkg.LockerFactory) plan.Resolver {
				sawScopes = append(sawScopes, runtimeScope)
				return &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}}
			}
			cfg.lockerFactory = func(scope.Scope) statepkg.Locker { return &moduleIndexTestLocker{} }
		}),
	))

	planJSON := awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.plan({ moduleName: 'core', action: 'install' })`)
	planID, _ := planJSON["id"].(string)
	if planID == "" {
		t.Fatalf("plan = %#v", planJSON)
	}
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.apply({ id: `+jsString(planID)+` })`)
	_ = awaitModuleManagementJSON(t, engine, `$choysum.moduleManagement.run({ id: `+jsString(planID)+` })`)
	if factoryCalls != 1 {
		t.Fatalf("compiler factory calls = %d, want 1 (leftover process executor?)", factoryCalls)
	}
	if counting.started != 1 {
		t.Fatalf("Start calls = %d, want 1", counting.started)
	}
	if len(sawScopes) < 2 {
		t.Fatalf("expected per-call Lifecycle/Resolver scopes, got %d", len(sawScopes))
	}

	// syncCatalog with processHost must not create/start a compiler.
	before := counting.started
	syncEngine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		WithModuleManagementProcessHost(),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.compilerExecutorFactory = func(scope.Scope) (jsexecutor.JsExecutor, error) {
				t.Fatal("syncCatalog must not create compiler")
				return nil, errors.New("unexpected")
			}
			cfg.opsHostFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory, ev bus.EventBus) *ops.Ops {
				return &ops.Ops{
					Scope:         runtimeScope,
					LockerFactory: lockerFactory,
					Events:        ev,
					SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
						return lifecycle.ModuleSourceSnapshotSyncStats{Total: 1, Success: 1}, nil
					},
				}
			}
		}),
	))
	got := awaitModuleManagementJSON(t, syncEngine, `$choysum.moduleManagement.syncCatalog({ originType: 'local' })`)
	if ok, _ := got["ok"].(bool); !ok {
		t.Fatalf("syncCatalog = %#v", got)
	}
	if counting.started != before {
		t.Fatalf("syncCatalog started compiler: %d → %d", before, counting.started)
	}
}

func TestQuickjsRuntimeOptionsHelpers(t *testing.T) {
	if got := newRuntimeOptions(scope.PathsRuntimeOptions{}, false); got.modulesPath != "" {
		t.Fatalf("newRuntimeOptions(no path) = %#v, want empty modulesPath", got)
	}
	if got := runtimeOptionsFromScope(nil); got.modulesPath != "" {
		t.Fatalf("runtimeOptionsFromScope(nil) = %#v, want empty modulesPath", got)
	}
	runtimeScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: "/workspace/modules"},
	}
	if got := runtimeOptionsFromScope(runtimeScope); got.modulesPath != "/workspace/modules" {
		t.Fatalf("runtimeOptionsFromScope(scope).modulesPath = %q, want /workspace/modules", got.modulesPath)
	}
	if (runtimeOptions{modulesPath: "   "}).hasModulesPath() {
		t.Fatal("hasModulesPath() = true for whitespace modulesPath")
	}
	if !(runtimeOptions{modulesPath: "/workspace/modules"}).hasModulesPath() {
		t.Fatal("hasModulesPath() = false for non-empty modulesPath")
	}
}

func awaitModuleManagementJSON(t *testing.T, engine *quickjsengine.QuickjsEngine, expr string) map[string]any {
	t.Helper()
	promise := engine.Ctx.Eval(expr)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("eval exception: %v", engine.Ctx.Exception())
	}
	ret := promise.Await()
	defer ret.Free()
	if ret.IsError() || ret.IsException() {
		msg := ret.String()
		if ret.IsException() {
			msg = engine.Ctx.Exception().Error()
		}
		t.Fatalf("promise rejected: %s", msg)
	}
	raw := ret.JSONStringify()
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return out
}

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

var (
	_ plan.Resolver     = (*mapResolver)(nil)
	_ bus.EventBus      = (*recordingBus)(nil)
	_ lifecycle.Service = moduleManagementTestManager{}
	_ statepkg.Locker   = (*moduleIndexTestLocker)(nil)
)
