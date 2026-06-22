// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scopetest

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func configSnapshotFixture() *config.Config {
	return &config.Config{
		ModulesPath:           "/tmp/modules",
		DistPath:              "/tmp/dist",
		TmpPath:               "/tmp/tmp",
		DefaultChoysumPath:    "/tmp/.choysum",
		ConfigPath:            "/tmp/config.yaml",
		ModuleCatalogIndexURL: "https://index.example.dev/v1/index.json",
		Compile:               &config.CompileConfig{BundleMode: "bundle"},
		Auth: &config.AuthConfig{
			GrpcEntryPolicy: map[string]*config.EntryMethodConfig{
				"auth.User/Login": {RecordRuleAllow: []config.EntryRecordRuleAllow{{Model: "auth.User", Ops: []string{"read"}}}},
			},
		},
		Server: &config.ServerConfig{
			Security: &config.SecurityConfig{
				CSP: &config.CSPConfig{Development: config.CSPDirectives{ScriptSrc: []string{"'self'"}}},
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
		FrontendEnv: map[string]any{"NESTED": map[string]any{"k": "v"}},
		BackendEnv:  map[string]any{"NESTED_LIST": []any{map[string]any{"k": "v"}}},
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: "file:/tmp/app.db?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL"},
	}
}

func TestConfigFromSnapshotReturnsDeepCopy(t *testing.T) {
	snap := snapshot.New(configSnapshotFixture())
	cfgCopy := ConfigFromSnapshot(snap)
	if cfgCopy == nil {
		t.Fatal("expected non-nil config copy")
	}

	cfgCopy.Auth.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0] = "write"
	cfgCopy.Server.Security.CSP.Development.ScriptSrc[0] = "'unsafe-inline'"
	cfgCopy.Task.Retention.TaskJob.Overrides["auth.User"].SucceededDays = 777
	cfgCopy.FrontendEnv["NESTED"].(map[string]any)["k"] = "changed"
	cfgCopy.BackendEnv["NESTED_LIST"].([]any)[0].(map[string]any)["k"] = "changed"
	cfgCopy.Db.DSN = "/tmp/changed.db"

	if got := snap.Auth.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0]; got != "read" {
		t.Fatalf("snapshot auth grpcEntryPolicy mutated via ConfigFromSnapshot: got %q", got)
	}
	if got := snap.Server.Security.CSP.Development.ScriptSrc[0]; got != "'self'" {
		t.Fatalf("snapshot CSP mutated via ConfigFromSnapshot: got %q", got)
	}
	if got := snap.Task.Retention.TaskJob.Overrides["auth.User"].SucceededDays; got != 1 {
		t.Fatalf("snapshot task retention override mutated via ConfigFromSnapshot: got %d", got)
	}
	if got := snap.FrontendEnv["NESTED"].(map[string]any)["k"]; got != "v" {
		t.Fatalf("snapshot frontend nested env mutated via ConfigFromSnapshot: got %v", got)
	}
	if got := snap.BackendEnv["NESTED_LIST"].([]any)[0].(map[string]any)["k"]; got != "v" {
		t.Fatalf("snapshot backend nested env mutated via ConfigFromSnapshot: got %v", got)
	}
	if got := snap.Db.DSN; got != "file:/tmp/app.db?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL" {
		t.Fatalf("snapshot db config mutated via ConfigFromSnapshot: got %q", got)
	}
}

func TestFactoryInputFromConfigExposesModulesPath(t *testing.T) {
	if got := FactoryInputFromConfig(nil); got != nil {
		t.Fatalf("FactoryInputFromConfig(nil) = %#v, want nil", got)
	}

	input := FactoryInputFromConfig(&config.Config{ModulesPath: "/workspace/modules", ModuleCatalogIndexURL: "https://index.example.dev/v1/index.json"})
	if input == nil {
		t.Fatal("FactoryInputFromConfig() returned nil input")
	}

	paths, ok := scope.PathsRuntimeOptionsFromInput(input)
	if !ok {
		t.Fatal("expected PathsRuntimeOptionsFromInput() to succeed")
	}
	if paths.ModulesPath != "/workspace/modules" {
		t.Fatalf("PathsRuntimeOptionsFromInput().ModulesPath = %q, want /workspace/modules", paths.ModulesPath)
	}
	if paths.ModuleCatalogIndexURL != "https://index.example.dev/v1/index.json" {
		t.Fatalf("PathsRuntimeOptionsFromInput().ModuleCatalogIndexURL = %q, want https://index.example.dev/v1/index.json", paths.ModuleCatalogIndexURL)
	}

	if got := (configFactoryInput{}).ModuleCatalogIndexURL(); got != "" {
		t.Fatalf("configFactoryInput{}.ModuleCatalogIndexURL() = %q, want empty", got)
	}
}
