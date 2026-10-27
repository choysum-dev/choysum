// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// filepathRel is filepath.Rel; tests replace it to cover Rel failures.
var filepathRel = filepath.Rel

// choysumGeneratedPlugin maps @choysum/generated/<rel> imports onto
// <defaultChoysumPath>/generated/<rel> so kit CSS and other workspace
// generated assets resolve outside modules/.
func choysumGeneratedPlugin(defaultChoysumPath string) api.Plugin {
	root := filepath.Clean(strings.TrimSpace(defaultChoysumPath))
	return api.Plugin{
		Name: "choysum-generated",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@choysum/generated/`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if root == "" || root == "." {
					return api.OnResolveResult{}, fmt.Errorf("@choysum/generated import requires defaultChoysumPath")
				}
				rel := strings.TrimPrefix(args.Path, ChoyGeneratedImportPrefix)
				rel = filepath.Clean(filepath.FromSlash(rel))
				if isUnsafeChoysumGeneratedRel(rel) {
					return api.OnResolveResult{}, fmt.Errorf("@choysum/generated path escapes generated root: %s", args.Path)
				}
				generatedRoot := filepath.Join(root, "generated")
				resolved := filepath.Join(generatedRoot, rel)
				if under, err := pathUnderRoot(generatedRoot, resolved); err != nil || !under {
					return api.OnResolveResult{}, fmt.Errorf("@choysum/generated path escapes generated root: %s", args.Path)
				}
				return api.OnResolveResult{
					Path:      resolved,
					Namespace: "file",
				}, nil
			})
		},
	}
}

// isUnsafeChoysumGeneratedRel reports import relatives that must not be joined
// onto <defaultChoysumPath>/generated. Includes "." / ".." traversal, absolute
// paths, and current-drive-rooted forms (e.g. `\etc\passwd` on Windows where
// filepath.IsAbs is false).
func isUnsafeChoysumGeneratedRel(rel string) bool {
	return rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, string(filepath.Separator))
}

// pathUnderRoot reports whether candidate stays inside root after Clean.
func pathUnderRoot(root, candidate string) (bool, error) {
	rel, err := filepathRel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}
