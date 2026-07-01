// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

func scopeInputFixtureConfig() *config.Config {
	return &config.Config{
		Compile: &config.CompileConfig{BundleMode: "bundle"},
		Auth: &config.AuthConfig{
			Enabled:             true,
			JobTokenAllowedSANs: []string{"task.choysum.internal"},
			HttpAuth:            &config.HttpAuthConfig{Enabled: true, ExcludedPaths: []string{"/health"}},
			GrpcEntryPolicy: map[string]*config.EntryMethodConfig{
				"auth.User/Login": {
					RecordRuleAllow: []config.EntryRecordRuleAllow{{Model: "auth.User", Ops: []string{"read"}}},
				},
			},
		},
		Server: &config.ServerConfig{
			Security: &config.SecurityConfig{
				CSP: &config.CSPConfig{
					Development: config.CSPDirectives{ScriptSrc: []string{"'self'"}},
				},
			},
		},
		Task: &config.TaskConfig{
			Retention: &config.TaskRetentionConfig{
				TaskJob: &config.TaskRetentionEntry{
					Overrides: map[string]*config.TaskRetentionPolicy{
						"auth.User": {SucceededDays: 1, FailedDays: 2, CancelledDays: 3},
					},
				},
			},
		},
		FrontendEnv: map[string]any{
			"APP_NAME": "demo",
			"NESTED":   map[string]any{"k": "v"},
		},
		BackendEnv: map[string]any{
			"NESTED_LIST": []any{map[string]any{"k": "v"}},
		},
	}
}

func TestCommandScopeInputPathPriorityAndNilOptions(t *testing.T) {
	input := NewCommandScopeInput(
		&ScopeInputConfigOptions{
			ModulesPath:                          "/options/modules",
			DistPath:                             "/options/dist",
			TmpPath:                              "/options/tmp",
			DefaultChoysumPath:                   "/options/default",
			ConfigPath:                           "/options/config.yaml",
			NPMRegistryURL:                       "https://registry.example.com",
			ModuleCatalogIndexURL:                "https://index.example.com/v1/index.json",
			BootstrapModuleInstallTimeoutSeconds: 333,
			Server:                               &config.ServerConfig{Environment: "production"},
		},
		Options{
			ModulesPath:           "/runtime/modules",
			TmpPath:               "/runtime/tmp",
			DefaultChoysumPath:    "/runtime/default",
			ModuleCatalogIndexURL: "https://index.runtime.example.com/v1/index.json",
		},
	)

	if got := input.Environment(); got != "production" {
		t.Fatalf("Environment() = %q, want %q", got, "production")
	}
	if got := input.ModulesPath(); got != "/runtime/modules" {
		t.Fatalf("ModulesPath() runtime override = %q, want %q", got, "/runtime/modules")
	}
	if got := input.TmpPath(); got != "/runtime/tmp" {
		t.Fatalf("TmpPath() runtime override = %q, want %q", got, "/runtime/tmp")
	}
	if got := input.DistPath(); got != "/options/dist" {
		t.Fatalf("DistPath() = %q, want %q", got, "/options/dist")
	}
	if got := input.DefaultChoysumPath(); got != "/runtime/default" {
		t.Fatalf("DefaultChoysumPath() runtime override = %q, want %q", got, "/runtime/default")
	}
	if got := input.ConfigPath(); got != "/options/config.yaml" {
		t.Fatalf("ConfigPath() = %q, want %q", got, "/options/config.yaml")
	}
	if got := input.NpmRegistryURL(); got != "https://registry.example.com" {
		t.Fatalf("NpmRegistryURL() = %q, want %q", got, "https://registry.example.com")
	}
	if got := input.ModuleCatalogIndexURL(); got != "https://index.runtime.example.com/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() runtime override = %q, want %q", got, "https://index.runtime.example.com/v1/index.json")
	}
	if got := input.BootstrapModuleInstallTimeoutSeconds(); got != 333 {
		t.Fatalf("BootstrapModuleInstallTimeoutSeconds() = %d, want %d", got, 333)
	}

	fallback := NewCommandScopeInput(
		&ScopeInputConfigOptions{ModulesPath: "/fallback/modules", TmpPath: "/fallback/tmp", DefaultChoysumPath: "/fallback/default", ModuleCatalogIndexURL: "https://index.fallback.example/v1/index.json"},
		Options{},
	)
	if got := fallback.ModulesPath(); got != "/fallback/modules" {
		t.Fatalf("ModulesPath() fallback = %q, want %q", got, "/fallback/modules")
	}
	if got := fallback.TmpPath(); got != "/fallback/tmp" {
		t.Fatalf("TmpPath() fallback = %q, want %q", got, "/fallback/tmp")
	}
	if got := fallback.DefaultChoysumPath(); got != "/fallback/default" {
		t.Fatalf("DefaultChoysumPath() fallback = %q, want %q", got, "/fallback/default")
	}
	if got := fallback.ModuleCatalogIndexURL(); got != "https://index.fallback.example/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() fallback = %q, want %q", got, "https://index.fallback.example/v1/index.json")
	}

	nilOptionsInput := NewCommandScopeInput(nil, Options{ModulesPath: "/runtime/modules", TmpPath: "/runtime/tmp"})
	if got := nilOptionsInput.ModulesPath(); got != "/runtime/modules" {
		t.Fatalf("ModulesPath() with nil options = %q, want %q", got, "/runtime/modules")
	}
	if got := nilOptionsInput.TmpPath(); got != "/runtime/tmp" {
		t.Fatalf("TmpPath() with nil options = %q, want %q", got, "/runtime/tmp")
	}
	if got := nilOptionsInput.Environment(); got != "" {
		t.Fatalf("Environment() with nil options = %q, want empty", got)
	}
	if got := nilOptionsInput.DistPath(); got != "" {
		t.Fatalf("DistPath() with nil options = %q, want empty", got)
	}
	if got := nilOptionsInput.NpmRegistryURL(); got != "" {
		t.Fatalf("NpmRegistryURL() with nil options = %q, want empty", got)
	}
	if got := nilOptionsInput.ModuleCatalogIndexURL(); got != "" {
		t.Fatalf("ModuleCatalogIndexURL() with nil options = %q, want empty", got)
	}
	if got := nilOptionsInput.BootstrapModuleInstallTimeoutSeconds(); got != 0 {
		t.Fatalf("BootstrapModuleInstallTimeoutSeconds() with nil options = %d, want 0", got)
	}
}

