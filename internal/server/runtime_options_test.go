// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestServerRuntimeOptionsDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	serverDefaults := config.NewDefaultServerConfig()
	compileDefaults := config.NewDefaultCompileConfig()

	base := defaultRuntimeOptions()
	if base.compileBundleMode != compileDefaults.BundleMode || base.bindAddress != serverDefaults.BindAddress || base.port != serverDefaults.Port {
		t.Fatalf("default runtime options mismatch: %#v", base)
	}
	if base.grpcClientMaxCachedConn != serverDefaults.GrpcClient.MaxCachedConns {
		t.Fatalf("default grpc client cache = %d, want %d", base.grpcClientMaxCachedConn, serverDefaults.GrpcClient.MaxCachedConns)
	}

	blankCompile := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{BundleMode: "   "}, true, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false)
	if blankCompile.compileBundleMode != compileDefaults.BundleMode {
		t.Fatalf("blank compile bundle mode should keep default, got %q", blankCompile.compileBundleMode)
	}

	override := newRuntimeOptions(
		scope.PathsRuntimeOptions{ModulesPath: "/workspace/modules", DistPath: "/workspace/dist"},
		true,
		scope.CompileRuntimeOptions{BundleMode: "application"},
		true,
		scope.ServerRuntimeOptions{
			BindAddress:              "   ",
			Port:                     0,
			EnableGzip:               false,
			EnabledTLS:               true,
			TLSCaFile:                "/tls/ca.pem",
			TLSServerName:            "api.internal",
			TLSCertFile:              "/tls/server.pem",
			TLSKeyFile:               "/tls/server.key",
			EnableGrpcWebProxy:       false,
			HotReload:                true,
			GrpcClientMaxCachedConns: 0,
			SecurityMissing:          false,
			CSP:                      nil,
			CSRF:                     nil,
		},
		true,
		scope.AuthRuntimeOptions{Enabled: true, HttpAuth: &config.HttpAuthConfig{Enabled: true}},
		true,
	)
	if override.modulesPath != "/workspace/modules" || override.distPath != "/workspace/dist" {
		t.Fatalf("override paths = %#v", override)
	}
	if override.compileBundleMode != "application" {
		t.Fatalf("compile bundle override = %q, want %q", override.compileBundleMode, "application")
	}
	if override.bindAddress != serverDefaults.BindAddress || override.port != serverDefaults.Port {
		t.Fatalf("blank bind/port should keep defaults, got bind=%q port=%d", override.bindAddress, override.port)
	}
	if !override.enabledTLS || override.enableGzip || !override.hotReload || override.enableGrpcWebProxy {
		t.Fatalf("server bool overrides not applied as expected: %#v", override)
	}
	if override.grpcClientMaxCachedConn != serverDefaults.GrpcClient.MaxCachedConns {
		t.Fatalf("grpc cache should keep default when override <= 0, got %d", override.grpcClientMaxCachedConn)
	}
	if override.cspEnabled || override.csrfEnabled {
		t.Fatalf("nil security config with SecurityMissing=false should disable csp/csrf, got %#v", override)
	}
	if !override.authEnabled || !override.httpAuthEnabled {
		t.Fatalf("auth overrides not applied: %#v", override)
	}

	serverWithSecurity := newRuntimeOptions(
		scope.PathsRuntimeOptions{},
		false,
		scope.CompileRuntimeOptions{},
		false,
		scope.ServerRuntimeOptions{
			BindAddress:              "0.0.0.0",
			Port:                     8088,
			GrpcClientMaxCachedConns: 64,
			CSP:                      &config.CSPConfig{Enabled: true},
			CSRF:                     &config.CSRFConfig{Enabled: false},
			SecurityMissing:          true,
		},
		true,
		scope.AuthRuntimeOptions{Enabled: true, HttpAuth: nil},
		true,
	)
	if serverWithSecurity.bindAddress != "0.0.0.0" || serverWithSecurity.port != 8088 || serverWithSecurity.grpcClientMaxCachedConn != 64 {
		t.Fatalf("explicit server overrides = %#v", serverWithSecurity)
	}
	if !serverWithSecurity.cspEnabled || serverWithSecurity.csrfEnabled {
		t.Fatalf("security pointer overrides not applied: %#v", serverWithSecurity)
	}
	if serverWithSecurity.httpAuthEnabled {
		t.Fatalf("httpAuthEnabled with nil HttpAuth should be false, got %#v", serverWithSecurity)
	}
}

