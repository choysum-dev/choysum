// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webplugin

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestWebPluginRuntimeOptionsNewAndValidate(t *testing.T) {
	t.Parallel()

	serverDefaults := config.NewDefaultServerConfig()
	base := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false)
	if base.webBaseURL != serverDefaults.WebBaseURL || base.serverEnvironment != serverDefaults.Environment || base.serverEnabledTLS != serverDefaults.EnabledTLS {
		t.Fatalf("default runtime options = %#v", base)
	}

	override := newRuntimeOptions(
		scope.PathsRuntimeOptions{ModulesPath: "/workspace/modules", DistPath: "/workspace/dist"},
		true,
		scope.ServerRuntimeOptions{WebBaseURL: "/portal/", Environment: "production", EnabledTLS: true},
		true,
	)
	if override.modulesPath != "/workspace/modules" || override.distPath != "/workspace/dist" {
		t.Fatalf("override paths = %#v", override)
	}
	if override.webBaseURL != "/portal/" || override.serverEnvironment != "production" || !override.serverEnabledTLS {
		t.Fatalf("override server options = %#v", override)
	}

	cases := []struct {
		name string
		opts runtimeOptions
		msg  string
	}{
		{name: "missing modules", opts: runtimeOptions{distPath: "/dist", webBaseURL: "/web/", serverEnvironment: "dev"}, msg: "modulesPath"},
		{name: "missing dist", opts: runtimeOptions{modulesPath: "/modules", webBaseURL: "/web/", serverEnvironment: "dev"}, msg: "distPath"},
		{name: "missing web base", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", serverEnvironment: "dev"}, msg: "webBaseURL"},
		{name: "missing environment", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", webBaseURL: "/web/"}, msg: "serverEnvironment"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	if err := (runtimeOptions{modulesPath: "/modules", distPath: "/dist", webBaseURL: "/web/", serverEnvironment: "dev"}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}

func TestWebPluginRuntimeOptionsFromScopeAndResolved(t *testing.T) {
	t.Parallel()

	nilScope := runtimeOptionsFromScope(nil)
	if nilScope.webBaseURL == "" {
		t.Fatalf("runtimeOptionsFromScope(nil) = %#v", nilScope)
	}

	runtimeScope := newTestScope(t).(*stubScope)
	runtimeScope.cfg.Server.WebBaseURL = "/portal/"
	runtimeScope.cfg.Server.Environment = "production"
	runtimeScope.cfg.Server.EnabledTLS = true
	runtimeScope.cfg.ModulesPath = "/workspace/modules"
	runtimeScope.cfg.DistPath = "/workspace/dist"

	fromScope := runtimeOptionsFromScope(runtimeScope)
	if fromScope.modulesPath != "/workspace/modules" || fromScope.distPath != "/workspace/dist" {
		t.Fatalf("runtimeOptionsFromScope(paths) = %#v", fromScope)
	}
	if fromScope.webBaseURL != "/portal/" || fromScope.serverEnvironment != "production" || !fromScope.serverEnabledTLS {
		t.Fatalf("runtimeOptionsFromScope(server) = %#v", fromScope)
	}

	explicit := runtimeOptions{modulesPath: "/explicit/modules", distPath: "/explicit/dist", webBaseURL: "/explicit/", serverEnvironment: "staging", serverEnabledTLS: true}
	plugin := &WebPlugin{BasePlugin: &esbplugins.BasePlugin{Env: runtimeScope}, runtimeOptions: explicit}
	if got := plugin.resolvedRuntimeOptions(); got != explicit {
		t.Fatalf("resolvedRuntimeOptions(explicit) = %#v, want %#v", got, explicit)
	}

	plugin = &WebPlugin{BasePlugin: &esbplugins.BasePlugin{Env: runtimeScope}}
	gotScope := plugin.resolvedRuntimeOptions()
	if gotScope.webBaseURL != "/portal/" || gotScope.modulesPath != "/workspace/modules" {
		t.Fatalf("resolvedRuntimeOptions(scope) = %#v", gotScope)
	}

	plugin = &WebPlugin{BasePlugin: &esbplugins.BasePlugin{}, runtimeOptions: runtimeOptions{modulesPath: "/only-runtime-field"}}
	gotRuntimeOnly := plugin.resolvedRuntimeOptions()
	if gotRuntimeOnly.modulesPath != "/only-runtime-field" || gotRuntimeOnly.webBaseURL != "" {
		t.Fatalf("resolvedRuntimeOptions(runtime-only) = %#v", gotRuntimeOnly)
	}

	var nilPlugin *WebPlugin
	gotNil := nilPlugin.resolvedRuntimeOptions()
	if gotNil.webBaseURL == "" {
		t.Fatalf("resolvedRuntimeOptions(nil plugin) = %#v", gotNil)
	}

	if hasRuntimeOptions(runtimeOptions{webBaseURL: "  "}) {
		t.Fatal("hasRuntimeOptions(blank webBaseURL) should be false")
	}
	if !hasRuntimeOptions(runtimeOptions{webBaseURL: "/web/"}) {
		t.Fatal("hasRuntimeOptions(non-blank webBaseURL) should be true")
	}
}
