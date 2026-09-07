// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tssourcemap "github.com/buke/typescript-go-internal/v7/pkg/sourcemap"
	xfmt "golang.org/x/exp/errors/fmt"
)

// ReportOptions configures WriteLcov (Go replacement for `nyc report`).
type ReportOptions struct {
	RepoRoot  string
	TmpRoot   string
	ReportDir string
	Reporters []string
	Includes  []string
	Excludes  []string
	RunID     string
}

// CheckOptions configures CheckCoverage (Go replacement for `nyc check-coverage`).
type CheckOptions struct {
	RepoRoot   string
	TmpRoot    string
	Includes   []string
	Excludes   []string
	RunID      string
	Lines      int
	Functions  int
	Branches   int
	Statements int
}

type coverageFileData struct {
	Path           string                    `json:"path"`
	StatementMap   map[string]coverageRange  `json:"statementMap"`
	FnMap          map[string]coverageFn     `json:"fnMap"`
	BranchMap      map[string]coverageBranch `json:"branchMap"`
	S              hitMap                    `json:"s"`
	F              hitMap                    `json:"f"`
	B              map[string][]int          `json:"b"`
	InputSourceMap *rawSourceMap             `json:"inputSourceMap,omitempty"`
	Hash           string                    `json:"hash,omitempty"`
	CoverageSchema string                    `json:"_coverageSchema,omitempty"`
}

// hitMap accepts Istanbul object maps or Array(n).fill(0) JSON arrays.
type hitMap map[string]int

func (h *hitMap) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		*h = hitMap{}
		return nil
	}
	if raw[0] == '[' {
		var arr []int
		if err := json.Unmarshal(raw, &arr); err != nil {
			return err
		}
		out := make(hitMap, len(arr))
		for i, v := range arr {
			out[strconv.Itoa(i)] = v
		}
		*h = out
		return nil
	}
	var obj map[string]int
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}
	*h = hitMap(obj)
	return nil
}

func (h hitMap) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]int(h))
}

type coverageRange struct {
	Start coveragePos `json:"start"`
	End   coveragePos `json:"end"`
}

type coveragePos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type coverageFn struct {
	Name string        `json:"name"`
	Decl coverageRange `json:"decl"`
	Loc  coverageRange `json:"loc"`
	Line int           `json:"line"`
}

type coverageBranch struct {
	Loc       coverageRange   `json:"loc"`
	Type      string          `json:"type"`
	Locations []coverageRange `json:"locations"`
	Line      int             `json:"line"`
}

type rawSourceMap struct {
	Version        int      `json:"version"`
	File           string   `json:"file,omitempty"`
	SourceRoot     string   `json:"sourceRoot,omitempty"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent,omitempty"`
	Names          []string `json:"names,omitempty"`
	Mappings       string   `json:"mappings"`
}

type fileCoverageStats struct {
	Path       string
	Statements hitCount
	Lines      hitCount
	Functions  hitCount
	Branches   hitCount
	LineHits   map[int]int
}

type hitCount struct {
	Covered int
	Total   int
}

func (h hitCount) Percent() float64 {
	if h.Total == 0 {
		return 100
	}
	return 100 * float64(h.Covered) / float64(h.Total)
}

