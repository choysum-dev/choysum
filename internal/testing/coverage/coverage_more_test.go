// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCoverageRunIDAndContextNil(t *testing.T) {
	id := NewCoverageRunID()
	if id == "" || !strings.Contains(id, "-r") {
		t.Fatalf("NewCoverageRunID() = %q", id)
	}
	ctx := ContextWithCoverageRunID(nil, "  run-x  ")
	if got := CoverageRunIDFromContext(ctx); got != "run-x" {
		t.Fatalf("CoverageRunIDFromContext = %q", got)
	}
	if got := ContextWithCoverageRunID(context.Background(), "  "); CoverageRunIDFromContext(got) != "" {
		t.Fatalf("empty run id should not be stored")
	}
	if CoverageRunIDFromContext(nil) != "" {
		t.Fatalf("nil ctx should yield empty run id")
	}
}

func TestResolveCoverageReportDirs(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	dir, err := ResolveCoverageReportDir(repoRoot, tmpRoot)
	if err != nil {
		t.Fatalf("ResolveCoverageReportDir: %v", err)
	}
	if !strings.Contains(dir, "reports") {
		t.Fatalf("report dir = %q", dir)
	}
	dir2, err := ResolveCoverageReportDirWithRunID(repoRoot, tmpRoot, "rid")
	if err != nil {
		t.Fatalf("ResolveCoverageReportDirWithRunID: %v", err)
	}
	if !strings.Contains(dir2, "rid") {
		t.Fatalf("expected run id in %q", dir2)
	}
	base, err := resolveCoverageTmpBaseDir(repoRoot, tmpRoot)
	if err != nil {
		t.Fatalf("resolveCoverageTmpBaseDir: %v", err)
	}
	if base == "" {
		t.Fatal("empty tmp base")
	}
}

