// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	xfmt "golang.org/x/exp/errors/fmt"
)

//go:embed instrument.cjs
var instrumentScript string

type coverageRunIDContextKey struct{}

// ContextWithCoverageRunID stores a run ID in context for coverage isolation.
func ContextWithCoverageRunID(ctx context.Context, runID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ctx
	}
	return context.WithValue(ctx, coverageRunIDContextKey{}, runID)
}

// CoverageRunIDFromContext reads the run ID from context.
func CoverageRunIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(coverageRunIDContextKey{}).(string)
	return strings.TrimSpace(v)
}

// NewCoverageRunID generates a unique run ID safe for file names.
func NewCoverageRunID() string {
	ts := time.Now().UTC().Format("20060102-150405.000")
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-r%x", ts, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-r%s", ts, hex.EncodeToString(buf))
}

// FindRepoRootFromCwd walks up from cwd until it finds go.mod.
// It falls back to the original cwd when not found.
func FindRepoRootFromCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	cur := cwd
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return cwd
		}
		cur = parent
	}
}

// InstrumentDistBundle instruments dist bundles in-place to enable Istanbul coverage.
//
// It instruments every existing target among:
// - dist/apps/<app>/index.js
// - dist/apps/<app>/tests.js
// - dist/bundles/index.js
// - dist/bundles/tests.js
//
// Each target writes/updates its corresponding .map file when possible.
//
// Note: this still relies on Node's module resolution to find `istanbul-lib-instrument`.
// We set cmd.Dir=repoRoot so `require()` can resolve from repoRoot/node_modules.
func InstrumentDistBundle(ctx context.Context, repoRoot string, distPath string, app string) error {
	return InstrumentDistBundleWithTmpRoot(ctx, repoRoot, distPath, app, "")
}

// InstrumentDistBundleWithTmpRoot instruments dist bundles and stores temporary
// helper scripts under TmpPath/testing/<workspace-hash>/<run-id>/coverage when tmpRoot is provided.
func InstrumentDistBundleWithTmpRoot(ctx context.Context, repoRoot string, distPath string, app string, tmpRoot string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}
	if strings.TrimSpace(distPath) == "" {
		return xfmt.Errorf("empty distPath")
	}
	if strings.TrimSpace(app) == "" {
		return xfmt.Errorf("empty app")
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return xfmt.Errorf("--coverage requires node in PATH")
	}

	targets := existingInstrumentTargets(distPath, app)
	if len(targets) == 0 {
		return xfmt.Errorf("instrument bundle for coverage: no dist bundle found for app %q under %s", app, distPath)
	}

	runID := testingpathing.TestingRunIDFromContext(ctx)
	scriptPath, cleanup, err := writeInstrumentScriptTempFileWithTmpRoot(repoRoot, tmpRoot, runID)
	if err != nil {
		return err
	}
	defer cleanup()

	for _, inPath := range targets {
		outMapPath := inPath + ".map"
		cmd := exec.CommandContext(ctx, nodePath, scriptPath, "--in", inPath, "--out", inPath, "--out-map", outMapPath)
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return xfmt.Errorf("instrument bundle for coverage (%s): %w", inPath, err)
		}
	}
	return nil
}

func existingInstrumentTargets(distPath string, app string) []string {
	candidates := []string{
		filepath.Join(distPath, "apps", app, "index.js"),
		filepath.Join(distPath, "apps", app, "tests.js"),
		filepath.Join(distPath, "bundles", "index.js"),
		filepath.Join(distPath, "bundles", "tests.js"),
	}
	targets := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			targets = append(targets, candidate)
		}
	}
	return targets
}

func resolveCoverageTmpBaseDir(repoRoot string, tmpRoot string) (string, error) {
	return resolveCoverageTmpBaseDirWithRunID(repoRoot, tmpRoot, "")
}

func resolveCoverageTmpBaseDirWithRunID(repoRoot string, tmpRoot string, runID string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}
	resolvedTmpRoot := strings.TrimSpace(tmpRoot)
	if resolvedTmpRoot == "" {
		resolvedTmpRoot = os.TempDir()
	}
	return testingpathing.ResolveTestingTmpDirWithRunID(repoRoot, resolvedTmpRoot, "coverage", runID)
}

func resolveCoverageNycOutputDir(repoRoot string, tmpRoot string) (string, error) {
	return resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, "")
}

func resolveCoverageNycOutputDirWithRunID(repoRoot string, tmpRoot string, runID string) (string, error) {
	coverageTmpDir, err := resolveCoverageTmpBaseDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return "", err
	}
	return filepath.Join(coverageTmpDir, "nyc_output"), nil
}

