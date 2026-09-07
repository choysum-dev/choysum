// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
)

// BundleOptions configures a thin FE unit esbuild (no Vue plugin / ModuleBuilder).
type BundleOptions struct {
	RepoRoot   string
	EntryPath  string
	Outfile    string
	Sourcemap  bool
	WorkingDir string
}

// BundleResult is the esbuild output for a FE unit fixture bundle.
type BundleResult struct {
	JS       string
	JSPath   string
	MapPath  string
	Warnings []string
}

// BuildFrontendUnitBundle bundles a FE unit entry with `@/*` → `<repo>/modules/*`.
// It does not resolve `.vue` SFCs (callers must keep entries free of illegal marks).
func BuildFrontendUnitBundle(opts BundleOptions) (*BundleResult, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	entry := strings.TrimSpace(opts.EntryPath)
	if repoRoot == "" {
		return nil, xfmt.Errorf("frontend bundle: empty repo root")
	}
	if entry == "" {
		return nil, xfmt.Errorf("frontend bundle: empty entry path")
	}
	modulesDir := filepath.Join(repoRoot, "modules")
	if st, err := os.Stat(modulesDir); err != nil || !st.IsDir() {
		return nil, xfmt.Errorf("frontend bundle: modules dir missing: %s", modulesDir)
	}

	outfile := strings.TrimSpace(opts.Outfile)
	if outfile == "" {
		outfile = filepath.Join(filepath.Dir(entry), "fe-unit.bundle.js")
	}
	if err := os.MkdirAll(filepath.Dir(outfile), 0o755); err != nil {
		return nil, xfmt.Errorf("frontend bundle: mkdir: %w", err)
	}

	buildOpts := api.BuildOptions{
		EntryPoints:   []string{entry},
		Bundle:        true,
		Write:         true,
		Outfile:       outfile,
		Platform:      api.PlatformNeutral,
		Format:        api.FormatIIFE,
		Target:        api.ES2020,
		LogLevel:      api.LogLevelWarning,
		AbsWorkingDir: strings.TrimSpace(opts.WorkingDir),
		Alias:         map[string]string{"@": modulesDir},
		External:      []string{"*.vue"},
		JSX:           api.JSXTransform,
	}
	if opts.Sourcemap {
		buildOpts.Sourcemap = api.SourceMapLinked
	}
	if buildOpts.AbsWorkingDir == "" {
		buildOpts.AbsWorkingDir = repoRoot
	}

	result := api.Build(buildOpts)
	warnings := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, w.Text)
	}
	if len(result.Errors) > 0 {
		msgs := make([]string, 0, len(result.Errors))
		for _, e := range result.Errors {
			msgs = append(msgs, e.Text)
		}
		return nil, xfmt.Errorf("frontend bundle: esbuild failed: %s", strings.Join(msgs, "; "))
	}

	jsBytes, err := os.ReadFile(outfile)
	if err != nil {
		return nil, xfmt.Errorf("frontend bundle: read outfile: %w", err)
	}
	out := &BundleResult{
		JS:       string(jsBytes),
		JSPath:   outfile,
		Warnings: warnings,
	}
	mapPath := outfile + ".map"
	if _, err := os.Stat(mapPath); err == nil {
		out.MapPath = mapPath
	}
	return out, nil
}

// WriteFrontendTestsEntry writes a generated entry that imports test files so
// choysumtest globals register cases at load time. Callers invoke
// globalThis.__choysum_test_run__ after Load (do not auto-run in the entry:
// QuickJS Load uses EvalAwait and must not await a top-level runner Promise).
func WriteFrontendTestsEntry(outPath string, testFiles []string) error {
	if strings.TrimSpace(outPath) == "" {
		return xfmt.Errorf("frontend bundle: empty entry out path")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return xfmt.Errorf("frontend bundle: mkdir entry: %w", err)
	}
	var b strings.Builder
	for _, f := range testFiles {
		imp := filepath.ToSlash(filepath.Clean(f))
		b.WriteString("import '")
		b.WriteString(imp)
		b.WriteString("';\n")
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return xfmt.Errorf("frontend bundle: write entry: %w", err)
	}
	return nil
}
