// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package compileconfig

import (
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

type BundleMode string

const (
	BundleModeBundle      BundleMode = "bundle"
	BundleModeApplication BundleMode = "application"
)

type CompileConfig struct {
	BundleMode  string `mapstructure:"bundleMode"`
	Production  bool   `mapstructure:"production"`
	Minify      bool   `mapstructure:"minify"`
	TreeShaking bool   `mapstructure:"treeShaking"`
	SourceMap   bool   `mapstructure:"sourcemap"`
}

func NormalizeBundleMode(mode string) (BundleMode, error) {
	normalized := BundleMode(strings.ToLower(strings.TrimSpace(mode)))
	if normalized == "" {
		normalized = BundleModeBundle
	}
	switch normalized {
	case BundleModeBundle, BundleModeApplication:
		return normalized, nil
	default:
		return "", xfmt.Errorf("invalid compile.bundleMode: %q (allowed: %q | %q)", mode, BundleModeBundle, BundleModeApplication)
	}
}

func NewDefaultCompileConfig() *CompileConfig {
	return &CompileConfig{
		BundleMode:  string(BundleModeBundle),
		Production:  true,
		Minify:      true,
		TreeShaking: true,
		SourceMap:   false,
	}
}
