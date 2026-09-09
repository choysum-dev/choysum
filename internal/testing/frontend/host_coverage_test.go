// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
)

type stubJsEngine struct {
	loadErr error
}

func (s stubJsEngine) Load([]*jsengine.JsScript) error { return s.loadErr }
func (s stubJsEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (s stubJsEngine) Close() error { return nil }

func TestInstallMinimalDOMGuards(t *testing.T) {
	if err := InstallMinimalDOM(nil); err == nil || !strings.Contains(err.Error(), "nil engine") {
		t.Fatalf("nil engine: %v", err)
	}
	if err := InstallMinimalDOM(stubJsEngine{loadErr: os.ErrInvalid}); err == nil || !strings.Contains(err.Error(), "InstallMinimalDOM") {
		t.Fatalf("load err: %v", err)
	}
}

func TestInstallMinimalConsoleGuards(t *testing.T) {
	if err := InstallMinimalConsole(nil); err == nil || !strings.Contains(err.Error(), "nil engine") {
		t.Fatalf("nil engine: %v", err)
	}
	if err := InstallMinimalConsole(stubJsEngine{loadErr: os.ErrInvalid}); err == nil || !strings.Contains(err.Error(), "InstallMinimalConsole") {
		t.Fatalf("load err: %v", err)
	}
}

func TestInstallMinimalConsolePolyfills(t *testing.T) {
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatal("expected QuickjsEngine")
	}

	// Drop host builtins so the console script installs its fallbacks.
	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "drop.js",
		Content: `
globalThis.TextEncoder = undefined;
globalThis.atob = undefined;
globalThis.localStorage = undefined;
globalThis.sessionStorage = undefined;
`,
	}}); err != nil {
		t.Fatalf("drop builtins: %v", err)
	}
	if err := InstallMinimalConsole(engine); err != nil {
		t.Fatal(err)
	}
	val := qjs.Ctx.Eval(`(function () {
  var enc = new TextEncoder().encode("é");
  var utf8 = Array.from(enc);
  localStorage.setItem("k", "local");
  sessionStorage.setItem("k", "session");
  return JSON.stringify({
    utf8: utf8,
    local: localStorage.getItem("k"),
    session: sessionStorage.getItem("k"),
    decoded: atob("YQ=="),
    sameStorage: localStorage === sessionStorage
  });
})()`)
	defer val.Free()
	if val.IsException() {
		t.Fatalf("eval exception: %v", qjs.Ctx.Exception())
	}
	got := val.String()
	if !strings.Contains(got, `"utf8":[195,169]`) {
		t.Fatalf("expected UTF-8 for é, got %s", got)
	}
	if !strings.Contains(got, `"local":"local"`) || !strings.Contains(got, `"session":"session"`) {
		t.Fatalf("expected independent storage, got %s", got)
	}
	if !strings.Contains(got, `"decoded":"a"`) {
		t.Fatalf("expected atob, got %s", got)
	}
	if !strings.Contains(got, `"sameStorage":false`) {
		t.Fatalf("expected distinct storage objects, got %s", got)
	}
	abortVal := qjs.Ctx.Eval(`(function () {
  var c = new AbortController();
  var hit = false;
  c.signal.addEventListener("abort", function () { hit = true; });
  c.abort();
  return JSON.stringify({ aborted: c.signal.aborted, hit: hit });
})()`)
	defer abortVal.Free()
	if abortVal.IsException() {
		t.Fatalf("abort eval: %v", qjs.Ctx.Exception())
	}
	abortGot := abortVal.String()
	if !strings.Contains(abortGot, `"aborted":true`) || !strings.Contains(abortGot, `"hit":true`) {
		t.Fatalf("expected AbortController polyfill, got %s", abortGot)
	}
}