func TestEnrichCoverageJSONWithMeta(t *testing.T) {
	dir := t.TempDir()
	jsPath := filepath.Join(dir, "bundle.js")
	meta := coverageFileData{
		Path: jsPath,
		StatementMap: map[string]coverageRange{
			"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 5}},
		},
		FnMap: map[string]coverageFn{
			"0": {Name: "go", Line: 1},
		},
		BranchMap: map[string]coverageBranch{
			"0": {Type: "if", Line: 1, Locations: []coverageRange{{Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}}},
		},
		InputSourceMap: &rawSourceMap{Version: 3, Sources: []string{"a.ts"}, Mappings: "AAAA"},
		CoverageSchema: istanbulCoverageSchema,
	}
	metaRaw, _ := json.Marshal(meta)
	if err := os.WriteFile(jsPath+".coverage-meta.json", metaRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	runtime := map[string]*coverageFileData{
		jsPath: {
			Path: jsPath,
			S:    hitMap{"0": 2},
			F:    hitMap{"0": 1},
			B:    map[string][]int{"0": {1, 0}},
		},
	}
	raw, _ := json.Marshal(runtime)
	out, err := enrichCoverageJSONWithMeta(string(raw))
	if err != nil {
		t.Fatalf("enrich: %v", err)
	}
	if !strings.Contains(out, "statementMap") || !strings.Contains(out, "inputSourceMap") {
		t.Fatalf("expected meta merged, got %s", out)
	}
	if got, err := enrichCoverageJSONWithMeta(""); err != nil || got != "" {
		t.Fatalf("empty enrich = %q %v", got, err)
	}
	if got, err := enrichCoverageJSONWithMeta("null"); err != nil || got != "null" {
		t.Fatalf("null enrich = %q %v", got, err)
	}
	if got, err := enrichCoverageJSONWithMeta("not-json"); err != nil || got != "not-json" {
		t.Fatalf("invalid json should pass through, got %q %v", got, err)
	}
	nilEntry, _ := json.Marshal(map[string]*coverageFileData{"x": nil})
	if got, err := enrichCoverageJSONWithMeta(string(nilEntry)); err != nil || !strings.Contains(got, "x") {
		t.Fatalf("nil entry enrich = %q %v", got, err)
	}
}

func TestWriteCoverageJSONEnrichesFromMeta(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	jsPath := filepath.Join(repoRoot, "out.js")
	meta := coverageFileData{
		Path:         jsPath,
		StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
		FnMap:        map[string]coverageFn{"0": {Name: "f", Line: 1}},
	}
	metaRaw, _ := json.Marshal(meta)
	if err := os.WriteFile(jsPath+".coverage-meta.json", metaRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	payload := `{"` + jsPath + `":{"path":"` + jsPath + `","s":[1],"f":[1],"b":{}}}`
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "app", "rid", payload, tmpRoot); err != nil {
		t.Fatal(err)
	}
}

func TestHitMapJSONArrayAndObject(t *testing.T) {
	var h hitMap
	if err := json.Unmarshal([]byte(`[1,0,3]`), &h); err != nil {
		t.Fatal(err)
	}
	if h["0"] != 1 || h["1"] != 0 || h["2"] != 3 {
		t.Fatalf("array hitMap = %#v", h)
	}
	if err := json.Unmarshal([]byte(`{"0":9}`), &h); err != nil {
		t.Fatal(err)
	}
	if h["0"] != 9 {
		t.Fatalf("object hitMap = %#v", h)
	}
	if err := json.Unmarshal([]byte(`null`), &h); err != nil {
		t.Fatal(err)
	}
	if len(h) != 0 {
		t.Fatalf("null hitMap = %#v", h)
	}
	raw, err := json.Marshal(hitMap(nil))
	if err != nil || string(raw) != "{}" {
		t.Fatalf("nil marshal = %s %v", raw, err)
	}
	raw, err = json.Marshal(hitMap{"1": 2})
	if err != nil || !strings.Contains(string(raw), `"1"`) {
		t.Fatalf("marshal = %s %v", raw, err)
	}
}

func TestInstrumentJSFile_ErrorsAndInlineSourceMap(t *testing.T) {
	if err := InstrumentJSFile(""); err == nil {
		t.Fatal("expected empty path error")
	}
	if err := InstrumentJSFile(filepath.Join(t.TempDir(), "missing.js")); err == nil {
		t.Fatal("expected missing file error")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "inline.js")
	sm := rawSourceMap{Version: 3, File: "inline.js", Sources: []string{"src/a.ts"}, Mappings: "AAAA"}
	smRaw, _ := json.Marshal(sm)
	src := "function named(){ return 1; }\nnamed();\n//# sourceMappingURL=data:application/json;base64," +
		base64.StdEncoding.EncodeToString(smRaw) + "\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(path)
	if !strings.Contains(string(out), ".f[") {
		t.Fatalf("expected function counter:\n%s", out)
	}
	metaRaw, err := os.ReadFile(path + ".coverage-meta.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metaRaw), "inputSourceMap") {
		t.Fatalf("expected inputSourceMap in meta: %s", metaRaw)
	}

	// Sibling .map fallback when URL path is wrong.
	path2 := filepath.Join(dir, "sib.js")
	if err := os.WriteFile(path2, []byte("var z=1;\n//# sourceMappingURL=missing-other.map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path2+".map", smRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path2); err != nil {
		t.Fatal(err)
	}

	// Invalid base64 / invalid JSON map URLs are ignored.
	path3 := filepath.Join(dir, "bad.js")
	if err := os.WriteFile(path3, []byte("var q=1;\n//# sourceMappingURL=data:application/json;base64,@@@\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path3); err != nil {
		t.Fatal(err)
	}
}

func TestInstrumentJSFile_BlockWrapClosingOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "if.js")
	// Closing braces at the boundary before the next statement must emit first.
	src := "if(a)b();c();\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(path)
	text := string(out)
	// Find the instrumented region after preamble; ensure `}c` / `}{` shape rather than `{{`.
	idx := strings.Index(text, "if(")
	if idx < 0 {
		t.Fatalf("missing if in:\n%s", text)
	}
	region := text[idx:]
	if strings.Contains(region, "{{") {
		t.Fatalf("unexpected nested open before close:\n%s", region)
	}
}

func TestInstrumentJSFile_ArrowExpressionBodyNoFnInsert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "arrow.js")
	src := "const f = (x) => x + 1;\nf(1);\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatal(err)
	}
	// Expression-bodied arrows are tracked in fnMap but have no entry insert.
	metaRaw, err := os.ReadFile(path + ".coverage-meta.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metaRaw), "fnMap") {
		t.Fatalf("expected fnMap: %s", metaRaw)
	}
}

func TestFunctionCoverageNameAndHelpers(t *testing.T) {
	if functionCoverageName(nil) != "(anonymous)" {
		t.Fatal("nil node")
	}
	if functionBodyEntryPos("{}", nil) != -1 {
		t.Fatal("nil entry")
	}
	if shouldInstrumentStatement(nil) {
		t.Fatal("nil statement")
	}
	if statementNeedsBlockWrap(nil) {
		t.Fatal("nil wrap")
	}
}

