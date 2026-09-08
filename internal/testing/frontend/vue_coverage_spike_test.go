// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/testing/coverage"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// TestVueSFCCoverageSpike_P0 proves narrowed-A P0 with the product host path:
// vuesfc → esbuild+real vue+sourcemap → InstrumentJSFile → QuickJS createApp().mount()
// (minimal DOM) → WriteLcov contains hits on SpikeCounter.vue script lines.
func TestVueSFCCoverageSpike_P0(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	fixtureDir := filepath.Join(filepath.Dir(thisFile), "testdata", "fixtures", "coverage")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	work := t.TempDir()
	for _, name := range []string{"SpikeCounter.vue", "entry.ts"} {
		raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	entryPath := filepath.Join(work, "entry.ts")
	outJS := filepath.Join(work, "out", "bundle.js")
	executor := newVueHostCompiler(t)

	bundle, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:      repoRoot,
		EntryPath:     entryPath,
		Outfile:       outJS,
		Sourcemap:     true,
		WorkingDir:    work,
		JsExecutor:    executor,
		WithVuePlugin: true,
	})
	if err != nil {
		t.Fatalf("BuildFrontendVueHostBundle: %v", err)
	}
	if bundle.MapPath == "" {
		t.Fatal("missing sourcemap")
	}

	if err := coverage.InstrumentJSFile(outJS); err != nil {
		t.Fatalf("InstrumentJSFile: %v", err)
	}
	bundleJS, err := os.ReadFile(outJS)
	if err != nil {
		t.Fatal(err)
	}

	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if err := PrepareVueHostEngine(engine); err != nil {
		t.Fatalf("PrepareVueHostEngine: %v", err)
	}
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: outJS, Content: string(bundleJS)},
	}); err != nil {
		t.Fatalf("Load instrumented bundle: %v", err)
	}

	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatalf("engine type %T", engine)
	}
	covVal := qjs.Ctx.Eval(`(typeof globalThis.__coverage__ === "undefined") ? "null" : JSON.stringify(globalThis.__coverage__)`)
	defer covVal.Free()
	if covVal.IsException() {
		t.Fatalf("read __coverage__: %v", qjs.Ctx.Exception())
	}
	coverageJSON := covVal.String()
	if coverageJSON == "" || coverageJSON == "null" {
		t.Fatal("expected non-empty __coverage__ after mount")
	}

	tmpRoot := t.TempDir()
	runID := "vue-cov-spike"
	if err := coverage.WriteCoverageJSONWithRunIDAndTmpRoot(work, "spike", runID, coverageJSON, tmpRoot); err != nil {
		t.Fatalf("WriteCoverageJSON: %v", err)
	}
	reportDir := filepath.Join(tmpRoot, "reports")
	if err := coverage.WriteLcov(t.Context(), coverage.ReportOptions{
		RepoRoot:  work,
		TmpRoot:   tmpRoot,
		ReportDir: reportDir,
		Reporters: []string{"lcovonly"},
		RunID:     runID,
	}); err != nil {
		t.Fatalf("WriteLcov: %v", err)
	}

	lcovPath := filepath.Join(reportDir, "lcov.info")
	lcovRaw, err := os.ReadFile(lcovPath)
	if err != nil {
		t.Fatalf("read lcov: %v", err)
	}
	lcov := string(lcovRaw)
	if !strings.Contains(lcov, "SpikeCounter.vue") {
		t.Fatalf("lcov missing SpikeCounter.vue; got:\n%s", lcov)
	}
	vueIdx := strings.Index(lcov, "SpikeCounter.vue")
	chunk := lcov[vueIdx:]
	if end := strings.Index(chunk, "\nend_of_record"); end >= 0 {
		chunk = chunk[:end]
	}
	hitLines := map[int]bool{}
	for _, rawLine := range strings.Split(chunk, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "DA:") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
		if len(parts) != 2 || parts[1] == "0" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(parts[0], "%d", &n); err == nil {
			hitLines[n] = true
		}
	}
	for _, want := range []int{8, 11, 14} {
		if !hitLines[want] {
			t.Fatalf("expected DA hit for SpikeCounter.vue script line %d; hits=%v chunk:\n%s\nfull:\n%s", want, hitLines, chunk, lcov)
		}
	}
}
