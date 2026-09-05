// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"slices"
)

// ambientDirName is the modules-relative directory for embedded ambient overlays.
const ambientDirName = ".typecheck-ambient"

// AmbientRoot returns the absolute ambient directory under modulesPath.
func AmbientRoot(modulesPath string) string {
	return filepath.Clean(filepath.Join(modulesPath, ambientDirName))
}

// BuiltInAmbientOverlays returns vite/client + subpath stub overlays keyed by
// absolute slash paths under AmbientRoot(modulesPath).
func BuiltInAmbientOverlays(modulesPath string) map[string]string {
	root := AmbientRoot(modulesPath)
	out := make(map[string]string, 2)
	if rel, content := ViteClientOverlay(); rel != "" {
		out[filepath.ToSlash(filepath.Join(root, filepath.FromSlash(rel)))] = content
	}
	if rel, content := SubpathStubOverlay(); rel != "" {
		out[filepath.ToSlash(filepath.Join(root, filepath.FromSlash(rel)))] = content
	}
	return out
}

// AmbientRootFiles returns sorted absolute ambient .d.ts paths to include as
// program roots for scopes that need web ambient (ScopeNoVue).
func AmbientRootFiles(modulesPath string) []string {
	overlays := BuiltInAmbientOverlays(modulesPath)
	files := make([]string, 0, len(overlays))
	for path := range overlays {
		files = append(files, normalizePathKey(path))
	}
	slices.Sort(files)
	return files
}

// mergeOverlays merges overlay maps; later maps win on key conflicts.
// Empty or nil inputs are ignored; an empty result is nil.
func mergeOverlays(maps ...map[string]string) map[string]string {
	var out map[string]string
	for _, m := range maps {
		if len(m) == 0 {
			continue
		}
		if out == nil {
			out = make(map[string]string, len(m))
		}
		for k, v := range m {
			key := normalizePathKey(k)
			if key == "" {
				continue
			}
			out[key] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
