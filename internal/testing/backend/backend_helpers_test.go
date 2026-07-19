// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type testNoopJsEngine struct{}

func (e *testNoopJsEngine) Load(_ []*jsengine.JsScript) error { return nil }
func (e *testNoopJsEngine) Execute(_ context.Context, _ *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *testNoopJsEngine) Close() error { return nil }

type testReportJsEngine struct {
	result     any
	executeErr error
}

func (e *testReportJsEngine) Load(_ []*jsengine.JsScript) error { return nil }
func (e *testReportJsEngine) Execute(_ context.Context, _ *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.executeErr != nil {
		return nil, e.executeErr
	}
	return &jsengine.JsResponse{Result: e.result}, nil
}
func (e *testReportJsEngine) Close() error { return nil }

type testFailLoadJsEngine struct {
	loadErr error
}

func (e *testFailLoadJsEngine) Load(_ []*jsengine.JsScript) error { return e.loadErr }
func (e *testFailLoadJsEngine) Execute(_ context.Context, _ *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *testFailLoadJsEngine) Close() error { return nil }

func testEngineName(t *testing.T, suffix string) string {
	t.Helper()
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-")
	return "backend-test-" + replacer.Replace(t.Name()) + "-" + suffix
}

func registerTestJsEngineFactory(name string, factory jsengine.JsEngineFactory) {
	jsengine.Register(name, func(_ jsengine.ScopeProvider, _ auth.Authenticator, _ ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
		return factory
	})
}

type testPassthroughTransactor struct {
	rootScope   scope.Scope
	requiredErr error
}

func (t testPassthroughTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	switch opts.Propagation {
	case scope.PropagationRequired:
		if t.requiredErr != nil {
			return t.requiredErr
		}
		txScope := t.rootScope
		if ctx != nil {
			txScope = t.rootScope.WithContext(ctx)
		}
		return fn(txScope, nil)
	case scope.PropagationRequiresNew:
		return scope.ErrRequiresNewUnsupported
	case scope.PropagationNested:
		return scope.ErrNestedUnsupported
	default:
		return fmt.Errorf("%w: %q", scope.ErrInvalidTransactionPropagation, opts.Propagation)
	}
}