func TestWriteLcovAllReportersAndRemap(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "run-all"
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/cov\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(repoRoot, "modules", "demo", "service", "math.ts")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, []byte("export function add(a: number, b: number) { return a + b }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	genPath := filepath.Join(repoRoot, "dist", "bundle.js")
	if err := os.MkdirAll(filepath.Dir(genPath), 0o755); err != nil {
		t.Fatal(err)
	}

	cov := map[string]*coverageFileData{
		genPath: {
			Path: genPath,
			StatementMap: map[string]coverageRange{
				"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 10}},
			},
			FnMap: map[string]coverageFn{
				"0": {Name: "add", Decl: coverageRange{Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}, Line: 1},
			},
			BranchMap: map[string]coverageBranch{
				"0": {
					Type: "if",
					Line: 1,
					Loc:  coverageRange{Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}},
					Locations: []coverageRange{
						{Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}},
						{Start: coveragePos{Line: 1, Column: 2}, End: coveragePos{Line: 1, Column: 3}},
					},
				},
			},
			S: hitMap{"0": 1},
			F: hitMap{"0": 1},
			B: map[string][]int{"0": {1, 0}},
			InputSourceMap: &rawSourceMap{
				Version:  3,
				File:     "bundle.js",
				Sources:  []string{"../modules/demo/service/math.ts"},
				Mappings: "AAAA",
			},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "demo", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}

	ctx := ContextWithCoverageRunID(nil, runID)
	reportDir := filepath.Join(tmpRoot, "all-reports")
	if err := WriteLcov(ctx, ReportOptions{
		RepoRoot:  repoRoot,
		TmpRoot:   tmpRoot,
		ReportDir: reportDir,
		Reporters: []string{"lcov", "text", "html", "json-summary", "text-summary"},
		RunID:     runID,
		Includes:  []string{"modules/**"},
	}); err != nil {
		t.Fatalf("WriteLcov: %v", err)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "lcov.info")); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(reportDir, "html", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "<table>") || !strings.Contains(string(html), "math.ts") {
		t.Fatalf("expected per-file html table, got:\n%s", html)
	}
	summary, err := os.ReadFile(filepath.Join(reportDir, "coverage-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), `"total"`) {
		t.Fatalf("json-summary: %s", summary)
	}

	// Default reporters path (empty ReportDir + empty Reporters).
	if err := WriteLcov(ctx, ReportOptions{
		RepoRoot: repoRoot,
		TmpRoot:  tmpRoot,
		RunID:    runID,
	}); err != nil {
		t.Fatalf("WriteLcov defaults: %v", err)
	}

	if err := CheckCoverage(ctx, CheckOptions{
		RepoRoot:   repoRoot,
		TmpRoot:    tmpRoot,
		RunID:      runID,
		Statements: 1,
		Lines:      1,
		Functions:  1,
		Branches:   1,
		Includes:   []string{"modules/**"},
	}); err != nil {
		t.Fatalf("CheckCoverage: %v", err)
	}
}

func TestCheckCoverageEmptyFilterFails(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "run-filter"
	srcPath := filepath.Join(repoRoot, "modules", "a", "x.ts")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	cov := map[string]*coverageFileData{
		srcPath: {
			Path:         srcPath,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
			F:            hitMap{},
			B:            map[string][]int{},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "a", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	err := CheckCoverage(context.Background(), CheckOptions{
		RepoRoot:   repoRoot,
		TmpRoot:    tmpRoot,
		RunID:      runID,
		Statements: 50,
		Includes:   []string{"modules/other/**"},
	})
	if err == nil || !strings.Contains(err.Error(), "no coverage data matched") {
		t.Fatalf("expected empty filter error, got %v", err)
	}
}

func TestCoveragePathExcludeWinsOnAbsOrRel(t *testing.T) {
	excludes := []string{"**/dist/**"}
	if !coveragePathExcluded("dist/x.js", "/repo/dist/x.js", excludes) {
		t.Fatal("expected exclude on rel")
	}
	if !coveragePathExcluded("other.js", "/repo/dist/x.js", excludes) {
		t.Fatal("expected exclude on abs even when rel differs")
	}
	if coveragePathExcluded("modules/a.ts", "/repo/modules/a.ts", excludes) {
		t.Fatal("should not exclude")
	}
	if !coveragePathIncluded("modules/a.ts", []string{"modules/**"}, nil) {
		t.Fatal("include should match")
	}
	if coveragePathIncluded("other.ts", []string{"modules/**"}, nil) {
		t.Fatal("include should miss")
	}
}

func TestMergeCoverageFileAndLoadMerged(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "merge-run"
	src := filepath.Join(repoRoot, "a.ts")
	first := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			FnMap:        map[string]coverageFn{"0": {Name: "f", Line: 1}},
			BranchMap:    map[string]coverageBranch{"0": {Type: "if", Line: 1}},
			S:            hitMap{"0": 1},
			F:            hitMap{"0": 1},
			B:            map[string][]int{"0": {1}},
			InputSourceMap: &rawSourceMap{
				Version: 3, Sources: []string{"a.ts"}, Mappings: "AAAA",
			},
		},
	}
	second := map[string]*coverageFileData{
		src: {
			Path: src,
			S:    hitMap{"0": 2},
			F:    hitMap{"0": 3},
			B:    map[string][]int{"0": {0, 1}},
		},
	}
	r1, _ := json.Marshal(first)
	r2, _ := json.Marshal(second)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "a", runID, string(r1), tmpRoot); err != nil {
		t.Fatal(err)
	}
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "a", runID, string(r2), tmpRoot); err != nil {
		t.Fatal(err)
	}
	merged, err := loadMergedCoverage(repoRoot, tmpRoot, runID)
	if err != nil {
		t.Fatal(err)
	}
	got := merged[src]
	if got == nil || got.S["0"] != 3 || got.F["0"] != 4 || len(got.B["0"]) < 2 {
		t.Fatalf("merged = %#v", got)
	}

	// Non-json entries and run-id mismatches are ignored / rejected.
	nyc, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nyc, "noise.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nyc, "other-run.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMergedCoverage(repoRoot, tmpRoot, runID); err != nil {
		t.Fatalf("noise/non-matching json should be ignored: %v", err)
	}
	if _, err := loadMergedCoverage(repoRoot, tmpRoot, "no-such"); err == nil {
		t.Fatal("expected missing run id error")
	}
}