func TestInstallMinimalConsole_HeadersAndSearchParamsPrototype(t *testing.T) {
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatal("expected QuickjsEngine")
	}
	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "drop.js",
		Content: `
globalThis.Headers = undefined;
globalThis.URLSearchParams = undefined;
globalThis.URL = undefined;
`,
	}}); err != nil {
		t.Fatalf("drop builtins: %v", err)
	}
	// Headers / URL / URLSearchParams ship with InstallMinimalConsole (host globals).
	if err := InstallMinimalConsole(engine); err != nil {
		t.Fatal(err)
	}
	val := qjs.Ctx.Eval(`(function () {
  var h1 = new Headers({ "X-A": "1" });
  h1.set("X-B", "2");
  var h2 = new Headers(h1);
  var ownHeaderKeys = Object.keys(h1).filter(function (k) { return k !== "_map"; });
  var sp1 = new URLSearchParams("a=1");
  sp1.set("b", "2");
  var sp2 = new URLSearchParams(sp1);
  var ownParamKeys = Object.keys(sp1).filter(function (k) { return k !== "_pairs"; });
  var url = new URL("http://choysum.local/p");
  url.searchParams.set("q", "1");
  return JSON.stringify({
    h2a: h2.get("x-a"),
    h2b: h2.get("x-b"),
    hHasSetOwn: Object.prototype.hasOwnProperty.call(h1, "set"),
    headerOwnExtras: ownHeaderKeys,
    sp2: sp2.toString(),
    spHasSetOwn: Object.prototype.hasOwnProperty.call(sp1, "set"),
    paramOwnExtras: ownParamKeys,
    href: url.href,
    noMethodQuery: sp2.get("set") == null && sp2.get("append") == null
  });
})()`)
	defer val.Free()
	if val.IsException() {
		t.Fatalf("eval exception: %v", qjs.Ctx.Exception())
	}
	got := val.String()
	if !strings.Contains(got, `"h2a":"1"`) || !strings.Contains(got, `"h2b":"2"`) {
		t.Fatalf("expected Headers clone via forEach, got %s", got)
	}
	if !strings.Contains(got, `"hHasSetOwn":false`) || !strings.Contains(got, `"spHasSetOwn":false`) {
		t.Fatalf("expected methods on prototype, got %s", got)
	}
	if !strings.Contains(got, `"headerOwnExtras":[]`) || !strings.Contains(got, `"paramOwnExtras":[]`) {
		t.Fatalf("expected no leaked own method keys, got %s", got)
	}
	if !strings.Contains(got, `"sp2":"a=1&b=2"`) || !strings.Contains(got, `"noMethodQuery":true`) {
		t.Fatalf("expected URLSearchParams clone without method names, got %s", got)
	}
	if !strings.Contains(got, `"href":"http://choysum.local/p?q=1"`) {
		t.Fatalf("expected URL.href synced with searchParams, got %s", got)
	}
}

func TestPrepareVueHostEngineBranches(t *testing.T) {
	if err := PrepareVueHostEngine(stubJsEngine{loadErr: os.ErrInvalid}); err == nil || !strings.Contains(err.Error(), "InstallMinimalConsole") {
		t.Fatalf("console load fail: %v", err)
	}
	if err := PrepareVueHostEngine(stubJsEngine{}); err != nil {
		t.Fatalf("non-quickjs engine: %v", err)
	}

	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })

	prev := bootstrapVueHostTimers
	bootstrapVueHostTimers = func(*quickjsengine.QuickjsEngine) bool { return false }
	t.Cleanup(func() { bootstrapVueHostTimers = prev })
	if err := PrepareVueHostEngine(engine); err == nil || !strings.Contains(err.Error(), "BootstrapTimers") {
		t.Fatalf("bootstrap fail: %v", err)
	}
	bootstrapVueHostTimers = prev

	if err := PrepareVueHostEngine(engine); err != nil {
		t.Fatalf("PrepareVueHostEngine: %v", err)
	}
}

func TestChoysumMountSourcePathSeams(t *testing.T) {
	prev := resolveChoysumMountSourcePath
	t.Cleanup(func() { resolveChoysumMountSourcePath = prev })

	resolveChoysumMountSourcePath = func() (string, error) { return "", os.ErrNotExist }
	if _, err := ChoysumMountSourcePath(); err == nil {
		t.Fatal("expected resolve error")
	}
	resolveChoysumMountSourcePath = prev

	prevCaller := runtimeCallerMount
	runtimeCallerMount = func(int) (uintptr, string, int, bool) { return 0, "", 0, false }
	t.Cleanup(func() { runtimeCallerMount = prevCaller })
	if _, err := defaultResolveChoysumMountSourcePath(); err == nil {
		t.Fatal("expected caller failure")
	}
	runtimeCallerMount = prevCaller

	prevStat := osStatMount
	osStatMount = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { osStatMount = prevStat })
	if _, err := defaultResolveChoysumMountSourcePath(); err == nil {
		t.Fatal("expected stat failure")
	}
	osStatMount = prevStat

	p, err := ChoysumMountSourcePath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if VueHostPackageVersion() == "" || ChoysumMountScript() == "" {
		t.Fatal("empty mount helpers")
	}
}