// ResolveCoverageReportDir returns the default coverage report output directory:
// TmpPath/testing/<workspace-hash>/<run-id>/coverage/reports.
func ResolveCoverageReportDir(repoRoot string, tmpRoot string) (string, error) {
	return ResolveCoverageReportDirWithRunID(repoRoot, tmpRoot, "")
}

// ResolveCoverageReportDirWithRunID returns the default coverage report output
// directory under the specified run-id subtree.
func ResolveCoverageReportDirWithRunID(repoRoot string, tmpRoot string, runID string) (string, error) {
	coverageTmpDir, err := resolveCoverageTmpBaseDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return "", err
	}
	return filepath.Join(coverageTmpDir, "reports"), nil
}

func writeInstrumentScriptTempFile(repoRoot string) (string, func(), error) {
	return writeInstrumentScriptTempFileWithTmpRoot(repoRoot, "", "")
}

func writeInstrumentScriptTempFileWithTmpRoot(repoRoot string, tmpRoot string, runID string) (string, func(), error) {
	if strings.TrimSpace(instrumentScript) == "" {
		return "", func() {}, xfmt.Errorf("embedded instrument script is empty")
	}

	baseDir := ""
	if coverageTmpDir, err := resolveCoverageTmpBaseDirWithRunID(repoRoot, tmpRoot, runID); err == nil {
		candidate := filepath.Join(coverageTmpDir, "instrument")
		if mkErr := os.MkdirAll(candidate, 0o755); mkErr == nil {
			baseDir = candidate
		}
	}

	f, err := os.CreateTemp(baseDir, "choysum-instrument-*.cjs")
	if err != nil {
		return "", func() {}, xfmt.Errorf("create temp instrument script: %w", err)
	}
	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }

	if _, err := f.WriteString(instrumentScript); err != nil {
		_ = f.Close()
		cleanup()
		return "", func() {}, xfmt.Errorf("write temp instrument script: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, xfmt.Errorf("close temp instrument script: %w", err)
	}
	return path, cleanup, nil
}

// WriteCoverageJSON writes coverage JSON (from JSON.stringify(globalThis.__coverage__)).
func WriteCoverageJSON(repoRoot string, app string, coverageJSON string) error {
	return WriteCoverageJSONWithRunID(repoRoot, app, "", coverageJSON)
}

// WriteCoverageJSONWithRunID writes coverage JSON and optionally tags filename with run-id.
func WriteCoverageJSONWithRunID(repoRoot string, app string, runID string, coverageJSON string) error {
	return WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, app, runID, coverageJSON, "")
}

// WriteCoverageJSONWithRunIDAndTmpRoot writes coverage JSON into
// TmpPath/testing/<workspace-hash>/<run-id>/coverage/nyc_output and optionally tags filename with run-id.
func WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot string, app string, runID string, coverageJSON string, tmpRoot string) error {
	if strings.TrimSpace(coverageJSON) == "" {
		return nil
	}
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}

	outDir, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return xfmt.Errorf("resolve coverage nyc_output dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return xfmt.Errorf("mkdir coverage nyc_output: %w", err)
	}

	runID = strings.TrimSpace(runID)
	name := fmt.Sprintf("choysum-%s-%d.json", app, time.Now().UnixNano())
	if runID != "" {
		name = fmt.Sprintf("choysum-%s-%s-%d.json", app, runID, time.Now().UnixNano())
	}
	outPath := filepath.Join(outDir, name)

	if err := os.WriteFile(outPath, []byte(coverageJSON), 0o644); err != nil {
		return xfmt.Errorf("write coverage: %w", err)
	}
	fmt.Fprintf(os.Stderr, "choysum test: wrote coverage %s\n", outPath)
	return nil
}

func nycCommonArgs(includes []string, excludes []string) []string {
	// Keep defaults conservative; advanced users can still run nyc manually.
	args := []string{
		"--exclude-after-remap",
	}

	for _, inc := range includes {
		inc = strings.TrimSpace(inc)
		if inc == "" {
			continue
		}
		args = append(args, "--include", inc)
	}

	// Default excludes (avoid counting generated/build/runtime artifacts).
	args = append(args,
		"--exclude", "**/*.test.ts",
		"--exclude", "**/*.d.ts",
		"--exclude", "**/.choysum/**",
		"--exclude", "**/dist/**",
		"--exclude", "**/node_modules/**",
	)

	for _, exc := range excludes {
		exc = strings.TrimSpace(exc)
		if exc == "" {
			continue
		}
		args = append(args, "--exclude", exc)
	}

	return args
}