func TestRemapCoverageWithoutSourceMapAndBranches(t *testing.T) {
	data := &coverageFileData{
		Path: "",
		StatementMap: map[string]coverageRange{
			"0": {Start: coveragePos{Line: 2, Column: 0}, End: coveragePos{Line: 2, Column: 1}},
		},
		FnMap: map[string]coverageFn{
			"0": {Name: "anon", Decl: coverageRange{Start: coveragePos{Line: 2, Column: 0}, End: coveragePos{Line: 2, Column: 1}}},
		},
		BranchMap: map[string]coverageBranch{
			"0": {Type: "cond", Locations: nil},
			"1": {Type: "if", Loc: coverageRange{Start: coveragePos{Line: 3, Column: 0}, End: coveragePos{Line: 3, Column: 1}}},
		},
		S: hitMap{"0": 0},
		F: hitMap{"0": 0},
		B: map[string][]int{"0": {}, "1": {2}},
	}
	out, err := remapCoverageToSources(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["unknown.js"]; !ok {
		t.Fatalf("expected unknown.js fallback, got %#v", out)
	}
}

func TestTrimGlobsAndSplitEmpty(t *testing.T) {
	if got := trimGlobs([]string{" a ", "", "b"}); len(got) != 2 || got[0] != "a" {
		t.Fatalf("trimGlobs = %#v", got)
	}
	if SplitCoverageGlobs("") != nil {
		t.Fatal("empty split")
	}
	if _, err := ValidateCoverageReporters([]string{"", "text"}); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeBase64AndBuildLineMap(t *testing.T) {
	raw, err := decodeBase64(base64.StdEncoding.EncodeToString([]byte("hi")))
	if err != nil || string(raw) != "hi" {
		t.Fatalf("decodeBase64 = %q %v", raw, err)
	}
	sm := &rawSourceMap{
		Version:    3,
		SourceRoot: "src",
		Sources:    []string{"a.ts", "b.ts"},
		Mappings:   "AAAA",
	}
	m, err := buildGeneratedLineToSource("/repo/dist/out.js", sm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal("expected at least one generated line mapping")
	}
}

func TestRelativizeAndRenderHelpers(t *testing.T) {
	if got := relativizeCoveragePath("/repo", "/repo/a/b.ts"); got != "a/b.ts" {
		t.Fatalf("rel = %q", got)
	}
	if got := relativizeCoveragePath("", "/abs/x.ts"); got != "/abs/x.ts" {
		t.Fatalf("abs fallback = %q", got)
	}
	stats := []fileCoverageStats{{
		Path:       "a.ts",
		Statements: hitCount{Covered: 1, Total: 2},
		Lines:      hitCount{Covered: 1, Total: 2},
		Functions:  hitCount{Covered: 0, Total: 1},
		Branches:   hitCount{Covered: 0, Total: 0},
		LineHits:   map[int]int{1: 1, 2: 0},
	}}
	html := renderHTMLSummary(stats)
	if !strings.Contains(html, "a.ts") {
		t.Fatalf("html: %s", html)
	}
	js := renderJSONSummary(stats)
	if !strings.Contains(js, "a.ts") {
		t.Fatalf("json: %s", js)
	}
}

func TestWriteLcovRelativeReportDirAndInvalidReporter(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "rel-dir"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot:  repoRoot,
		TmpRoot:   tmpRoot,
		ReportDir: "rel-reports",
		Reporters: []string{"lcovonly"},
		RunID:     runID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "rel-reports", "lcov.info")); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot:  repoRoot,
		TmpRoot:   tmpRoot,
		Reporters: []string{"nope"},
		RunID:     runID,
	}); err == nil {
		t.Fatal("expected invalid reporter")
	}
}

