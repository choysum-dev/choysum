// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysummount"
)

// FrozenVTUSubsetAPIs is the PR-unit-vue-host mount surface (changes need a follow-up PR).
var FrozenVTUSubsetAPIs = []string{
	"mount",
	"shallowMount",
	"stubs",
	"flushPromises",
	"wrapper.find",
	"wrapper.trigger",
	"wrapper.unmount",
}

// Test seams for ChoysumMountSourcePath (overridden in unit tests).
var (
	resolveChoysumMountSourcePath = defaultResolveChoysumMountSourcePath
	runtimeCallerMount            = runtime.Caller
	osStatMount                   = os.Stat
)

func defaultResolveChoysumMountSourcePath() (string, error) {
	_, thisFile, _, ok := runtimeCallerMount(0)
	if !ok {
		return "", os.ErrNotExist
	}
	// this file: internal/testing/frontend/choysum_mount.go
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	p := filepath.Join(repoRoot, "pkg", "jsengine", "scripts", "choysummount", "choysummount.js")
	if _, err := osStatMount(p); err != nil {
		return "", err
	}
	return p, nil
}

// ChoysumMountSourcePath returns the on-disk path to choysummount.js for esbuild entry/alias.
func ChoysumMountSourcePath() (string, error) {
	return resolveChoysumMountSourcePath()
}

// ChoysumMountScript returns the embedded mount host source (for diagnostics / future Load).
func ChoysumMountScript() string {
	return choysummount.ChoysumMountScript
}

// VueHostPackageVersion is the pinned Vue used by BuildFrontendVueHostBundle.
func VueHostPackageVersion() string {
	return choysummount.VuePackageVersion
}
