// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
)

func feQjsRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func feQjsFixtureDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "fe_qjs")
}

func TestUseQJSFrontendEngine(t *testing.T) {
	t.Setenv(EnvFEUnitEngine, "")
	if UseQJSFrontendEngine() {
		t.Fatal("empty env should be false")
	}
	t.Setenv(EnvFEUnitEngine, "qjs")
	if !UseQJSFrontendEngine() {
		t.Fatal("qjs should enable")
	}
	t.Setenv(EnvFEUnitEngine, "QJS")
	if !UseQJSFrontendEngine() {
		t.Fatal("case-insensitive qjs")
	}
	t.Setenv(EnvFEUnitEngine, "vitest")
	if UseQJSFrontendEngine() {
		t.Fatal("vitest must not enable qjs")
	}
}

func TestValidateFrontendTestDependencies_QJSSkipsNpx(t *testing.T) {
	t.Setenv(EnvFEUnitEngine, "qjs")
	if err := ValidateFrontendTestDependencies("", "", false); err != nil {
		t.Fatalf("qjs path should skip vitest deps: %v", err)
	}
}

func TestBuildFrontendUnitBundle_WithVue(t *testing.T) {
	repoRoot := feQjsRepoRoot(t)
	fixture := filepath.Join(feQjsFixtureDir(t), "vue")
	work := t.TempDir()
	for _, name := range []string{"Counter.vue", "counter.test.ts"} {
		raw, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entry := filepath.Join(work, "entry.ts")
	if err := WriteFrontendTestsEntry(entry, []string{filepath.Join(work, "counter.test.ts")}); err != nil {
		t.Fatal(err)
	}
	executor := newVueHostCompiler(t)
	outJS := filepath.Join(work, "out", "vue.bundle.js")
	bundle, err := BuildFrontendUnitBundle(BundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    outJS,
		Sourcemap:  true,
		WorkingDir: work,
		Vue:        true,
		JsExecutor: executor,
		CacheDir:   t.TempDir(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "download failed") {
			t.Skipf("esm.sh unavailable: %v", err)
		}
		t.Fatalf("BuildFrontendUnitBundle Vue: %v", err)
	}
	if bundle.MapPath == "" {
		t.Fatal("expected sourcemap")
	}
	mapRaw, err := os.ReadFile(bundle.MapPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mapRaw), "Counter.vue") {
		t.Fatalf("sourcemap sources should include Counter.vue; map=%s", truncate(string(mapRaw), 400))
	}
	if _, err := BuildFrontendUnitBundle(BundleOptions{
		RepoRoot:  repoRoot,
		EntryPath: entry,
		Vue:       true,
	}); err == nil || !strings.Contains(err.Error(), "JsExecutor") {
		t.Fatalf("Vue without executor: %v", err)
	}
}

func TestRunFrontendQJS_FeQjsMath(t *testing.T) {
	repoRoot := feQjsRepoRoot(t)
	fixture := feQjsFixtureDir(t)
	work := t.TempDir()
	for _, name := range []string{"math.ts", "math.test.ts"} {
		raw, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	junit := filepath.Join(t.TempDir(), "math.xml")
	failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
		RepoRoot:   repoRoot,
		App:        "fe_qjs_math",
		TestFiles:  []string{filepath.Join(work, "math.test.ts")},
		WorkingDir: work,
		JUnitPath:  junit,
		TmpRoot:    t.TempDir(),
		Keep:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatal("expected math fixture to pass")
	}
	raw, err := os.ReadFile(junit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "add sums two numbers") {
		t.Fatalf("junit missing case: %s", raw)
	}
}