func (t testPassthroughTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (t testPassthroughTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (t testPassthroughTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

type testStubScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *testStubScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *testStubScope) Transactor() scope.Transactor {
	return testPassthroughTransactor{rootScope: e}
}
func (e *testStubScope) Session() *scope.Session { return nil }
func (e *testStubScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *testStubScope) Context() context.Context { return e.ctx }
func (e *testStubScope) Logger() *slog.Logger     { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func (e *testStubScope) Config() *config.Config   { return e.cfg }
func (e *testStubScope) FactoryInput() scope.FactoryInput {
	if e == nil {
		return nil
	}
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func TestBackendPathHelpers(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	dirPath := filepath.Join(root, "dir")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", filePath, err)
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dirPath, err)
	}

	if !fileExists(filePath) {
		t.Fatal("expected fileExists to return true for file")
	}
	if fileExists(dirPath) || fileExists("") {
		t.Fatal("expected fileExists to reject directories and empty paths")
	}
	if !dirExists(dirPath) {
		t.Fatal("expected dirExists to return true for directory")
	}
	if dirExists(filePath) || dirExists("") {
		t.Fatal("expected dirExists to reject files and empty paths")
	}
	if dirExists(filepath.Join(root, "missing")) {
		t.Fatal("expected dirExists to reject missing paths")
	}
}

func TestShouldSkipTestScanDir(t *testing.T) {
	if !shouldSkipTestScanDir("node_modules") || !shouldSkipTestScanDir("dist") || !shouldSkipTestScanDir(".choysum") || !shouldSkipTestScanDir("tmp") {
		t.Fatalf("expected known dirs to be skipped")
	}
	if shouldSkipTestScanDir("service") {
		t.Fatalf("did not expect service dir to be skipped")
	}
}

func TestResolveBundleModeForTests(t *testing.T) {
	distRoot := filepath.Join(t.TempDir(), "dist")
	runtimeScope := &testStubScope{
		ctx: context.Background(),
		cfg: &config.Config{
			DistPath: distRoot,
			Compile:  &config.CompileConfig{BundleMode: "bundle"},
		},
	}

	t.Run("defaults to bundle when unset", func(t *testing.T) {
		got := resolveBundleModeForTests(&testStubScope{ctx: context.Background(), cfg: &config.Config{}})
		if got != "application" {
			t.Fatalf("resolveBundleModeForTests returned %q, want application", got)
		}
	})

	t.Run("returns explicit non bundle mode", func(t *testing.T) {
		got := resolveBundleModeForTests(&testStubScope{ctx: context.Background(), cfg: &config.Config{Compile: &config.CompileConfig{BundleMode: "application"}}})
		if got != "application" {
			t.Fatalf("resolveBundleModeForTests returned %q, want application", got)
		}
	})

	t.Run("falls back to application when bundle artifacts are missing", func(t *testing.T) {
		got := resolveBundleModeForTests(runtimeScope)
		if got != "application" {
			t.Fatalf("resolveBundleModeForTests returned %q, want application", got)
		}
	})

	t.Run("keeps bundle when bundle artifacts exist", func(t *testing.T) {
		bundlesDir := filepath.Join(distRoot, "bundles")
		if err := os.MkdirAll(bundlesDir, 0o755); err != nil {
			t.Fatalf("MkdirAll bundles dir: %v", err)
		}
		if err := os.MkdirAll(config.APIRootFromDist(distRoot), 0o755); err != nil {
			t.Fatalf("MkdirAll api root: %v", err)
		}
		if err := os.WriteFile(filepath.Join(bundlesDir, "index.js"), []byte("ok"), 0o644); err != nil {
			t.Fatalf("WriteFile bundles index: %v", err)
		}
		got := resolveBundleModeForTests(runtimeScope)
		if got != "bundle" {
			t.Fatalf("resolveBundleModeForTests returned %q, want bundle", got)
		}
	})
}

func TestBackendUtilityHelpers(t *testing.T) {
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	tmpRoot := filepath.Join(t.TempDir(), "choysum-custom-tmp")
	wantBackendTmpDir, err := testingpathing.ResolveTestingTmpDir(workspaceRoot, tmpRoot, "backend")
	if err != nil {
		t.Fatalf("ResolveTestingTmpDir(backend): %v", err)
	}
	gotBackendTmpDir, err := backendTmpDir(context.Background(), modulesPath, tmpRoot)
	if err != nil {
		t.Fatalf("backendTmpDir(): %v", err)
	}
	if gotBackendTmpDir != wantBackendTmpDir {
		t.Fatalf("backendTmpDir() = %q, want %q", gotBackendTmpDir, wantBackendTmpDir)
	}

	wantBackendTestsIndexTmpDir := filepath.Join(wantBackendTmpDir, "tests-index", "auth_role")
	gotBackendTestsIndexTmpDir, err := backendTestsIndexTmpDir(context.Background(), modulesPath, tmpRoot, "auth role")
	if err != nil {
		t.Fatalf("backendTestsIndexTmpDir(): %v", err)
	}
	if gotBackendTestsIndexTmpDir != wantBackendTestsIndexTmpDir {
		t.Fatalf("backendTestsIndexTmpDir() = %q, want %q", gotBackendTestsIndexTmpDir, wantBackendTestsIndexTmpDir)
	}

	testDBTmpDir, err := testingpathing.ResolveTestingTmpDir(workspaceRoot, tmpRoot, "testdb")
	if err != nil {
		t.Fatalf("ResolveTestingTmpDir(testdb): %v", err)
	}
	wantUnitTestDBDir := filepath.Join(testDBTmpDir, "auth")
	gotUnitTestDBDir, err := unitTestDBTmpDir(context.Background(), modulesPath, tmpRoot, "auth")
	if err != nil {
		t.Fatalf("unitTestDBTmpDir(): %v", err)
	}
	if gotUnitTestDBDir != wantUnitTestDBDir {
		t.Fatalf("unitTestDBTmpDir() = %q, want %q", gotUnitTestDBDir, wantUnitTestDBDir)
	}

	unitTmpDir, err := testingpathing.ResolveTestingTmpDir(workspaceRoot, tmpRoot, "unit")
	if err != nil {
		t.Fatalf("ResolveTestingTmpDir(unit): %v", err)
	}
	gotUnitRunRoot, err := unitTestRunRoot(context.Background(), modulesPath, tmpRoot, "auth")
	if err != nil {
		t.Fatalf("unitTestRunRoot(): %v", err)
	}
	if filepath.Dir(gotUnitRunRoot) != unitTmpDir {
		t.Fatalf("unitTestRunRoot() parent = %q, want %q", filepath.Dir(gotUnitRunRoot), unitTmpDir)
	}
	if !strings.HasPrefix(filepath.Base(gotUnitRunRoot), "auth-") {
		t.Fatalf("unitTestRunRoot() base = %q, want auth-<nanos> prefix", filepath.Base(gotUnitRunRoot))
	}
	if apiRoot := config.APIRootFromDist(filepath.Join(gotUnitRunRoot, "dist")); apiRoot != filepath.Join(gotUnitRunRoot, "api") {
		t.Fatalf("APIRootFromDist(isolated dist) = %q, want sibling api under run root", apiRoot)
	}

	quoted := quotePostgresIdent(`db"name`)
	if quoted != `"db""name"` {
		t.Fatalf("quotePostgresIdent returned %q", quoted)
	}

	name := makePostgresTestDBName("App Name-123")
	if !strings.HasPrefix(name, "choysum_test_") {
		t.Fatalf("makePostgresTestDBName returned %q", name)
	}
	if len(name) > 63 {
		t.Fatalf("expected postgres db name to fit limit, got len=%d name=%q", len(name), name)
	}

	longName := makePostgresTestDBName(strings.Repeat("verylongname", 10))
	if len(longName) > 63 {
		t.Fatalf("expected trimmed postgres db name, got len=%d name=%q", len(longName), longName)
	}
}

func TestPostgresDSNHelpers(t *testing.T) {
	t.Run("setPostgresDatabaseInDSN rejects empty dsn", func(t *testing.T) {
		_, err := setPostgresDatabaseInDSN("", "db")
		if err == nil || !strings.Contains(err.Error(), "empty dsn") {
			t.Fatalf("expected empty dsn error, got %v", err)
		}
	})

	t.Run("setPostgresDatabaseInDSN updates url dsn", func(t *testing.T) {
		got, err := setPostgresDatabaseInDSN("postgres://user:pass@localhost:5432/olddb?sslmode=disable", "newdb")
		if err != nil {
			t.Fatalf("setPostgresDatabaseInDSN returned error: %v", err)
		}
		if !strings.Contains(got, "/newdb?") {
			t.Fatalf("expected URL dsn to use new database, got %q", got)
		}
	})

	t.Run("setPostgresDatabaseInDSN updates key value dsn", func(t *testing.T) {
		got, err := setPostgresDatabaseInDSN("host=localhost dbname=olddb sslmode=disable", "newdb")
		if err != nil {
			t.Fatalf("setPostgresDatabaseInDSN returned error: %v", err)
		}
		if !strings.Contains(got, "dbname=newdb") {
			t.Fatalf("expected key value dsn to use new db, got %q", got)
		}
	})

	t.Run("setPostgresDatabaseInDSN appends dbname when missing", func(t *testing.T) {
		got, err := setPostgresDatabaseInDSN("host=localhost sslmode=disable", "newdb")
		if err != nil {
			t.Fatalf("setPostgresDatabaseInDSN returned error: %v", err)
		}
		if !strings.Contains(got, "dbname=newdb") {
			t.Fatalf("expected dbname to be appended, got %q", got)
		}
	})

	t.Run("splitPostgresKVDSN keeps quoted values together", func(t *testing.T) {
		tokens := splitPostgresKVDSN(`host=localhost password='two words' application_name="choysum test"`)
		want := []string{"host=localhost", "password='two words'", `application_name="choysum test"`}
		if strings.Join(tokens, "|") != strings.Join(want, "|") {
			t.Fatalf("splitPostgresKVDSN returned %#v, want %#v", tokens, want)
		}
	})
}

func TestDropPostgresDatabaseGuards(t *testing.T) {
	t.Run("empty admin dsn is a no-op", func(t *testing.T) {
		if err := dropPostgresDatabase(context.Background(), "", "choysum_test_app"); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("refuses non test databases", func(t *testing.T) {
		err := dropPostgresDatabase(context.Background(), "postgres://localhost/postgres", "prod")
		if err == nil || !strings.Contains(err.Error(), "refuse to drop non-test database") {
			t.Fatalf("expected refusal error, got %v", err)
		}
	})

	t.Run("nil context still attempts drop and returns connection error", func(t *testing.T) {
		err := dropPostgresDatabase(context.TODO(), "postgres://127.0.0.1:1/postgres?sslmode=disable", "choysum_test_app")
		if err == nil {
			t.Fatalf("expected non-nil error for unreachable postgres")
		}
	})
}

func TestNewCompilerExecutorBranches(t *testing.T) {
	t.Run("missing js engine factory returns create error", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			Server: &config.ServerConfig{JsEngineFactory: "missing-factory"},
		}}

		jsExec, err := newCompilerExecutor(runtimeScope)
		if jsExec != nil {
			t.Fatalf("expected nil executor on create error")
		}
		if err == nil || !strings.Contains(err.Error(), "create compiler executor") {
			t.Fatalf("expected wrapped create error, got %v", err)
		}
	})

	t.Run("engine load failure returns start error", func(t *testing.T) {
		engineName := testEngineName(t, "load-fail")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testFailLoadJsEngine{loadErr: errors.New("load boom")}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			Server: &config.ServerConfig{JsEngineFactory: engineName},
		}}

		jsExec, err := newCompilerExecutor(runtimeScope)
		if jsExec != nil {
			t.Fatalf("expected nil executor on start error")
		}
		if err == nil || !strings.Contains(err.Error(), "start compiler executor") || !strings.Contains(err.Error(), "load boom") {
			t.Fatalf("expected wrapped start error with load boom, got %v", err)
		}
	})

	t.Run("happy path starts and stops executor", func(t *testing.T) {
		engineName := testEngineName(t, "noop")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			Server: &config.ServerConfig{JsEngineFactory: engineName},
		}}

		jsExec, err := newCompilerExecutor(runtimeScope)
		if err != nil {
			t.Fatalf("expected successful compiler executor start, got %v", err)
		}
		if jsExec == nil {
			t.Fatalf("expected non-nil executor")
		}
		if err := jsExec.Stop(); err != nil {
			t.Fatalf("expected successful stop, got %v", err)
		}
	})
}

