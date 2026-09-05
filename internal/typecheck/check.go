// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"os"
	"path/filepath"
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
		return Result{}, err
	}

	fs := newTypecheckFS(opts.Overlays)
	if coreAmbient := filepath.ToSlash(filepath.Join(modulesPath, "core", "types", "$choysum.d.ts")); fs.FileExists(coreAmbient) {
		files = appendUniqueSlash(files, coreAmbient)
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
