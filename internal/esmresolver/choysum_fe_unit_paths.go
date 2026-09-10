// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"os"
	"path/filepath"
)

// choysumFEUnitTsconfigPathEntries maps bare Choysum package names to repo-relative
// declaration files (from modules/). Runtime resolution still uses host_bundle /
// e2e-bundle esbuild aliases; these paths exist so IDEs can typecheck
// *.test.ts / e2e/*.spec.ts imports. Entries whose target .d.ts is missing are
// omitted (temp-dir type-fetch tests).
func choysumFEUnitTsconfigPathEntries(modulesDir string) (map[string][]string, error) {
	modulesDir = filepath.Clean(modulesDir)
	repoRoot := filepath.Clean(filepath.Join(modulesDir, ".."))
	abs := map[string]string{
		"@choysum/test-utils": filepath.Join(repoRoot, "pkg", "jsengine", "scripts", "choysummount", "choysummount.d.ts"),
		"@choysum/page-mount": filepath.Join(repoRoot, "internal", "testing", "frontend", "testdata", "stubs", "page_mount.d.ts"),
		"@choysum/e2e":        filepath.Join(repoRoot, "pkg", "jsengine", "scripts", "choysume2e", "choysume2e.d.ts"),
	}
	out := make(map[string][]string, len(abs))
	for pkg, target := range abs {
		if _, err := os.Stat(target); err != nil {
			continue
		}
		rel, err := filepathRel(modulesDir, target)
		if err != nil {
			return nil, err
		}
		out[pkg] = []string{filepath.ToSlash(rel)}
	}
	return out, nil
}

// applyChoysumFEUnitTsconfigPaths ensures IDE path mappings for Choysum QJS
// helpers (@choysum/test-utils, @choysum/page-mount, @choysum/e2e).
// Returns how many path entries were inserted or updated.
func applyChoysumFEUnitTsconfigPaths(modulesDir string, paths map[string]interface{}) (int, error) {
	entries, err := choysumFEUnitTsconfigPathEntries(modulesDir)
	if err != nil {
		return 0, err
	}
	applied := 0
	for pkg, want := range entries {
		if tsconfigPathMappingEquals(paths[pkg], want) {
			continue
		}
		paths[pkg] = want
		applied++
	}
	return applied, nil
}
