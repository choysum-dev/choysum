// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
)

// Test seams for rare OS / build failures (overridden in unit tests).
var (
	osMkdirAll   = os.MkdirAll
	osReadFile   = os.ReadFile
	osWriteFile  = os.WriteFile
	osStatBundle = os.Stat
	jsonMarshal  = json.Marshal
	esbuildBuild = api.Build
	filepathRel  = filepath.Rel
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
	st, err := osStatBundle(modulesDir)
	if err != nil {
		return nil, xfmt.Errorf("frontend bundle: stat modules dir %s: %w", modulesDir, err)
	}
	if !st.IsDir() {
		return nil, xfmt.Errorf("frontend bundle: modules path is not a directory: %s", modulesDir)
	}

	outfile := strings.TrimSpace(opts.Outfile)
	if outfile == "" {
		outfile = filepath.Join(filepath.Dir(entry), "fe-unit.bundle.js")
	}
	if err := osMkdirAll(filepath.Dir(outfile), 0o755); err != nil {
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

	result := esbuildBuild(buildOpts)
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

	jsBytes, err := osReadFile(outfile)
	if err != nil {
		return nil, xfmt.Errorf("frontend bundle: read outfile: %w", err)
	}
	out := &BundleResult{
		JS:       string(jsBytes),
		JSPath:   outfile,
		Warnings: warnings,
	}
	if opts.Sourcemap {
		mapPath := outfile + ".map"
		if _, err := osStatBundle(mapPath); err == nil {
			out.MapPath = mapPath
		}
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
	if err := osMkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return xfmt.Errorf("frontend bundle: mkdir entry: %w", err)
	}
	entryDir, err := filepathAbs(filepath.Dir(outPath))
	if err != nil {
		entryDir = filepath.Dir(outPath)
	}
	var b strings.Builder
	for _, f := range testFiles {
		absFile, absErr := filepathAbs(filepath.Clean(f))
		if absErr != nil {
			absFile = filepath.Clean(f)
		}
		rel, relErr := filepathRel(entryDir, absFile)
		if relErr != nil {
			return xfmt.Errorf("frontend bundle: relative import path: %w", relErr)
		}
		imp := filepath.ToSlash(rel)
		if !strings.HasPrefix(imp, ".") {
			imp = "./" + imp
		}
		encoded, err := jsonMarshal(imp)
		if err != nil {
			return xfmt.Errorf("frontend bundle: encode import path: %w", err)
		}
		b.WriteString("import ")
		b.Write(encoded)
		b.WriteString(";\n")
	}
	if err := osWriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return xfmt.Errorf("frontend bundle: write entry: %w", err)
	}
	return nil
}
