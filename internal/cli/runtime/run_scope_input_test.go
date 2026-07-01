// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

func TestRunScopeInputComprehensiveGettersAndIsolation(t *testing.T) {
	cfg := scopeInputFixtureConfig()
	cfg.ModulesPath = "/cfg/modules"
	cfg.DistPath = "/cfg/dist"
	cfg.TmpPath = "/cfg/tmp"
	cfg.DefaultChoysumPath = "/cfg/choysum"
	cfg.ConfigPath = "/cfg/config.yaml"
	cfg.NPMRegistryURL = "https://registry.cfg.example"
	cfg.ModuleCatalogIndexURL = "https://index.cfg.example/v1/index.json"
	cfg.BootstrapModuleInstallTimeoutSeconds = 777
	cfg.ESMUpstreamURL = "https://esm.cfg.example"
	cfg.Log = &config.LogConfig{Format: "json", Level: "debug"}
	cfg.Db = &config.DbConfig{
		Dialect:         "sqlite",
		DSN:             "/cfg/choysum.sqlite",
		MaxOpenConns:    21,
		MaxIdleConns:    12,
		ConnMaxLifetime: 345,
	}
	cfg.Server = &config.ServerConfig{
		Environment:        "development",
		BindAddress:        "127.0.0.1",
		Port:               8080,
		EnableGzip:         true,
		EnabledTLS:         true,
		TLSCaFile:          "/cfg/ca.pem",
		TLSServerName:      "cfg.local",
		TLSCertFile:        "/cfg/cert.pem",
		TLSKeyFile:         "/cfg/key.pem",
		EnableGrpcWebProxy: true,
		HotReload:          true,
		GrpcClient:         &config.GrpcClientConfig{MaxCachedConns: 55},
		Security:           config.NewDefaultSecurityConfig(),
		Register:           "local",
		RuntimeEngine:      "default",
		WebBaseURL:         "/web",
		RootRedirectURL:    "",
		JsEngineFactory:    "quickjs",
		JsExecutorFactory:  "default",
	}

	options := NewScopeInputConfigOptions(snapshot.New(cfg))
	input := NewRunScopeInput(
		options,
		Options{
			ModulesPath:           "/runtime/modules",
			TmpPath:               "/runtime/tmp",
			DefaultChoysumPath:    "/runtime/choysum",
			ModuleCatalogIndexURL: "https://index.runtime.example/v1/index.json",
		},
		RunServerOptions{BindAddress: "0.0.0.0", Port: 9527, EnabledTLS: false},
		RunDBOptions{Dialect: "postgres", DSN: "postgres://runtime-db/app", AllowCreate: false},
	)

	if got := input.ConfigOptions(); got == nil {
		t.Fatal("ConfigOptions() should not be nil")
	}
	if got := input.CLIOptions(); got.ModulesPath != "/runtime/modules" {
		t.Fatalf("CLIOptions().ModulesPath = %q, want %q", got.ModulesPath, "/runtime/modules")
	}
	if got := input.ServerOptions(); got.BindAddress != "0.0.0.0" {
		t.Fatalf("ServerOptions().BindAddress = %q, want %q", got.BindAddress, "0.0.0.0")
	}
	if got := input.DBOptions(); got.Dialect != "postgres" {
		t.Fatalf("DBOptions().Dialect = %q, want %q", got.Dialect, "postgres")
	}

	if got := input.Environment(); got != "development" {
		t.Fatalf("Environment() = %q, want %q", got, "development")
	}
	if got := input.ModulesPath(); got != "/runtime/modules" {
		t.Fatalf("ModulesPath() = %q, want %q", got, "/runtime/modules")
	}
	if got := input.DistPath(); got != "/cfg/dist" {
		t.Fatalf("DistPath() = %q, want %q", got, "/cfg/dist")
	}
	if got := input.TmpPath(); got != "/runtime/tmp" {
		t.Fatalf("TmpPath() = %q, want %q", got, "/runtime/tmp")
	}
	if got := input.DefaultChoysumPath(); got != "/runtime/choysum" {
		t.Fatalf("DefaultChoysumPath() = %q, want %q", got, "/runtime/choysum")
	}
	if got := input.ConfigPath(); got != "/cfg/config.yaml" {
		t.Fatalf("ConfigPath() = %q, want %q", got, "/cfg/config.yaml")
	}
	if got := input.ESMUpstreamURL(); got != "https://esm.cfg.example" {
		t.Fatalf("ESMUpstreamURL() = %q, want %q", got, "https://esm.cfg.example")
	}
	if got := input.NpmRegistryURL(); got != "https://registry.cfg.example" {
		t.Fatalf("NpmRegistryURL() = %q, want %q", got, "https://registry.cfg.example")
	}
	if got := input.ModuleCatalogIndexURL(); got != "https://index.runtime.example/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() = %q, want %q", got, "https://index.runtime.example/v1/index.json")
	}
	if got := input.BootstrapModuleInstallTimeoutSeconds(); got != 777 {
		t.Fatalf("BootstrapModuleInstallTimeoutSeconds() = %d, want 777", got)
	}

	if got := input.CompileBundleMode(); got != "bundle" {
		t.Fatalf("CompileBundleMode() = %q, want %q", got, "bundle")
	}
	if !input.AuthEnabled() {
		t.Fatal("AuthEnabled() expected true")
	}
	if got := input.AuthGrpcAuthentication(); got {
		t.Fatal("AuthGrpcAuthentication() expected false by default fixture")
	}
	if got := input.AuthInternalKey(); got != "" {
		t.Fatalf("AuthInternalKey() = %q, want empty", got)
	}
	if got := input.ServerEnabledTLS(); got != true {
		t.Fatalf("ServerEnabledTLS() = %v, want true from config", got)
	}
	if got := input.ServerBindAddress(); got != "0.0.0.0" {
		t.Fatalf("ServerBindAddress() = %q, want %q", got, "0.0.0.0")
	}
	if got := input.ServerPort(); got != 9527 {
		t.Fatalf("ServerPort() = %d, want %d", got, 9527)
	}
	if got := input.ServerEnableGzip(); !got {
		t.Fatal("ServerEnableGzip() expected true")
	}
	if got := input.ServerTLSCaFile(); got != "/cfg/ca.pem" {
		t.Fatalf("ServerTLSCaFile() = %q, want %q", got, "/cfg/ca.pem")
	}
	if got := input.ServerTLSServerName(); got != "cfg.local" {
		t.Fatalf("ServerTLSServerName() = %q, want %q", got, "cfg.local")
	}
	if got := input.ServerTLSCertFile(); got != "/cfg/cert.pem" {
		t.Fatalf("ServerTLSCertFile() = %q, want %q", got, "/cfg/cert.pem")
	}
	if got := input.ServerTLSKeyFile(); got != "/cfg/key.pem" {
		t.Fatalf("ServerTLSKeyFile() = %q, want %q", got, "/cfg/key.pem")
	}
	if got := input.ServerEnableGrpcWebProxy(); !got {
		t.Fatal("ServerEnableGrpcWebProxy() expected true")
	}
	if got := input.ServerHotReload(); !got {
		t.Fatal("ServerHotReload() expected true")
	}
	if got := input.ServerGrpcClientMaxCachedConns(); got != 55 {
		t.Fatalf("ServerGrpcClientMaxCachedConns() = %d, want %d", got, 55)
	}
	if got := input.ServerSecurityMissing(); got {
		t.Fatal("ServerSecurityMissing() expected false")
	}

	if got := input.DatabaseDialect(); got != "postgres" {
		t.Fatalf("DatabaseDialect() = %q, want %q", got, "postgres")
	}
	if got := input.DatabaseDSN(); got != "postgres://runtime-db/app" {
		t.Fatalf("DatabaseDSN() = %q, want %q", got, "postgres://runtime-db/app")
	}
	if got := input.DatabaseMaxOpenConns(); got != 21 {
		t.Fatalf("DatabaseMaxOpenConns() = %d, want %d", got, 21)
	}
	if got := input.DatabaseMaxIdleConns(); got != 12 {
		t.Fatalf("DatabaseMaxIdleConns() = %d, want %d", got, 12)
	}
	if got := input.DatabaseConnMaxLifetimeSeconds(); got != 345 {
		t.Fatalf("DatabaseConnMaxLifetimeSeconds() = %d, want %d", got, 345)
	}

	if got := input.AuthHttpAuth(); got == nil {
		t.Fatal("AuthHttpAuth() should not be nil")
	}
	if got := input.AuthConfig(); got == nil {
		t.Fatal("AuthConfig() should not be nil")
	}
	if got := input.AuthJobTokenAllowedSANs(); len(got) == 0 {
		t.Fatal("AuthJobTokenAllowedSANs() should not be empty")
	}
	if got := input.AuthGrpcEntryPolicy(); len(got) == 0 {
		t.Fatal("AuthGrpcEntryPolicy() should not be empty")
	}
	if got := input.ServerConfig(); got == nil {
		t.Fatal("ServerConfig() should not be nil")
	}
	if got := input.ServerCSPConfig(); got == nil {
		t.Fatal("ServerCSPConfig() should not be nil")
	}
	if got := input.ServerHSTSConfig(); got == nil {
		t.Fatal("ServerHSTSConfig() should not be nil")
	}
	if got := input.ServerCSRFConfig(); got == nil {
		t.Fatal("ServerCSRFConfig() should not be nil")
	}
	if got := input.TaskConfig(); got == nil {
		t.Fatal("TaskConfig() should not be nil")
	}
	if got := input.LogConfig(); got == nil {
		t.Fatal("LogConfig() should not be nil")
	}
	if got := input.FrontendEnv(); got == nil {
		t.Fatal("FrontendEnv() should not be nil")
	}
	if got := input.BackendEnv(); got == nil {
		t.Fatal("BackendEnv() should not be nil")
	}
	if got := input.CompileConfig(); got == nil {
		t.Fatal("CompileConfig() should not be nil")
	}

	compileCopy := input.CompileConfig()
	compileCopy.BundleMode = "mutated"
	if got := input.CompileConfig().BundleMode; got != "bundle" {
		t.Fatalf("CompileConfig leaked mutation, got %q", got)
	}
	frontendCopy := input.FrontendEnv()
	frontendCopy["APP_NAME"] = "changed"
	if got := input.FrontendEnv()["APP_NAME"]; got != "demo" {
		t.Fatalf("FrontendEnv leaked mutation, got %v", got)
	}
}