// WriteLcov merges Istanbul-shaped coverage JSON from nyc_output and writes
// reporters under ReportDir. Text reporters print to stderr (TAP-safe).
func WriteLcov(ctx context.Context, opts ReportOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = CoverageRunIDFromContext(ctx)
	}

	reportDir := strings.TrimSpace(opts.ReportDir)
	var err error
	if reportDir == "" {
		reportDir, err = ResolveCoverageReportDirWithRunID(repoRoot, opts.TmpRoot, runID)
		if err != nil {
			return xfmt.Errorf("resolve coverage report dir: %w", err)
		}
	} else if !filepath.IsAbs(reportDir) {
		reportDir = filepath.Join(repoRoot, reportDir)
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return xfmt.Errorf("mkdir coverage report dir: %w", err)
	}

	reporters := opts.Reporters
	if len(reporters) == 0 {
		reporters = []string{"text", "lcovonly"}
	}
	validated, err := ValidateCoverageReporters(reporters)
	if err != nil {
		return err
	}

	merged, err := loadMergedCoverage(repoRoot, opts.TmpRoot, runID)
	if err != nil {
		return err
	}

	stats, err := computeCoverageStats(repoRoot, merged, trimGlobs(opts.Includes), mergeExcludeGlobs(opts.Excludes))
	if err != nil {
		return err
	}
	if len(stats) == 0 {
		return xfmt.Errorf("no coverage data matched include/exclude filters")
	}

	for _, reporter := range validated {
		switch reporter {
		case "lcovonly", "lcov":
			lcovPath := filepath.Join(reportDir, "lcov.info")
			if err := os.WriteFile(lcovPath, []byte(renderLcov(stats)), 0o644); err != nil {
				return xfmt.Errorf("write lcov.info: %w", err)
			}
			fmt.Fprintf(os.Stderr, "choysum test: wrote coverage report %s\n", lcovPath)
		case "text", "text-summary":
			fmt.Fprint(os.Stderr, renderTextSummary(stats))
		case "html":
			htmlDir := filepath.Join(reportDir, "html")
			if err := os.MkdirAll(htmlDir, 0o755); err != nil {
				return xfmt.Errorf("mkdir html report dir: %w", err)
			}
			if err := os.WriteFile(filepath.Join(htmlDir, "index.html"), []byte(renderHTMLSummary(stats)), 0o644); err != nil {
				return xfmt.Errorf("write html report: %w", err)
			}
		case "json-summary":
			if err := os.WriteFile(filepath.Join(reportDir, "coverage-summary.json"), []byte(renderJSONSummary(stats)), 0o644); err != nil {
				return xfmt.Errorf("write json-summary: %w", err)
			}
		}
	}
	return nil
}

// CheckCoverage fails when remapped coverage is below configured thresholds.
// A threshold of 0 disables that metric (same as nyc CLI flags).
func CheckCoverage(ctx context.Context, opts CheckOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Lines <= 0 && opts.Functions <= 0 && opts.Branches <= 0 && opts.Statements <= 0 {
		return xfmt.Errorf("--coverage-check requires at least one threshold flag (e.g. --coverage-lines 80)")
	}

	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = CoverageRunIDFromContext(ctx)
	}

	merged, err := loadMergedCoverage(repoRoot, opts.TmpRoot, runID)
	if err != nil {
		return err
	}
	stats, err := computeCoverageStats(repoRoot, merged, trimGlobs(opts.Includes), mergeExcludeGlobs(opts.Excludes))
	if err != nil {
		return err
	}
	if len(stats) == 0 {
		return xfmt.Errorf("no coverage data matched include/exclude filters")
	}

	totals := aggregateTotals(stats)
	var failures []string
	if opts.Statements > 0 && totals.Statements.Percent() < float64(opts.Statements) {
		failures = append(failures, fmt.Sprintf("statements: want >= %d%%, got %.2f%%", opts.Statements, totals.Statements.Percent()))
	}
	if opts.Lines > 0 && totals.Lines.Percent() < float64(opts.Lines) {
		failures = append(failures, fmt.Sprintf("lines: want >= %d%%, got %.2f%%", opts.Lines, totals.Lines.Percent()))
	}
	if opts.Functions > 0 && totals.Functions.Percent() < float64(opts.Functions) {
		failures = append(failures, fmt.Sprintf("functions: want >= %d%%, got %.2f%%", opts.Functions, totals.Functions.Percent()))
	}
	if opts.Branches > 0 && totals.Branches.Percent() < float64(opts.Branches) {
		failures = append(failures, fmt.Sprintf("branches: want >= %d%%, got %.2f%%", opts.Branches, totals.Branches.Percent()))
	}
	if len(failures) > 0 {
		return xfmt.Errorf("coverage check failed: %s", strings.Join(failures, "; "))
	}
	fmt.Fprintf(os.Stderr, "choysum test: coverage check passed (statements %.2f%% lines %.2f%% functions %.2f%% branches %.2f%%)\n",
		totals.Statements.Percent(), totals.Lines.Percent(), totals.Functions.Percent(), totals.Branches.Percent())
	return nil
}

func trimGlobs(globs []string) []string {
	out := make([]string, 0, len(globs))
	for _, g := range globs {
		g = strings.TrimSpace(g)
		if g != "" {
			out = append(out, g)
		}
	}
	return out
}

