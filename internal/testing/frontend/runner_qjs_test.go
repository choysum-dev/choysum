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
	return filepath.Join(filepath.Dir(thisFile), "testdata", "fixtures", "runner")
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
	reportDir := filepath.Join(t.TempDir(), "reports")
	tmpRoot := t.TempDir()
	failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
		RepoRoot:          repoRoot,
		App:               "fe_qjs_vue",
		TestFiles:         []string{filepath.Join(work, "counter.test.ts")},
		WorkingDir:        work,
		Coverage:          true,
		CoverageReport:    true,
		CoverageReportDir: reportDir,
		TmpRoot:           tmpRoot,
		ForceVue:          true,
		Keep:              true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatal("expected vue fixture to pass")
	}
	lcovPath := filepath.Join(reportDir, "fe", "fe_qjs_vue", "lcov.info")
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

func TestFrontendQJS_LcovPath_FeApp(t *testing.T) {
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
	reportDir := filepath.Join(t.TempDir(), "cov-root")
	failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
		RepoRoot:          repoRoot,
		App:               "web",
		TestFiles:         []string{filepath.Join(work, "counter.test.ts")},
		WorkingDir:        work,
		Coverage:          true,
		CoverageReport:    true,
		CoverageReportDir: reportDir,
		TmpRoot:           t.TempDir(),
		ForceVue:          true,
	})
	if err != nil || failed {
		t.Fatalf("run: failed=%v err=%v", failed, err)
	}
	lcovPath := filepath.Join(reportDir, "fe", "web", "lcov.info")
	if _, err := os.Stat(lcovPath); err != nil {
		t.Fatalf("expected %s: %v", lcovPath, err)
	}
	// Must not write lcov at the coverage root (would collide with BE).
	if _, err := os.Stat(filepath.Join(reportDir, "lcov.info")); err == nil {
		t.Fatal("FE must not write lcov.info at CoverageReportDir root")
	}
	raw, err := os.ReadFile(lcovPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Counter.vue") {
		t.Fatalf("expected .vue hits in %s", lcovPath)
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
	if !frontendTestsNeedVue(files) {
		t.Fatal("auth FE corpus uses .vue / choysumMount; needVue should be true")
	}
}

func TestRunOneAppFrontendTests_DefaultsToQJS(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
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
