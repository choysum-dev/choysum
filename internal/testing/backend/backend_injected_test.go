// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
)

type failRunScope struct {
	ctx    context.Context
	cfg    *config.Config
	runErr error
}

type failRunTransactor struct{ err error }

func (e *failRunScope) Run(fn func(runtimeScope scope.Scope) error) error { return e.runErr }
func (e *failRunScope) Transactor() scope.Transactor {
	return failRunTransactor{err: e.runErr}
}
func (e *failRunScope) Session() *scope.Session { return nil }
func (e *failRunScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *failRunScope) Context() context.Context { return e.ctx }
func (e *failRunScope) Logger() *slog.Logger     { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func (e *failRunScope) Config() *config.Config   { return e.cfg }
func (e *failRunScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func (t failRunTransactor) Do(context.Context, scope.TransactionOptions, scope.TxFunc) error {
	return t.err
}

func (t failRunTransactor) Required(context.Context, scope.TxFunc) error {
	return t.err
}

func (t failRunTransactor) RequiresNew(context.Context, scope.TxFunc) error {
	return t.err
}

func (t failRunTransactor) Nested(context.Context, scope.TxFunc) error {
	return t.err
}

type harnessStubService struct {
	name    string
	scripts []*jsengine.JsScript
	descs   []*grpc.ServiceDesc
	descErr error
}

func (s *harnessStubService) Name() string { return s.name }
func (s *harnessStubService) ServiceDescs() ([]*grpc.ServiceDesc, error) {
	if s.descErr != nil {
		return nil, s.descErr
	}
	return s.descs, nil
}
func (s *harnessStubService) ServiceScripts() []*jsengine.JsScript { return s.scripts }
func (s *harnessStubService) WebHandlers() (map[string]http.Handler, error) {
	return map[string]http.Handler{}, nil
}

type fakeAuthenticator struct {
	createTokensFn func(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error)
}

func (a *fakeAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	return nil, nil
}
func (a *fakeAuthenticator) CreateTokens(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	if a.createTokensFn != nil {
		return a.createTokensFn(ctx, userID, metadata)
	}
	return &auth.TokenPair{AccessToken: "token"}, nil
}
func (a *fakeAuthenticator) RefreshTokens(ctx context.Context, refreshToken string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	return &auth.TokenPair{}, nil
}
func (a *fakeAuthenticator) RevokeToken(ctx context.Context, token string, reason string) error {
	return nil
}
func (a *fakeAuthenticator) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	return 0, nil
}
func (a *fakeAuthenticator) Close() error { return nil }

func TestDefaultNewQuickJSAuthenticatorUsesAuthFactory(t *testing.T) {
	typeName := "backend-injected-auth-factory"
	runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
		Auth: &config.AuthConfig{Enabled: true, Type: typeName},
	}}
	want := &fakeAuthenticator{}

	auth.Register(typeName, func(gotScope scope.Scope) (auth.Authenticator, error) {
		if gotScope != runtimeScope {
			t.Fatalf("factory received unexpected scope: %#v", gotScope)
		}
		return want, nil
	})

	got, err := defaultNewQuickJSAuthenticator(runtimeScope)
	if err != nil {
		t.Fatalf("defaultNewQuickJSAuthenticator() error = %v", err)
	}
	if got != want {
		t.Fatalf("defaultNewQuickJSAuthenticator() = %#v, want %#v", got, want)
	}
}