func TestNewRuntimeExecutorBranches(t *testing.T) {
	t.Run("missing js engine factory returns create error", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			Server: &config.ServerConfig{JsEngineFactory: "missing-factory"},
		}}

		jsExec, err := newRuntimeExecutor(runtimeScope, nil)
		if jsExec != nil {
			t.Fatalf("expected nil executor on create error")
		}
		if err == nil || !strings.Contains(err.Error(), "create runtime executor") {
			t.Fatalf("expected wrapped create error, got %v", err)
		}
	})

	t.Run("happy path creates startable executor", func(t *testing.T) {
		engineName := testEngineName(t, "runtime-noop")
		registerTestJsEngineFactory(engineName, func() (jsengine.JsEngine, error) {
			return &testNoopJsEngine{}, nil
		})

		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{
			Server: &config.ServerConfig{JsEngineFactory: engineName},
		}}

		jsExec, err := newRuntimeExecutor(runtimeScope, nil)
		if err != nil {
			t.Fatalf("expected successful runtime executor creation, got %v", err)
		}
		if jsExec == nil {
			t.Fatalf("expected non-nil executor")
		}
		if err := jsExec.Start(); err != nil {
			t.Fatalf("expected successful runtime executor start, got %v", err)
		}
		if err := jsExec.Stop(); err != nil {
			t.Fatalf("expected successful runtime executor stop, got %v", err)
		}
	})
}

