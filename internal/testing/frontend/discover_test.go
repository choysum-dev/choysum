// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
)

func TestDiscoverAndScanIllegalFrontendMarks(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}

	pure := filepath.Join(web, "pure.test.ts")
	if err := os.WriteFile(pure, []byte("import { describe, it } from 'vitest'\nit('ok', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	illegal := filepath.Join(web, "bad.test.ts")
	illegalSrc := strings.Join([]string{
		"// @vitest-environment happy-dom",
		"import { mount } from '@vue/test-utils'",
		"import Comp from './Comp.vue'",
		"import 'happy-dom'",
		"mount(Comp)",
		"",
	}, "\n")
	if err := os.WriteFile(illegal, []byte(illegalSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverFrontendTests(repo, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("DiscoverFrontendTests = %#v", files)
	}

	hits, err := ScanIllegalFrontendMarks(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected illegal hits")
	}
	kinds := map[IllegalKind]bool{}
	for _, h := range hits {
		kinds[h.Kind] = true
	}
	for _, want := range []IllegalKind{IllegalDOMEnvironment, IllegalVTU, IllegalVueImport, IllegalDOMPackage} {
		if !kinds[want] {
			t.Fatalf("missing kind %s in %#v", want, hits)
		}
	}

	warn := FormatIllegalMarksWarn(hits, repo)
	if !strings.Contains(warn, "illegal FE unit mark") {
		t.Fatalf("warn = %q", warn)
	}
	ann := FormatIllegalMarksGitHubAnnotations(hits, repo)
	if !strings.Contains(ann, "::warning file=") {
		t.Fatalf("annotations = %q", ann)
	}

	if _, err := CheckIllegalFrontendMarks(repo, "demo", ScanModeError); err == nil {
		t.Fatal("expected error mode failure")
	}
	if hits2, err := CheckIllegalFrontendMarks(repo, "demo", ScanModeWarn); err != nil || len(hits2) == 0 {
		t.Fatalf("warn mode = %v hits=%d", err, len(hits2))
	}

	pureHits, err := ScanIllegalFrontendMarks([]string{pure})
	if err != nil {
		t.Fatal(err)
	}
	if len(pureHits) != 0 {
		t.Fatalf("pure file hits = %#v", pureHits)
	}
}

func TestDiscoverFrontendTestsEmptyApp(t *testing.T) {
	repo := t.TempDir()
	files, err := DiscoverFrontendTests(repo, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("got %#v", files)
	}
	if _, err := DiscoverFrontendTests("", "x"); err == nil {
		t.Fatal("expected empty repo error")
	}
}

func TestBuildFrontendUnitBundleAndChoysumtestFixture(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixtureDir := filepath.Join(filepath.Dir(thisFile), "testdata", "fe_qjs")
	testFile := filepath.Join(fixtureDir, "math.test.ts")
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}

	outDir := t.TempDir()
	entry := filepath.Join(outDir, "entry.ts")
	if err := WriteFrontendTestsEntry(entry, []string{testFile}); err != nil {
		t.Fatal(err)
	}
	outfile := filepath.Join(outDir, "bundle.js")
	bundle, err := BuildFrontendUnitBundle(BundleOptions{
		RepoRoot:  repoRoot,
		EntryPath: entry,
		Outfile:   outfile,
		Sourcemap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bundle.JS, "add sums two numbers") {
		snip := bundle.JS
		if len(snip) > 400 {
			snip = snip[:400]
		}
		t.Fatalf("unexpected bundle:\n%s", snip)
	}

	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
		{FileName: outfile, Content: bundle.JS},
	}); err != nil {
		t.Fatalf("Load: %v", err)
	}

	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatalf("engine type %T", engine)
	}
	// Wrap in an async IIFE: bare top-level await is unreliable with EvalAwait here.
	val := qjs.Ctx.Eval(`(async () => {
  const r = await globalThis.__choysum_test_run__();
  return JSON.stringify(r);
})()`, quickjs.EvalAwait(true))
	defer val.Free()
	if val.IsException() {
		t.Fatalf("eval report: %v", qjs.Ctx.Exception())
	}
	raw := val.String()
	var report struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	}
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("parse report %q: %v", raw, err)
	}
	if report.Total < 2 || report.Failed != 0 || report.Passed != report.Total {
		t.Fatalf("report = %+v raw=%s", report, raw)
	}
}
