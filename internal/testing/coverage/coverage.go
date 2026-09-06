// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	xfmt "golang.org/x/exp/errors/fmt"
)

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
	runIDValue, _ := ctx.Value(coverageRunIDContextKey{}).(string)
	return strings.TrimSpace(runIDValue)
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
	scanDir := cwd
	for {
		if _, err := os.Stat(filepath.Join(scanDir, "go.mod")); err == nil {
			return scanDir
		}
		parent := filepath.Dir(scanDir)
		if parent == scanDir {
			return cwd
		}
		scanDir = parent
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
func InstrumentDistBundle(ctx context.Context, repoRoot string, distPath string, app string) error {
	return InstrumentDistBundleWithTmpRoot(ctx, repoRoot, distPath, app, "")
}

// InstrumentDistBundleWithTmpRoot instruments dist bundles with pure-Go Istanbul
// instrumentation. tmpRoot is accepted for API compatibility with callers; the
// Go instrumenter does not write helper scripts under tmp.
func InstrumentDistBundleWithTmpRoot(ctx context.Context, repoRoot string, distPath string, app string, tmpRoot string) error {
	_ = ctx
	_ = repoRoot
	_ = tmpRoot
	if strings.TrimSpace(distPath) == "" {
		return xfmt.Errorf("empty distPath")
	}
	if strings.TrimSpace(app) == "" {
		return xfmt.Errorf("empty app")
	}

	targets := existingInstrumentTargets(distPath, app)
	if len(targets) == 0 {
		return xfmt.Errorf("instrument bundle for coverage: no dist bundle found for app %q under %s", app, distPath)
	}

	for _, inPath := range targets {
		if err := InstrumentJSFile(inPath); err != nil {
			return xfmt.Errorf("instrument bundle for coverage (%s): %w", inPath, err)
		}
	}
	return nil
}

// PreflightInstrumentationPrerequisites validates coverage instrumentation
// prerequisites. Pure-Go instrumentation has no Node/npm prerequisites.
func PreflightInstrumentationPrerequisites(repoRoot string) error {
	_ = repoRoot
	return nil
}

func existingInstrumentTargets(distPath string, app string) []string {
	// Instrument product bundles only. Test bundles are large, mostly excluded
	// from gates via **/*.test.ts, and duplicate app code already covered via index.js.
	instrumentTargetCandidates := []string{
		filepath.Join(distPath, "apps", app, "index.js"),
		filepath.Join(distPath, "bundles", "index.js"),
	}
	instrumentTargets := make([]string, 0, len(instrumentTargetCandidates))
	for _, instrumentTargetPath := range instrumentTargetCandidates {
		if _, err := os.Stat(instrumentTargetPath); err == nil {
			instrumentTargets = append(instrumentTargets, instrumentTargetPath)
		}
	}
	return instrumentTargets
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
	enriched, err := enrichCoverageJSONWithMeta(coverageJSON)
	if err != nil {
		return xfmt.Errorf("enrich coverage meta: %w", err)
	}
	coverageJSON = enriched
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

func enrichCoverageJSONWithMeta(coverageJSON string) (string, error) {
	coverageJSON = strings.TrimSpace(coverageJSON)
	if coverageJSON == "" || coverageJSON == "null" {
		return coverageJSON, nil
	}
	var fileMap map[string]*coverageFileData
	if err := json.Unmarshal([]byte(coverageJSON), &fileMap); err != nil {
		return coverageJSON, nil
	}
	changed := false
	for path, data := range fileMap {
		if data == nil {
			continue
		}
		metaPath := strings.TrimSpace(data.Path)
		if metaPath == "" {
			metaPath = path
		}
		metaPath = metaPath + ".coverage-meta.json"
		raw, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta coverageFileData
		if err := json.Unmarshal(raw, &meta); err != nil {
			continue
		}
		if len(data.StatementMap) == 0 && len(meta.StatementMap) > 0 {
			data.StatementMap = meta.StatementMap
			changed = true
		}
		if len(data.FnMap) == 0 && len(meta.FnMap) > 0 {
			data.FnMap = meta.FnMap
			changed = true
		}
		if len(data.BranchMap) == 0 && len(meta.BranchMap) > 0 {
			data.BranchMap = meta.BranchMap
			changed = true
		}
		if data.InputSourceMap == nil && meta.InputSourceMap != nil {
			data.InputSourceMap = meta.InputSourceMap
			changed = true
		}
		if data.CoverageSchema == "" && meta.CoverageSchema != "" {
			data.CoverageSchema = meta.CoverageSchema
			changed = true
		}
	}
	if !changed {
		return coverageJSON, nil
	}
	out, err := json.Marshal(fileMap)
	if err != nil {
		return "", err
	}
	return string(out), nil
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
	globs := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		globs = append(globs, p)
	}
	return globs
}

// SplitCoverageReporters is an alias of SplitCoverageGlobs.
func SplitCoverageReporters(v string) []string { return SplitCoverageGlobs(v) }

// ValidateCoverageReporters allows a conservative set to avoid typos silently producing no output.
func ValidateCoverageReporters(reporters []string) ([]string, error) {
	allowedReporterSet := map[string]struct{}{
		"text":         {},
		"html":         {},
		"lcov":         {},
		"lcovonly":     {},
		"text-summary": {},
		"json-summary": {},
	}
	seenReporterSet := map[string]struct{}{}
	validatedReporters := make([]string, 0, len(reporters))
	for _, reporter := range reporters {
		reporter = strings.ToLower(strings.TrimSpace(reporter))
		if reporter == "" {
			continue
		}
		if _, ok := allowedReporterSet[reporter]; !ok {
			return nil, xfmt.Errorf("unsupported coverage reporter %q", reporter)
		}
		if _, seen := seenReporterSet[reporter]; seen {
			continue
		}
		seenReporterSet[reporter] = struct{}{}
		validatedReporters = append(validatedReporters, reporter)
	}
	return validatedReporters, nil
}