func TestRunFrontendQJS_VueCoverage(t *testing.T) {
	work := t.TempDir()
	testFile := filepath.Join(work, "counter.test.ts")
	vuePath := filepath.Join(work, "Counter.vue")
	if err := os.WriteFile(testFile, []byte("import { mount } from '@choysum/test-utils';\nimport C from './Counter.vue';\ntest('x', () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vuePath, []byte("<script setup lang=\"ts\">\nconst n = 1;\n</script>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	covJSON := `{"` + vuePath + `":{"path":"` + vuePath + `","s":{"0":1,"1":1},"f":{"0":1},"b":{},"statementMap":{"0":{"start":{"line":1,"column":0},"end":{"line":1,"column":10}},"1":{"start":{"line":2,"column":0},"end":{"line":2,"column":10}}},"fnMap":{"0":{"name":"setup","decl":{"start":{"line":1,"column":0},"end":{"line":1,"column":5}},"loc":{"start":{"line":1,"column":0},"end":{"line":2,"column":10}}}},"branchMap":{}}}`

	prevBuild := buildFrontendUnitBundleQJS
	buildFrontendUnitBundleQJS = func(opts BundleOptions) (*BundleResult, error) {
		js := "globalThis.__choysum_test_run__ = async () => ({ total:1, passed:1, failed:0, cases:[{name:'mounts Counter',ok:true,durationMs:1}], coverageJSON: " + strconvQuote(covJSON) + " });"
		if err := os.WriteFile(opts.Outfile, []byte(js), 0o644); err != nil {
			return nil, err
		}
		return &BundleResult{JS: js, JSPath: opts.Outfile}, nil
	}
	prevComp := newFrontendCompilerExecutorQ
	newFrontendCompilerExecutorQ = func(context.Context) (jsexecutor.JsExecutor, error) { return stubJsExecutor{}, nil }
	prevPrep := prepareVueHostEngineQJS
	prepareVueHostEngineQJS = func(jsengine.JsEngine) error { return nil }
	prevPF := preflightCoverageQJS
	preflightCoverageQJS = func(string) error { return nil }
	prevInst := instrumentJSFileQJS
	instrumentJSFileQJS = func(string) error { return nil }
	t.Cleanup(func() {
		buildFrontendUnitBundleQJS = prevBuild
		newFrontendCompilerExecutorQ = prevComp
		prepareVueHostEngineQJS = prevPrep
		preflightCoverageQJS = prevPF
		instrumentJSFileQJS = prevInst
	})

	reportDir := filepath.Join(t.TempDir(), "reports")
	failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
		RepoRoot:          work,
		App:               "fe_qjs_vue",
		TestFiles:         []string{testFile},
		WorkingDir:        work,
		Coverage:          true,
		CoverageReport:    true,
		CoverageReportDir: reportDir,
		TmpRoot:           t.TempDir(),
		ForceVue:          true,
		Keep:              true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatal("expected vue fixture to pass")
	}
	lcovPath := filepath.Join(reportDir, "lcov.info")
	lcovRaw, err := os.ReadFile(lcovPath)
	if err != nil {
		t.Fatalf("read lcov: %v", err)
	}
	lcov := string(lcovRaw)
	if !strings.Contains(lcov, "Counter.vue") {
		t.Fatalf("lcov missing Counter.vue:\n%s", truncate(lcov, 800))
	}
	if !strings.Contains(lcov, "DA:") || !strings.Contains(lcov, ",1") {
		t.Fatalf("expected script line hits in lcov:\n%s", truncate(lcov, 800))
	}
}

func TestRunFrontendQJS_DiscoverAuthSmoke(t *testing.T) {
	repoRoot := feQjsRepoRoot(t)
	files, err := DiscoverFrontendTests(repoRoot, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected auth FE tests")
	}
	// Corpus is still Vitest; only assert discover + Vue detection wiring.
	if !frontendTestsNeedVue(files) {
		t.Fatal("auth FE corpus still uses VTU/.vue; needVue should be true")
	}
}

func TestRunOneAppFrontendTests_RoutesToQJS(t *testing.T) {
	t.Setenv(EnvFEUnitEngine, "qjs")
	repoRoot := feQjsRepoRoot(t)
	// No modules/fe_qjs_math/web — discover empty → ok with no tests.
	failed, err := RunOneAppFrontendTests(context.Background(), repoRoot, "fe_qjs_math", "", "", false, false, false, false, "", 0, 0, 0, 0, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatal("empty discover should not fail")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
