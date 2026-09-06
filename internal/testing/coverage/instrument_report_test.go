// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstrumentJSFile_StatementCountersAndVoid0(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.js")
	src := "function add(a, b) {\n  return a + b;\n}\nvar x = add(1, 2);\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := InstrumentJSFile(path); err != nil {
		t.Fatalf("InstrumentJSFile: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "globalThis") {
		t.Fatalf("expected globalThis coverage scope, got:\n%s", text)
	}
	if !strings.Contains(text, "__coverage__") {
		t.Fatalf("expected __coverage__, got:\n%s", text)
	}
	if !strings.Contains(text, ";void 0;") {
		t.Fatalf("expected trailing ;void 0;, got:\n%s", text)
	}
	if !strings.Contains(text, ".s[") {
		t.Fatalf("expected statement counters, got:\n%s", text)
	}
	metaRaw, err := os.ReadFile(path + ".coverage-meta.json")
	if err != nil {
		t.Fatalf("expected coverage meta sidecar: %v", err)
	}
	if !strings.Contains(string(metaRaw), "statementMap") {
		t.Fatalf("expected statementMap in coverage meta, got:\n%s", metaRaw)
	}
}

func TestInstrumentJSFile_InheritsExternalSourceMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.js")
	mapPath := path + ".map"
	src := "var a = 1;\n//# sourceMappingURL=bundle.js.map\n"
	sm := rawSourceMap{
		Version:  3,
		File:     "bundle.js",
		Sources:  []string{"src/a.ts"},
		Mappings: "AAAA",
	}
	mapBytes, _ := json.Marshal(sm)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile js: %v", err)
	}
	if err := os.WriteFile(mapPath, mapBytes, 0o644); err != nil {
		t.Fatalf("WriteFile map: %v", err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatalf("InstrumentJSFile: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(out), "sourceMappingURL=bundle.js.map") {
		t.Fatalf("expected sourceMappingURL preserved, got:\n%s", out)
	}
	metaRaw, err := os.ReadFile(path + ".coverage-meta.json")
	if err != nil {
		t.Fatalf("expected coverage meta sidecar: %v", err)
	}
	if !strings.Contains(string(metaRaw), "inputSourceMap") {
		t.Fatalf("expected inputSourceMap in coverage meta, got:\n%s", metaRaw)
	}
}

func TestWriteLcovAndCheckCoverage_FromNycOutput(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "run-test1"
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/cov\n"), 0o644); err != nil {
		t.Fatalf("go.mod: %v", err)
	}

	srcPath := filepath.Join(repoRoot, "modules", "demo", "service", "math.ts")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte("export function add(a: number, b: number) { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Use the TypeScript path directly (no remap) so statement hits are stable for gates.
	cov := map[string]*coverageFileData{
		srcPath: {
			Path: srcPath,
			StatementMap: map[string]coverageRange{
				"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 10}},
				"1": {Start: coveragePos{Line: 2, Column: 0}, End: coveragePos{Line: 2, Column: 10}},
			},
			FnMap: map[string]coverageFn{
				"0": {Name: "add", Decl: coverageRange{Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}, Loc: coverageRange{Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 10}}, Line: 1},
			},
			BranchMap: map[string]coverageBranch{},
			S:         map[string]int{"0": 1, "1": 0},
			F:         map[string]int{"0": 1},
			B:         map[string][]int{},
		},
	}
	raw, err := json.Marshal(cov)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "demo", runID, string(raw), tmpRoot); err != nil {
		t.Fatalf("WriteCoverageJSON: %v", err)
	}

	ctx := ContextWithCoverageRunID(context.Background(), runID)
	reportDir := filepath.Join(tmpRoot, "reports")
	if err := WriteLcov(ctx, ReportOptions{
		RepoRoot:  repoRoot,
		TmpRoot:   tmpRoot,
		ReportDir: reportDir,
		Reporters: []string{"lcovonly", "text-summary"},
		RunID:     runID,
	}); err != nil {
		t.Fatalf("WriteLcov: %v", err)
	}
	lcovPath := filepath.Join(reportDir, "lcov.info")
	lcov, err := os.ReadFile(lcovPath)
	if err != nil {
		t.Fatalf("read lcov: %v", err)
	}
	if !strings.Contains(string(lcov), "SF:modules/demo/service/math.ts") {
		t.Fatalf("expected source path in lcov, got:\n%s", lcov)
	}
	if !strings.Contains(string(lcov), "DA:1,1") || !strings.Contains(string(lcov), "DA:2,0") {
		t.Fatalf("expected DA hit lines in lcov, got:\n%s", lcov)
	}

	if err := CheckCoverage(ctx, CheckOptions{
		RepoRoot:   repoRoot,
		TmpRoot:    tmpRoot,
		RunID:      runID,
		Statements: 40, // 1/2 = 50% should pass
		Lines:      40,
	}); err != nil {
		t.Fatalf("CheckCoverage should pass: %v", err)
	}
	if err := CheckCoverage(ctx, CheckOptions{
		RepoRoot:   repoRoot,
		TmpRoot:    tmpRoot,
		RunID:      runID,
		Statements: 80, // 50% < 80%
	}); err == nil {
		t.Fatalf("expected CheckCoverage to fail at 80%% statements")
	}
}

func TestMatchCoverageGlob(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"**/dist/**", "foo/dist/bar.js", true},
		{"**/*.test.ts", "modules/a/service/x.test.ts", true},
		{"**/*.d.ts", "modules/a/types.d.ts", true},
		{"modules/**/*.ts", "modules/a/service/x.ts", true},
		{"**/node_modules/**", "a/node_modules/b/c.js", true},
		{"**/*.test.ts", "modules/a/service/x.ts", false},
	}
	for _, tc := range cases {
		if got := matchCoverageGlob(tc.pattern, tc.path); got != tc.want {
			t.Fatalf("matchCoverageGlob(%q, %q)=%v want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestInstrumentJSFileLargeBundleTiming(t *testing.T) {
	candidates := []string{"/tmp/cov-tests.js", "/tmp/cov-idx.js"}
	var src string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			src = c
			break
		}
	}
	if src == "" {
		t.Skip("no large bundle fixture under /tmp")
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "index.js")
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	if err := InstrumentJSFile(dst); err != nil {
		t.Fatal(err)
	}
	t.Logf("instrumented %s (%d bytes) in %v; out=%d", src, len(raw), time.Since(t0), mustSize(t, dst))
}

func mustSize(t *testing.T, path string) int64 {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return st.Size()
}
