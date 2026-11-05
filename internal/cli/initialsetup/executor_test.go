// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/module/evolution/systemidentity"
	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type executorTestScope struct {
	ctx    context.Context
	db     *gorm.DB
	config *config.Config
}

func (e *executorTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *executorTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *executorTestScope) Session() *scope.Session { return &scope.Session{DB: e.db} }

func (e *executorTestScope) WithContext(ctx context.Context) scope.Scope {
	return &executorTestScope{ctx: ctx, db: e.db, config: e.config}
}

func (e *executorTestScope) Context() context.Context { return e.ctx }

func (e *executorTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *executorTestScope) Config() *config.Config { return e.config }

func (e *executorTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.config)
}

func newExecutorTestScope(t *testing.T, distPath string, bundleMode string) *executorTestScope {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "setup.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	cfg := &config.Config{DistPath: distPath}
	if bundleMode != "" {
		cfg.Compile = &config.CompileConfig{BundleMode: bundleMode}
	}
	return &executorTestScope{ctx: context.Background(), db: db, config: cfg}
}

type fakeJsExecutor struct {
	scripts  []*jsengine.JsScript
	startErr error
	stopped  int
	execErr  error
	resp     *jsengine.JsResponse
	requests []*jsengine.JsRequest
	execCtx  context.Context
}

func (f *fakeJsExecutor) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	f.execCtx = ctx
	f.requests = append(f.requests, req)
	return f.resp, f.execErr
}

func (f *fakeJsExecutor) GetJsScripts() []*jsengine.JsScript            { return f.scripts }
func (f *fakeJsExecutor) SetJsScripts(scripts []*jsengine.JsScript)     { f.scripts = scripts }
func (f *fakeJsExecutor) Reload(...*jsengine.JsScript) error            { return nil }
func (f *fakeJsExecutor) AppendJsScripts(scripts ...*jsengine.JsScript) {}
func (f *fakeJsExecutor) Start() error                                  { return f.startErr }

func (f *fakeJsExecutor) Stop() error {
	f.stopped++
	return nil
}

type planCall struct {
	distRoot   string
	bundleMode string
}

func stubPlan(t *testing.T, decision runplan.RunDecision, err error) *[]planCall {
	t.Helper()
	calls := &[]planCall{}
	prev := planServe
	planServe = func(distRoot string, bundleMode string, _ *slog.Logger, _ []string) (*distmanifest.DistManifestV2, runplan.RunDecision, error) {
		*calls = append(*calls, planCall{distRoot: distRoot, bundleMode: bundleMode})
		return nil, decision, err
	}
	t.Cleanup(func() { planServe = prev })
	return calls
}