func TestRunScopeInputPathFallbackAndRegistryURL(t *testing.T) {
	input := NewRunScopeInput(
		&ScopeInputConfigOptions{
			ModulesPath:                          "/options/modules",
			TmpPath:                              "/options/tmp",
			DefaultChoysumPath:                   "/options/default",
			NPMRegistryURL:                       "https://registry.options.example",
			ModuleCatalogIndexURL:                "https://index.options.example/v1/index.json",
			BootstrapModuleInstallTimeoutSeconds: 444,
			Server:                               &config.ServerConfig{BindAddress: "127.0.0.1", Port: 8080},
			Db:                                   &config.DbConfig{Dialect: "sqlite", DSN: "/options/db.sqlite"},
		},
		Options{DefaultChoysumPath: "/runtime/default", ModuleCatalogIndexURL: "https://index.runtime.example/v1/index.json"},
		RunServerOptions{BindAddress: "0.0.0.0", Port: 9000},
		RunDBOptions{Dialect: "mysql", DSN: "mysql://runtime"},
	)

	if got := input.ModulesPath(); got != "/options/modules" {
		t.Fatalf("ModulesPath() fallback = %q, want %q", got, "/options/modules")
	}
	if got := input.DefaultChoysumPath(); got != "/runtime/default" {
		t.Fatalf("DefaultChoysumPath() runtime override = %q, want %q", got, "/runtime/default")
	}
	if got := input.NpmRegistryURL(); got != "https://registry.options.example" {
		t.Fatalf("NpmRegistryURL() fallback = %q, want %q", got, "https://registry.options.example")
	}
	if got := input.ModuleCatalogIndexURL(); got != "https://index.runtime.example/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() runtime override = %q, want %q", got, "https://index.runtime.example/v1/index.json")
	}
	if got := input.BootstrapModuleInstallTimeoutSeconds(); got != 444 {
		t.Fatalf("BootstrapModuleInstallTimeoutSeconds() fallback = %d, want %d", got, 444)
	}
	if got := input.ServerBindAddress(); got != "0.0.0.0" {
		t.Fatalf("ServerBindAddress() resolved options = %q, want %q", got, "0.0.0.0")
	}
	if got := input.ServerPort(); got != 9000 {
		t.Fatalf("ServerPort() resolved options = %d, want %d", got, 9000)
	}
	if got := input.DatabaseDialect(); got != "mysql" {
		t.Fatalf("DatabaseDialect() resolved options = %q, want %q", got, "mysql")
	}
	if got := input.DatabaseDSN(); got != "mysql://runtime" {
		t.Fatalf("DatabaseDSN() resolved options = %q, want %q", got, "mysql://runtime")
	}

	nilOptions := NewRunScopeInput(nil, Options{}, RunServerOptions{}, RunDBOptions{})
	if got := nilOptions.ModulesPath(); got != "" {
		t.Fatalf("ModulesPath() with nil options = %q, want empty", got)
	}
	if got := nilOptions.NpmRegistryURL(); got != "" {
		t.Fatalf("NpmRegistryURL() with nil options = %q, want empty", got)
	}
	if got := nilOptions.ModuleCatalogIndexURL(); got != "" {
		t.Fatalf("ModuleCatalogIndexURL() with nil options = %q, want empty", got)
	}
	if got := nilOptions.BootstrapModuleInstallTimeoutSeconds(); got != 0 {
		t.Fatalf("BootstrapModuleInstallTimeoutSeconds() with nil options = %d, want 0", got)
	}
}

