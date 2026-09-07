// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/pkg/jsexecutor"
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
)

// BundleOptions configures a thin FE unit esbuild (Vue optional; no ModuleBuilder).
type BundleOptions struct {
	RepoRoot   string
	EntryPath  string
	Outfile    string
	Sourcemap  bool
	WorkingDir string
	// Vue enables vueplugin + real vue + @choysum/test-utils alias (PR-unit-fe-runner).
	Vue bool
	// JsExecutor is required when Vue is true.
	JsExecutor jsexecutor.ScriptExecutor
	// CacheDir for esmresolver when Vue is true.
	CacheDir string
}

// BundleResult is the esbuild output for a FE unit fixture bundle.
type BundleResult struct {
	JS       string
	JSPath   string
	MapPath  string
	Warnings []string
}

// BuildFrontendUnitBundle bundles a FE unit entry with `@/*` → `<repo>/modules/*`.
// When Vue is false, `.vue` stays external (pure TS fixtures). When Vue is true,
// delegates to BuildFrontendVueHostBundle (vuesfc + real vue + choysumMount).
func BuildFrontendUnitBundle(opts BundleOptions) (*BundleResult, error) {
	if opts.Vue {
		if opts.JsExecutor == nil {
			return nil, xfmt.Errorf("frontend bundle: JsExecutor required when Vue is true")
		}
		return BuildFrontendVueHostBundle(VueHostBundleOptions{
			RepoRoot:      opts.RepoRoot,
			EntryPath:     opts.EntryPath,
			Outfile:       opts.Outfile,
			Sourcemap:     opts.Sourcemap,
			WorkingDir:    opts.WorkingDir,
			CacheDir:      opts.CacheDir,
			JsExecutor:    opts.JsExecutor,
			WithVuePlugin: true,
		})
	}

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
//
// Imports use absolute paths (same as the BE tests index). Relative paths from
// os.TempDir entries break under macOS /var → /private/var when AbsWorkingDir
// is the repo root.
func WriteFrontendTestsEntry(outPath string, testFiles []string) error {
	if strings.TrimSpace(outPath) == "" {
		return xfmt.Errorf("frontend bundle: empty entry out path")
	}
	if err := osMkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return xfmt.Errorf("frontend bundle: mkdir entry: %w", err)
	}
	var b strings.Builder
	for _, f := range testFiles {
		absFile, absErr := filepathAbs(filepath.Clean(f))
		if absErr != nil {
			return xfmt.Errorf("frontend bundle: resolve test file path: %w", absErr)
		}
		imp := filepath.ToSlash(absFile)
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