// RunNycReport runs `npx --no-install nyc report` with common include/exclude defaults.
func RunNycReport(ctx context.Context, repoRoot string, reportDir string, reporters []string, includes []string, excludes []string, tmpRoot string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}

	if _, err := exec.LookPath("node"); err != nil {
		return xfmt.Errorf("--coverage-report requires node in PATH")
	}
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return xfmt.Errorf("--coverage-report requires npx in PATH (npm)")
	}
	runID := strings.TrimSpace(CoverageRunIDFromContext(ctx))

	// Force nyc output to stderr to avoid polluting TAP stdout.
	if strings.TrimSpace(reportDir) == "" {
		reportDir, err = ResolveCoverageReportDirWithRunID(repoRoot, tmpRoot, runID)
		if err != nil {
			return xfmt.Errorf("resolve coverage report dir: %w", err)
		}
	} else if !filepath.IsAbs(reportDir) {
		reportDir = filepath.Join(repoRoot, reportDir)
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return xfmt.Errorf("mkdir coverage report dir: %w", err)
	}

	if len(reporters) == 0 {
		reporters = []string{"text", "html"}
	}
	validated, err := ValidateCoverageReporters(reporters)
	if err != nil {
		return err
	}

	if runID == "" {
		fmt.Fprintln(os.Stderr, "choysum test: warning: missing run-id, using latest fallback (TmpPath/testing/<workspace-hash>/coverage/nyc_output newest file only)")
	}
	reportTempDir, cleanup, err := prepareNycReportTempDir(repoRoot, runID, tmpRoot)
	if err != nil {
		return xfmt.Errorf("prepare nyc report input: %w", err)
	}
	defer cleanup()

	args := []string{"--no-install", "nyc", "report", "--temp-dir", reportTempDir, "--report-dir", reportDir}
	for _, r := range validated {
		args = append(args, "--reporter", r)
	}
	args = append(args, nycCommonArgs(includes, excludes)...)

	cmd := exec.CommandContext(ctx, npxPath, args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return xfmt.Errorf("nyc report failed (did you run npm install?): %w", err)
	}
	return nil
}

func prepareNycReportTempDir(repoRoot string, runID string, tmpRoot string) (string, func(), error) {
	nycOutputDir, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return "", func() {}, xfmt.Errorf("resolve nyc_output dir: %w", err)
	}
	entries, err := os.ReadDir(nycOutputDir)
	if err != nil {
		return "", func() {}, xfmt.Errorf("read nyc_output: %w", err)
	}

	runID = strings.TrimSpace(runID)
	if runID != "" {
		token := "-" + runID + "-"
		reportTempDir, cleanup, err := createNycReportTempDirWithRunID(repoRoot, tmpRoot, runID)
		if err != nil {
			return "", func() {}, err
		}
		copied := 0
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if !strings.Contains(entry.Name(), token) {
				continue
			}
			srcPath := filepath.Join(nycOutputDir, entry.Name())
			dstPath := filepath.Join(reportTempDir, entry.Name())
			if err := copyFile(srcPath, dstPath); err != nil {
				cleanup()
				return "", func() {}, xfmt.Errorf("copy run-scoped coverage json: %w", err)
			}
			copied++
		}
		if copied == 0 {
			cleanup()
			return "", func() {}, xfmt.Errorf("no coverage json found in nyc_output for run-id %q", runID)
		}
		return reportTempDir, cleanup, nil
	}

	latestPath := ""
	latestName := ""
	var latestModTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", func() {}, xfmt.Errorf("stat coverage file %s: %w", entry.Name(), err)
		}
		if latestPath == "" || info.ModTime().After(latestModTime) || (info.ModTime().Equal(latestModTime) && entry.Name() > latestName) {
			latestPath = filepath.Join(nycOutputDir, entry.Name())
			latestName = entry.Name()
			latestModTime = info.ModTime()
		}
	}

	if latestPath == "" {
		return "", func() {}, xfmt.Errorf("no coverage json found in nyc_output")
	}

	reportTempDir, cleanup, err := createNycReportTempDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return "", func() {}, err
	}

	dstPath := filepath.Join(reportTempDir, filepath.Base(latestPath))
	if err := copyFile(latestPath, dstPath); err != nil {
		cleanup()
		return "", func() {}, xfmt.Errorf("copy latest coverage json: %w", err)
	}

	return reportTempDir, cleanup, nil
}