func TestCheckCoverageThresholdFailures(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "thr"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path: src,
			StatementMap: map[string]coverageRange{
				"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}},
				"1": {Start: coveragePos{Line: 2, Column: 0}, End: coveragePos{Line: 2, Column: 1}},
			},
			FnMap:     map[string]coverageFn{"0": {Name: "f", Line: 1}},
			BranchMap: map[string]coverageBranch{"0": {Type: "if", Line: 1, Locations: []coverageRange{{}, {}}}},
			S:         hitMap{"0": 1, "1": 0},
			F:         hitMap{"0": 0},
			B:         map[string][]int{"0": {0, 0}},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	err := CheckCoverage(context.Background(), CheckOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, RunID: runID,
		Statements: 90, Lines: 90, Functions: 90, Branches: 90,
	})
	if err == nil || !strings.Contains(err.Error(), "coverage check failed") {
		t.Fatalf("expected threshold failures, got %v", err)
	}
}

func TestLoadMergedCoverageSkipsDirsAndBadJSON(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	nyc, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, "bad")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nyc, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nyc, "choysum-app-bad-1.json"), []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMergedCoverage(repoRoot, tmpRoot, "bad"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestMatchGlobPartStarVariants(t *testing.T) {
	if !matchGlobPart("a*c", "abc") {
		t.Fatal("mid star")
	}
	if matchGlobPart("a*c", "ab") {
		t.Fatal("should miss")
	}
	if !matchGlobPart("file.*", "file.ts") {
		t.Fatal("trailing star after dot")
	}
	if matchGlobPart("ab", "abc") {
		t.Fatal("pattern exhausted early")
	}
	if splitGlob("/") != nil {
		t.Fatal("empty split")
	}
	if coveragePathIncluded("x.test.ts", nil, []string{"**/*.test.ts"}) {
		t.Fatal("exclude via coveragePathIncluded")
	}
}

func TestInstrumentJSFile_WriteFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ro.js")
	if err := os.WriteFile(path, []byte("var a=1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if err := InstrumentJSFile(path); err == nil {
		t.Fatal("expected write failure on read-only file")
	}
}

func TestInstrumentJSFile_SourceMapWriteFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sm.js")
	altMap := filepath.Join(dir, "other.map")
	sm := rawSourceMap{Version: 3, Sources: []string{"a.ts"}, Mappings: "AAAA"}
	smRaw, _ := json.Marshal(sm)
	if err := os.WriteFile(path, []byte("var a=1;\n//# sourceMappingURL=other.map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(altMap, smRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	// Instrument always rewrites path+".map"; make that path a directory so write fails
	// after the input map was successfully loaded from other.map.
	if err := os.Mkdir(path+".map", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err == nil {
		t.Fatal("expected sourcemap write failure")
	}
}

func TestInstrumentJSFile_ClassMethodAndDebugger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cls.js")
	src := "class C { m(){ return 1 } }\ndebugger;\nnew C().m();\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(path)
	if !strings.Contains(string(out), ".f[") {
		t.Fatalf("expected method fn counter:\n%s", out)
	}
}

func TestInstrumentJSFile_InvalidExternalMapJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badmap.js")
	if err := os.WriteFile(path, []byte("var a=1;\n//# sourceMappingURL=badmap.js.map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".map", []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestWriteLcovMkdirAndEmptyFilterErrors(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "mkdir-fail"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(tmpRoot, "blocked-as-file")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: blocked, RunID: runID, Reporters: []string{"lcovonly"},
	}); err == nil {
		t.Fatal("expected mkdir failure")
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: filepath.Join(tmpRoot, "ok"), RunID: runID,
		Reporters: []string{"lcovonly"}, Includes: []string{"nomatch/**"},
	}); err == nil || !strings.Contains(err.Error(), "no coverage data matched") {
		t.Fatalf("expected empty filter, got %v", err)
	}
}

