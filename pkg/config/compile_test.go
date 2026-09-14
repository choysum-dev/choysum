// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"os"
	"testing"
)

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

func TestCompileSourceMapEnvMatrix(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{name: "unset keeps default false", env: "", want: false},
		{name: "true enables", env: "true", want: true},
		{name: "false disables", env: "false", want: false},
		{name: "1 enables", env: "1", want: true},
		{name: "0 disables", env: "0", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := writeTestConfig(t, "default_choysum_path: ./.choysum-custom\n")
			t.Setenv("CHOYSUM_TEST_COMPILE_SOURCEMAP", tc.env)
			if tc.env == "" {
				t.Setenv("CHOYSUM_TEST_COMPILE_SOURCEMAP", "")
				_ = os.Unsetenv("CHOYSUM_TEST_COMPILE_SOURCEMAP")
			}
			cfg := defaultConfig()
			if err := cfg.unmarshal(cfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
				t.Fatalf("unmarshal() error = %v", err)
			}
			if cfg.Compile == nil {
				t.Fatal("expected compile config")
			}
			if cfg.Compile.SourceMap != tc.want {
				t.Fatalf("SourceMap = %v, want %v (env=%q)", cfg.Compile.SourceMap, tc.want, tc.env)
			}
		})
	}
}