func loadMergedCoverage(repoRoot, tmpRoot, runID string) (map[string]*coverageFileData, error) {
	nycOutputDir, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return nil, xfmt.Errorf("resolve nyc_output dir: %w", err)
	}
	entries, err := os.ReadDir(nycOutputDir)
	if err != nil {
		return nil, xfmt.Errorf("read nyc_output: %w", err)
	}

	runID = strings.TrimSpace(runID)
	token := ""
	if runID != "" {
		token = "-" + runID + "-"
	}

	merged := map[string]*coverageFileData{}
	matched := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if token != "" && !strings.Contains(entry.Name(), token) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(nycOutputDir, entry.Name()))
		if err != nil {
			return nil, xfmt.Errorf("read coverage json %s: %w", entry.Name(), err)
		}
		var fileMap map[string]*coverageFileData
		if err := json.Unmarshal(raw, &fileMap); err != nil {
			return nil, xfmt.Errorf("parse coverage json %s: %w", entry.Name(), err)
		}
		for path, data := range fileMap {
			if data == nil {
				continue
			}
			if data.Path == "" {
				data.Path = path
			}
			mergeCoverageFile(merged, path, data)
		}
		matched++
	}
	if matched == 0 {
		if runID != "" {
			return nil, xfmt.Errorf("no coverage json found in nyc_output for run-id %q", runID)
		}
		return nil, xfmt.Errorf("no coverage json found in nyc_output")
	}
	return merged, nil
}

func mergeCoverageFile(dst map[string]*coverageFileData, path string, src *coverageFileData) {
	existing, ok := dst[path]
	if !ok {
		dst[path] = cloneCoverageFile(src)
		return
	}
	for id, hits := range src.S {
		existing.S[id] += hits
	}
	for id, hits := range src.F {
		existing.F[id] += hits
	}
	for id, hits := range src.B {
		cur := existing.B[id]
		if len(cur) < len(hits) {
			grown := make([]int, len(hits))
			copy(grown, cur)
			cur = grown
			existing.B[id] = cur
		}
		for i, h := range hits {
			if i < len(cur) {
				cur[i] += h
			}
		}
	}
	if existing.InputSourceMap == nil && src.InputSourceMap != nil {
		existing.InputSourceMap = src.InputSourceMap
	}
	if len(existing.StatementMap) == 0 {
		existing.StatementMap = src.StatementMap
	}
	if len(existing.FnMap) == 0 {
		existing.FnMap = src.FnMap
	}
	if len(existing.BranchMap) == 0 {
		existing.BranchMap = src.BranchMap
	}
}

func cloneCoverageFile(src *coverageFileData) *coverageFileData {
	out := &coverageFileData{
		Path:           src.Path,
		Hash:           src.Hash,
		CoverageSchema: src.CoverageSchema,
		InputSourceMap: src.InputSourceMap,
		StatementMap:   map[string]coverageRange{},
		FnMap:          map[string]coverageFn{},
		BranchMap:      map[string]coverageBranch{},
		S:              hitMap{},
		F:              hitMap{},
		B:              map[string][]int{},
	}
	for k, v := range src.StatementMap {
		out.StatementMap[k] = v
	}
	for k, v := range src.FnMap {
		out.FnMap[k] = v
	}
	for k, v := range src.BranchMap {
		out.BranchMap[k] = v
	}
	for k, v := range src.S {
		out.S[k] = v
	}
	for k, v := range src.F {
		out.F[k] = v
	}
	for k, v := range src.B {
		cp := make([]int, len(v))
		copy(cp, v)
		out.B[k] = cp
	}
	return out
}

