// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import compileconfig "github.com/choysum-dev/choysum/internal/module/artifact/config/compile"

type BundleMode = compileconfig.BundleMode

const (
	BundleModeBundle      = compileconfig.BundleModeBundle
	BundleModeApplication = compileconfig.BundleModeApplication
)

type CompileConfig = compileconfig.CompileConfig

func NormalizeCompileBundleMode(mode string) (BundleMode, error) {
	return compileconfig.NormalizeBundleMode(mode)
}

func NewDefaultCompileConfig() *CompileConfig {
	return compileconfig.NewDefaultCompileConfig()
}
