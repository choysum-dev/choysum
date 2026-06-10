// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

func e2eScopeInputMutationFixtureConfig() *config.Config {
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

func TestRuntimeScopeInputMutationIsolation(t *testing.T) {
	input := newRuntimeScopeInput(snapshot.New(e2eScopeInputMutationFixtureConfig()), e2eRuntimeOptions{})

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

func TestRuntimeScopeInputModuleCatalogIndexURLBranches(t *testing.T) {
	input := newRuntimeScopeInput(snapshot.New(&config.Config{ModuleCatalogIndexURL: "https://index.example.dev/v1/index.json"}), e2eRuntimeOptions{})
	if got := input.ModuleCatalogIndexURL(); got != "https://index.example.dev/v1/index.json" {
		t.Fatalf("ModuleCatalogIndexURL() = %q, want https://index.example.dev/v1/index.json", got)
	}

	nilOptionsInput := newRuntimeScopeInput(nil, e2eRuntimeOptions{})
	if got := nilOptionsInput.ModuleCatalogIndexURL(); got != "" {
		t.Fatalf("ModuleCatalogIndexURL() with nil options = %q, want empty", got)
	}
}