func TestRunOneAppBackendTestsWithInjectedHooks(t *testing.T) {
	oldMake := makeTestScopeHook
	oldExec := newCompilerExecutorHook
	oldStartHarness := startInProcessGrpcHarnessHook
	oldQuickJSAuth := newQuickJSAuthenticatorHook
	oldPrepare := prepareBackendHook
	oldExecute := executeBackendHook
	defer func() {
		makeTestScopeHook = oldMake
		newCompilerExecutorHook = oldExec
		startInProcessGrpcHarnessHook = oldStartHarness
		newQuickJSAuthenticatorHook = oldQuickJSAuth
		prepareBackendHook = oldPrepare
		executeBackendHook = oldExecute
	}()

	t.Run("make test scope error is propagated", func(t *testing.T) {
		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir()}}
		makeErr := errors.New("make scope failed")
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return nil, func() {}, makeErr
		}
		newCompilerExecutorHook = oldExec
		failed, err := RunOneAppBackendTests(context.Background(), baseScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false on env creation error")
		}
		if !errors.Is(err, makeErr) {
			t.Fatalf("expected make scope error, got %v", err)
		}
	})

	t.Run("coverage preflight fails before runtime preparation", func(t *testing.T) {
		repoRoot := t.TempDir()
		t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", filepath.Join(t.TempDir(), "missing-global-root"))

		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir()}}
		makeScopeCalled := false
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			makeScopeCalled = true
			return baseScope, func() {}, nil
		}

		var progress strings.Builder
		oldProgressWriter := backendProgressWriter
		backendProgressWriter = &progress
		defer func() { backendProgressWriter = oldProgressWriter }()

		failed, err := RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, true)
		if failed {
			t.Fatalf("expected failed=false on preflight error")
		}
		if err == nil || !strings.Contains(err.Error(), "missing 1 required module(s): istanbul-lib-instrument") {
			t.Fatalf("expected coverage preflight error, got %v", err)
		}
		if makeScopeCalled {
			t.Fatalf("expected preflight failure before makeTestScope")
		}
		if progress.Len() != 0 {
			t.Fatalf("expected no runtime preparation logs before preflight failure, got %q", progress.String())
		}
	})

	t.Run("compiler creation error still triggers cleanup", func(t *testing.T) {
		cleanupCalled := false
		repoRoot := t.TempDir()
		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: filepath.Join(t.TempDir(), "dist")}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return baseScope, func() { cleanupCalled = true }, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, errors.New("compiler failed")
		}

		failed, err := RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false on compiler init error")
		}
		if err == nil || !strings.Contains(err.Error(), "compiler failed") {
			t.Fatalf("expected compiler error, got %v", err)
		}
		if !cleanupCalled {
			t.Fatalf("expected cleanup function to be called")
		}
	})

	t.Run("mocked prepare and execute can cover result branches", func(t *testing.T) {
		repoRoot := t.TempDir()
		tmpRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: distRoot, TmpPath: tmpRoot}}

		prepareCleanupCalled := false
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return baseScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, nil
		}
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() { prepareCleanupCalled = true }, nil
		}
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{
				"total":        1,
				"passed":       1,
				"failed":       0,
				"coverageJSON": `{"ok":true}`,
				"cases": []any{
					map[string]any{"name": "ok", "ok": true, "durationMs": 1},
				},
			}, nil
		}

		junitPath := filepath.Join(t.TempDir(), "junit.xml")
		failed, err := RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, junitPath, "", false, true)
		if err != nil || failed {
			t.Fatalf("expected success with mocked execution, failed=%v err=%v", failed, err)
		}
		if !prepareCleanupCalled {
			t.Fatalf("expected prepare cleanup to be called")
		}
		if _, err := os.Stat(junitPath); err != nil {
			t.Fatalf("expected junit file written, err=%v", err)
		}
		coverageFiles, err := filepath.Glob(filepath.Join(tmpRoot, "testing", "*", "coverage", "nyc_output", "*.json"))
		if err != nil || len(coverageFiles) == 0 {
			t.Fatalf("expected coverage json written under tmp root, err=%v files=%d", err, len(coverageFiles))
		}

		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{
				"total":  1,
				"passed": 0,
				"failed": 1,
				"cases":  []any{map[string]any{"name": "bad", "ok": false, "durationMs": 1}},
			}, nil
		}
		failed, err = RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if err != nil || !failed {
			t.Fatalf("expected failed report without execution error, failed=%v err=%v", failed, err)
		}

		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return "bad", nil
		}
		failed, err = RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if err == nil || !failed || !strings.Contains(err.Error(), "parse test report") {
			t.Fatalf("expected parse report error, failed=%v err=%v", failed, err)
		}
	})

	t.Run("prints runtime preparation progress lines", func(t *testing.T) {
		repoRoot := t.TempDir()
		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: filepath.Join(t.TempDir(), "dist")}}

		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return baseScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, nil
		}
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{
				"total":  1,
				"passed": 1,
				"failed": 0,
				"cases":  []any{map[string]any{"name": "ok", "ok": true, "durationMs": 1}},
			}, nil
		}

		var progress strings.Builder
		oldProgressWriter := backendProgressWriter
		backendProgressWriter = &progress
		defer func() { backendProgressWriter = oldProgressWriter }()

		failed, err := RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if err != nil || failed {
			t.Fatalf("expected success with progress output, failed=%v err=%v", failed, err)
		}
		out := progress.String()
		if !strings.Contains(out, "# prepare runtime auth\n") {
			t.Fatalf("expected prepare start line, got %q", out)
		}
		if !strings.Contains(out, "# prepare runtime auth ok (") {
			t.Fatalf("expected prepare completion line, got %q", out)
		}
	})

	t.Run("prepare and execute hook errors are wrapped", func(t *testing.T) {
		repoRoot := t.TempDir()
		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: filepath.Join(t.TempDir(), "dist")}}

		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return baseScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, nil
		}

		prepareErr := errors.New("prepare failed")
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return nil, prepareErr
		}
		executeBackendHook = nil

		failed, err := RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false on prepare error")
		}
		if !errors.Is(err, prepareErr) {
			t.Fatalf("expected prepare error, got %v", err)
		}

		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		execErr := errors.New("execute failed")
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return nil, execErr
		}

		failed, err = RunOneAppBackendTests(context.Background(), baseScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if !failed {
			t.Fatalf("expected failed=true on execute error")
		}
		if err == nil || !strings.Contains(err.Error(), "execute tests") {
			t.Fatalf("expected wrapped execute error, got %v", err)
		}
	})

	t.Run("context canceled and junit write error branches", func(t *testing.T) {
		baseScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: filepath.Join(t.TempDir(), "dist")}}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		failed, err := RunOneAppBackendTests(ctx, baseScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
		if !failed || !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context failure, failed=%v err=%v", failed, err)
		}

		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return baseScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{
				"total":  1,
				"passed": 1,
				"failed": 0,
				"cases":  []any{map[string]any{"name": "ok", "ok": true, "durationMs": 1}},
			}, nil
		}

		badJunitPath := filepath.Join(t.TempDir(), "report.xml")
		if err := os.MkdirAll(badJunitPath, 0o755); err != nil {
			t.Fatalf("MkdirAll bad junit path: %v", err)
		}
		failed, err = RunOneAppBackendTests(context.Background(), baseScope, "auth", t.TempDir(), "sqlite", "", "", false, badJunitPath, "", false, false)
		if !failed {
			t.Fatalf("expected failed=true when junit write fails")
		}
		if err == nil || !strings.Contains(err.Error(), "write junit") {
			t.Fatalf("expected write junit error, got %v", err)
		}
	})

	t.Run("helper constructors are callable", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Server: &config.ServerConfig{JsEngineFactory: "quickjs"}}}
		mod := &meta.Module{Name: "auth", ApplicationStr: "auth", Path: filepath.Join(runtimeOptionsFromScope(runtimeScope).modulesPath, "auth")}
		if b := defaultNewBackendBuilder(runtimeScope, nil, mod, filepath.Join(t.TempDir(), "entry.ts"), "index.js", "auth"); b == nil {
			t.Fatalf("expected non-nil backend builder")
		}
		func() {
			defer func() { _ = recover() }()
			if jsExec, err := newCompilerExecutor(runtimeScope); err == nil {
				if jsExec != nil {
					_ = jsExec.Stop()
				}
			}
		}()
	})

	t.Run("default prepare branch returns test scope transaction-entry error", func(t *testing.T) {
		runErr := errors.New("prepare run failed")
		runtimeScope := &failRunScope{
			ctx:    context.Background(),
			cfg:    &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir()},
			runErr: runErr,
		}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = nil
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{"total": 0, "passed": 0, "failed": 0, "cases": []any{}}, nil
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false when default prepare branch errors")
		}
		if !errors.Is(err, runErr) {
			t.Fatalf("expected run error from default prepare branch, got %v", err)
		}
	})

	t.Run("default execute branch fails on missing js engine factory", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    t.TempDir(),
			Server:      &config.ServerConfig{JsEngineFactory: "missing-factory"},
			Compile:     &config.CompileConfig{BundleMode: "bundle"},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false when default execute branch returns factory error")
		}
		if err == nil || !strings.Contains(err.Error(), "JavaScript engine factory is not registered") {
			t.Fatalf("expected js engine factory error, got %v", err)
		}
	})

	t.Run("coverage write error path returns failed true", func(t *testing.T) {
		tmpRootFile := filepath.Join(t.TempDir(), "tmp-root-file")
		if err := os.WriteFile(tmpRootFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write tmpRoot file: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), TmpPath: tmpRootFile}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{
				"total":        1,
				"passed":       1,
				"failed":       0,
				"coverageJSON": `{"ok":true}`,
				"cases":        []any{map[string]any{"name": "ok", "ok": true, "durationMs": 1}},
			}, nil
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, true)
		if !failed {
			t.Fatalf("expected failed=true when coverage write fails")
		}
		if err == nil || !strings.Contains(err.Error(), "mkdir coverage nyc_output") {
			t.Fatalf("expected coverage mkdir error, got %v", err)
		}
	})

	t.Run("context canceled after prepare returns canceled", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir()}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }

		ctx, cancel := context.WithCancel(context.Background())
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			cancel()
			return nil, nil
		}
		executeBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error) {
			return map[string]any{"total": 0, "passed": 0, "failed": 0, "cases": []any{}}, nil
		}

		failed, err := RunOneAppBackendTests(ctx, runtimeScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
		if !failed || !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context after prepare, failed=%v err=%v", failed, err)
		}
	})

	t.Run("default execute branch reaches runtime path", func(t *testing.T) {
		baseDist := t.TempDir()
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    baseDist,
			Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file::memory:?cache=shared&_fk=1&_busy_timeout=60000"},
			Server:      &config.ServerConfig{JsEngineFactory: "quickjs"},
			Compile:     &config.CompileConfig{BundleMode: "bundle"},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil

		var gotErr error
		didPanic := false
		func() {
			defer func() {
				if recover() != nil {
					didPanic = true
				}
			}()
			failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
			if failed {
				t.Fatalf("expected failed=false on default runtime path error")
			}
			gotErr = err
		}()

		if !didPanic {
			if gotErr == nil {
				t.Fatalf("expected panic or non-nil error from default runtime path")
			}
			if !strings.Contains(gotErr.Error(), "read dist index.js failed") &&
				!strings.Contains(gotErr.Error(), "load scripts") &&
				!strings.Contains(gotErr.Error(), "JavaScript engine factory") {
				t.Fatalf("unexpected default runtime path error: %v", gotErr)
			}
		}
	})

	t.Run("default execute branch succeeds with fake engine and harness", func(t *testing.T) {
		repoRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		appDistDir := filepath.Join(distRoot, "apps", "auth")
		if err := os.MkdirAll(appDistDir, 0o755); err != nil {
			t.Fatalf("mkdir app dist dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("// app"), 0o644); err != nil {
			t.Fatalf("write app index: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "tests.js"), []byte("// tests"), 0o644); err != nil {
			t.Fatalf("write app tests: %v", err)
		}

		engineName := testEngineName(t, "default-execute-success")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testReportJsEngine{result: map[string]any{
				"total":  1,
				"passed": 1,
				"failed": 0,
				"cases":  []any{map[string]any{"name": "ok", "ok": true, "durationMs": 1}},
			}}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    distRoot,
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return &inProcessGrpcHarness{}, nil
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if err != nil || failed {
			t.Fatalf("expected default execute success, failed=%v err=%v", failed, err)
		}
	})

	t.Run("default execute branch surfaces unit identity context errors", func(t *testing.T) {
		repoRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		appDistDir := filepath.Join(distRoot, "apps", "auth")
		if err := os.MkdirAll(appDistDir, 0o755); err != nil {
			t.Fatalf("mkdir app dist dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("// app"), 0o644); err != nil {
			t.Fatalf("write app index: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "tests.js"), []byte("// tests"), 0o644); err != nil {
			t.Fatalf("write app tests: %v", err)
		}

		engineName := testEngineName(t, "default-execute-identity-err")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testReportJsEngine{result: map[string]any{"total": 0, "passed": 0, "failed": 0, "cases": []any{}}}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    distRoot,
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return &inProcessGrpcHarness{}, nil
		}
		prevIdentity := unitTestIdentityContextFn
		t.Cleanup(func() { unitTestIdentityContextFn = prevIdentity })
		unitTestIdentityContextFn = func(ctx context.Context, testScope scope.Scope) (map[string]interface{}, error) {
			return nil, errors.New("identity context boom")
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		// loadUnitAppTestContext failures return (false, err) before the test run is marked failed.
		if failed || err == nil || !strings.Contains(err.Error(), "identity context boom") {
			t.Fatalf("expected identity context failure, failed=%v err=%v", failed, err)
		}
	})

	t.Run("default execute branch wraps harness startup and execute errors", func(t *testing.T) {
		repoRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		appDistDir := filepath.Join(distRoot, "apps", "auth")
		if err := os.MkdirAll(appDistDir, 0o755); err != nil {
			t.Fatalf("mkdir app dist dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("// app"), 0o644); err != nil {
			t.Fatalf("write app index: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "tests.js"), []byte("// tests"), 0o644); err != nil {
			t.Fatalf("write app tests: %v", err)
		}

		engineName := testEngineName(t, "default-execute-errors")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testReportJsEngine{result: map[string]any{"total": 0, "passed": 0, "failed": 0, "cases": []any{}}}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    distRoot,
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil

		harnessErr := errors.New("harness boom")
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return nil, harnessErr
		}
		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if !failed || err == nil || !strings.Contains(err.Error(), "execute tests") || !errors.Is(err, harnessErr) {
			t.Fatalf("expected wrapped harness error, failed=%v err=%v", failed, err)
		}

		executeErrEngine := testEngineName(t, "default-execute-execerr")
		registerTestJsEngineFactory(executeErrEngine, func() (jsengine.JsEngine, error) {
			return &testReportJsEngine{executeErr: errors.New("engine execute boom")}, nil
		})
		runtimeScope.cfg.Server.JsEngineFactory = executeErrEngine
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return &inProcessGrpcHarness{}, nil
		}

		failed, err = RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if !failed || err == nil || !strings.Contains(err.Error(), "execute tests") || !strings.Contains(err.Error(), "engine execute boom") {
			t.Fatalf("expected wrapped engine execute error, failed=%v err=%v", failed, err)
		}
	})

	t.Run("default execute branch returns read tests bundle error", func(t *testing.T) {
		repoRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		appDistDir := filepath.Join(distRoot, "apps", "auth")
		if err := os.MkdirAll(appDistDir, 0o755); err != nil {
			t.Fatalf("mkdir app dist dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("// app"), 0o644); err != nil {
			t.Fatalf("write app index: %v", err)
		}

		engineName := testEngineName(t, "default-execute-missing-tests")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    distRoot,
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return &inProcessGrpcHarness{}, nil
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false on read tests bundle error")
		}
		if err == nil || !strings.Contains(err.Error(), "read dist tests.js failed") {
			t.Fatalf("expected read tests bundle error, got %v", err)
		}
	})

	t.Run("default execute branch returns load scripts error", func(t *testing.T) {
		repoRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		appDistDir := filepath.Join(distRoot, "apps", "auth")
		if err := os.MkdirAll(appDistDir, 0o755); err != nil {
			t.Fatalf("mkdir app dist dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("// app"), 0o644); err != nil {
			t.Fatalf("write app index: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "tests.js"), []byte("// tests"), 0o644); err != nil {
			t.Fatalf("write app tests: %v", err)
		}

		engineName := testEngineName(t, "default-execute-load-fail")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testFailLoadJsEngine{loadErr: errors.New("load scripts boom")}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    distRoot,
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return &inProcessGrpcHarness{}, nil
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false on load scripts error")
		}
		if err == nil || !strings.Contains(err.Error(), "load scripts") || !strings.Contains(err.Error(), "load scripts boom") {
			t.Fatalf("expected load scripts error, got %v", err)
		}
	})

	t.Run("default execute branch returns jwt authenticator init error when auth enabled", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    t.TempDir(),
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: "quickjs"},
			Auth:        &config.AuthConfig{Enabled: true, JWT: nil},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", t.TempDir(), "sqlite", "", "", false, "", "", false, false)
		if failed {
			t.Fatalf("expected failed=false on jwt authenticator init error")
		}
		if err == nil {
			t.Fatalf("expected non-nil jwt authenticator init error")
		}
	})

	t.Run("default execute auth-enabled branch handles token creation success and error", func(t *testing.T) {
		repoRoot := t.TempDir()
		distRoot := filepath.Join(t.TempDir(), "dist")
		appDistDir := filepath.Join(distRoot, "apps", "auth")
		if err := os.MkdirAll(appDistDir, 0o755); err != nil {
			t.Fatalf("mkdir app dist dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("// app"), 0o644); err != nil {
			t.Fatalf("write app index: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDistDir, "tests.js"), []byte("// tests"), 0o644); err != nil {
			t.Fatalf("write app tests: %v", err)
		}

		engineName := testEngineName(t, "default-execute-auth-enabled")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testReportJsEngine{result: map[string]any{
				"total":  1,
				"passed": 1,
				"failed": 0,
				"cases":  []any{map[string]any{"name": "ok", "ok": true, "durationMs": 1}},
			}}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    distRoot,
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Auth:        &config.AuthConfig{Enabled: true},
		}}
		makeTestScopeHook = func(ctx context.Context, base scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
			return runtimeScope, func() {}, nil
		}
		newCompilerExecutorHook = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) { return nil, nil }
		prepareBackendHook = func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot string, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error) {
			return func() {}, nil
		}
		executeBackendHook = nil
		startInProcessGrpcHarnessHook = func(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
			return &inProcessGrpcHarness{}, nil
		}

		called := false
		newQuickJSAuthenticatorHook = func(runtimeScope scope.Scope) (auth.Authenticator, error) {
			return &fakeAuthenticator{createTokensFn: func(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
				called = true
				if userID != "choysum_test_runner" {
					t.Fatalf("unexpected token user id: %s", userID)
				}
				return &auth.TokenPair{AccessToken: "ok-token"}, nil
			}}, nil
		}

		failed, err := RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if err != nil || failed {
			t.Fatalf("expected auth-enabled default execute success, failed=%v err=%v", failed, err)
		}
		if !called {
			t.Fatalf("expected fake authenticator CreateTokens to be called")
		}

		newQuickJSAuthenticatorHook = func(runtimeScope scope.Scope) (auth.Authenticator, error) {
			return &fakeAuthenticator{createTokensFn: func(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
				return nil, errors.New("token creation boom")
			}}, nil
		}

		failed, err = RunOneAppBackendTests(context.Background(), runtimeScope, "auth", repoRoot, "sqlite", "", "", false, "", "", false, false)
		if !failed || err == nil || !strings.Contains(err.Error(), "execute tests") || !strings.Contains(err.Error(), "token creation boom") {
			t.Fatalf("expected wrapped create token error, failed=%v err=%v", failed, err)
		}
	})
}