func TestUnitTestRuntimeLogger(t *testing.T) {
	t.Run("drops js console passthrough noise when debug disabled", func(t *testing.T) {
		var buf bytes.Buffer
		baseLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

		unitTestRuntimeLogger(baseLogger).Warn("js console message",
			"emitter", jsConsoleEmitter,
			"passthrough", true,
			"console_text", "boom",
		)

		if got := buf.String(); got != "" {
			t.Fatalf("expected js console passthrough warning to be suppressed, got %q", got)
		}
	})

	t.Run("keeps non-js warnings when debug disabled", func(t *testing.T) {
		var buf bytes.Buffer
		baseLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

		unitTestRuntimeLogger(baseLogger).Warn("backend tests failed", "app", "auth")

		if got := buf.String(); !strings.Contains(got, "backend tests failed") {
			t.Fatalf("expected non-js warning to be preserved, got %q", got)
		}
	})

	t.Run("keeps js console passthrough noise when debug enabled", func(t *testing.T) {
		var buf bytes.Buffer
		baseLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		unitTestRuntimeLogger(baseLogger).Warn("js console message",
			"emitter", jsConsoleEmitter,
			"passthrough", true,
			"console_text", "boom",
		)

		if got := buf.String(); !strings.Contains(got, "js console message") {
			t.Fatalf("expected js console passthrough warning to be preserved with debug enabled, got %q", got)
		}
	})
}