func stubExecutor(t *testing.T, executor *fakeJsExecutor, err error) {
	t.Helper()
	prev := newRuntimeExecutor
	newRuntimeExecutor = func(scope.Scope, auth.Authenticator, ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		if err != nil {
			return nil, err
		}
		return executor, nil
	}
	t.Cleanup(func() { newRuntimeExecutor = prev })
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func scriptNames(scripts []*jsengine.JsScript) []string {
	out := make([]string, 0, len(scripts))
	for _, s := range scripts {
		out = append(out, s.FileName)
	}
	return out
}

func TestLoadServeScriptsRequiresDistPath(t *testing.T) {
	if _, err := loadServeScripts(newExecutorTestScope(t, " ", "")); err == nil || !strings.Contains(err.Error(), "dist path is not configured") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadServeScriptsEmptyDistIsNotReady(t *testing.T) {
	if _, err := loadServeScripts(newExecutorTestScope(t, t.TempDir(), "")); err == nil || !strings.Contains(err.Error(), "not ready to serve") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadServeScriptsPlanError(t *testing.T) {
	stubPlan(t, runplan.RunDecision{}, errors.New("bad manifest"))
	if _, err := loadServeScripts(newExecutorTestScope(t, t.TempDir(), "")); err == nil || err.Error() != "bad manifest" {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadServeScriptsBundleMode(t *testing.T) {
	dist := t.TempDir() + " "
	calls := stubPlan(t, runplan.RunDecision{RunMode: runplan.RunModeApplication, CompileBundleMode: " Bundle "}, nil)
	writeFile(t, config.BundlesIndexJS(dist), "globalThis.bundle = 1")

	scripts, err := loadServeScripts(newExecutorTestScope(t, dist, ""))
	if err != nil {
		t.Fatalf("loadServeScripts: %v", err)
	}
	if got := scriptNames(scripts); len(got) != 1 || got[0] != config.BundlesIndexJS(dist) || scripts[0].Content != "globalThis.bundle = 1" {
		t.Fatalf("scripts = %v", got)
	}
	want := planCall{distRoot: dist, bundleMode: config.NewDefaultCompileConfig().BundleMode}
	if len(*calls) != 1 || (*calls)[0] != want {
		t.Fatalf("plan calls = %+v, want %+v", *calls, want)
	}
}

func TestLoadServeScriptsAppModeSkipsMissingApps(t *testing.T) {
	dist := t.TempDir()
	calls := stubPlan(t, runplan.RunDecision{RunMode: runplan.RunModeApplication, CompileBundleMode: "app", ServeTargets: []string{"base", "auth"}}, nil)
	writeFile(t, config.AppIndexJS(dist, "base"), "globalThis.base = 1")

	scripts, err := loadServeScripts(newExecutorTestScope(t, dist, "app"))
	if err != nil {
		t.Fatalf("loadServeScripts: %v", err)
	}
	if got := scriptNames(scripts); len(got) != 1 || got[0] != config.AppIndexJS(dist, "base") {
		t.Fatalf("scripts = %v", got)
	}
	if (*calls)[0].bundleMode != "app" {
		t.Fatalf("bundle mode = %q, want scope compile option", (*calls)[0].bundleMode)
	}
}

func TestLoadServeScriptsReadError(t *testing.T) {
	dist := t.TempDir()
	stubPlan(t, runplan.RunDecision{RunMode: runplan.RunModeApplication, CompileBundleMode: "app", ServeTargets: []string{"base"}}, nil)
	if err := os.MkdirAll(config.AppIndexJS(dist, "base"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := loadServeScripts(newExecutorTestScope(t, dist, "")); err == nil {
		t.Fatal("unreadable index.js must fail")
	}
}

func readyBundleDist(t *testing.T) string {
	t.Helper()
	dist := t.TempDir()
	stubPlan(t, runplan.RunDecision{RunMode: runplan.RunModeApplication, CompileBundleMode: "bundle"}, nil)
	writeFile(t, config.BundlesIndexJS(dist), "globalThis.bundle = 1")
	return dist
}

func TestOpenExecutorCallerStartsAndStops(t *testing.T) {
	runtimeScope := newExecutorTestScope(t, readyBundleDist(t), "")
	executor := &fakeJsExecutor{}
	stubExecutor(t, executor, nil)

	caller, closeCaller, err := openExecutorCaller(runtimeScope)
	if err != nil {
		t.Fatalf("openExecutorCaller: %v", err)
	}
	if got := scriptNames(executor.scripts); len(got) != 1 {
		t.Fatalf("executor scripts = %v", got)
	}
	if ec, ok := caller.(executorCaller); !ok || ec.executor != executor || ec.scope != runtimeScope {
		t.Fatalf("caller = %#v", caller)
	}
	closeCaller()
	if executor.stopped != 1 {
		t.Fatalf("stopped = %d", executor.stopped)
	}
}

func TestOpenExecutorCallerErrors(t *testing.T) {
	t.Run("scripts", func(t *testing.T) {
		stubExecutor(t, &fakeJsExecutor{}, nil)
		if _, _, err := openExecutorCaller(newExecutorTestScope(t, "", "")); err == nil {
			t.Fatal("missing dist must fail")
		}
	})
	t.Run("create", func(t *testing.T) {
		runtimeScope := newExecutorTestScope(t, readyBundleDist(t), "")
		stubExecutor(t, nil, errors.New("no engine"))
		if _, _, err := openExecutorCaller(runtimeScope); err == nil || err.Error() != "no engine" {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("start", func(t *testing.T) {
		runtimeScope := newExecutorTestScope(t, readyBundleDist(t), "")
		executor := &fakeJsExecutor{startErr: errors.New("boot failed")}
		stubExecutor(t, executor, nil)
		if _, _, err := openExecutorCaller(runtimeScope); err == nil || err.Error() != "boot failed" {
			t.Fatalf("err = %v", err)
		}
		if executor.stopped != 1 {
			t.Fatalf("executor must be stopped after a failed start, stopped = %d", executor.stopped)
		}
	})
}

func TestScopeDepsReadsWorkspace(t *testing.T) {
	runtimeScope := newExecutorTestScope(t, "", "")
	deps := ScopeDeps(runtimeScope)
	if phase, err := deps.Phase(); err != nil || phase != provision.PhaseEmpty {
		t.Fatalf("Phase = %q, %v", phase, err)
	}
	if mode, err := deps.SetupMode(); err != nil || mode != "" {
		t.Fatalf("SetupMode = %q, %v", mode, err)
	}
	if _, _, err := deps.OpenCaller(); err == nil {
		t.Fatal("OpenCaller without dist must fail")
	}
}

func TestExecutorCallerBuildsSystemRequest(t *testing.T) {
	runtimeScope := newExecutorTestScope(t, "", "")
	executor := &fakeJsExecutor{resp: &jsengine.JsResponse{Result: map[string]any{"ok": true}}}
	caller := executorCaller{executor: executor, scope: runtimeScope}

	got, err := caller.Call(context.Background(), serviceSetupOptions, "zh_CN", map[string]any{})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got.(map[string]any)["ok"] != true {
		t.Fatalf("result = %v", got)
	}
	req := executor.requests[0]
	if req.Service != serviceSetupOptions || len(req.Args) != 1 || !strings.HasPrefix(req.Id, "cli-setup-") {
		t.Fatalf("request = %+v", req)
	}
	ctxPayload := req.Context
	reqCtx := ctxPayload["ctx"].(map[string]any)
	if reqCtx["tz"] != "UTC" || reqCtx["lang"] != "zh_CN" {
		t.Fatalf("ctx = %v", reqCtx)
	}
	reqMeta := ctxPayload["req"].(map[string]any)
	if reqMeta["requestId"] != req.Id || reqMeta["traceId"] != req.Id || reqMeta["kind"] != "grpc" || reqMeta["depth"] != 0 {
		t.Fatalf("req = %v", reqMeta)
	}
	if _, ok := scope.ScopeFromContext(executor.execCtx); !ok {
		t.Fatal("exec context must carry the transaction scope")
	}
}

func TestExecutorCallerOmitsLangAndNilResponse(t *testing.T) {
	executor := &fakeJsExecutor{}
	caller := executorCaller{executor: executor, scope: newExecutorTestScope(t, "", "")}
	got, err := caller.Call(context.Background(), serviceAppSettingSet, "", setupModeKey, nil)
	if err != nil || got != nil {
		t.Fatalf("Call = %v, %v", got, err)
	}
	if _, ok := executor.requests[0].Context["ctx"].(map[string]any)["lang"]; ok {
		t.Fatal("empty lang must be omitted")
	}
}

func TestExecutorCallerExecuteError(t *testing.T) {
	boom := errors.New("boom")
	caller := executorCaller{executor: &fakeJsExecutor{execErr: boom}, scope: newExecutorTestScope(t, "", "")}
	_, err := caller.Call(context.Background(), serviceCompleteSetup, "")
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "call "+serviceCompleteSetup+": ") {
		t.Fatalf("err = %v", err)
	}
}

func TestExecutorCallerIdentityError(t *testing.T) {
	runtimeScope := newExecutorTestScope(t, "", "")
	if err := runtimeScope.db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate meta_module: %v", err)
	}
	if err := runtimeScope.db.Create(&meta.Module{Name: "auth", Status: meta.Installed}).Error; err != nil {
		t.Fatalf("insert auth module: %v", err)
	}
	executor := &fakeJsExecutor{}
	caller := executorCaller{executor: executor, scope: runtimeScope}
	_, err := caller.Call(context.Background(), serviceSetupOptions, "")
	if !errors.Is(err, systemidentity.ErrBootstrapAdminUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if len(executor.requests) != 0 {
		t.Fatal("no request may run without the system identity")
	}
}
