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
	if err := ctx.Err(); err != nil {
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

	scope := opts.Scope
	files, err := CollectRootFiles(modulesPath, opts.App, scope)
	if err != nil {
		if !errors.Is(err, ErrNoRootFiles) || len(opts.Overlays) == 0 {
			return Result{}, err
		}
		files = nil
	}

	fs := newTypecheckFS(opts.Overlays)
	files = appendOverlayServiceRoots(files, modulesPath, opts.App, scope, opts.Overlays)
	if coreAmbient := filepath.ToSlash(filepath.Join(modulesPath, "core", "types", "$choysum.d.ts")); fs.FileExists(coreAmbient) {
		files = appendUniqueSlash(files, coreAmbient)
	}
	if len(files) == 0 {
		return Result{}, ErrNoRootFiles
	}

	compilerOpts, err := BuildCompilerOptions(modulesPath, repoRoot)
	if err != nil {
		return Result{}, err
	}

	host := newHost(modulesPath, fs)
	program := buildProgram(host, files, compilerOpts)
	diags, err := collectDiagnostics(ctx, program)
	if err != nil {
		return Result{}, err
	}
	return toResult(diags), nil
}

// appendOverlayServiceRoots adds overlay-only paths that match ScopeService
// collection rules (app-root *.ts / service/**), including virtual files.
func appendOverlayServiceRoots(files []string, modulesPath, app string, scope Scope, overlays map[string]string) []string {
	if scope != ScopeService || len(overlays) == 0 {
		return files
	}
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
		rel, ok := strings.CutPrefix(norm, appRoot+"/")
		if !ok {
			continue
		}
		if !strings.Contains(rel, "/") {
			files = appendUniqueSlash(files, norm)
			continue
		}
		if !strings.HasPrefix(rel, "service/") {
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