func TestSpikeBuildScopeCoverage(t *testing.T) {
	cfg := &config.Config{Server: &config.ServerConfig{JsEngineFactory: "quickjs"}}
	s := &spikeBuildScope{ctx: context.Background(), cfg: cfg}
	if err := s.Run(func(scope.Scope) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if s.Session() != nil || s.Transactor() != nil {
		t.Fatal("expected nil session/transactor")
	}
	cloned := s.WithContext(context.WithValue(context.Background(), struct{}{}, 1))
	if cloned.Context() == nil || cloned.Logger() == nil {
		t.Fatal("context/logger")
	}
	in := s.FactoryInput()
	fi, ok := in.(*spikeFactoryInput)
	if !ok || fi == nil || fi.ServerConfig() == nil {
		t.Fatal("factory input")
	}
	_ = fi.Environment()
	_ = fi.ModulesPath()
	_ = fi.DistPath()
	_ = fi.TmpPath()
	_ = fi.DefaultChoysumPath()
	_ = fi.ConfigPath()
	_ = fi.ESMUpstreamURL()
	_ = fi.NpmRegistryURL()
	_ = fi.ModuleCatalogIndexURL()
	_ = fi.CompileConfig()
	_ = fi.AuthConfig()
	_ = fi.TaskConfig()
	_ = fi.LogConfig()

	empty := &spikeBuildScope{ctx: context.Background(), cfg: nil}
	if empty.FactoryInput() != nil {
		t.Fatal("nil cfg factory input")
	}
}

func TestBuildFrontendVueHostBundleGuards(t *testing.T) {
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{}); err == nil || !strings.Contains(err.Error(), "empty repo root") {
		t.Fatalf("empty repo: %v", err)
	}
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{RepoRoot: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "empty entry path") {
		t.Fatalf("empty entry: %v", err)
	}

	prevResolve := resolveChoysumMountSourcePath
	resolveChoysumMountSourcePath = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { resolveChoysumMountSourcePath = prevResolve })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{RepoRoot: t.TempDir(), EntryPath: "x.ts"}); err == nil || !strings.Contains(err.Error(), "choysummount path") {
		t.Fatalf("mount path: %v", err)
	}
	resolveChoysumMountSourcePath = func() (string, error) { return "/tmp/choysummount.js", nil }
	prevAbs := hostFilepathAbs
	hostFilepathAbs = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { hostFilepathAbs = prevAbs })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{RepoRoot: t.TempDir(), EntryPath: "x.ts"}); err == nil || !strings.Contains(err.Error(), "abs choysummount") {
		t.Fatalf("abs: %v", err)
	}
	hostFilepathAbs = prevAbs
	resolveChoysumMountSourcePath = prevResolve

	repo := vueHostRepoRoot(t)
	fixtureDir := vueHostFixtureDir(t)
	entry := filepath.Join(fixtureDir, "entry_mount.ts")

	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:      repo,
		EntryPath:     entry,
		WithVuePlugin: true,
	}); err == nil || !strings.Contains(err.Error(), "JsExecutor required") {
		t.Fatalf("vueplugin without executor: %v", err)
	}

	t.Setenv("CHOYSUM_HOME", "")
	prevHome := hostUserHomeDir
	hostUserHomeDir = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { hostUserHomeDir = prevHome })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:  repo,
		EntryPath: entry,
	}); err == nil || !strings.Contains(err.Error(), "cache dir") {
		t.Fatalf("home: %v", err)
	}
	hostUserHomeDir = prevHome

	t.Setenv("CHOYSUM_HOME", "")
	hostUserHomeDir = func() (string, error) { return t.TempDir(), nil }
	t.Cleanup(func() { hostUserHomeDir = prevHome })
	prevBuildHome := esbuildBuild
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		_ = os.WriteFile(opts.Outfile, []byte("/*home*/"), 0o644)
		return api.BuildResult{}
	}
	t.Cleanup(func() { esbuildBuild = prevBuildHome })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:  repo,
		EntryPath: entry,
		Outfile:   filepath.Join(t.TempDir(), "home.js"),
	}); err != nil {
		t.Fatalf("home cache: %v", err)
	}
	esbuildBuild = prevBuildHome
	hostUserHomeDir = prevHome
	t.Setenv("CHOYSUM_HOME", t.TempDir())

	prevMkdir := osMkdirAll
	osMkdirAll = func(string, os.FileMode) error { return os.ErrPermission }
	t.Cleanup(func() { osMkdirAll = prevMkdir })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    filepath.Join(t.TempDir(), "out", "x.js"),
		CacheDir:   t.TempDir(),
		WorkingDir: fixtureDir,
	}); err == nil || !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("mkdir: %v", err)
	}
	osMkdirAll = prevMkdir

	prevBuild := esbuildBuild
	esbuildBuild = func(api.BuildOptions) api.BuildResult {
		return api.BuildResult{Errors: []api.Message{{Text: "boom"}}}
	}
	t.Cleanup(func() { esbuildBuild = prevBuild })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    filepath.Join(t.TempDir(), "bundle.js"),
		CacheDir:   t.TempDir(),
		WorkingDir: fixtureDir,
	}); err == nil || !strings.Contains(err.Error(), "esbuild") {
		t.Fatalf("esbuild err: %v", err)
	}

	esbuildBuild = func(api.BuildOptions) api.BuildResult {
		return api.BuildResult{Warnings: []api.Message{{Text: "soft"}}}
	}
	prevRead := osReadFile
	osReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { osReadFile = prevRead })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    filepath.Join(t.TempDir(), "bundle.js"),
		CacheDir:   t.TempDir(),
		WorkingDir: fixtureDir,
	}); err == nil || !strings.Contains(err.Error(), "read outfile") {
		t.Fatalf("read: %v", err)
	}
	osReadFile = prevRead

	outFile := filepath.Join(t.TempDir(), "ok.js")
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		_ = os.WriteFile(opts.Outfile, []byte("/*ok*/"), 0o644)
		return api.BuildResult{Warnings: []api.Message{{Text: "warn-line"}}}
	}
	prevStat := osStatBundle
	osStatBundle = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { osStatBundle = prevStat })
	got, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    outFile,
		Sourcemap:  true,
		CacheDir:   t.TempDir(),
		WorkingDir: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.JS != "/*ok*/" || len(got.Warnings) != 1 || got.MapPath != "" {
		t.Fatalf("got = %#v", got)
	}

	// WorkingDir outside repoRoot: TsconfigRaw @/* must still resolve under repoRoot/modules.
	workOutside := t.TempDir()
	var capturedTsconfig string
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		capturedTsconfig = opts.TsconfigRaw
		if opts.AbsWorkingDir != workOutside {
			t.Fatalf("AbsWorkingDir = %q, want %q", opts.AbsWorkingDir, workOutside)
		}
		_ = os.WriteFile(opts.Outfile, []byte("/*rel*/"), 0o644)
		return api.BuildResult{}
	}
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    filepath.Join(t.TempDir(), "rel.js"),
		CacheDir:   t.TempDir(),
		WorkingDir: workOutside,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedTsconfig, `"@/*"`) || strings.Contains(capturedTsconfig, `"./modules/*"`) {
		t.Fatalf("expected @/* relpath from WorkingDir to repo modules, got %s", capturedTsconfig)
	}
	osStatBundle = func(string) (os.FileInfo, error) {
		return os.Stat(outFile)
	}
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		_ = os.WriteFile(opts.Outfile, []byte("/*map*/"), 0o644)
		_ = os.WriteFile(opts.Outfile+".map", []byte("{}"), 0o644)
		return api.BuildResult{}
	}
	got2, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    outFile,
		Sourcemap:  true,
		CacheDir:   t.TempDir(),
		WorkingDir: fixtureDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got2.MapPath == "" {
		t.Fatal("expected map path")
	}
	got3, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:   repo,
		EntryPath:  entry,
		Outfile:    outFile,
		Sourcemap:  false,
		CacheDir:   t.TempDir(),
		WorkingDir: fixtureDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got3.MapPath != "" {
		t.Fatalf("Sourcemap=false must not attach stale map: %#v", got3)
	}
	osStatBundle = prevStat
	esbuildBuild = prevBuild

	// Default outfile path (empty Outfile → beside entry) via mocked build.
	prevBuildDefault := esbuildBuild
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		if !strings.HasSuffix(opts.Outfile, "vue-host.bundle.js") {
			t.Fatalf("default outfile = %q", opts.Outfile)
		}
		_ = os.WriteFile(opts.Outfile, []byte("/*default*/"), 0o644)
		return api.BuildResult{}
	}
	tmpEntry := filepath.Join(t.TempDir(), "x.ts")
	if err := os.WriteFile(tmpEntry, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	def, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:  repo,
		EntryPath: tmpEntry,
		CacheDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(def.JSPath, "vue-host.bundle.js") {
		t.Fatalf("default path: %#v", def)
	}
	esbuildBuild = prevBuildDefault

	executor := newVueHostCompiler(t)
	bundled, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:      repo,
		EntryPath:     entry,
		Outfile:       filepath.Join(t.TempDir(), "vue-host.bundle.js"),
		Sourcemap:     true,
		WorkingDir:    fixtureDir,
		JsExecutor:    executor,
		WithVuePlugin: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundled.MapPath == "" || !strings.Contains(bundled.JSPath, "vue-host.bundle.js") {
		t.Fatalf("outfile/map: %#v", bundled)
	}
}