func TestRunScopeInputNilBranches(t *testing.T) {
	input := NewRunScopeInput(nil, Options{}, RunServerOptions{}, RunDBOptions{})

	if got := input.Environment(); got != "" {
		t.Fatalf("Environment() = %q, want empty", got)
	}
	if got := input.ModulesPath(); got != "" {
		t.Fatalf("ModulesPath() = %q, want empty", got)
	}
	if got := input.DistPath(); got != "" {
		t.Fatalf("DistPath() = %q, want empty", got)
	}
	if got := input.TmpPath(); got != "" {
		t.Fatalf("TmpPath() = %q, want empty", got)
	}
	if got := input.DefaultChoysumPath(); got != "" {
		t.Fatalf("DefaultChoysumPath() = %q, want empty", got)
	}
	if got := input.ConfigPath(); got != "" {
		t.Fatalf("ConfigPath() = %q, want empty", got)
	}
	if got := input.ESMUpstreamURL(); got != "" {
		t.Fatalf("ESMUpstreamURL() = %q, want empty", got)
	}
	if got := input.NpmRegistryURL(); got != "" {
		t.Fatalf("NpmRegistryURL() = %q, want empty", got)
	}
	if got := input.ModuleCatalogIndexURL(); got != "" {
		t.Fatalf("ModuleCatalogIndexURL() = %q, want empty", got)
	}
	if got := input.BootstrapModuleInstallTimeoutSeconds(); got != 0 {
		t.Fatalf("BootstrapModuleInstallTimeoutSeconds() = %d, want 0", got)
	}
	if got := input.CompileBundleMode(); got != "" {
		t.Fatalf("CompileBundleMode() = %q, want empty", got)
	}
	if got := input.CompileConfig(); got != nil {
		t.Fatalf("CompileConfig() = %#v, want nil", got)
	}
	if input.AuthEnabled() {
		t.Fatal("AuthEnabled() should be false")
	}
	if got := input.AuthConfig(); got != nil {
		t.Fatalf("AuthConfig() = %#v, want nil", got)
	}
	if got := input.AuthHttpAuth(); got != nil {
		t.Fatalf("AuthHttpAuth() = %#v, want nil", got)
	}
	if got := input.AuthGrpcAuthentication(); got {
		t.Fatal("AuthGrpcAuthentication() should be false")
	}
	if got := input.AuthInternalKey(); got != "" {
		t.Fatalf("AuthInternalKey() = %q, want empty", got)
	}
	if got := input.AuthJobTokenAllowedSANs(); got != nil {
		t.Fatalf("AuthJobTokenAllowedSANs() = %#v, want nil", got)
	}
	if got := input.AuthGrpcEntryPolicy(); got != nil {
		t.Fatalf("AuthGrpcEntryPolicy() = %#v, want nil", got)
	}
	if input.ServerEnabledTLS() {
		t.Fatal("ServerEnabledTLS() should be false")
	}
	if got := input.ServerConfig(); got != nil {
		t.Fatalf("ServerConfig() = %#v, want nil", got)
	}
	if got := input.ServerBindAddress(); got != "" {
		t.Fatalf("ServerBindAddress() = %q, want empty", got)
	}
	if got := input.ServerPort(); got != 0 {
		t.Fatalf("ServerPort() = %d, want 0", got)
	}
	if input.ServerEnableGzip() {
		t.Fatal("ServerEnableGzip() should be false")
	}
	if got := input.ServerTLSCaFile(); got != "" {
		t.Fatalf("ServerTLSCaFile() = %q, want empty", got)
	}
	if got := input.ServerTLSServerName(); got != "" {
		t.Fatalf("ServerTLSServerName() = %q, want empty", got)
	}
	if got := input.ServerTLSCertFile(); got != "" {
		t.Fatalf("ServerTLSCertFile() = %q, want empty", got)
	}
	if got := input.ServerTLSKeyFile(); got != "" {
		t.Fatalf("ServerTLSKeyFile() = %q, want empty", got)
	}
	if input.ServerEnableGrpcWebProxy() {
		t.Fatal("ServerEnableGrpcWebProxy() should be false")
	}
	if input.ServerHotReload() {
		t.Fatal("ServerHotReload() should be false")
	}
	if got := input.ServerGrpcClientMaxCachedConns(); got != 0 {
		t.Fatalf("ServerGrpcClientMaxCachedConns() = %d, want 0", got)
	}
	if got := input.ServerCSPConfig(); got != nil {
		t.Fatalf("ServerCSPConfig() = %#v, want nil", got)
	}
	if got := input.ServerHSTSConfig(); got != nil {
		t.Fatalf("ServerHSTSConfig() = %#v, want nil", got)
	}
	if got := input.ServerCSRFConfig(); got != nil {
		t.Fatalf("ServerCSRFConfig() = %#v, want nil", got)
	}
	if !input.ServerSecurityMissing() {
		t.Fatal("ServerSecurityMissing() should be true")
	}
	if got := input.TaskConfig(); got != nil {
		t.Fatalf("TaskConfig() = %#v, want nil", got)
	}
	if got := input.LogConfig(); got != nil {
		t.Fatalf("LogConfig() = %#v, want nil", got)
	}
	if got := input.FrontendEnv(); got != nil {
		t.Fatalf("FrontendEnv() = %#v, want nil", got)
	}
	if got := input.BackendEnv(); got != nil {
		t.Fatalf("BackendEnv() = %#v, want nil", got)
	}
	if got := input.DatabaseDialect(); got != "" {
		t.Fatalf("DatabaseDialect() = %q, want empty", got)
	}
	if got := input.DatabaseDSN(); got != "" {
		t.Fatalf("DatabaseDSN() = %q, want empty", got)
	}
	if got := input.DatabaseMaxOpenConns(); got != 0 {
		t.Fatalf("DatabaseMaxOpenConns() = %d, want 0", got)
	}
	if got := input.DatabaseMaxIdleConns(); got != 0 {
		t.Fatalf("DatabaseMaxIdleConns() = %d, want 0", got)
	}
	if got := input.DatabaseConnMaxLifetimeSeconds(); got != 0 {
		t.Fatalf("DatabaseConnMaxLifetimeSeconds() = %d, want 0", got)
	}
}

