// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"context"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestWebRuntimeOptionsNewAndValidate(t *testing.T) {
	t.Parallel()

	compileDefaults := config.NewDefaultCompileConfig()
	serverDefaults := config.NewDefaultServerConfig()
	base := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false, scope.CompileRuntimeOptions{}, false)
	if base.frontendEnv == nil {
		t.Fatal("default frontendEnv should be initialized")
	}
	if len(base.frontendEnv) != 0 {
		t.Fatalf("default frontendEnv length = %d, want 0", len(base.frontendEnv))
	}
	if base.webBaseURL != serverDefaults.WebBaseURL {
		t.Fatalf("default webBaseURL = %q, want %q", base.webBaseURL, serverDefaults.WebBaseURL)
	}
	if base.compileSourceMap != compileDefaults.SourceMap || base.compileMinify != compileDefaults.Minify || base.compileTreeShaking != compileDefaults.TreeShaking {
		t.Fatalf("default compile flags = %#v, want defaults from config", base)
	}

	override := newRuntimeOptions(
		scope.PathsRuntimeOptions{ModulesPath: "/workspace/modules", DistPath: "/workspace/dist", DefaultChoysumPath: "/workspace/.choysum"},
		true,
		scope.ServerRuntimeOptions{WebBaseURL: "/portal/"},
		true,
		scope.RuntimeEnvironmentOptions{FrontendEnv: map[string]any{"VITE_API_BASE": "/api"}},
		true,
		scope.CompileRuntimeOptions{SourceMap: false, Minify: true, TreeShaking: false},
		true,
	)
	if override.modulesPath != "/workspace/modules" || override.distPath != "/workspace/dist" || override.defaultChoysumPath != "/workspace/.choysum" {
		t.Fatalf("override path fields = %#v", override)
	}
	if override.webBaseURL != "/portal/" {
		t.Fatalf("override webBaseURL = %q, want %q", override.webBaseURL, "/portal/")
	}
	if override.frontendEnv["VITE_API_BASE"] != "/api" {
		t.Fatalf("override frontend env = %#v", override.frontendEnv)
	}
	if override.compileSourceMap || !override.compileMinify || override.compileTreeShaking {
		t.Fatalf("override compile flags = %#v", override)
	}

	nilEnv := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{FrontendEnv: nil}, true, scope.CompileRuntimeOptions{}, false)
	if nilEnv.frontendEnv == nil || len(nilEnv.frontendEnv) != 0 {
		t.Fatalf("nil FrontendEnv override should keep initialized empty map, got %#v", nilEnv.frontendEnv)
	}

	cases := []struct {
		name string
		opts runtimeOptions
		msg  string
	}{
		{name: "missing modules", opts: runtimeOptions{distPath: "/dist", defaultChoysumPath: "/root", webBaseURL: "/web/"}, msg: "modulesPath"},
		{name: "missing dist", opts: runtimeOptions{modulesPath: "/modules", defaultChoysumPath: "/root", webBaseURL: "/web/"}, msg: "distPath"},
		{name: "missing default path", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", webBaseURL: "/web/"}, msg: "defaultChoysumPath"},
		{name: "missing web base", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", defaultChoysumPath: "/root"}, msg: "webBaseURL"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	if err := (runtimeOptions{modulesPath: "/modules", distPath: "/dist", defaultChoysumPath: "/root", webBaseURL: "/web/"}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}

func TestWebRuntimeOptionsFromScopeAndResolved(t *testing.T) {
	t.Parallel()

	nilScope := runtimeOptionsFromScope(nil)
	if nilScope.frontendEnv == nil {
		t.Fatal("runtimeOptionsFromScope(nil) should initialize frontendEnv")
	}

	runtimeScope := &testScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath:        "/workspace/modules",
			DistPath:           "/workspace/dist",
			DefaultChoysumPath: "/workspace/.choysum",
			Server:             &config.ServerConfig{WebBaseURL: "/portal/"},
			FrontendEnv:        map[string]any{"VITE_MODE": "test"},
			Compile:            &config.CompileConfig{SourceMap: false, Minify: true, TreeShaking: false},
		},
	}
	fromScope := runtimeOptionsFromScope(runtimeScope)
	if fromScope.modulesPath != "/workspace/modules" || fromScope.distPath != "/workspace/dist" || fromScope.defaultChoysumPath != "/workspace/.choysum" {
		t.Fatalf("runtimeOptionsFromScope(paths) = %#v", fromScope)
	}
	if fromScope.webBaseURL != "/portal/" || fromScope.frontendEnv["VITE_MODE"] != "test" {
		t.Fatalf("runtimeOptionsFromScope(server/env) = %#v", fromScope)
	}
	if fromScope.compileSourceMap || !fromScope.compileMinify || fromScope.compileTreeShaking {
		t.Fatalf("runtimeOptionsFromScope(compile) = %#v", fromScope)
	}

	explicit := runtimeOptions{distPath: "/explicit/dist", modulesPath: "/explicit/modules", defaultChoysumPath: "/explicit/root", webBaseURL: "/explicit/"}
	builder := &WebModuleBuilder{runtimeOptions: explicit, runtimeScope: runtimeScope}
	gotExplicit := builder.resolvedRuntimeOptions()
	if gotExplicit.distPath != explicit.distPath || gotExplicit.webBaseURL != explicit.webBaseURL {
		t.Fatalf("resolvedRuntimeOptions(explicit) = %#v", gotExplicit)
	}

	builder = &WebModuleBuilder{runtimeScope: runtimeScope}
	gotScope := builder.resolvedRuntimeOptions()
	if gotScope.distPath != "/workspace/dist" || gotScope.webBaseURL != "/portal/" {
		t.Fatalf("resolvedRuntimeOptions(scope) = %#v", gotScope)
	}

	builder = &WebModuleBuilder{runtimeOptions: runtimeOptions{modulesPath: "/only-runtime-field"}}
	gotRuntimeOnly := builder.resolvedRuntimeOptions()
	if gotRuntimeOnly.modulesPath != "/only-runtime-field" {
		t.Fatalf("resolvedRuntimeOptions(runtime-only) = %#v", gotRuntimeOnly)
	}

	var nilBuilder *WebModuleBuilder
	gotNil := nilBuilder.resolvedRuntimeOptions()
	if gotNil.frontendEnv == nil {
		t.Fatalf("resolvedRuntimeOptions(nil builder) = %#v", gotNil)
	}

	if hasRuntimeOptions(runtimeOptions{distPath: "  "}) {
		t.Fatal("hasRuntimeOptions(blank distPath) should be false")
	}
	if !hasRuntimeOptions(runtimeOptions{distPath: "/dist"}) {
		t.Fatal("hasRuntimeOptions(non-blank distPath) should be true")
	}
}