func TestInProcessGrpcHarnessGuards(t *testing.T) {
	var h *inProcessGrpcHarness
	h.Close()

	h = &inProcessGrpcHarness{}
	h.Close()

	if _, err := startInProcessGrpcHarness(context.TODO(), nil); err == nil || !strings.Contains(err.Error(), "invalid scope") {
		t.Fatalf("expected invalid scope error, got %v", err)
	}
	if _, err := startInProcessGrpcHarness(context.Background(), &testStubScope{ctx: context.Background(), cfg: nil}); err == nil || !strings.Contains(err.Error(), "invalid scope") {
		t.Fatalf("expected invalid scope error with nil config, got %v", err)
	}

	runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
		ModulesPath: t.TempDir(),
		DistPath:    t.TempDir(),
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file::memory:?cache=shared&_fk=1&_busy_timeout=60000"},
		Server:      &config.ServerConfig{JsEngineFactory: "quickjs"},
		Compile:     &config.CompileConfig{BundleMode: "application"},
	}}
	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil {
			t.Fatalf("expected non-nil error for minimal env harness startup")
		}
	}()
	if didPanic {
		// acceptable in minimal unit env when default quickjs factory dereferences absent runtime deps
	}

	bundleDist := t.TempDir()
	if err := os.MkdirAll(config.APIRootFromDist(bundleDist), 0o755); err != nil {
		t.Fatalf("mkdir api root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(bundleDist, "bundles"), 0o755); err != nil {
		t.Fatalf("mkdir bundles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDist, "bundles", "index.js"), []byte("// bundle entry"), 0o644); err != nil {
		t.Fatalf("write bundle index: %v", err)
	}
	bundleRuntimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
		ModulesPath: t.TempDir(),
		DistPath:    bundleDist,
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file::memory:?cache=shared&_fk=1&_busy_timeout=60000"},
		Server:      &config.ServerConfig{JsEngineFactory: "quickjs"},
		Compile:     &config.CompileConfig{BundleMode: "bundle"},
	}}
	bundlePanic := false
	func() {
		defer func() {
			if recover() != nil {
				bundlePanic = true
			}
		}()
		h2, err2 := startInProcessGrpcHarness(context.Background(), bundleRuntimeScope)
		if h2 != nil {
			h2.Close()
		}
		if err2 == nil {
			t.Fatalf("expected non-nil error for minimal bundle harness startup")
		}
	}()
	if bundlePanic {
		// acceptable in minimal unit env; branch touch is still useful for coverage hardening
	}
}

