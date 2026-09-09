// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysume2e

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed choysume2e.js
var ChoysumE2EScript string

// SourcePath returns the on-disk path to choysume2e.js for esbuild aliasing.
func SourcePath() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	p := filepath.Join(filepath.Dir(thisFile), "choysume2e.js")
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

// PlaywrightShimPath returns the Node ESM shim used when Playwright resolves `@choysum/e2e`.
func PlaywrightShimPath() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	p := filepath.Join(filepath.Dir(thisFile), "choysume2e_pw_shim.mjs")
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}