func TestRunScopeInputRemainingFallbackBranches(t *testing.T) {
	t.Run("new run server options defaults and overrides", func(t *testing.T) {
		defaults := config.NewDefaultServerConfig()

		got := NewRunServerOptions(&config.ServerConfig{BindAddress: " ", Port: 0, EnabledTLS: true})
		if got.BindAddress != defaults.BindAddress {
			t.Fatalf("NewRunServerOptions(default bind) = %q, want %q", got.BindAddress, defaults.BindAddress)
		}
		if got.Port != defaults.Port {
			t.Fatalf("NewRunServerOptions(default port) = %d, want %d", got.Port, defaults.Port)
		}
		if !got.EnabledTLS {
			t.Fatal("NewRunServerOptions() should keep explicit EnabledTLS=true")
		}

		overrides := NewRunServerOptions(&config.ServerConfig{BindAddress: "127.0.0.1", Port: 9001, EnabledTLS: false})
		if overrides.BindAddress != "127.0.0.1" || overrides.Port != 9001 {
			t.Fatalf("NewRunServerOptions(override) = %#v, want bind=127.0.0.1 port=9001", overrides)
		}
	})

	t.Run("run server options validate separate branches", func(t *testing.T) {
		if err := (RunServerOptions{BindAddress: "", Port: 1}).Validate(); err == nil {
			t.Fatal("RunServerOptions.Validate() should fail when bindAddress is empty")
		}
		if err := (RunServerOptions{BindAddress: "127.0.0.1", Port: 0}).Validate(); err == nil {
			t.Fatal("RunServerOptions.Validate() should fail when port is non-positive")
		}
	})

	t.Run("tmp path and database dialect fallback", func(t *testing.T) {
		input := NewRunScopeInput(
			&ScopeInputConfigOptions{TmpPath: "/cfg/tmp", Db: &config.DbConfig{Dialect: "sqlite"}},
			Options{},
			RunServerOptions{},
			RunDBOptions{},
		)

		if got := input.TmpPath(); got != "/cfg/tmp" {
			t.Fatalf("TmpPath() fallback = %q, want %q", got, "/cfg/tmp")
		}
		if got := input.DatabaseDialect(); got != "sqlite" {
			t.Fatalf("DatabaseDialect() fallback = %q, want %q", got, "sqlite")
		}
	})
}