func TestMakeTestScope(t *testing.T) {
	const envName = "unitest-backend-helper"
	scope.Register(envName, func(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
		var cfg *config.Config
		switch testInput := input.(type) {
		case testRuntimeScopeInput:
			cfg = scopetest.ConfigFromSnapshot(testInput.cfg)
		case *testRuntimeScopeInput:
			cfg = scopetest.ConfigFromSnapshot(testInput.cfg)
		}
		return &testStubScope{ctx: ctx, cfg: cfg}
	})

	baseDistPath := filepath.Join(t.TempDir(), "dist")
	if err := os.MkdirAll(baseDistPath, 0o755); err != nil {
		t.Fatalf("MkdirAll base DistPath: %v", err)
	}
	baseScope := &testStubScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath: filepath.Join(t.TempDir(), "modules"),
			DistPath:    baseDistPath,
			TmpPath:     filepath.Join(t.TempDir(), "tmp-root"),
			Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file:base.sqlite"},
			Server:      &config.ServerConfig{Environment: envName},
		},
	}

	t.Run("rejects invalid base scope", func(t *testing.T) {
		_, cleanup, err := makeTestScope(context.Background(), nil, "auth", "sqlite", "", "", false)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), "invalid base scope") {
			t.Fatalf("expected invalid base scope error, got %v", err)
		}
	})

	t.Run("rejects missing db config", func(t *testing.T) {
		_, cleanup, err := makeTestScope(context.Background(), &testStubScope{ctx: context.Background(), cfg: &config.Config{}}, "auth", "sqlite", "", "", false)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), "config missing db") {
			t.Fatalf("expected missing db config error, got %v", err)
		}
	})

	t.Run("rejects unsupported db dialect", func(t *testing.T) {
		_, cleanup, err := makeTestScope(context.Background(), baseScope, "auth", "mysql", "", "", false)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), "unsupported --db mysql") {
			t.Fatalf("expected unsupported dialect error, got %v", err)
		}
	})

	t.Run("requires postgres dsn", func(t *testing.T) {
		_, cleanup, err := makeTestScope(context.Background(), baseScope, "auth", "postgres", "", "", false)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), "requires --db-dsn") {
			t.Fatalf("expected postgres dsn error, got %v", err)
		}
	})

	t.Run("postgres admin connection failure is wrapped", func(t *testing.T) {
		_, cleanup, err := makeTestScope(context.Background(), baseScope, "auth", "postgres", "", "postgres://127.0.0.1:1/postgres?sslmode=disable", false)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), "connect postgres (admin)") {
			t.Fatalf("expected postgres admin connect error, got %v", err)
		}
	})

	t.Run("builds sqlite test scope and cleanup removes temp db", func(t *testing.T) {
		childScope, cleanup, err := makeTestScope(context.Background(), baseScope, "auth", "sqlite", "", "", false)
		if err != nil {
			t.Fatalf("makeTestScope returned error: %v", err)
		}
		childDBOpts, hasChildDBOpts := scope.DatabaseRuntimeOptionsFromScope(childScope)
		if childScope == nil || !hasChildDBOpts {
			t.Fatalf("expected initialized child scope, got %#v", childScope)
		}
		if childDBOpts.Dialect != "sqlite" {
			t.Fatalf("expected sqlite dialect, got %q", childDBOpts.Dialect)
		}
		const prefix = "file:"
		if !strings.HasPrefix(childDBOpts.DSN, prefix) {
			t.Fatalf("expected sqlite dsn, got %q", childDBOpts.DSN)
		}
		sqlitePath := strings.Split(strings.TrimPrefix(childDBOpts.DSN, prefix), "?")[0]
		baseOpts := runtimeOptionsFromScope(baseScope)
		wantTmpDir, err := unitTestDBTmpDir(context.Background(), baseOpts.modulesPath, baseOpts.tmpPath, "auth")
		if err != nil {
			t.Fatalf("unitTestDBTmpDir() error: %v", err)
		}
		if gotDir := filepath.Dir(sqlitePath); gotDir != wantTmpDir {
			t.Fatalf("expected sqlite path dir %q, got %q", wantTmpDir, gotDir)
		}
		if _, err := os.Stat(filepath.Dir(sqlitePath)); err != nil {
			t.Fatalf("expected sqlite temp dir to exist: %v", err)
		}

		childOpts := runtimeOptionsFromScope(childScope)
		baseDist := strings.TrimSpace(baseOpts.distPath)
		if childOpts.distPath == "" || childOpts.distPath == baseDist {
			t.Fatalf("expected isolated DistPath, got %q (base %q)", childOpts.distPath, baseDist)
		}
		runRoot := filepath.Dir(childOpts.distPath)
		if filepath.Base(childOpts.distPath) != "dist" {
			t.Fatalf("expected DistPath to end with /dist, got %q", childOpts.distPath)
		}
		if st, err := os.Stat(childOpts.distPath); err != nil || !st.IsDir() {
			t.Fatalf("expected isolated dist dir to exist: %v", err)
		}
		wantAPIRoot := filepath.Join(runRoot, "api")
		if got := config.APIRootFromDist(childOpts.distPath); got != wantAPIRoot {
			t.Fatalf("APIRootFromDist(isolated) = %q, want %q", got, wantAPIRoot)
		}
		marker := filepath.Join(childOpts.distPath, "marker.txt")
		if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile marker: %v", err)
		}

		if err := os.WriteFile(sqlitePath, []byte("db"), 0o644); err != nil {
			t.Fatalf("WriteFile sqlitePath: %v", err)
		}
		if err := os.WriteFile(sqlitePath+"-wal", []byte("wal"), 0o644); err != nil {
			t.Fatalf("WriteFile sqlitePath-wal: %v", err)
		}
		if err := os.WriteFile(sqlitePath+"-shm", []byte("shm"), 0o644); err != nil {
			t.Fatalf("WriteFile sqlitePath-shm: %v", err)
		}
		cleanup()
		if _, err := os.Stat(sqlitePath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected cleanup to remove sqlite path, stat err=%v", err)
		}
		if _, err := os.Stat(sqlitePath + "-wal"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected cleanup to remove sqlite wal path, stat err=%v", err)
		}
		if _, err := os.Stat(sqlitePath + "-shm"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected cleanup to remove sqlite shm path, stat err=%v", err)
		}
		if _, err := os.Stat(runRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected cleanup to remove isolated run root, stat err=%v", err)
		}
		if _, err := os.Stat(baseDist); err != nil {
			t.Fatalf("expected base DistPath to remain untouched, stat err=%v", err)
		}
	})

	t.Run("keep retains isolated dist run root", func(t *testing.T) {
		childScope, cleanup, err := makeTestScope(context.Background(), baseScope, "auth", "sqlite", "", "", true)
		if err != nil {
			t.Fatalf("makeTestScope returned error: %v", err)
		}
		childOpts := runtimeOptionsFromScope(childScope)
		runRoot := filepath.Dir(childOpts.distPath)
		marker := filepath.Join(childOpts.distPath, "keep-marker.txt")
		if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
			t.Fatalf("WriteFile marker: %v", err)
		}
		cleanup()
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("expected keep to retain isolated dist marker, got %v", err)
		}
		if _, err := os.Stat(runRoot); err != nil {
			t.Fatalf("expected keep to retain isolated run root, got %v", err)
		}
		_ = os.RemoveAll(runRoot)
	})

	t.Run("CLI bind uses shared home outside production choysum path", func(t *testing.T) {
		cliTmp := filepath.Join(t.TempDir(), "cli-test-tmp")
		t.Setenv(testingpathing.EnvCLITestTMP, cliTmp)
		productionHome := filepath.Join(t.TempDir(), ".choysum")
		baseWithHome := &testStubScope{
			ctx: context.Background(),
			cfg: &config.Config{
				ModulesPath:        filepath.Join(t.TempDir(), "modules"),
				DistPath:           filepath.Join(productionHome, "dist"),
				TmpPath:            filepath.Join(productionHome, "tmp"),
				DefaultChoysumPath: productionHome,
				Db:                 &config.DbConfig{Dialect: "sqlite", DSN: "file:base.sqlite"},
				Server:             &config.ServerConfig{Environment: envName},
			},
		}
		ctx, testTmp, runHome, err := testingpathing.BindCLITestRuntimePaths(context.Background(), filepath.Dir(baseWithHome.cfg.ModulesPath))
		if err != nil {
			t.Fatalf("BindCLITestRuntimePaths: %v", err)
		}
		if filepath.Clean(testTmp) != filepath.Clean(cliTmp) {
			t.Fatalf("testTmp = %q, want %q", testTmp, cliTmp)
		}
		childScope, cleanup, err := makeTestScope(ctx, baseWithHome, "auth", "sqlite", "", "", false)
		if err != nil {
			t.Fatalf("makeTestScope returned error: %v", err)
		}
		defer cleanup()

		pathOpts, ok := scope.PathsRuntimeOptionsFromScope(childScope)
		if !ok {
			t.Fatal("expected paths runtime options on child scope")
		}
		if pathOpts.DefaultChoysumPath != runHome {
			t.Fatalf("DefaultChoysumPath = %q, want shared run home %q", pathOpts.DefaultChoysumPath, runHome)
		}
		if pathOpts.DefaultChoysumPath == productionHome {
			t.Fatalf("DefaultChoysumPath must not stay on production home %q", productionHome)
		}
		if !strings.HasPrefix(filepath.Clean(pathOpts.DistPath), filepath.Clean(cliTmp)+string(filepath.Separator)) {
			t.Fatalf("DistPath = %q, want under CLI test tmp %q", pathOpts.DistPath, cliTmp)
		}
		if strings.HasPrefix(filepath.Clean(pathOpts.DistPath), filepath.Clean(productionHome)+string(filepath.Separator)) {
			t.Fatalf("DistPath must not be under production home: %q", pathOpts.DistPath)
		}
		if filepath.Base(runHome) != testingpathing.CLITestingRunHomeKind {
			t.Fatalf("run home base = %q, want %q", filepath.Base(runHome), testingpathing.CLITestingRunHomeKind)
		}
		pkgLink := filepath.Join(runHome, "pkg")
		if st, err := os.Lstat(pkgLink); err != nil {
			t.Fatalf("lstat home/pkg: %v", err)
		} else if st.Mode()&os.ModeSymlink == 0 {
			t.Fatal("expected home/pkg to be a symlink to the persistent CLI pkg cache")
		}
		pkgCache, err := testingpathing.ResolveCLITestingPkgCache(cliTmp)
		if err != nil {
			t.Fatalf("ResolveCLITestingPkgCache: %v", err)
		}
		target, err := os.Readlink(pkgLink)
		if err != nil {
			t.Fatalf("readlink home/pkg: %v", err)
		}
		if filepath.Clean(target) != filepath.Clean(pkgCache) {
			t.Fatalf("home/pkg target = %q, want %q", target, pkgCache)
		}
	})

	t.Run("forces sourcemap on and minify off for unit scope", func(t *testing.T) {
		baseWithMinify := &testStubScope{
			ctx: context.Background(),
			cfg: &config.Config{
				ModulesPath: filepath.Join(t.TempDir(), "modules"),
				DistPath:    filepath.Join(t.TempDir(), "dist"),
				TmpPath:     filepath.Join(t.TempDir(), "tmp-root"),
				Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file:base.sqlite"},
				Compile: &config.CompileConfig{
					BundleMode:  "application",
					SourceMap:   false,
					Minify:      true,
					TreeShaking: true,
				},
				Server: &config.ServerConfig{Environment: envName},
			},
		}

		childScope, cleanup, err := makeTestScope(context.Background(), baseWithMinify, "auth", "sqlite", "", "", false)
		if err != nil {
			t.Fatalf("makeTestScope returned error: %v", err)
		}
		defer cleanup()

		compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(childScope)
		if !hasCompileOpts {
			t.Fatalf("expected child scope compile runtime options")
		}
		if !compileOpts.SourceMap {
			t.Fatalf("expected child scope sourcemap=true, got false")
		}
		if compileOpts.Minify {
			t.Fatalf("expected child scope minify=false, got true")
		}
		if compileOpts.BundleMode != "application" {
			t.Fatalf("expected bundle mode to be preserved, got %q", compileOpts.BundleMode)
		}
		if !compileOpts.TreeShaking {
			t.Fatalf("expected tree shaking to be preserved, got false")
		}
	})

	t.Run("keeps explicit sqlite file when requested", func(t *testing.T) {
		keepPath := filepath.Join(t.TempDir(), "keep.sqlite")
		childScope, cleanup, err := makeTestScope(context.Background(), baseScope, "auth", "sqlite", keepPath, "", true)
		if err != nil {
			t.Fatalf("makeTestScope returned error: %v", err)
		}
		childDBOpts, hasChildDBOpts := scope.DatabaseRuntimeOptionsFromScope(childScope)
		childDSN := ""
		if hasChildDBOpts {
			childDSN = childDBOpts.DSN
		}
		if !hasChildDBOpts || !strings.Contains(childDSN, keepPath) {
			t.Fatalf("expected explicit sqlite path in dsn, got %q", childDSN)
		}
		if err := os.WriteFile(keepPath, []byte("db"), 0o644); err != nil {
			t.Fatalf("WriteFile keepPath: %v", err)
		}
		cleanup()
		if _, err := os.Stat(keepPath); err != nil {
			t.Fatalf("expected keep sqlite file to remain, got %v", err)
		}
	})

	t.Run("returns initialize test scope error when server environment missing", func(t *testing.T) {
		badBase := &testStubScope{
			ctx: context.Background(),
			cfg: &config.Config{
				ModulesPath: filepath.Join(t.TempDir(), "modules"),
				DistPath:    filepath.Join(t.TempDir(), "dist"),
				TmpPath:     filepath.Join(t.TempDir(), "tmp-root"),
				Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file:base.sqlite"},
				Server:      &config.ServerConfig{Environment: "missing-env-factory"},
			},
		}

		_, cleanup, err := makeTestScope(context.Background(), badBase, "auth", "sqlite", "", "", false)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to initialize test scope") {
			t.Fatalf("expected failed initialize test scope error, got %v", err)
		}
	})
}

