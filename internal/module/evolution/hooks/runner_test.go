// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	modulegenerator "github.com/choysum-dev/choysum/internal/module/artifact/generate"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"google.golang.org/grpc/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type hooksTestScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
}

func (e *hooksTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *hooksTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *hooksTestScope) Session() *scope.Session {
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
func (e *hooksTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	return &hooksTestScope{ctx: ctx, cfg: e.cfg, logger: e.logger, session: e.session}
}
func (e *hooksTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *hooksTestScope) Logger() *slog.Logger   { return e.logger }
func (e *hooksTestScope) Config() *config.Config { return e.cfg }

func (e *hooksTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type fakeIdentity struct{ userID string }

func (i fakeIdentity) GetUserID() string                   { return i.userID }
func (i fakeIdentity) GetTokenID() string                  { return "token" }
func (i fakeIdentity) GetMetadata() map[string]interface{} { return nil }
func (i fakeIdentity) IsValid() bool                       { return strings.TrimSpace(i.userID) != "" }

type hooksSelectiveEngine struct {
	loadedScripts []*jsengine.JsScript
	execute       func(ctx context.Context, req *jsengine.JsRequest, loaded []*jsengine.JsScript) (*jsengine.JsResponse, error)
}

type hooksReloadCountingExecutor struct {
	inner interface {
		Execute(ctx context.Context, request *jsengine.JsRequest) (*jsengine.JsResponse, error)
		GetJsScripts() []*jsengine.JsScript
		SetJsScripts(scripts []*jsengine.JsScript)
		Reload(scripts ...*jsengine.JsScript) error
	}
	reloadCalls int
}

func (e *hooksReloadCountingExecutor) Execute(ctx context.Context, request *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return e.inner.Execute(ctx, request)
}

func (e *hooksReloadCountingExecutor) GetJsScripts() []*jsengine.JsScript {
	return e.inner.GetJsScripts()
}

func (e *hooksReloadCountingExecutor) SetJsScripts(scripts []*jsengine.JsScript) {
	e.inner.SetJsScripts(scripts)
}

func (e *hooksReloadCountingExecutor) Reload(scripts ...*jsengine.JsScript) error {
	e.reloadCalls++
	return e.inner.Reload(scripts...)
}

func (e *hooksSelectiveEngine) Load(scripts []*jsengine.JsScript) error {
	e.loadedScripts = append([]*jsengine.JsScript(nil), scripts...)
	return nil
}

func (e *hooksSelectiveEngine) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.execute != nil {
		return e.execute(ctx, req, e.loadedScripts)
	}
	return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
}

func (e *hooksSelectiveEngine) Close() error { return nil }

func newHooksTestScope(t *testing.T) *hooksTestScope {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hooks.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	distRoot := filepath.Join(t.TempDir(), "dist")
	return &hooksTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath: filepath.Join(t.TempDir(), "modules"),
			DistPath:    distRoot,
			Log:         config.NewDefaultLogConfig(),
			Db:          &config.DbConfig{Dialect: "sqlite"},
			Server:      config.NewDefaultServerConfig(),
			Auth:        config.NewDefaultAuthConfig(),
			Compile:     config.NewDefaultCompileConfig(),
		},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func newHooksTestExecutorWithEngine(t *testing.T, testRuntimeScope *hooksTestScope, engine jsengine.JsEngine) jsexecutor.JsExecutor {
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

func writeHooksRuntimeBundle(t *testing.T, testRuntimeScope *hooksTestScope, content string) string {
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

func TestParseHookError(t *testing.T) {
	code, msg := parseHookError(nil)
	if code != "" || msg != "" {
		t.Fatalf("expected empty result, got %q %q", code, msg)
	}

	code, msg = parseHookError(errTest("plain error"))
	if code != "" || msg != "plain error" {
		t.Fatalf("unexpected result: %q %q", code, msg)
	}

	code, msg = parseHookError(errTest("HOOK_TIMEOUT: exceeded"))
	if code != "HOOK_TIMEOUT" || msg != "exceeded" {
		t.Fatalf("unexpected result: %q %q", code, msg)
	}
}

func TestDefaultRequired(t *testing.T) {
	if !defaultRequired(PhasePreInit) {
		t.Fatalf("expected pre_init required")
	}
	if !defaultRequired(PhasePreUpgrade) {
		t.Fatalf("expected pre_upgrade required")
	}
	if !defaultRequired(PhasePreUninstall) {
		t.Fatalf("expected pre_uninstall required")
	}
	if defaultRequired(PhasePostInit) {
		t.Fatalf("expected post_init not required")
	}
	if defaultRequired(PhasePostUpgrade) {
		t.Fatalf("expected post_upgrade not required")
	}
	if defaultRequired(PhasePostUninstall) {
		t.Fatalf("expected post_uninstall not required")
	}
}

func TestDefaultTimeout(t *testing.T) {
	if got := defaultTimeout(PhasePreInit); got != 30*time.Second {
		t.Fatalf("expected 30s, got %s", got)
	}
	if got := defaultTimeout(PhasePreUpgrade); got != 30*time.Second {
		t.Fatalf("expected 30s, got %s", got)
	}
	if got := defaultTimeout(PhasePreUninstall); got != 30*time.Second {
		t.Fatalf("expected 30s, got %s", got)
	}
	if got := defaultTimeout(PhasePostInit); got != 60*time.Second {
		t.Fatalf("expected 60s, got %s", got)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestNewRunnerAndNormalizeConfig(t *testing.T) {
	if runner, err := NewRunner(nil, nil, nil); err != nil || runner != nil {
		t.Fatalf("expected nil runner for nil env/module, got %#v err=%v", runner, err)
	}

	testRuntimeScope := newHooksTestScope(t)
	mod := &meta.IrModule{Name: "base", ApplicationStr: "core"}
	runner, err := NewRunner(testRuntimeScope, nil, mod)
	if err != nil || runner == nil {
		t.Fatalf("expected runner, got %#v err=%v", runner, err)
	}

	preCfg := normalizeConfig(PhasePreInit)
	if !preCfg.Required || preCfg.Timeout <= 0 || preCfg.Retry != 0 {
		t.Fatalf("unexpected pre-init config: %#v", preCfg)
	}
	postCfg := normalizeConfig(PhasePostInit)
	if postCfg.Required || postCfg.Timeout != 60*1000000000 {
		t.Fatalf("unexpected post-init config: %#v", postCfg)
	}
}

func TestRunnerBuildScriptsAndContexts(t *testing.T) {
	testRuntimeScope := newHooksTestScope(t)
	testRuntimeScope.cfg.Auth.InternalKey = "internal-secret"
	runner := &Runner{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.0.0"}}

	if script := runner.buildHookEnvScript(); script == nil || !strings.Contains(script.Content, `CHOYSUM_MODULE_NAME = "base"`) {
		t.Fatalf("unexpected hook env script: %#v", script)
	}
	if script := runner.buildHookWrapperScript(); script == nil || !strings.Contains(script.Content, `__choysum_hook__ = async function (app, moduleName, phase)`) || !strings.Contains(script.Content, `HOOK_UNSUPPORTED`) {
		t.Fatalf("unexpected hook wrapper script: %#v", script)
	}
	runner.module.ApplicationStr = ""
	runner.module.Name = ""
	if script := runner.buildHookEnvScript(); script != nil {
		t.Fatalf("expected nil hook env script when module name is missing, got %#v", script)
	}
	if script := runner.buildHookWrapperScript(); script == nil {
		t.Fatalf("expected generic hook wrapper script when module info is missing")
	}

	runner.module.Name = "base"
	runner.module.ApplicationStr = "core"
	ctx := auth.ContextWithIdentity(context.Background(), fakeIdentity{userID: "u-1"})
	jsCtx := runner.buildJsContext(ctx, RunOptions{FromVersion: "0.9.0"})
	identity, ok := jsCtx.payload["identity"].(map[string]any)
	if !ok || identity["userId"] != "u-1" {
		t.Fatalf("unexpected js identity payload: %#v", jsCtx.payload)
	}
	moduleInfo := jsCtx.payload["module"].(map[string]any)
	if moduleInfo["fromVersion"] != "0.9.0" || moduleInfo["name"] != "base" {
		t.Fatalf("unexpected js module payload: %#v", moduleInfo)
	}
	if strings.TrimSpace(jsCtx.requestId) == "" {
		t.Fatal("expected request id to be generated")
	}

	execCtx := runner.buildExecContext(context.Background())
	if _, ok := scope.SessionFromContext(execCtx); !ok {
		t.Fatal("expected session to be propagated into execution context")
	}
	if key, ok := auth.InternalKeyFromContext(execCtx); !ok || key != "internal-secret" {
		t.Fatalf("unexpected internal key in exec context: %q ok=%v", key, ok)
	}
	md, ok := metadata.FromIncomingContext(execCtx)
	if !ok || md.Get("x-choysum-depth")[0] != "0" {
		t.Fatalf("unexpected grpc metadata: %#v", md)
	}

	ctxWithToken := auth.ContextWithAccessToken(context.Background(), "token")
	execCtx = runner.buildExecContext(ctxWithToken)
	if _, ok := auth.InternalKeyFromContext(execCtx); ok {
		t.Fatal("did not expect internal key to be injected when access token exists")
	}
}

func TestRunnerBuildExecContext_PrefersContextScope(t *testing.T) {
	testRuntimeScope := newHooksTestScope(t)
	runner := &Runner{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.0.0"}}
	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hooks_runtime.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	runtimeSession := &scope.Session{DB: runtimeDB}

	runtimeScope := &hooksTestScope{ctx: context.Background(), cfg: testRuntimeScope.cfg, logger: testRuntimeScope.logger, session: runtimeSession}
	execCtx := runner.buildExecContext(scope.ContextWithScope(context.Background(), runtimeScope))
	sess, ok := scope.SessionFromContext(execCtx)
	if !ok || sess != runtimeSession {
		t.Fatalf("expected runtime session to be preserved, got %#v ok=%v", sess, ok)
	}
}

func TestResolveScriptsLoadDistScriptsAndBuildModuleEntryScript(t *testing.T) {
	testRuntimeScope := newHooksTestScope(t)
	runner := &Runner{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}}
	provided := []*jsengine.JsScript{{FileName: "provided.js", Content: "console.log(1)"}}
	scripts, err := runner.resolveScripts(context.Background(), PhasePostInit, RunOptions{Scripts: provided}, false)
	if err != nil || len(scripts) != 1 || scripts[0].FileName != "provided.js" {
		t.Fatalf("resolveScripts(provided) = %#v err=%v", scripts, err)
	}

	runner.module.ServiceEntryPoint = ""
	if script, err := runner.buildModuleEntryScript(context.Background()); err != nil || script != nil {
		t.Fatalf("expected nil buildModuleEntryScript for empty entry, got %#v err=%v", script, err)
	}
	if scripts, err := runner.resolveScripts(context.Background(), PhasePostInit, RunOptions{}, false); err != nil || scripts != nil {
		t.Fatalf("expected nil resolveScripts when nothing is available, got %#v err=%v", scripts, err)
	}

	runner.module.ServiceEntryPoint = "service/index.ts"
	if _, err := runner.resolveScripts(context.Background(), PhasePreInit, RunOptions{}, true); err == nil || !strings.Contains(err.Error(), "required hook phase pre_init") {
		t.Fatalf("expected required phase build failure, got %v", err)
	}

	if _, err := LoadDistScripts(nil, runner.module); err == nil {
		t.Fatal("expected LoadDistScripts to reject missing env")
	}
	if scripts, err := LoadDistScripts(testRuntimeScope, &meta.IrModule{Name: "base"}); err != nil || scripts != nil {
		t.Fatalf("expected nil scripts when service entry point is empty, got %#v err=%v", scripts, err)
	}

	bundlePath := config.BundlesIndexJS(testRuntimeScope.cfg.DistPath)
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}
	if err := os.WriteFile(bundlePath, []byte("console.log('bundle')"), 0o644); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	mod := &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}
	scripts, err = LoadDistScripts(testRuntimeScope, mod)
	if err != nil || len(scripts) != 1 || scripts[0].FileName != bundlePath {
		t.Fatalf("unexpected bundled scripts: %#v err=%v", scripts, err)
	}

	testRuntimeScope.cfg.Compile.BundleMode = "split"
	appPath := config.AppIndexJS(testRuntimeScope.cfg.DistPath, "core")
	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	if err := os.WriteFile(appPath, []byte("console.log('app')"), 0o644); err != nil {
		t.Fatalf("write app file: %v", err)
	}
	scripts, err = LoadDistScripts(testRuntimeScope, mod)
	if err != nil || len(scripts) != 1 || scripts[0].FileName != appPath {
		t.Fatalf("unexpected split scripts: %#v err=%v", scripts, err)
	}
}

func TestBuildModuleEntryScript_PrefersContextSessionForBuilderRuntimeState(t *testing.T) {
	testRuntimeScope := newHooksTestScope(t)
	testRuntimeScope.cfg.DefaultChoysumPath = filepath.Join(t.TempDir(), ".choysum")
	if err := os.MkdirAll(testRuntimeScope.cfg.ModulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testRuntimeScope.cfg.ModulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":".","paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	entryPoint := filepath.Join(testRuntimeScope.cfg.ModulesPath, "base", "service", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entryPoint), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPoint, []byte("export const hook = {};\n"), 0o644); err != nil {
		t.Fatalf("write entry point: %v", err)
	}

	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hooks_runtime_builder.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	if err := runtimeDB.AutoMigrate(&meta.IrModule{}); err != nil {
		t.Fatalf("migrate runtime modules: %v", err)
	}
	_, _, serviceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "crm", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(crm) error = %v", err)
	}
	serviceIndex := filepath.Join(serviceDir, "index.ts")
	if err := os.MkdirAll(filepath.Dir(serviceIndex), 0o755); err != nil {
		t.Fatalf("mkdir service index dir: %v", err)
	}
	if err := os.WriteFile(serviceIndex, []byte("globalThis.__hookInjected = (globalThis.__hookInjected || 0) + 1;\nexport {};\n"), 0o644); err != nil {
		t.Fatalf("write service index: %v", err)
	}
	if err := runtimeDB.Create(&meta.IrModule{Name: "crm_mod", ApplicationStr: "crm", Status: meta.Installed, ServiceEntryPoint: "./service/index.ts"}).Error; err != nil {
		t.Fatalf("seed runtime installed module: %v", err)
	}

	runner := &Runner{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}}
	baseScript, err := runner.buildModuleEntryScript(context.Background())
	if err != nil {
		t.Fatalf("buildModuleEntryScript(base) error = %v", err)
	}
	runtimeScope := &hooksTestScope{ctx: context.Background(), cfg: testRuntimeScope.cfg, logger: testRuntimeScope.logger, session: &scope.Session{DB: runtimeDB}}
	runtimeCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	runtimeScript, err := runner.buildModuleEntryScript(runtimeCtx)
	if err != nil {
		t.Fatalf("buildModuleEntryScript(runtime) error = %v", err)
	}
	if baseScript == nil || runtimeScript == nil {
		t.Fatalf("expected both scripts to be built, base=%#v runtime=%#v", baseScript, runtimeScript)
	}
	if strings.Contains(baseScript.Content, "__hookInjected") {
		t.Fatalf("expected base build to skip runtime-only import, got: %s", baseScript.Content)
	}
	if !strings.Contains(runtimeScript.Content, "__hookInjected") {
		t.Fatalf("expected runtime build to include runtime-only import, got: %s", runtimeScript.Content)
	}
}

func TestScriptFromBuildResult(t *testing.T) {
	if script, err := ScriptFromBuildResult(nil); err != nil || script != nil {
		t.Fatalf("expected nil script for nil result, got %#v err=%v", script, err)
	}
	result := &module.BuildResult{EsbuildResult: &api.BuildResult{OutputFiles: []api.OutputFile{{Path: "chunk.css", Contents: []byte("body{}")}, {Path: "other.js", Contents: []byte("console.log('x')")}, {Path: "index.js", Contents: []byte("console.log('index')")}}}}
	script, err := ScriptFromBuildResult(result)
	if err != nil || script == nil || script.FileName != "index.js" || !strings.Contains(script.Content, "index") {
		t.Fatalf("unexpected selected script: %#v err=%v", script, err)
	}
}

func TestRunnerRunPhaseExecutionPaths(t *testing.T) {
	if err := (*Runner)(nil).RunPhase(context.Background(), PhasePreInit, RunOptions{}); err != nil {
		t.Fatalf("expected nil runner RunPhase to be no-op, got %v", err)
	}

	t.Run("executes hook with runtime scripts and restores executor state", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		bundlePath := writeHooksRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
		module := &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.0.0", ServiceEntryPoint: "service/index.ts"}

		var loadedByService = map[string][]string{}
		var requests []*jsengine.JsRequest
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, req *jsengine.JsRequest, loaded []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			fileNames := make([]string, 0, len(loaded))
			for _, script := range loaded {
				fileNames = append(fileNames, script.FileName)
			}
			loadedByService[req.Service] = append([]string(nil), fileNames...)
			requests = append(requests, req)
			return &jsengine.JsResponse{Id: req.Id, Result: nil}, nil
		}}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		prevScripts := []*jsengine.JsScript{{FileName: "prev.js", Content: "prev"}}
		executor.SetJsScripts(prevScripts)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: module}

		err := runner.RunPhase(context.Background(), PhasePostInit, RunOptions{FromVersion: "0.9.0"})
		if err != nil {
			t.Fatalf("RunPhase() error = %v", err)
		}
		if !reflect.DeepEqual(executor.GetJsScripts(), prevScripts) {
			t.Fatalf("expected scripts restored, got %#v", executor.GetJsScripts())
		}
		if got := loadedByService["__choysum_hook__"]; !reflect.DeepEqual(got, []string{bundlePath, "hook_wrapper.js"}) {
			t.Fatalf("unexpected loaded scripts: %#v", got)
		}
		if len(requests) != 1 || requests[0].Service != "__choysum_hook__" {
			t.Fatalf("unexpected requests: %#v", requests)
		}
		moduleInfo, _ := requests[0].Context["module"].(map[string]any)
		if moduleInfo["fromVersion"] != "0.9.0" {
			t.Fatalf("unexpected request module payload: %#v", moduleInfo)
		}
	})

	t.Run("fails fast on required phase when module-entry build fails", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		engine := &hooksSelectiveEngine{}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}}

		err := runner.RunPhase(context.Background(), PhasePreInit, RunOptions{})
		if err == nil || !strings.Contains(err.Error(), "required hook phase pre_init") {
			t.Fatalf("expected required phase fail-fast build error, got %v", err)
		}
	})

	t.Run("rejects nil js executor", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		runner := &Runner{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "base", ApplicationStr: "core"}}
		if err := runner.RunPhase(context.Background(), PhasePreInit, RunOptions{}); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
			t.Fatalf("expected js executor error, got %v", err)
		}
	})

	t.Run("ignores non-required hook failure", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		writeHooksRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			return nil, errors.New("hook boom")
		}}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}}

		if err := runner.RunPhase(context.Background(), PhasePostInit, RunOptions{}); err != nil {
			t.Fatalf("expected post hook failure to be ignored, got %v", err)
		}
	})

	t.Run("returns required hook failure", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			return nil, errors.New("HOOK_UNSUPPORTED: missing pre hook")
		}}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}}

		err := runner.RunPhase(context.Background(), PhasePreInit, RunOptions{Scripts: []*jsengine.JsScript{{FileName: "provided.js", Content: ""}}})
		if err == nil || !strings.Contains(err.Error(), "HOOK_UNSUPPORTED") {
			t.Fatalf("expected required hook error, got %v", err)
		}
	})
}

