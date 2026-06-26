// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type moduleIndexTestScope struct {
	ctx        context.Context
	logger     *slog.Logger
	cfg        *config.Config
	transactor scope.Transactor
}

func (e *moduleIndexTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *moduleIndexTestScope) Transactor() scope.Transactor {
	if e.transactor != nil {
		return e.transactor
	}
	return scopetest.NewPassthroughTransactor(e)
}
func (e *moduleIndexTestScope) Session() *scope.Session { return nil }
func (e *moduleIndexTestScope) WithContext(ctx context.Context) scope.Scope {
	return &moduleIndexTestScope{ctx: ctx, logger: e.logger, cfg: e.cfg, transactor: e.transactor}
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

func registerModuleManagementTestJsEngine(name string) {
	jsengine.Register(name, func(jsengine.ScopeProvider, auth.Authenticator, ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
		return quickjsengine.NewFactory()
	})
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

func TestSyncModuleIndexLocalUsesInjectedLockerFactory(t *testing.T) {
	runtimeScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.Default(),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	locker := &moduleIndexTestLocker{acquireErr: statepkg.ErrLeaseBusy}

	stats, err := syncModuleIndexLocal(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return locker
	})
	if err == nil {
		t.Fatal("expected syncModuleIndexLocal() to return lease conflict")
	}
	if stats.Total != 0 || stats.Success != 0 || stats.Failed != 0 {
		t.Fatalf("unexpected stats = %+v", stats)
	}
	if info := err.Error(); !strings.Contains(info, "LEASE_CONFLICT") {
		t.Fatalf("expected LEASE_CONFLICT error, got %v", err)
	}
	if locker.acquired != 1 {
		t.Fatalf("locker Acquire calls = %d, want 1", locker.acquired)
	}
	if locker.released != 0 {
		t.Fatalf("locker Release calls = %d, want 0 on failed acquire", locker.released)
	}
}

func TestNewModuleManagerForModuleManagementPassesResolvedLockerFactory(t *testing.T) {
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
		return nil
	}

	_ = newModuleLifecycleForModuleManagement(nil, nil, cfg)
	if !called {
		t.Fatal("expected module manager factory to be called")
	}
}

func TestWithModuleManagementProviderUsesExecContextBoundRuntimeScope(t *testing.T) {
	type ctxKey struct{}
	runtimeCtx := context.WithValue(context.Background(), ctxKey{}, "runtime")
	engineName := "module-management-provider-test-engine"
	registerModuleManagementTestJsEngine(engineName)
	serverCfg := config.NewDefaultServerConfig()
	serverCfg.JsEngineFactory = engineName
	baseScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{Server: serverCfg},
	}

	var factoryScope scope.Scope
	var installCtx context.Context
	engine := newTestQuickjsEngine(t, WithModuleManagementProvider(
		jsengine.StaticScopeProvider(baseScope),
		moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
			cfg.moduleLifecycleFactory = func(runtimeScope scope.Scope, _ jsexecutor.JsExecutor, _ statepkg.LockerFactory) lifecycle.Service {
				factoryScope = runtimeScope
				return moduleManagementTestManager{install: func(ctx context.Context, name string) error {
					installCtx = ctx
					if name != "auth" {
						return errors.New("unexpected module name")
					}
					return nil
				}}
			}
		}),
	))

	evalString(t, engine, `(function() {
		globalThis.$choysum = globalThis.$choysum || {};
		globalThis.$choysum.__rpc__ = async function(req) {
			return {
				id: req.id,
				result: await $choysum.moduleManagement.install({ moduleName: req.service, withDemo: false }),
				context: {}
			};
		};
		return "ok";
	})()`)

	resp, err := engine.Execute(runtimeCtx, &jsengine.JsRequest{Id: "req-1", Service: "auth"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result object, got %#v", resp.Result)
	}
	if got, ok := result["ok"].(bool); !ok || !got {
		t.Fatalf("expected successful module management result, got %#v", result)
	}
	if factoryScope == nil {
		t.Fatal("expected module manager factory to receive a runtime scope")
	}
	if got := factoryScope.Context().Value(ctxKey{}); got != "runtime" {
		t.Fatalf("expected factory scope ctx to carry runtime marker, got %#v", got)
	}
	if installCtx == nil || installCtx.Value(ctxKey{}) != "runtime" {
		t.Fatalf("expected install ctx to carry runtime marker, got %#v", installCtx)
	}
}

func TestNormalizeModuleIndexOriginType(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults all", in: "", want: "all"},
		{name: "all", in: "all", want: "all"},
		{name: "registry", in: "registry", want: "registry"},
		{name: "trimmed uppercase local", in: "  LOCAL ", want: "local"},
		{name: "unsupported invalid", in: "remote", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeModuleIndexOriginType(tt.in); got != tt.want {
				t.Fatalf("normalizeModuleIndexOriginType(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRunModuleIndexSyncAllUsesPerOriginTransactions(t *testing.T) {
	runtimeScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	counting := &moduleIndexCountingTransactor{inner: scopetest.NewPassthroughTransactor(runtimeScope)}
	runtimeScope.transactor = counting

	called := make([]string, 0, 2)
	result, err := runModuleIndexSync(context.Background(), runtimeScope, "all", func(_ context.Context, _ scope.Scope, originType string) (lifecycle.ModuleIndexSyncStats, error) {
		called = append(called, originType)
		if originType == "registry" {
			return lifecycle.ModuleIndexSyncStats{Total: 1, Success: 0, Failed: 1}, errors.New("registry unavailable")
		}
		return lifecycle.ModuleIndexSyncStats{Total: 2, Success: 2, Failed: 0}, nil
	})
	if err != nil {
		t.Fatalf("runModuleIndexSync(all) error = %v", err)
	}
	if counting.requiredCalls != 2 {
		t.Fatalf("required transaction calls = %d, want 2", counting.requiredCalls)
	}
	if !reflect.DeepEqual(called, []string{"registry", "local"}) {
		t.Fatalf("origin call order = %#v, want [registry local]", called)
	}
	if !result.Ok {
		t.Fatalf("result.Ok = false, want true; result = %#v", result)
	}
	if result.Total != 3 || result.Success != 2 || result.Failed != 1 {
		t.Fatalf("unexpected stats result = %#v", result)
	}
	if !strings.Contains(result.Error, "registry") {
		t.Fatalf("expected partial error to mention registry, got %q", result.Error)
	}
}

func TestRunModuleIndexSyncSingleOriginUsesSingleTransaction(t *testing.T) {
	runtimeScope := &moduleIndexTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	counting := &moduleIndexCountingTransactor{inner: scopetest.NewPassthroughTransactor(runtimeScope)}
	runtimeScope.transactor = counting

	result, err := runModuleIndexSync(context.Background(), runtimeScope, "local", func(_ context.Context, _ scope.Scope, originType string) (lifecycle.ModuleIndexSyncStats, error) {
		if originType != "local" {
			t.Fatalf("originType = %q, want local", originType)
		}
		return lifecycle.ModuleIndexSyncStats{Total: 5, Success: 4, Failed: 1}, nil
	})
	if err != nil {
		t.Fatalf("runModuleIndexSync(local) error = %v", err)
	}
	if counting.requiredCalls != 1 {
		t.Fatalf("required transaction calls = %d, want 1", counting.requiredCalls)
	}
	if !result.Ok || result.Total != 5 || result.Success != 4 || result.Failed != 1 {
		t.Fatalf("unexpected result = %#v", result)
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