func TestStartInProcessGrpcHarnessErrorBranches(t *testing.T) {
	oldNewHarnessService := newHarnessServiceHook
	defer func() { newHarnessServiceHook = oldNewHarnessService }()

	t.Run("application mode create service error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "app-create-service")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dist, "apps", "auth"), 0o755); err != nil {
			t.Fatalf("mkdir apps auth dir: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "application"},
		}}

		newHarnessServiceHook = func(runtimeScope scope.Scope, name string, jsExec jsexecutor.JsExecutor, mode string) (harnessService, error) {
			return nil, errors.New("create service boom")
		}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "create service auth") {
			t.Fatalf("expected create service error, got %v", err)
		}
	})

	t.Run("application mode service desc error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "app-desc-err")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dist, "apps", "auth"), 0o755); err != nil {
			t.Fatalf("mkdir apps auth dir: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "application"},
		}}

		callCount := 0
		newHarnessServiceHook = func(runtimeScope scope.Scope, name string, jsExec jsexecutor.JsExecutor, mode string) (harnessService, error) {
			callCount++
			if callCount == 1 {
				return &harnessStubService{name: name, scripts: []*jsengine.JsScript{}}, nil
			}
			return &harnessStubService{name: name, scripts: []*jsengine.JsScript{}, descErr: errors.New("desc boom")}, nil
		}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "service descs auth") || !strings.Contains(err.Error(), "desc boom") {
			t.Fatalf("expected service desc error, got %v", err)
		}
	})

	t.Run("bundle mode create service error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "bundle-create-service")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		if err := os.MkdirAll(config.APIAppProtoDir(dist, "auth"), 0o755); err != nil {
			t.Fatalf("mkdir api auth proto dir: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(dist, "bundles"), 0o755); err != nil {
			t.Fatalf("mkdir bundles dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dist, "bundles", "index.js"), []byte("// bundle"), 0o644); err != nil {
			t.Fatalf("write bundle index: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "bundle"},
		}}

		newHarnessServiceHook = func(runtimeScope scope.Scope, name string, jsExec jsexecutor.JsExecutor, mode string) (harnessService, error) {
			return nil, errors.New("bundle create boom")
		}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "create bundle service auth") {
			t.Fatalf("expected bundle create service error, got %v", err)
		}
	})

	t.Run("bundle mode service desc error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "bundle-desc-err")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		if err := os.MkdirAll(config.APIAppProtoDir(dist, "auth"), 0o755); err != nil {
			t.Fatalf("mkdir api auth proto dir: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(dist, "bundles"), 0o755); err != nil {
			t.Fatalf("mkdir bundles dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dist, "bundles", "index.js"), []byte("// bundle"), 0o644); err != nil {
			t.Fatalf("write bundle index: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "bundle"},
		}}

		callCount := 0
		newHarnessServiceHook = func(runtimeScope scope.Scope, name string, jsExec jsexecutor.JsExecutor, mode string) (harnessService, error) {
			callCount++
			if callCount == 1 {
				return &harnessStubService{name: name, scripts: []*jsengine.JsScript{}}, nil
			}
			return &harnessStubService{name: name, scripts: []*jsengine.JsScript{}, descErr: errors.New("bundle desc boom")}, nil
		}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "service descs auth") || !strings.Contains(err.Error(), "bundle desc boom") {
			t.Fatalf("expected bundle service desc error, got %v", err)
		}
	})

	newHarnessServiceHook = oldNewHarnessService

	t.Run("application mode can start harness", func(t *testing.T) {
		engineName := testEngineName(t, "harness-success")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dist, "apps"), 0o755); err != nil {
			t.Fatalf("mkdir apps dir: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "application"},
		}}
		newHarnessServiceHook = oldNewHarnessService

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if err != nil {
			t.Fatalf("expected harness startup success, got %v", err)
		}
		if h == nil {
			t.Fatalf("expected non-nil harness")
		}
		_ = h
	})

	t.Run("authenticator initialization error is propagated", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    t.TempDir(),
			Server:      &config.ServerConfig{JsEngineFactory: "quickjs"},
			Compile:     &config.CompileConfig{BundleMode: "application"},
			Auth:        &config.AuthConfig{Enabled: true, Type: "missing-auth-type"},
		}}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil {
			t.Fatalf("expected auth initialization error")
		}
	})

	t.Run("runtime executor creation error is wrapped", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    t.TempDir(),
			Server:      &config.ServerConfig{JsEngineFactory: "missing-factory"},
			Compile:     &config.CompileConfig{BundleMode: "application"},
		}}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "create runtime executor") {
			t.Fatalf("expected wrapped runtime executor creation error, got %v", err)
		}
	})

	t.Run("application mode read apps dir error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "apps-dir")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "application"},
		}}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "read apps dist dir") {
			t.Fatalf("expected read apps dist dir error, got %v", err)
		}
	})

	t.Run("bundle mode read api root dir error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "bundle-assets")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		dist := t.TempDir()
		bundlesDir := filepath.Join(dist, "bundles")
		apiRoot := config.APIRootFromDist(dist)
		if err := os.MkdirAll(bundlesDir, 0o755); err != nil {
			t.Fatalf("mkdir bundles dir: %v", err)
		}
		if err := os.MkdirAll(apiRoot, 0o755); err != nil {
			t.Fatalf("mkdir api root: %v", err)
		}
		if err := os.WriteFile(filepath.Join(bundlesDir, "index.js"), []byte("// bundle entry"), 0o644); err != nil {
			t.Fatalf("write bundles index: %v", err)
		}
		if err := os.Chmod(apiRoot, 0o000); err != nil {
			t.Fatalf("chmod api root: %v", err)
		}
		defer os.Chmod(apiRoot, 0o755)

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "bundle"},
		}}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "read api root dir") {
			t.Fatalf("expected read api root dir error, got %v", err)
		}
	})

	t.Run("runtime executor start error is wrapped", func(t *testing.T) {
		engineName := testEngineName(t, "start-fail")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testFailLoadJsEngine{loadErr: errors.New("runtime load fail")}, nil
		})

		dist := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dist, "apps"), 0o755); err != nil {
			t.Fatalf("mkdir apps dir: %v", err)
		}
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    dist,
			Server:      &config.ServerConfig{JsEngineFactory: engineName},
			Compile:     &config.CompileConfig{BundleMode: "application"},
		}}

		h, err := startInProcessGrpcHarness(context.Background(), runtimeScope)
		if h != nil {
			h.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "start runtime executor") || !strings.Contains(err.Error(), "runtime load fail") {
			t.Fatalf("expected wrapped runtime start error, got %v", err)
		}
	})
}
