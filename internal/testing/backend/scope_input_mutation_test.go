// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func backendScopeInputMutationFixtureConfig() *config.Config {
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

func TestTestRuntimeScopeInputMutationIsolation(t *testing.T) {
	input := newTestRuntimeScopeInput(snapshot.New(backendScopeInputMutationFixtureConfig()))

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

func TestNewTestRuntimeScopeInputFromScopeCopiesRuntimeAndDBOptions(t *testing.T) {
	runtimeScope := &testStubScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath:           "/workspace/modules",
			DistPath:              "/workspace/dist",
			TmpPath:               "/workspace/tmp",
			DefaultChoysumPath:    "/workspace/.choysum",
			ConfigPath:            "/workspace/config.yaml",
			NPMRegistryURL:        "https://registry.npmjs.org",
			ModuleCatalogIndexURL: "https://index.example.dev/v1/index.json",
			Server: &config.ServerConfig{
				Environment: "test",
				BindAddress: "127.0.0.1",
				Port:        8088,
			},
			Auth: &config.AuthConfig{Enabled: true, InternalKey: "k"},
		},
	}

	input := newTestRuntimeScopeInputFromScope(runtimeScope, scope.DatabaseRuntimeOptions{
		Dialect:                "sqlite",
		DSN:                    "file:test.db",
		MaxOpenConns:           8,
		MaxIdleConns:           4,
		ConnMaxLifetimeSeconds: 120,
	})

	if got := input.ModulesPath(); got != "/workspace/modules" {
		t.Fatalf("ModulesPath() = %q, want /workspace/modules", got)
	}
	if got := input.DistPath(); got != "/workspace/dist" {
		t.Fatalf("DistPath() = %q, want /workspace/dist", got)
	}
	if got := input.TmpPath(); got != "/workspace/tmp" {
		t.Fatalf("TmpPath() = %q, want /workspace/tmp", got)
	}
	if got := input.DefaultChoysumPath(); got != "/workspace/.choysum" {
		t.Fatalf("DefaultChoysumPath() = %q, want /workspace/.choysum", got)
	}
	if got := input.ConfigPath(); got != "/workspace/config.yaml" {
		t.Fatalf("ConfigPath() = %q, want /workspace/config.yaml", got)
	}
	if got := input.NpmRegistryURL(); got != "https://registry.npmjs.org" {
		t.Fatalf("NpmRegistryURL() = %q, want https://registry.npmjs.org", got)
	}
	if got := input.ModuleCatalogIndexURL(); got != "https://index.example.dev/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() = %q, want https://index.example.dev/v1/index.json", got)
	}
	if got := input.Environment(); got != "test" {
		t.Fatalf("Environment() = %q, want test", got)
	}
	if !input.AuthEnabled() {
		t.Fatal("AuthEnabled() = false, want true")
	}
	if got := input.AuthInternalKey(); got != "k" {
		t.Fatalf("AuthInternalKey() = %q, want k", got)
	}

	compileCfg := input.CompileConfig()
	if compileCfg == nil {
		t.Fatal("CompileConfig() = nil, want default compile config")
	}
	if !compileCfg.SourceMap {
		t.Fatal("CompileConfig().SourceMap = false, want true")
	}
	if compileCfg.Minify {
		t.Fatal("CompileConfig().Minify = true, want false")
	}

	if got := input.DatabaseDialect(); got != "sqlite" {
		t.Fatalf("DatabaseDialect() = %q, want sqlite", got)
	}
	if got := input.DatabaseDSN(); got != "file:test.db" {
		t.Fatalf("DatabaseDSN() = %q, want file:test.db", got)
	}
	if got := input.DatabaseMaxOpenConns(); got != 8 {
		t.Fatalf("DatabaseMaxOpenConns() = %d, want 8", got)
	}
	if got := input.DatabaseMaxIdleConns(); got != 4 {
		t.Fatalf("DatabaseMaxIdleConns() = %d, want 4", got)
	}
	if got := input.DatabaseConnMaxLifetimeSeconds(); got != 120 {
		t.Fatalf("DatabaseConnMaxLifetimeSeconds() = %d, want 120", got)
	}
}

func TestNewTestRuntimeScopeInputFromScopeNilScope(t *testing.T) {
	input := newTestRuntimeScopeInputFromScope(nil, scope.DatabaseRuntimeOptions{})
	if got := input.ModulesPath(); got != "" {
		t.Fatalf("ModulesPath() = %q, want empty", got)
	}
	if got := input.DatabaseDialect(); got != "" {
		t.Fatalf("DatabaseDialect() = %q, want empty", got)
	}
	if got := input.ModuleCatalogIndexURL(); got != "" {
		t.Fatalf("ModuleCatalogIndexURL() = %q, want empty", got)
	}
	if input.CompileConfig() != nil {
		t.Fatalf("CompileConfig() = %#v, want nil", input.CompileConfig())
	}
}

func TestTestRuntimeScopeInputModulesPathFallsBackToSnapshotConfig(t *testing.T) {
	input := testRuntimeScopeInput{cfg: snapshot.New(&config.Config{ModulesPath: "/snapshot/modules"})}
	if got := input.ModulesPath(); got != "/snapshot/modules" {
		t.Fatalf("ModulesPath() = %q, want /snapshot/modules", got)
	}
}