func TestListTestFilesAndGeneratedEntry(t *testing.T) {
	serviceDir := filepath.Join(t.TempDir(), "service")

	files, err := listTestFiles(serviceDir)
	if err != nil {
		t.Fatalf("listTestFiles(non-existent) error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files for non-existent dir, got %d", len(files))
	}

	if err := os.MkdirAll(filepath.Join(serviceDir, "a"), 0o755); err != nil {
		t.Fatalf("mkdir service subdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(serviceDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "a", "ok.test.ts"), []byte("test"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "a", "not-test.ts"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write non-test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "node_modules", "ignored.test.ts"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write ignored test file: %v", err)
	}

	files, err = listTestFiles(serviceDir)
	if err != nil {
		t.Fatalf("listTestFiles error: %v", err)
	}
	if len(files) != 1 || !strings.HasSuffix(files[0], filepath.ToSlash("a/ok.test.ts")) && !strings.HasSuffix(files[0], filepath.Join("a", "ok.test.ts")) {
		t.Fatalf("unexpected discovered test files: %#v", files)
	}

	modulesPath := t.TempDir()
	app := "auth"
	serviceRoot := filepath.Join(modulesPath, app, "service")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("mkdir serviceRoot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "one.test.ts"), []byte("test1"), 0o644); err != nil {
		t.Fatalf("write one.test.ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "two.test.ts"), []byte("test2"), 0o644); err != nil {
		t.Fatalf("write two.test.ts: %v", err)
	}

	tmpRoot := filepath.Join(t.TempDir(), "tmp-root")
	runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath, DistPath: filepath.Join(t.TempDir(), "dist"), TmpPath: tmpRoot}}
	entry, cleanup, err := resolveOrGenerateTestsEntryPoint(context.Background(), runtimeScope, app)
	if err != nil {
		t.Fatalf("resolveOrGenerateTestsEntryPoint error: %v", err)
	}
	wantEntryDir, err := backendTestsIndexTmpDir(context.Background(), modulesPath, tmpRoot, app)
	if err != nil {
		t.Fatalf("backendTestsIndexTmpDir error: %v", err)
	}
	if filepath.Dir(entry) != wantEntryDir {
		t.Fatalf("generated entry dir = %q, want %q", filepath.Dir(entry), wantEntryDir)
	}
	content, err := os.ReadFile(entry)
	if err != nil {
		t.Fatalf("read generated entry: %v", err)
	}
	text := string(content)
	oneImport := "import '" + filepath.ToSlash(filepath.Join(serviceRoot, "one.test.ts")) + "';"
	twoImport := "import '" + filepath.ToSlash(filepath.Join(serviceRoot, "two.test.ts")) + "';"
	if !strings.Contains(text, oneImport) || !strings.Contains(text, twoImport) || !strings.Contains(text, "export async function Run") {
		t.Fatalf("unexpected generated entry content: %s", text)
	}
	cleanup()
	if _, err := os.Stat(entry); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected generated entry removed by cleanup, got err=%v", err)
	}
	if _, err := os.Stat(wantEntryDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected generated entry dir removed by cleanup when empty, got err=%v", err)
	}

	if err := os.Remove(filepath.Join(serviceRoot, "one.test.ts")); err != nil {
		t.Fatalf("remove one.test.ts: %v", err)
	}
	if err := os.Remove(filepath.Join(serviceRoot, "two.test.ts")); err != nil {
		t.Fatalf("remove two.test.ts: %v", err)
	}
	_, cleanup2, err := resolveOrGenerateTestsEntryPoint(context.Background(), runtimeScope, app)
	cleanup2()
	if err == nil || !strings.Contains(err.Error(), "no test files found") {
		t.Fatalf("expected no test files error, got %v", err)
	}
}

