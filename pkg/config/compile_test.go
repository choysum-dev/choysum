// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestCompileConfigNormalizedBundleMode(t *testing.T) {
	t.Run("defaults empty to bundle", func(t *testing.T) {
		mode, err := NormalizeCompileBundleMode("")
		if err != nil {
			t.Fatalf("NormalizeCompileBundleMode() error = %v", err)
		}
		if mode != BundleModeBundle {
			t.Fatalf("mode = %q, want %q", mode, BundleModeBundle)
		}
	})

	t.Run("trims and lowercases application", func(t *testing.T) {
		mode, err := NormalizeCompileBundleMode(" Application ")
		if err != nil {
			t.Fatalf("NormalizeCompileBundleMode() error = %v", err)
		}
		if mode != BundleModeApplication {
			t.Fatalf("mode = %q, want %q", mode, BundleModeApplication)
		}
	})

	t.Run("rejects invalid mode", func(t *testing.T) {
		if _, err := NormalizeCompileBundleMode("zip"); err == nil {
			t.Fatal("expected invalid mode error")
		}
	})
}

func TestNewDefaultCompileConfig(t *testing.T) {
	cfg := NewDefaultCompileConfig()
	if cfg.BundleMode != string(BundleModeBundle) {
		t.Fatalf("BundleMode = %q, want %q", cfg.BundleMode, BundleModeBundle)
	}
	if !cfg.Production || !cfg.Minify || !cfg.TreeShaking || cfg.SourceMap {
		t.Fatalf("unexpected compile defaults: %#v", cfg)
	}
}
