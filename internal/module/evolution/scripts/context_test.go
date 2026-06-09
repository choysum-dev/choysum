// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	modulemetadata "github.com/choysum-dev/choysum/internal/module/metadata"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type scriptsSelectiveEngine struct {
	loadedScripts []*jsengine.JsScript
	execute       func(req *jsengine.JsRequest, loaded []*jsengine.JsScript) (*jsengine.JsResponse, error)
}

func (e *scriptsSelectiveEngine) Load(scripts []*jsengine.JsScript) error {
	e.loadedScripts = append([]*jsengine.JsScript(nil), scripts...)
	return nil
}

func (e *scriptsSelectiveEngine) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.execute != nil {
		return e.execute(req, e.loadedScripts)
	}
	return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
}

func (e *scriptsSelectiveEngine) Close() error { return nil }

type scriptsTestScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
}

func (e *scriptsTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *scriptsTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *scriptsTestScope) Session() *scope.Session {
	if e == nil {
		return nil
	}
	if tx, ok := scope.TransactionFromContext(e.ctx); ok {
		if sess := tx.Session(); sess != nil {
			return sess
		}
	}
	if sess, ok := scope.SessionFromContext(e.ctx); ok {
		return sess
	}
	return e.session
}
func (e *scriptsTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	return &scriptsTestScope{ctx: ctx, cfg: e.cfg, logger: e.logger, session: e.session}
}
func (e *scriptsTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *scriptsTestScope) Logger() *slog.Logger   { return e.logger }
func (e *scriptsTestScope) Config() *config.Config { return e.cfg }

func (e *scriptsTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type scriptsFakeIdentity struct{ userID string }

func (i scriptsFakeIdentity) GetUserID() string                   { return i.userID }
func (i scriptsFakeIdentity) GetTokenID() string                  { return "token" }
func (i scriptsFakeIdentity) GetMetadata() map[string]interface{} { return nil }
func (i scriptsFakeIdentity) IsValid() bool                       { return strings.TrimSpace(i.userID) != "" }

func newScriptsTestScope(t *testing.T) *scriptsTestScope {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration_scripts.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&modulemetadata.IrModuleMigrationHistory{}); err != nil {
		t.Fatalf("migrate history table: %v", err)
	}
	return &scriptsTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath: filepath.Join(t.TempDir(), "modules"),
			DistPath:    filepath.Join(t.TempDir(), "dist"),
			Log:         config.NewDefaultLogConfig(),
			Db:          &config.DbConfig{Dialect: "sqlite"},
			Auth:        config.NewDefaultAuthConfig(),
			Server:      config.NewDefaultServerConfig(),
			Compile:     config.NewDefaultCompileConfig(),
		},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func newScriptsTestExecutorWithEngine(t *testing.T, testRuntimeScope *scriptsTestScope, engine jsengine.JsEngine) jsexecutor.JsExecutor {
	t.Helper()
	executor, err := jsexecutor.NewCompilerExecutor(testRuntimeScope, jsexecutor.WithJsEngine(func() (jsengine.JsEngine, error) {
		return engine, nil
	}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("start executor: %v", err)
	}
	t.Cleanup(func() {
		if err := executor.Stop(); err != nil {
			t.Fatalf("stop executor: %v", err)
		}
	})
	return executor
}

func writeScriptsRuntimeBundle(t *testing.T, testRuntimeScope *scriptsTestScope, content string) string {
	t.Helper()
	bundlePath := config.BundlesIndexJS(testRuntimeScope.cfg.DistPath)
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}
	if err := os.WriteFile(bundlePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	return bundlePath
}

func TestBuildJsContextAndExecContext(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	testRuntimeScope.cfg.Auth.InternalKey = "internal-secret"
	mod := &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.0.0", ServiceEntryPoint: "service/index.ts"}

	ctx := auth.ContextWithIdentity(context.Background(), scriptsFakeIdentity{userID: "u-1"})
	built := BuildJsContext(ctx, testRuntimeScope, mod, "1.0.0", "0.9.0")
	identity := built.payload["identity"].(map[string]any)
	if identity["userId"] != "u-1" {
		t.Fatalf("unexpected identity payload: %#v", built.payload)
	}
	modulePayload := built.payload["module"].(map[string]any)
	if modulePayload["name"] != "base" || modulePayload["fromVersion"] != "0.9.0" {
		t.Fatalf("unexpected module payload: %#v", modulePayload)
	}
	if strings.TrimSpace(built.requestId) == "" {
		t.Fatal("expected request id to be generated")
	}
	if _, ok := scope.SessionFromContext(built.execCtx); !ok {
		t.Fatal("expected session to be propagated")
	}
	if key, ok := auth.InternalKeyFromContext(built.execCtx); !ok || key != "internal-secret" {
		t.Fatalf("unexpected internal key in exec context: %q ok=%v", key, ok)
	}
	if md, ok := metadata.FromIncomingContext(built.execCtx); !ok || md.Get("x-choysum-depth")[0] != "0" {
		t.Fatalf("unexpected incoming metadata: %#v", md)
	}

	ctxWithToken := auth.ContextWithAccessToken(context.Background(), "token")
	execCtx := BuildExecContext(ctxWithToken, testRuntimeScope)
	if _, ok := auth.InternalKeyFromContext(execCtx); ok {
		t.Fatal("did not expect internal key injection when access token exists")
	}
}

func TestBuildExecContext_PrefersContextScope(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration_scripts_runtime.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	runtimeSession := &scope.Session{DB: runtimeDB}

	runtimeScope := &scriptsTestScope{ctx: context.Background(), cfg: testRuntimeScope.cfg, logger: testRuntimeScope.logger, session: runtimeSession}
	execCtx := BuildExecContext(scope.ContextWithScope(context.Background(), runtimeScope), testRuntimeScope)
	sess, ok := scope.SessionFromContext(execCtx)
	if !ok || sess != runtimeSession {
		t.Fatalf("expected runtime session to be preserved, got %#v ok=%v", sess, ok)
	}
}

func TestLoadRuntimeScripts(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	mod := &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.0.0", ServiceEntryPoint: "service/index.ts"}

	if _, err := LoadRuntimeScripts(nil, mod); err == nil {
		t.Fatal("expected missing env error")
	}
	if scripts, err := LoadRuntimeScripts(testRuntimeScope, &meta.IrModule{Name: "base"}); err != nil || scripts != nil {
		t.Fatalf("expected nil scripts for empty entrypoint, got %#v err=%v", scripts, err)
	}

	bundlePath := config.BundlesIndexJS(testRuntimeScope.cfg.DistPath)
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}
	if err := os.WriteFile(bundlePath, []byte("console.log('bundle')"), 0o644); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	scripts, err := LoadRuntimeScripts(testRuntimeScope, mod)
	if err != nil || len(scripts) != 1 || scripts[0].FileName != bundlePath {
		t.Fatalf("unexpected bundle scripts: %#v err=%v", scripts, err)
	}

	testRuntimeScope.cfg.Compile.BundleMode = "split"
	appPath := config.AppIndexJS(testRuntimeScope.cfg.DistPath, "core")
	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	if err := os.WriteFile(appPath, []byte("console.log('app')"), 0o644); err != nil {
		t.Fatalf("write app file: %v", err)
	}
	scripts, err = LoadRuntimeScripts(testRuntimeScope, mod)
	if err != nil || len(scripts) != 1 || scripts[0].FileName != appPath {
		t.Fatalf("unexpected split scripts: %#v err=%v", scripts, err)
	}
}