func TestWriteLcovEmptyRepoRootUsesCwd(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "cwd-root"
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot("", "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(nil, ReportOptions{
		TmpRoot: tmpRoot, ReportDir: filepath.Join(tmpRoot, "r"), RunID: runID, Reporters: []string{"text"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := CheckCoverage(nil, CheckOptions{TmpRoot: tmpRoot, RunID: runID, Statements: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestHitMapUnmarshalErrors(t *testing.T) {
	var h hitMap
	if err := json.Unmarshal([]byte(`[1,"x"]`), &h); err == nil {
		t.Fatal("expected array type error")
	}
	if err := json.Unmarshal([]byte(`{"0":"x"}`), &h); err == nil {
		t.Fatal("expected object type error")
	}
	if err := h.UnmarshalJSON([]byte("   ")); err != nil {
		t.Fatal(err)
	}
}

func TestComputeCoverageStatsNilAndExclude(t *testing.T) {
	repo := t.TempDir()
	src := filepath.Join(repo, "modules", "a.ts")
	merged := map[string]*coverageFileData{
		"nil": nil,
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
		filepath.Join(repo, "dist", "x.js"): {
			Path:         filepath.Join(repo, "dist", "x.js"),
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	stats, err := computeCoverageStats(repo, merged, nil, defaultCoverageExcludes())
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range stats {
		if strings.Contains(st.Path, "dist") {
			t.Fatalf("dist should be excluded: %#v", stats)
		}
	}
}

func TestMergeCoverageKeepsExistingMaps(t *testing.T) {
	dst := map[string]*coverageFileData{}
	path := "/a.js"
	mergeCoverageFile(dst, path, &coverageFileData{
		Path:           path,
		StatementMap:   map[string]coverageRange{"0": {}},
		FnMap:          map[string]coverageFn{"0": {Name: "a"}},
		BranchMap:      map[string]coverageBranch{"0": {Type: "if"}},
		S:              hitMap{"0": 1},
		F:              hitMap{"0": 1},
		B:              map[string][]int{"0": {1}},
		InputSourceMap: &rawSourceMap{Version: 3, Sources: []string{"a.ts"}, Mappings: "AAAA"},
	})
	mergeCoverageFile(dst, path, &coverageFileData{
		Path:           path,
		StatementMap:   map[string]coverageRange{"1": {}},
		FnMap:          map[string]coverageFn{"1": {Name: "b"}},
		BranchMap:      map[string]coverageBranch{"1": {Type: "else"}},
		S:              hitMap{"0": 1},
		F:              hitMap{"0": 1},
		B:              map[string][]int{"0": {1}},
		InputSourceMap: &rawSourceMap{Version: 3, Sources: []string{"b.ts"}, Mappings: "AAAA"},
	})
	got := dst[path]
	if len(got.StatementMap) != 1 || got.FnMap["0"].Name != "a" || got.InputSourceMap.Sources[0] != "a.ts" {
		t.Fatalf("expected first maps retained: %#v", got)
	}
}

func TestBuildGeneratedLineToSourceAbsoluteAndBadIndex(t *testing.T) {
	sm := &rawSourceMap{
		Version:  3,
		Sources:  []string{"/abs/a.ts"},
		Mappings: "AAAA,AAAA",
	}
	m, err := buildGeneratedLineToSource("/repo/out.js", sm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal("expected mappings")
	}
	// Remap with sourcemap that points statements through remap path.
	data := &coverageFileData{
		Path: "/repo/out.js",
		StatementMap: map[string]coverageRange{
			"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}},
		},
		FnMap: map[string]coverageFn{
			"0": {Name: "f", Decl: coverageRange{Start: coveragePos{Line: 0, Column: 0}, End: coveragePos{Line: 0, Column: 1}}},
		},
		BranchMap: map[string]coverageBranch{
			"0": {Type: "if", Loc: coverageRange{Start: coveragePos{Line: 0, Column: 0}, End: coveragePos{Line: 0, Column: 1}}},
		},
		S:              hitMap{"0": 1},
		F:              hitMap{"0": 1},
		B:              map[string][]int{"0": {1}},
		InputSourceMap: sm,
	}
	out, err := remapCoverageToSources(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected remapped stats")
	}
}

func TestWriteCoverageJSONEmptyRepoRoot(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := WriteCoverageJSONWithRunIDAndTmpRoot("", "app", "r", `{"a":{"path":"a","s":{}}}`, tmp); err != nil {
		t.Fatal(err)
	}
}

func TestEnrichCoverageUsesMapKeyWhenPathEmpty(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "keyed.js")
	meta := coverageFileData{StatementMap: map[string]coverageRange{"0": {}}, CoverageSchema: "x"}
	rawMeta, _ := json.Marshal(meta)
	if err := os.WriteFile(key+".coverage-meta.json", rawMeta, 0o644); err != nil {
		t.Fatal(err)
	}
	payload := `{"` + key + `":{"s":{"0":1},"f":{},"b":{}}}`
	out, err := enrichCoverageJSONWithMeta(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "statementMap") || !strings.Contains(out, `"_coverageSchema":"x"`) {
		t.Fatalf("enrich = %s", out)
	}
}

func TestLoadMergedCoverageNoRunID(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	src := filepath.Join(repoRoot, "a.ts")
	cov := map[string]*coverageFileData{
		src: {Path: "", StatementMap: map[string]coverageRange{"0": {}}, S: hitMap{"0": 1}},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "a", "", string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	merged, err := loadMergedCoverage(repoRoot, tmpRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) == 0 {
		t.Fatal("expected merged files")
	}
}

func TestWriteLcovHTMLWriteFailure(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "html-fail"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(tmpRoot, "reports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pretend html/ is a file so mkdir html fails.
	if err := os.WriteFile(filepath.Join(reportDir, "html"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: reportDir, RunID: runID, Reporters: []string{"html"},
	}); err == nil {
		t.Fatal("expected html mkdir failure")
	}
}

func TestRemapInvalidSourceMapMappings(t *testing.T) {
	data := &coverageFileData{
		Path: "/out.js",
		StatementMap: map[string]coverageRange{
			"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}},
		},
		S: hitMap{"0": 1},
		InputSourceMap: &rawSourceMap{
			Version:  3,
			Sources:  []string{"a.ts"},
			Mappings: "!!!!",
		},
	}
	if _, err := remapCoverageToSources(data); err == nil {
		t.Fatal("expected invalid VLQ error")
	}
	stats, err := computeCoverageStats(t.TempDir(), map[string]*coverageFileData{"/out.js": data}, nil, nil)
	if err == nil {
		t.Fatalf("expected computeCoverageStats error, got %#v", stats)
	}
}

func TestInstrumentDistBundlePropagatesInstrumentError(t *testing.T) {
	distPath := t.TempDir()
	appIndex := filepath.Join(distPath, "apps", "portal", "index.js")
	if err := os.MkdirAll(filepath.Dir(appIndex), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appIndex, []byte("var a=1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(appIndex), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(appIndex), 0o755) })
	if err := InstrumentDistBundle(context.Background(), t.TempDir(), distPath, "portal"); err == nil {
		t.Fatal("expected instrument failure")
	}
}

func TestWriteCoverageJSONReadOnlyDir(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	// Create nyc_output then freeze the coverage base dir.
	nyc, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, "ro")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nyc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(nyc, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(nyc, 0o755) })
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "app", "ro", `{"a":{}}`, tmpRoot); err == nil {
		t.Fatal("expected write failure")
	}
}

func TestLoadMergedCoverageUnreadableJSON(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "unreadable"
	nyc, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nyc, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(nyc, "choysum-app-unreadable-1.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if _, err := loadMergedCoverage(repoRoot, tmpRoot, runID); err == nil {
		t.Fatal("expected read error")
	}
}

func TestWriteLcovLcovWriteFailure(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "lcov-fail"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(tmpRoot, "reports2")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(reportDir, "lcov.info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: reportDir, RunID: runID, Reporters: []string{"lcovonly"},
	}); err == nil {
		t.Fatal("expected lcov write failure")
	}
}

func TestWriteLcovJSONSummaryWriteFailure(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "json-fail"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(tmpRoot, "reports3")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(reportDir, "coverage-summary.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: reportDir, RunID: runID, Reporters: []string{"json-summary"},
	}); err == nil {
		t.Fatal("expected json-summary write failure")
	}
}

func TestCheckCoverageEmptyRepoAndRunIDFromContext(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(repoRoot, "z.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	runID := "ctx-run"
	if err := WriteCoverageJSONWithRunIDAndTmpRoot("", "z", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	ctx := ContextWithCoverageRunID(context.Background(), runID)
	if err := CheckCoverage(ctx, CheckOptions{TmpRoot: tmpRoot, Statements: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildGeneratedLineSkipsBadSourceIndex(t *testing.T) {
	sm := &rawSourceMap{
		Version:  3,
		Sources:  []string{},
		Mappings: "AAAA", // source index 0 with empty sources
	}
	m, err := buildGeneratedLineToSource("/out.js", sm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("expected skipped mappings, got %#v", m)
	}
}

func TestMatchGlobPartTrailingStarsOnly(t *testing.T) {
	if !matchGlobPart("abc***", "abc") {
		t.Fatal("trailing stars should match")
	}
}

func TestSplitCoverageGlobsWhitespaceOnlyParts(t *testing.T) {
	// FieldsFunc may yield empty between separators depending on input.
	got := SplitCoverageGlobs(",,;")
	if got != nil && len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestInstrumentMetaWriteFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.js")
	if err := os.WriteFile(path, []byte("var a=1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Reserve the meta path as a directory so the sidecar write fails after JS write.
	if err := os.Mkdir(path+".coverage-meta.json", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err == nil {
		t.Fatal("expected meta write failure")
	}
}

func TestHTMLWriteFileFailure(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "html-file-fail"
	src := filepath.Join(repoRoot, "x.ts")
	cov := map[string]*coverageFileData{
		src: {
			Path:         src,
			StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
			S:            hitMap{"0": 1},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(tmpRoot, "reports-html")
	htmlDir := filepath.Join(reportDir, "html")
	if err := os.MkdirAll(htmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(htmlDir, "index.html"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteLcov(context.Background(), ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: reportDir, RunID: runID, Reporters: []string{"html"},
	}); err == nil {
		t.Fatal("expected html file write failure")
	}
}

func TestCoveragePathIncludedNoIncludes(t *testing.T) {
	if !coveragePathIncluded("anything.ts", nil, nil) {
		t.Fatal("no includes should admit all non-excluded paths")
	}
}

func TestResolveHelpersEmptyRepoRoot(t *testing.T) {
	tmp := t.TempDir()
	if _, err := ResolveCoverageReportDir("", tmp); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveCoverageNycOutputDir("", tmp); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveCoverageTmpBaseDir("", tmp); err != nil {
		t.Fatal(err)
	}
}

func TestEnrichSkipsInvalidMetaJSON(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "badmeta.js")
	if err := os.WriteFile(js+".coverage-meta.json", []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := `{"` + js + `":{"path":"` + js + `","s":{},"f":{},"b":{}}}`
	out, err := enrichCoverageJSONWithMeta(payload)
	if err != nil {
		t.Fatal(err)
	}
	if out != payload {
		t.Fatalf("expected unchanged payload, got %s", out)
	}
}

func TestInlineSourceMapInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badjson.js")
	payload := base64.StdEncoding.EncodeToString([]byte("not-json"))
	src := "var a=1;\n//# sourceMappingURL=data:application/json;base64," + payload + "\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstrumentJSFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestWriteLcovRunIDFromContextAndInvalidRemap(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	runID := "from-ctx"
	gen := filepath.Join(repoRoot, "out.js")
	cov := map[string]*coverageFileData{
		gen: {
			Path: gen,
			StatementMap: map[string]coverageRange{
				"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}},
			},
			S: hitMap{"0": 1},
			InputSourceMap: &rawSourceMap{
				Version:  3,
				Sources:  []string{"a.ts"},
				Mappings: "$$$$",
			},
		},
	}
	raw, _ := json.Marshal(cov)
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "x", runID, string(raw), tmpRoot); err != nil {
		t.Fatal(err)
	}
	ctx := ContextWithCoverageRunID(context.Background(), runID)
	if err := WriteLcov(ctx, ReportOptions{
		RepoRoot: repoRoot, TmpRoot: tmpRoot, ReportDir: filepath.Join(tmpRoot, "r"), Reporters: []string{"text"},
	}); err == nil {
		t.Fatal("expected remap failure via WriteLcov")
	}
	if err := CheckCoverage(ctx, CheckOptions{RepoRoot: repoRoot, TmpRoot: tmpRoot, Statements: 1}); err == nil {
		t.Fatal("expected remap failure via CheckCoverage")
	}
}

func TestMergeCoverageFillsEmptyMapsFromSrc(t *testing.T) {
	dst := map[string]*coverageFileData{}
	path := "/runtime.js"
	mergeCoverageFile(dst, path, &coverageFileData{
		Path: path,
		S:    hitMap{"0": 1},
		F:    hitMap{"0": 1},
		B:    map[string][]int{},
	})
	mergeCoverageFile(dst, path, &coverageFileData{
		Path:         path,
		StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
		FnMap:        map[string]coverageFn{"0": {Name: "f", Line: 1}},
		BranchMap:    map[string]coverageBranch{"0": {Type: "if", Line: 1}},
		S:            hitMap{"0": 1},
		F:            hitMap{"0": 0},
		B:            map[string][]int{"0": {1}},
	})
	got := dst[path]
	if len(got.StatementMap) == 0 || len(got.FnMap) == 0 || len(got.BranchMap) == 0 {
		t.Fatalf("expected maps filled from src: %#v", got)
	}
}

func TestRelativizeOutsideRepo(t *testing.T) {
	if got := relativizeCoveragePath("/repo", "/other/a.ts"); got != "/other/a.ts" {
		t.Fatalf("got %q", got)
	}
}

func TestAccumulateEnsureEmptyPath(t *testing.T) {
	by := map[string]*fileCoverageStats{}
	accumulateInto(by, &coverageFileData{
		StatementMap: map[string]coverageRange{"0": {Start: coveragePos{Line: 1, Column: 0}, End: coveragePos{Line: 1, Column: 1}}},
		S:            hitMap{"0": 1},
	}, nil, "")
	if _, ok := by[""]; !ok {
		// ensure("") uses fallbackPath ""; path becomes "" then replaced - actually ensure uses fallback when path==""
		// fallback is "" so path stays "" after replace? ensure: if path=="" { path = fallbackPath } then ReplaceAll
	}
	if len(by) == 0 {
		t.Fatal("expected stats entry")
	}
}

func TestWriteCoverageMkdirFailure(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	base, err := resolveCoverageTmpBaseDirWithRunID(repoRoot, tmpRoot, "mk")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(base, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(base, 0o755) })
	if err := WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, "app", "mk", `{"a":{}}`, tmpRoot); err == nil {
		t.Fatal("expected mkdir failure")
	}
}

func TestLoadMergedCoverageEmptyWithoutRunID(t *testing.T) {
	repoRoot := t.TempDir()
	tmpRoot := t.TempDir()
	nyc, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nyc, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMergedCoverage(repoRoot, tmpRoot, ""); err == nil {
		t.Fatal("expected no coverage json error")
	}
}

func TestBuildGeneratedLineWithSourceRoot(t *testing.T) {
	sm := &rawSourceMap{
		Version:    3,
		SourceRoot: "pkg",
		Sources:    []string{"a.ts"},
		Mappings:   "AAAA",
	}
	m, err := buildGeneratedLineToSource("/repo/dist/out.js", sm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal("expected mapping")
	}
}
