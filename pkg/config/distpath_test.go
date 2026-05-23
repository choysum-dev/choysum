// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"path/filepath"
	"testing"
)

func TestDistPathHelpers(t *testing.T) {
	root := filepath.Join("tmp", "dist")
	app := "portal"

	checks := []struct {
		name string
		got  string
		want string
	}{
		{name: "AppsDir", got: AppsDir(root), want: filepath.Join(root, "apps")},
		{name: "AppDir", got: AppDir(root, app), want: filepath.Join(root, "apps", app)},
		{name: "AppIndexJS", got: AppIndexJS(root, app), want: filepath.Join(root, "apps", app, "index.js")},
		{name: "AppAssetsDir", got: AppAssetsDir(root, app), want: filepath.Join(root, "apps", app, "assets")},
		{name: "BundlesDir", got: BundlesDir(root), want: filepath.Join(root, "bundles")},
		{name: "BundlesIndexJS", got: BundlesIndexJS(root), want: filepath.Join(root, "bundles", "index.js")},
		{name: "BundlesAssetsDir", got: BundlesAssetsDir(root), want: filepath.Join(root, "bundles", "assets")},
		{name: "BundlesAssetsAppDir", got: BundlesAssetsAppDir(root, app), want: filepath.Join(root, "bundles", "assets", app)},
		{name: "APIRootFromDist", got: APIRootFromDist(root), want: filepath.Join("tmp", "api")},
		{name: "APIAppProtoDir", got: APIAppProtoDir(root, app), want: filepath.Join("tmp", "api", app, "proto")},
		{name: "WebDir", got: WebDir(root), want: filepath.Join(root, "web")},
	}

	for _, check := range checks {
		if check.got != check.want {
			t.Fatalf("%s() = %q, want %q", check.name, check.got, check.want)
		}
	}
}
