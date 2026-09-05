// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Check typechecks an application's TypeScript roots using
// typescript-go-internal. It does not invoke Node or vue-tsc.
// ScopeNoVue includes web TS/TSX plus embedded vite/client and subpath ambient.
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
	var ambientOverlays map[string]string
	if scope == ScopeNoVue {
		ambientOverlays = BuiltInAmbientOverlays(modulesPath)
		overlays = mergeOverlays(overlays, ambientOverlays)
	}
	fs := newTypecheckFS(overlays)
	files = appendOverlayRoots(files, modulesPath, opts.App, scope, overlays, fs.UseCaseSensitiveFileNames())
	for _, ambient := range sortedOverlayPaths(ambientOverlays) {
		files = appendUniqueSlash(files, ambient)
	}
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
	if len(out) == 0 {
		return nil
	}
	return out
}

// appendOverlayRoots adds overlay-only paths that match the scope collection
// rules (app-root / service / web), including virtual files.
func appendOverlayRoots(files []string, modulesPath, app string, scope Scope, overlays map[string]string, caseSensitive bool) []string {
	switch scope {
	case ScopeService, ScopeNoVue:
	default:
		return files
	}
	if len(overlays) == 0 {
		return files
	}
	app = strings.TrimSpace(app)
	appRoot := filepath.ToSlash(filepath.Join(modulesPath, app))
	overlayKeys := make([]string, 0, len(overlays))
	for k := range overlays {
		overlayKeys = append(overlayKeys, k)
	}
	slices.Sort(overlayKeys)
	for _, path := range overlayKeys {
		norm := normalizePathKey(path)
		if norm == "" || shouldSkipTSFileName(filepath.Base(norm)) {
			continue
		}
		lower := strings.ToLower(norm)
		isTSX := strings.HasSuffix(lower, ".tsx")
		isTS := strings.HasSuffix(lower, ".ts")
		if !(isTS || (isTSX && scope == ScopeNoVue)) {
			continue
		}
		rel, ok := cutPathPrefix(norm, appRoot+"/", caseSensitive)
		if !ok {
			continue
		}
		if !strings.Contains(rel, "/") {
			// App-root roots are .ts / .d.ts only (not .tsx), matching CollectRootFiles.
			if isTSX {
				continue
			}
			files = appendUniqueSlash(files, norm)
			continue
		}
		relLower := rel
		if !caseSensitive {
			relLower = strings.ToLower(rel)
		}
		switch {
		case strings.HasPrefix(relLower, "service/"):
			if isTSX {
				continue
			}
		case scope == ScopeNoVue && strings.HasPrefix(relLower, "web/"):
			// web allows .ts and .tsx
		default:
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
	if !strings.EqualFold(path[:len(prefix)], prefix) {
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
