// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Check typechecks an application's service TypeScript roots using
// typescript-go-internal. It does not invoke Node or vue-tsc.
func Check(ctx context.Context, opts Options) (Result, error) {
	if err := validateOptions(opts); err != nil {
		return Result{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	modulesPath, err := absPath(opts.ModulesPath)
	if err != nil {
		return Result{}, err
	}
	repoRoot, err := absPath(opts.RepoRoot)
	if err != nil {
		return Result{}, err
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	scope := opts.Scope
	files, err := CollectRootFiles(ctx, modulesPath, opts.App, scope)
	if err != nil {
		if !errors.Is(err, ErrNoRootFiles) || len(opts.Overlays) == 0 {
			return Result{}, err
		}
		files = nil
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	overlays := resolveOverlaysAgainstModules(opts.Overlays, modulesPath)
	fs := newTypecheckFS(overlays)
	files = appendOverlayServiceRoots(files, modulesPath, opts.App, scope, overlays, fs.UseCaseSensitiveFileNames())
	if coreAmbient := filepath.ToSlash(filepath.Join(modulesPath, "core", "types", "$choysum.d.ts")); fs.FileExists(coreAmbient) {
		files = appendUniqueSlash(files, coreAmbient)
	}
	if len(files) == 0 {
		return Result{}, ErrNoRootFiles
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	compilerOpts, err := BuildCompilerOptions(modulesPath, repoRoot)
	if err != nil {
		return Result{}, err
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	host := newHost(modulesPath, fs)
	program := buildProgram(host, files, compilerOpts)
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}
	diags, err := collectDiagnostics(ctx, program)
	if err != nil {
		return Result{}, err
	}
	return toResult(diags), nil
}

// Test hook for mid-phase cancellation coverage.
var ctxErr = func(ctx context.Context) error { return ctx.Err() }

// resolveOverlaysAgainstModules makes overlay keys absolute under modulesPath
// so relative overlay paths do not resolve against the process CWD.
func resolveOverlaysAgainstModules(overlays map[string]string, modulesPath string) map[string]string {
	if len(overlays) == 0 {
		return nil
	}
	out := make(map[string]string, len(overlays))
	for k, v := range overlays {
		path := strings.TrimSpace(k)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(modulesPath, path)
		}
		out[normalizePathKey(path)] = v
	}
	return out
}

// appendOverlayServiceRoots adds overlay-only paths that match ScopeService
// collection rules (app-root *.ts / service/**), including virtual files.
func appendOverlayServiceRoots(files []string, modulesPath, app string, scope Scope, overlays map[string]string, caseSensitive bool) []string {
	if scope != ScopeService || len(overlays) == 0 {
		return files
	}
	app = strings.TrimSpace(app)
	appRoot := filepath.ToSlash(filepath.Join(modulesPath, app))
	for path := range overlays {
		norm := normalizePathKey(path)
		if norm == "" || shouldSkipTSFileName(filepath.Base(norm)) {
			continue
		}
		lower := strings.ToLower(norm)
		if !strings.HasSuffix(lower, ".ts") {
			continue
		}
		rel, ok := cutPathPrefix(norm, appRoot+"/", caseSensitive)
		if !ok {
			continue
		}
		if !strings.Contains(rel, "/") {
			files = appendUniqueSlash(files, norm)
			continue
		}
		svcPrefix := "service/"
		if caseSensitive {
			if !strings.HasPrefix(rel, svcPrefix) {
				continue
			}
		} else if !strings.HasPrefix(strings.ToLower(rel), svcPrefix) {
			continue
		}
		parts := strings.Split(rel, "/")
		skip := false
		for _, part := range parts[:len(parts)-1] {
			if shouldSkipScanDir(part) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		files = appendUniqueSlash(files, norm)
	}
	return files
}

func cutPathPrefix(path, prefix string, caseSensitive bool) (string, bool) {
	if caseSensitive {
		return strings.CutPrefix(path, prefix)
	}
	if len(path) < len(prefix) {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix)) {
		return "", false
	}
	return path[len(prefix):], true
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func appendUniqueSlash(files []string, path string) []string {
	path = filepath.ToSlash(path)
	for _, f := range files {
		if f == path {
			return files
		}
	}
	return append(files, path)
}
