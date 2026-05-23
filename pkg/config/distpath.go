// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "path/filepath"

// Layout (under Config.DistPath):
// - apps/<app>
// - bundles
// - web

func AppsDir(distRoot string) string {
	return filepath.Join(distRoot, "apps")
}

func AppDir(distRoot string, app string) string {
	return filepath.Join(distRoot, "apps", app)
}

func AppIndexJS(distRoot string, app string) string {
	return filepath.Join(distRoot, "apps", app, "index.js")
}

func AppAssetsDir(distRoot string, app string) string {
	return filepath.Join(distRoot, "apps", app, "assets")
}

func BundlesDir(distRoot string) string {
	return filepath.Join(distRoot, "bundles")
}

func BundlesIndexJS(distRoot string) string {
	return filepath.Join(distRoot, "bundles", "index.js")
}

func BundlesAssetsDir(distRoot string) string {
	return filepath.Join(distRoot, "bundles", "assets")
}

func BundlesAssetsAppDir(distRoot string, app string) string {
	return filepath.Join(distRoot, "bundles", "assets", app)
}

// APIRootFromDist returns the runtime API root adjacent to dist root.
// Example: dist=/repo/.choysum/dist => api=/repo/.choysum/api
func APIRootFromDist(distRoot string) string {
	return filepath.Join(filepath.Dir(distRoot), "api")
}

func APIAppProtoDir(distRoot string, app string) string {
	return filepath.Join(APIRootFromDist(distRoot), app, "proto")
}

func WebDir(distRoot string) string {
	return filepath.Join(distRoot, "web")
}
