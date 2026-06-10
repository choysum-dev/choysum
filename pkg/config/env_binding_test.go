// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestEnvNameForPath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want string
	}{
		{name: "snake case path", path: []string{"default_choysum_path"}, want: "CHOYSUM_DEFAULT_CHOYSUM_PATH"},
		{name: "npm registry url", path: []string{"npm_registry_url"}, want: "CHOYSUM_NPM_REGISTRY_URL"},
		{name: "module catalog index url", path: []string{"module_catalog_index_url"}, want: "CHOYSUM_MODULE_CATALOG_INDEX_URL"},
		{name: "camel case segment", path: []string{"server", "hotReload"}, want: "CHOYSUM_SERVER_HOT_RELOAD"},
		{name: "acronym segment", path: []string{"server", "jsEngineFactory"}, want: "CHOYSUM_SERVER_JS_ENGINE_FACTORY"},
		{name: "nested auth", path: []string{"auth", "internalKey"}, want: "CHOYSUM_AUTH_INTERNAL_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envNameForPath("CHOYSUM", tt.path); got != tt.want {
				t.Fatalf("envNameForPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestConfigUnmarshalEnvOnlyCompileAndServerKeys(t *testing.T) {
	envOnlyCfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
modules_path: from-config
`)

	t.Setenv("CHOYSUM_TEST_COMPILE_MINIFY", "false")
	t.Setenv("CHOYSUM_TEST_COMPILE_SOURCEMAP", "true")
	t.Setenv("CHOYSUM_TEST_SERVER_HOT_RELOAD", "true")

	cfg := defaultConfig()
	if err := cfg.unmarshal(envOnlyCfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
		t.Fatalf("unmarshal() error = %v", err)
	}
	if cfg.Compile == nil {
		t.Fatalf("expected compile config after unmarshal, got %#v", cfg)
	}
	if cfg.Compile.Minify {
		t.Fatal("expected compile.minify env override to disable minify")
	}
	if !cfg.Compile.SourceMap {
		t.Fatal("expected compile.sourcemap env override to enable sourcemap")
	}
	if cfg.Server == nil {
		t.Fatalf("expected server config after unmarshal, got %#v", cfg)
	}
	if !cfg.Server.HotReload {
		t.Fatal("expected server.hotReload env override to enable hot reload")
	}
}