func createNycReportTempDir(repoRoot string, tmpRoot string) (string, func(), error) {
	return createNycReportTempDirWithRunID(repoRoot, tmpRoot, "")
}

func createNycReportTempDirWithRunID(repoRoot string, tmpRoot string, runID string) (string, func(), error) {
	coverageTmpDir, err := resolveCoverageTmpBaseDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return "", func() {}, xfmt.Errorf("resolve coverage tmp dir: %w", err)
	}
	tempBaseDir := filepath.Join(coverageTmpDir, "report")
	if err := os.MkdirAll(tempBaseDir, 0o755); err != nil {
		tempBaseDir = ""
	}
	reportTempDir, err := os.MkdirTemp(tempBaseDir, "nyc-report-latest-")
	if err != nil {
		if tempBaseDir != "" {
			reportTempDir, err = os.MkdirTemp("", "nyc-report-latest-")
		}
		if err != nil {
			return "", func() {}, xfmt.Errorf("create temp nyc report dir: %w", err)
		}
	}
	cleanup := func() { _ = os.RemoveAll(reportTempDir) }

	return reportTempDir, cleanup, nil
}

func copyFile(srcPath string, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

// RunNycCheckCoverage runs `npx --no-install nyc check-coverage` with thresholds.
func RunNycCheckCoverage(ctx context.Context, repoRoot string, includes []string, excludes []string, lines int, functions int, branches int, statements int) error {
	return RunNycCheckCoverageWithTmpRoot(ctx, repoRoot, includes, excludes, lines, functions, branches, statements, "")
}

// RunNycCheckCoverageWithTmpRoot runs `npx --no-install nyc check-coverage` with thresholds
// and reads nyc output from TmpPath/testing/<workspace-hash>/<run-id>/coverage/nyc_output.
func RunNycCheckCoverageWithTmpRoot(ctx context.Context, repoRoot string, includes []string, excludes []string, lines int, functions int, branches int, statements int, tmpRoot string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if lines <= 0 && functions <= 0 && branches <= 0 && statements <= 0 {
		return xfmt.Errorf("--coverage-check requires at least one threshold flag (e.g. --coverage-lines 80)")
	}

	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}

	if _, err := exec.LookPath("node"); err != nil {
		return xfmt.Errorf("--coverage-check requires node in PATH")
	}
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return xfmt.Errorf("--coverage-check requires npx in PATH (npm)")
	}
	runID := strings.TrimSpace(CoverageRunIDFromContext(ctx))
	nycOutputDir, err := resolveCoverageNycOutputDirWithRunID(repoRoot, tmpRoot, runID)
	if err != nil {
		return xfmt.Errorf("resolve nyc_output dir: %w", err)
	}

	args := []string{"--no-install", "nyc", "check-coverage", "--temp-dir", nycOutputDir}
	args = append(args, nycCommonArgs(includes, excludes)...)
	if lines > 0 {
		args = append(args, "--lines", fmt.Sprintf("%d", lines))
	}
	if functions > 0 {
		args = append(args, "--functions", fmt.Sprintf("%d", functions))
	}
	if branches > 0 {
		args = append(args, "--branches", fmt.Sprintf("%d", branches))
	}
	if statements > 0 {
		args = append(args, "--statements", fmt.Sprintf("%d", statements))
	}

	cmd := exec.CommandContext(ctx, npxPath, args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return xfmt.Errorf("nyc check-coverage failed: %w", err)
	}
	return nil
}

// SplitCoverageGlobs accepts comma/semicolon/whitespace-separated values.
func SplitCoverageGlobs(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	parts := strings.FieldsFunc(v, func(r rune) bool {
		switch {
		case r == ',', r == ';':
			return true
		case unicode.IsSpace(r):
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// SplitCoverageReporters is an alias of SplitCoverageGlobs.
func SplitCoverageReporters(v string) []string { return SplitCoverageGlobs(v) }

func ValidateCoverageReporters(reporters []string) ([]string, error) {
	// We only allow a conservative set to avoid typos silently producing no output.
	allowed := map[string]struct{}{
		"text":         {},
		"html":         {},
		"lcov":         {},
		"lcovonly":     {},
		"text-summary": {},
		"json-summary": {},
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(reporters))
	for _, r := range reporters {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		r = strings.ToLower(r)
		if _, ok := allowed[r]; !ok {
			return nil, xfmt.Errorf("unsupported coverage reporter %q (allowed: text, html, lcov, lcovonly, text-summary, json-summary)", r)
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out, nil
}
