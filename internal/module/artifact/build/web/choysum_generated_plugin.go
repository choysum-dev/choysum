// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

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
					return api.OnResolveResult{}, nil
				}
				rel := strings.TrimPrefix(args.Path, ChoyGeneratedImportPrefix)
				rel = filepath.FromSlash(rel)
				return api.OnResolveResult{
					Path:      filepath.Join(root, "generated", rel),
					Namespace: "file",
				}, nil
			})
		},
	}
}