func TestBuildAppBundleEntryMissing(t *testing.T) {
	runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Compile: &config.CompileConfig{BundleMode: "bundle"}}}
	err := buildAppBundle(context.Background(), runtimeScope, nil, "auth", "tests.js", "__tests__", filepath.Join(t.TempDir(), "missing.ts"))
	if err == nil || !strings.Contains(err.Error(), "entry point not found") {
		t.Fatalf("expected missing entry point error, got %v", err)
	}
}

type bundleStub struct {
	bundleToDirErr error
	bundleErr      error
	buildErr       error
	called         string
}

func (s *bundleStub) BundleToDirCtx(ctx context.Context, distAppDir string) (*module.BuildResult, error) {
	s.called = "dir"
	return &module.BuildResult{}, s.bundleToDirErr
}

func (s *bundleStub) Bundle() (*module.BuildResult, error) {
	s.called = "bundle"
	return &module.BuildResult{}, s.bundleErr
}

func (s *bundleStub) Build() (*module.BuildResult, error) {
	s.called = "build"
	return &module.BuildResult{}, s.buildErr
}

type buildOnlyStub struct {
	err    error
	called bool
}

func (s *buildOnlyStub) Build() (*module.BuildResult, error) {
	s.called = true
	return &module.BuildResult{}, s.err
}