func TestExecuteWithScriptsHandlesFailureModes(t *testing.T) {
	t.Run("treats ordinary hook exception as retriable", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			return nil, errors.New("hook boom")
		}}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		prevScripts := []*jsengine.JsScript{{FileName: "prev.js", Content: "prev"}}
		executor.SetJsScripts(prevScripts)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: &meta.IrModule{Name: "base", ApplicationStr: "core"}}
		lastErr := error(nil)

		err := runner.executeWithScripts(context.Background(), []*jsengine.JsScript{{FileName: "hook.js", Content: "1"}}, &jsengine.JsRequest{Id: "1", Service: "__choysum_hook__"}, &lastErr, false)
		if !errors.Is(err, errHookRetriable) {
			t.Fatalf("expected retriable error, got %v", err)
		}
		if lastErr == nil || !strings.Contains(lastErr.Error(), "HOOK_EXCEPTION: hook boom") {
			t.Fatalf("unexpected lastErr: %v", lastErr)
		}
		if !reflect.DeepEqual(executor.GetJsScripts(), prevScripts) {
			t.Fatalf("expected scripts restored, got %#v", executor.GetJsScripts())
		}
	})

	t.Run("treats canceled context as timeout", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			return nil, context.Canceled
		}}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: &meta.IrModule{Name: "base", ApplicationStr: "core"}}
		lastErr := error(nil)

		err := runner.executeWithScripts(context.Background(), nil, &jsengine.JsRequest{Id: "1", Service: "__choysum_hook__"}, &lastErr, false)
		if !errors.Is(err, errHookRetriable) {
			t.Fatalf("expected retriable timeout, got %v", err)
		}
		if lastErr == nil || !strings.Contains(lastErr.Error(), "HOOK_TIMEOUT") {
			t.Fatalf("unexpected lastErr: %v", lastErr)
		}
	})

	t.Run("returns non-retriable unsupported hook error", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			return nil, errors.New("HOOK_UNSUPPORTED: missing hook")
		}}
		executor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: executor, module: &meta.IrModule{Name: "base", ApplicationStr: "core"}}
		lastErr := error(nil)

		err := runner.executeWithScripts(context.Background(), nil, &jsengine.JsRequest{Id: "1", Service: "__choysum_hook__"}, &lastErr, false)
		if err == nil || errors.Is(err, errHookRetriable) || !strings.Contains(err.Error(), "HOOK_UNSUPPORTED") {
			t.Fatalf("expected non-retriable unsupported error, got %v", err)
		}
		if lastErr == nil || !strings.Contains(lastErr.Error(), "HOOK_UNSUPPORTED") {
			t.Fatalf("unexpected lastErr: %v", lastErr)
		}
	})

	t.Run("reuses loaded scripts when requested", func(t *testing.T) {
		testRuntimeScope := newHooksTestScope(t)
		engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{Id: "1", Result: nil}, nil
		}}
		baseExecutor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
		countingExecutor := &hooksReloadCountingExecutor{inner: baseExecutor}
		runner := &Runner{runtimeScope: testRuntimeScope, jsExecutor: countingExecutor, module: &meta.IrModule{Name: "base", ApplicationStr: "core"}}
		lastErr := error(nil)
		scripts := []*jsengine.JsScript{{FileName: "hook.js", Content: "1"}}

		if err := runner.executeWithScripts(context.Background(), scripts, &jsengine.JsRequest{Id: "1", Service: "__choysum_hook__"}, &lastErr, true); err != nil {
			t.Fatalf("first executeWithScripts error = %v", err)
		}
		if err := runner.executeWithScripts(context.Background(), scripts, &jsengine.JsRequest{Id: "2", Service: "__choysum_hook__"}, &lastErr, true); err != nil {
			t.Fatalf("second executeWithScripts error = %v", err)
		}
		if countingExecutor.reloadCalls != 1 {
			t.Fatalf("expected one reload for reused scripts, got %d", countingExecutor.reloadCalls)
		}
	})
}