func computeCoverageStats(repoRoot string, merged map[string]*coverageFileData, includes, excludes []string) ([]fileCoverageStats, error) {
	bySource := map[string]*fileCoverageStats{}

	for _, data := range merged {
		if data == nil {
			continue
		}
		perFile, err := remapCoverageToSources(data)
		if err != nil {
			return nil, err
		}
		for sourcePath, fileStats := range perFile {
			rel := relativizeCoveragePath(repoRoot, sourcePath)
			// Exclude wins on either path form; includes OR across rel/abs.
			if coveragePathExcluded(rel, sourcePath, excludes) {
				continue
			}
			if len(includes) > 0 && !coveragePathIncluded(rel, includes, nil) && !coveragePathIncluded(sourcePath, includes, nil) {
				continue
			}
			agg, ok := bySource[rel]
			if !ok {
				agg = &fileCoverageStats{Path: rel, LineHits: map[int]int{}}
				bySource[rel] = agg
			}
			agg.Statements.Covered += fileStats.Statements.Covered
			agg.Statements.Total += fileStats.Statements.Total
			agg.Functions.Covered += fileStats.Functions.Covered
			agg.Functions.Total += fileStats.Functions.Total
			agg.Branches.Covered += fileStats.Branches.Covered
			agg.Branches.Total += fileStats.Branches.Total
			for line, hits := range fileStats.LineHits {
				if cur, ok := agg.LineHits[line]; !ok || hits > cur {
					agg.LineHits[line] = hits
				}
			}
		}
	}

	out := make([]fileCoverageStats, 0, len(bySource))
	for _, st := range bySource {
		st.Lines = hitCount{}
		for _, hits := range st.LineHits {
			st.Lines.Total++
			if hits > 0 {
				st.Lines.Covered++
			}
		}
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func relativizeCoveragePath(repoRoot, path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	repoRoot = strings.ReplaceAll(strings.TrimSpace(repoRoot), "\\", "/")
	if repoRoot != "" {
		if rel, err := filepath.Rel(repoRoot, path); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
			return strings.ReplaceAll(rel, "\\", "/")
		}
	}
	return path
}

type sourceLineRef struct {
	Path string
	Line int
}

func remapCoverageToSources(data *coverageFileData) (map[string]*fileCoverageStats, error) {
	fallback := data.Path
	if fallback == "" {
		fallback = "unknown.js"
	}
	bySource := map[string]*fileCoverageStats{}

	var remap lineRemapper
	if data.InputSourceMap != nil && strings.TrimSpace(data.InputSourceMap.Mappings) != "" {
		lineMap, err := buildGeneratedLineToSource(fallback, data.InputSourceMap)
		if err != nil {
			return nil, err
		}
		remap = func(genLine int) (string, int, bool) {
			ref, ok := lineMap[genLine]
			if !ok {
				return "", 0, false
			}
			return ref.Path, ref.Line, true
		}
	}

	accumulateInto(bySource, data, remap, fallback)
	return bySource, nil
}

func buildGeneratedLineToSource(generatedPath string, sm *rawSourceMap) (map[int]sourceLineRef, error) {
	dec := tssourcemap.DecodeMappings(sm.Mappings)
	out := map[int]sourceLineRef{}
	baseDir := filepath.Dir(generatedPath)
	for mapping, done := dec.Next(); !done; mapping, done = dec.Next() {
		if mapping == nil || !mapping.IsSourceMapping() {
			continue
		}
		idx := int(mapping.SourceIndex)
		if idx < 0 || idx >= len(sm.Sources) {
			continue
		}
		srcPath := sm.Sources[idx]
		if sm.SourceRoot != "" {
			srcPath = filepath.Join(sm.SourceRoot, srcPath)
		}
		if !filepath.IsAbs(srcPath) {
			srcPath = filepath.Join(baseDir, srcPath)
		}
		srcPath = filepath.Clean(srcPath)
		genLine := mapping.GeneratedLine + 1
		srcLine := mapping.SourceLine + 1
		if _, exists := out[genLine]; !exists {
			out[genLine] = sourceLineRef{Path: srcPath, Line: srcLine}
		}
	}
	if err := dec.Error(); err != nil {
		return nil, xfmt.Errorf("decode inputSourceMap mappings: %w", err)
	}
	return out, nil
}

type lineRemapper func(genLine int) (sourcePath string, sourceLine int, ok bool)

func accumulateInto(bySource map[string]*fileCoverageStats, data *coverageFileData, remap lineRemapper, fallbackPath string) {
	ensure := func(path string) *fileCoverageStats {
		if path == "" {
			path = fallbackPath
		}
		path = strings.ReplaceAll(path, "\\", "/")
		st, ok := bySource[path]
		if !ok {
			st = &fileCoverageStats{Path: path, LineHits: map[int]int{}}
			bySource[path] = st
		}
		return st
	}

	for id, rng := range data.StatementMap {
		hits := data.S[id]
		path := fallbackPath
		line := rng.Start.Line
		if remap != nil {
			if srcPath, srcLine, ok := remap(rng.Start.Line); ok {
				path = srcPath
				line = srcLine
			}
		}
		st := ensure(path)
		st.Statements.Total++
		if hits > 0 {
			st.Statements.Covered++
		}
		if line > 0 {
			if cur, ok := st.LineHits[line]; !ok || hits > cur {
				st.LineHits[line] = hits
			}
		}
	}

	for id, fn := range data.FnMap {
		hits := data.F[id]
		path := fallbackPath
		if remap != nil {
			line := fn.Line
			if line <= 0 {
				line = fn.Decl.Start.Line
			}
			if srcPath, _, ok := remap(line); ok {
				path = srcPath
			}
		}
		st := ensure(path)
		st.Functions.Total++
		if hits > 0 {
			st.Functions.Covered++
		}
	}

	for id, br := range data.BranchMap {
		hits := data.B[id]
		path := fallbackPath
		if remap != nil {
			line := br.Line
			if line <= 0 {
				line = br.Loc.Start.Line
			}
			if srcPath, _, ok := remap(line); ok {
				path = srcPath
			}
		}
		st := ensure(path)
		locs := len(br.Locations)
		if locs == 0 {
			locs = len(hits)
		}
		if locs == 0 {
			locs = 1
		}
		st.Branches.Total += locs
		for i := 0; i < locs; i++ {
			h := 0
			if i < len(hits) {
				h = hits[i]
			}
			if h > 0 {
				st.Branches.Covered++
			}
		}
	}
}

func aggregateTotals(stats []fileCoverageStats) fileCoverageStats {
	var totals fileCoverageStats
	for _, st := range stats {
		totals.Statements.Covered += st.Statements.Covered
		totals.Statements.Total += st.Statements.Total
		totals.Lines.Covered += st.Lines.Covered
		totals.Lines.Total += st.Lines.Total
		totals.Functions.Covered += st.Functions.Covered
		totals.Functions.Total += st.Functions.Total
		totals.Branches.Covered += st.Branches.Covered
		totals.Branches.Total += st.Branches.Total
	}
	return totals
}

func renderLcov(stats []fileCoverageStats) string {
	var b strings.Builder
	for _, st := range stats {
		b.WriteString("TN:\n")
		b.WriteString("SF:")
		b.WriteString(st.Path)
		b.WriteByte('\n')

		fnIDs := make([]int, 0)
		// Functions are aggregated only as counts; emit synthetic FN lines from line hits is not required.
		// Emit DA lines from LineHits.
		lines := make([]int, 0, len(st.LineHits))
		for line := range st.LineHits {
			lines = append(lines, line)
		}
		sort.Ints(lines)
		_ = fnIDs
		b.WriteString("FNF:")
		b.WriteString(strconv.Itoa(st.Functions.Total))
		b.WriteByte('\n')
		b.WriteString("FNH:")
		b.WriteString(strconv.Itoa(st.Functions.Covered))
		b.WriteByte('\n')
		for _, line := range lines {
			fmt.Fprintf(&b, "DA:%d,%d\n", line, st.LineHits[line])
		}
		b.WriteString("LF:")
		b.WriteString(strconv.Itoa(st.Lines.Total))
		b.WriteByte('\n')
		b.WriteString("LH:")
		b.WriteString(strconv.Itoa(st.Lines.Covered))
		b.WriteByte('\n')
		b.WriteString("BRF:")
		b.WriteString(strconv.Itoa(st.Branches.Total))
		b.WriteByte('\n')
		b.WriteString("BRH:")
		b.WriteString(strconv.Itoa(st.Branches.Covered))
		b.WriteByte('\n')
		b.WriteString("end_of_record\n")
	}
	return b.String()
}

func renderTextSummary(stats []fileCoverageStats) string {
	totals := aggregateTotals(stats)
	var b strings.Builder
	fmt.Fprintf(&b, "=============================== Coverage summary ===============================\n")
	fmt.Fprintf(&b, "Statements   : %6.2f%% ( %d/%d )\n", totals.Statements.Percent(), totals.Statements.Covered, totals.Statements.Total)
	fmt.Fprintf(&b, "Branches     : %6.2f%% ( %d/%d )\n", totals.Branches.Percent(), totals.Branches.Covered, totals.Branches.Total)
	fmt.Fprintf(&b, "Functions    : %6.2f%% ( %d/%d )\n", totals.Functions.Percent(), totals.Functions.Covered, totals.Functions.Total)
	fmt.Fprintf(&b, "Lines        : %6.2f%% ( %d/%d )\n", totals.Lines.Percent(), totals.Lines.Covered, totals.Lines.Total)
	fmt.Fprintf(&b, "================================================================================\n")
	return b.String()
}

// renderHTMLSummary writes a per-file summary table. It is not a full nyc-style
// annotated source browser; use lcov + an external viewer for line detail.
func renderHTMLSummary(stats []fileCoverageStats) string {
	totals := aggregateTotals(stats)
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>Coverage</title>")
	b.WriteString("<style>body{font-family:sans-serif;margin:1.5rem}table{border-collapse:collapse;width:100%}")
	b.WriteString("th,td{border:1px solid #ccc;padding:.4rem .6rem;text-align:left}th{background:#f5f5f5}")
	b.WriteString("td.num{text-align:right}</style></head><body>")
	b.WriteString("<h1>Coverage summary</h1>")
	fmt.Fprintf(&b, "<p>Statements %.2f%% · Lines %.2f%% · Functions %.2f%% · Branches %.2f%%</p>",
		totals.Statements.Percent(), totals.Lines.Percent(), totals.Functions.Percent(), totals.Branches.Percent())
	b.WriteString("<table><thead><tr><th>File</th><th>Statements</th><th>Lines</th><th>Functions</th><th>Branches</th></tr></thead><tbody>")
	for _, st := range stats {
		fmt.Fprintf(&b, "<tr><td>%s</td><td class=\"num\">%.2f%% (%d/%d)</td><td class=\"num\">%.2f%% (%d/%d)</td><td class=\"num\">%.2f%% (%d/%d)</td><td class=\"num\">%.2f%% (%d/%d)</td></tr>",
			st.Path,
			st.Statements.Percent(), st.Statements.Covered, st.Statements.Total,
			st.Lines.Percent(), st.Lines.Covered, st.Lines.Total,
			st.Functions.Percent(), st.Functions.Covered, st.Functions.Total,
			st.Branches.Percent(), st.Branches.Covered, st.Branches.Total,
		)
	}
	b.WriteString("</tbody></table></body></html>\n")
	return b.String()
}

func renderJSONSummary(stats []fileCoverageStats) string {
	totals := aggregateTotals(stats)
	type metric struct {
		Total   int     `json:"total"`
		Covered int     `json:"covered"`
		Pct     float64 `json:"pct"`
	}
	type fileEntry struct {
		Lines      metric `json:"lines"`
		Statements metric `json:"statements"`
		Functions  metric `json:"functions"`
		Branches   metric `json:"branches"`
	}
	out := map[string]fileEntry{
		"total": {
			Lines:      metric{Total: totals.Lines.Total, Covered: totals.Lines.Covered, Pct: totals.Lines.Percent()},
			Statements: metric{Total: totals.Statements.Total, Covered: totals.Statements.Covered, Pct: totals.Statements.Percent()},
			Functions:  metric{Total: totals.Functions.Total, Covered: totals.Functions.Covered, Pct: totals.Functions.Percent()},
			Branches:   metric{Total: totals.Branches.Total, Covered: totals.Branches.Covered, Pct: totals.Branches.Percent()},
		},
	}
	for _, st := range stats {
		out[st.Path] = fileEntry{
			Lines:      metric{Total: st.Lines.Total, Covered: st.Lines.Covered, Pct: st.Lines.Percent()},
			Statements: metric{Total: st.Statements.Total, Covered: st.Statements.Covered, Pct: st.Statements.Percent()},
			Functions:  metric{Total: st.Functions.Total, Covered: st.Functions.Covered, Pct: st.Functions.Percent()},
			Branches:   metric{Total: st.Branches.Total, Covered: st.Branches.Covered, Pct: st.Branches.Percent()},
		}
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw) + "\n"
}