func TestBuildAppBundleWithInjectedBuilder(t *testing.T) {
	entry := filepath.Join(t.TempDir(), "entry.ts")
	if err := os.WriteFile(entry, []byte("export default {};"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	oldBuilder := newBackendBuilderHook
	defer func() { newBackendBuilderHook = oldBuilder }()

	t.Run("bundle mode tests.js uses BundleToDirCtx", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Compile: &config.CompileConfig{BundleMode: "bundle"}}}
		stub := &bundleStub{}
		newBackendBuilderHook = func(runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, mod *meta.IrModule, entryPoint, outFileName, globalName string) any {
			return stub
		}
		err := buildAppBundle(context.Background(), runtimeScope, nil, "auth", "tests.js", "__tests__", entry)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if stub.called != "dir" {
			t.Fatalf("expected BundleToDirCtx path, got %q", stub.called)
		}
	})

	t.Run("bundle to dir error is wrapped", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Compile: &config.CompileConfig{BundleMode: "bundle"}}}
		stub := &bundleStub{bundleToDirErr: errors.New("dir failed")}
		newBackendBuilderHook = func(runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, mod *meta.IrModule, entryPoint, outFileName, globalName string) any {
			return stub
		}
		err := buildAppBundle(context.Background(), runtimeScope, nil, "auth", "tests.js", "__tests__", entry)
		if err == nil || !strings.Contains(err.Error(), "to bundles dir") {
			t.Fatalf("expected bundle-to-dir error, got %v", err)
		}
	})

	t.Run("bundle mode tests.js without BundleToDirCtx returns explicit error", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Compile: &config.CompileConfig{BundleMode: "bundle"}}}
		newBackendBuilderHook = func(runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, mod *meta.IrModule, entryPoint, outFileName, globalName string) any {
			return struct{}{}
		}
		err := buildAppBundle(context.Background(), runtimeScope, nil, "auth", "tests.js", "__tests__", entry)
		if err == nil || !strings.Contains(err.Error(), "does not support BundleToDirCtx") {
			t.Fatalf("expected missing BundleToDirCtx error, got %v", err)
		}
	})

	t.Run("non-tests bundle path uses Bundle", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Compile: &config.CompileConfig{BundleMode: "bundle"}}}
		stub := &bundleStub{bundleErr: errors.New("bundle failed")}
		newBackendBuilderHook = func(runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, mod *meta.IrModule, entryPoint, outFileName, globalName string) any {
			return stub
		}
		err := buildAppBundle(context.Background(), runtimeScope, nil, "auth", "index.js", "auth", entry)
		if err == nil || !strings.Contains(err.Error(), "bundle auth (index.js)") {
			t.Fatalf("expected bundle error, got %v", err)
		}
		if stub.called != "bundle" {
			t.Fatalf("expected Bundle path, got %q", stub.called)
		}
	})

	t.Run("fallback build path uses Build", func(t *testing.T) {
		runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DistPath: t.TempDir(), Compile: &config.CompileConfig{BundleMode: "application"}}}
		stub := &buildOnlyStub{err: errors.New("build failed")}
		newBackendBuilderHook = func(runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, mod *meta.IrModule, entryPoint, outFileName, globalName string) any {
			return stub
		}
		err := buildAppBundle(context.Background(), runtimeScope, nil, "auth", "index.js", "auth", entry)
		if err == nil || !strings.Contains(err.Error(), "build auth (index.js)") {
			t.Fatalf("expected build error, got %v", err)
		}
		if !stub.called {
			t.Fatalf("expected fallback Build path")
		}
	})
}