func TestRunScopeInputResolvedGetterFallbacks(t *testing.T) {
	input := NewRunScopeInput(
		&ScopeInputConfigOptions{
			DefaultChoysumPath:    "/options/default",
			ModuleCatalogIndexURL: "https://index.options.example/v1/index.json",
			Server:                &config.ServerConfig{BindAddress: "127.0.0.1", Port: 8081},
			Db:                    &config.DbConfig{Dialect: "sqlite", DSN: "/options/fallback.sqlite"},
		},
		Options{},
		RunServerOptions{},
		RunDBOptions{},
	)

	if got := input.DefaultChoysumPath(); got != "/options/default" {
		t.Fatalf("DefaultChoysumPath() fallback = %q, want %q", got, "/options/default")
	}
	if got := input.ModuleCatalogIndexURL(); got != "https://index.options.example/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() fallback = %q, want %q", got, "https://index.options.example/v1/index.json")
	}
	if got := input.ServerBindAddress(); got != "127.0.0.1" {
		t.Fatalf("ServerBindAddress() fallback = %q, want %q", got, "127.0.0.1")
	}
	if got := input.ServerPort(); got != 8081 {
		t.Fatalf("ServerPort() fallback = %d, want %d", got, 8081)
	}
	if got := input.DatabaseDSN(); got != "/options/fallback.sqlite" {
		t.Fatalf("DatabaseDSN() fallback = %q, want %q", got, "/options/fallback.sqlite")
	}
}

func TestRunScopeOptionsValidation(t *testing.T) {
	t.Run("run server options defaults and validate", func(t *testing.T) {
		opts := NewRunServerOptions(nil)
		if opts.BindAddress == "" || opts.Port <= 0 {
			t.Fatalf("NewRunServerOptions(nil) = %#v, expected default bindAddress and positive port", opts)
		}
		if err := opts.Validate(); err != nil {
			t.Fatalf("RunServerOptions.Validate() error = %v", err)
		}
	})

	t.Run("run server options invalid", func(t *testing.T) {
		if err := (RunServerOptions{}).Validate(); err == nil {
			t.Fatal("RunServerOptions.Validate() expected error for empty bindAddress/port")
		}
	})

	t.Run("run db options validate", func(t *testing.T) {
		if err := (RunDBOptions{Dialect: "sqlite"}).Validate(); err != nil {
			t.Fatalf("RunDBOptions.Validate() error = %v", err)
		}
		if err := (RunDBOptions{}).Validate(); err == nil {
			t.Fatal("RunDBOptions.Validate() expected error for empty dialect")
		}
	})
}

func TestCommandScopeInputMutationIsolation(t *testing.T) {
	input := NewCommandScopeInput(NewScopeInputConfigOptions(snapshot.New(scopeInputFixtureConfig())), Options{})

	compileCopy := input.CompileConfig()
	compileCopy.BundleMode = "application"
	if got := input.CompileConfig().BundleMode; got != "bundle" {
		t.Fatalf("CompileConfig leaked mutation, got %q", got)
	}

	authCopy := input.AuthConfig()
	authCopy.JobTokenAllowedSANs[0] = "mutated"
	authCopy.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0] = "write"
	if got := input.AuthConfig().JobTokenAllowedSANs[0]; got != "task.choysum.internal" {
		t.Fatalf("AuthConfig leaked SAN mutation, got %q", got)
	}
	if got := input.AuthConfig().GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0]; got != "read" {
		t.Fatalf("AuthConfig leaked grpcEntryPolicy mutation, got %q", got)
	}

	cspCopy := input.ServerCSPConfig()
	cspCopy.Development.ScriptSrc[0] = "'unsafe-inline'"
	if got := input.ServerCSPConfig().Development.ScriptSrc[0]; got != "'self'" {
		t.Fatalf("ServerCSPConfig leaked mutation, got %q", got)
	}

	taskCopy := input.TaskConfig()
	taskCopy.Retention.TaskJob.Overrides["auth.User"].SucceededDays = 999
	if got := input.TaskConfig().Retention.TaskJob.Overrides["auth.User"].SucceededDays; got != 1 {
		t.Fatalf("TaskConfig leaked overrides mutation, got %d", got)
	}

	frontendCopy := input.FrontendEnv()
	frontendCopy["APP_NAME"] = "mutated"
	frontendCopy["NESTED"].(map[string]any)["k"] = "mutated"
	if got := input.FrontendEnv()["APP_NAME"]; got != "demo" {
		t.Fatalf("FrontendEnv leaked top-level mutation, got %v", got)
	}
	if got := input.FrontendEnv()["NESTED"].(map[string]any)["k"]; got != "v" {
		t.Fatalf("FrontendEnv leaked nested mutation, got %v", got)
	}

	backendCopy := input.BackendEnv()
	backendCopy["NESTED_LIST"].([]any)[0].(map[string]any)["k"] = "mutated"
	if got := input.BackendEnv()["NESTED_LIST"].([]any)[0].(map[string]any)["k"]; got != "v" {
		t.Fatalf("BackendEnv leaked nested list/map mutation, got %v", got)
	}
}