func TestServerRuntimeOptionsFromScopeAndResolved(t *testing.T) {
	t.Parallel()

	nilScope := runtimeOptionsFromScope(nil)
	if nilScope.bindAddress == "" || nilScope.port <= 0 {
		t.Fatalf("runtimeOptionsFromScope(nil) = %#v", nilScope)
	}

	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.ModulesPath = "/workspace/modules"
	runtimeScope.cfg.DistPath = "/workspace/dist"
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.Server.BindAddress = "0.0.0.0"
	runtimeScope.cfg.Server.Port = 9090
	runtimeScope.cfg.Server.GrpcClient = &config.GrpcClientConfig{MaxCachedConns: 88}
	runtimeScope.cfg.Server.Security = &config.SecurityConfig{
		CSP:  &config.CSPConfig{Enabled: true},
		CSRF: &config.CSRFConfig{Enabled: false},
	}
	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.HttpAuth = &config.HttpAuthConfig{Enabled: true}

	fromScope := runtimeOptionsFromScope(runtimeScope)
	if fromScope.modulesPath != "/workspace/modules" || fromScope.distPath != "/workspace/dist" {
		t.Fatalf("runtimeOptionsFromScope(paths) = %#v", fromScope)
	}
	if fromScope.compileBundleMode != "application" || fromScope.bindAddress != "0.0.0.0" || fromScope.port != 9090 {
		t.Fatalf("runtimeOptionsFromScope(compile/server) = %#v", fromScope)
	}
	if fromScope.grpcClientMaxCachedConn != 88 || !fromScope.cspEnabled || fromScope.csrfEnabled {
		t.Fatalf("runtimeOptionsFromScope(security/grpc) = %#v", fromScope)
	}
	if !fromScope.authEnabled || !fromScope.httpAuthEnabled {
		t.Fatalf("runtimeOptionsFromScope(auth) = %#v", fromScope)
	}

	explicit := runtimeOptions{bindAddress: "127.0.0.1", port: 8080, modulesPath: "/explicit/modules", distPath: "/explicit/dist", compileBundleMode: "bundle", grpcClientMaxCachedConn: 1}
	srv := &GRPCWebServer{runtimeOptions: explicit, runtimeScope: runtimeScope}
	gotExplicit := srv.resolvedRuntimeOptions()
	if gotExplicit.bindAddress != explicit.bindAddress || gotExplicit.port != explicit.port {
		t.Fatalf("resolvedRuntimeOptions(explicit) = %#v", gotExplicit)
	}

	srv = &GRPCWebServer{runtimeScope: runtimeScope}
	gotScope := srv.resolvedRuntimeOptions()
	if gotScope.bindAddress != "0.0.0.0" || gotScope.port != 9090 {
		t.Fatalf("resolvedRuntimeOptions(scope) = %#v", gotScope)
	}

	srv = &GRPCWebServer{runtimeOptions: runtimeOptions{bindAddress: "runtime-only"}}
	gotRuntimeOnly := srv.resolvedRuntimeOptions()
	if gotRuntimeOnly.bindAddress != "runtime-only" {
		t.Fatalf("resolvedRuntimeOptions(runtime-only) = %#v", gotRuntimeOnly)
	}

	var nilServer *GRPCWebServer
	gotNil := nilServer.resolvedRuntimeOptions()
	if gotNil.bindAddress == "" || gotNil.port <= 0 {
		t.Fatalf("resolvedRuntimeOptions(nil server) = %#v", gotNil)
	}

	if hasRuntimeOptions(runtimeOptions{bindAddress: "  "}) {
		t.Fatal("hasRuntimeOptions(blank bindAddress) should be false")
	}
	if !hasRuntimeOptions(runtimeOptions{bindAddress: "127.0.0.1"}) {
		t.Fatal("hasRuntimeOptions(non-blank bindAddress) should be true")
	}
}

func TestServerRuntimeOptionsValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts runtimeOptions
		msg  string
	}{
		{name: "missing modules", opts: runtimeOptions{distPath: "/dist", compileBundleMode: "bundle", bindAddress: "127.0.0.1", port: 1, grpcClientMaxCachedConn: 1}, msg: "modulesPath"},
		{name: "missing dist", opts: runtimeOptions{modulesPath: "/modules", compileBundleMode: "bundle", bindAddress: "127.0.0.1", port: 1, grpcClientMaxCachedConn: 1}, msg: "distPath"},
		{name: "missing bundle mode", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", bindAddress: "127.0.0.1", port: 1, grpcClientMaxCachedConn: 1}, msg: "compileBundleMode"},
		{name: "missing bind address", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", compileBundleMode: "bundle", port: 1, grpcClientMaxCachedConn: 1}, msg: "bindAddress"},
		{name: "invalid port", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", compileBundleMode: "bundle", bindAddress: "127.0.0.1", grpcClientMaxCachedConn: 1}, msg: "port"},
		{name: "invalid grpc cache", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", compileBundleMode: "bundle", bindAddress: "127.0.0.1", port: 1}, msg: "grpcClientMaxCachedConn"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	if err := (runtimeOptions{modulesPath: "/modules", distPath: "/dist", compileBundleMode: "bundle", bindAddress: "127.0.0.1", port: 8080, grpcClientMaxCachedConn: 8}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}
