// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package snapshot

import "testing"

func TestConfigSnapshotCopyGettersReturnIsolatedCopies(t *testing.T) {
	snapshot := New(snapshotFixtureConfig())
	if snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}

	authCopy := snapshot.CopyAuthConfig()
	authCopy.JobTokenAllowedSANs[0] = "mutated-san"
	authCopy.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0] = "write"
	if got := snapshot.Auth.JobTokenAllowedSANs[0]; got != "task.choysum.internal" {
		t.Fatalf("snapshot JobTokenAllowedSANs mutated: got %q", got)
	}
	if got := snapshot.Auth.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0]; got != "read" {
		t.Fatalf("snapshot auth grpcEntryPolicy mutated via getter: got %q", got)
	}

	serverCopy := snapshot.CopyServerConfig()
	serverCopy.Security.CSP.Development.ScriptSrc[0] = "'unsafe-inline'"
	if got := snapshot.Server.Security.CSP.Development.ScriptSrc[0]; got != "'self'" {
		t.Fatalf("snapshot CSP mutated via getter: got %q", got)
	}

	taskCopy := snapshot.CopyTaskConfig()
	taskCopy.Retention.TaskJob.Overrides["auth.User"].SucceededDays = 999
	if got := snapshot.Task.Retention.TaskJob.Overrides["auth.User"].SucceededDays; got != 1 {
		t.Fatalf("snapshot task retention override mutated via getter: got %d", got)
	}

	frontendEnvCopy := snapshot.CopyFrontendEnv()
	frontendEnvCopy["APP_NAME"] = "mutated"
	frontendEnvCopy["NESTED"].(map[string]any)["k"] = "mutated"
	if got := snapshot.FrontendEnv["APP_NAME"]; got != "demo" {
		t.Fatalf("snapshot frontend env mutated via getter: got %v", got)
	}
	if got := snapshot.FrontendEnv["NESTED"].(map[string]any)["k"]; got != "v" {
		t.Fatalf("snapshot frontend nested env mutated via getter: got %v", got)
	}

	backendEnvCopy := snapshot.CopyBackendEnv()
	backendEnvCopy["NESTED_LIST"].([]any)[0].(map[string]any)["k"] = "mutated"
	if got := snapshot.BackendEnv["NESTED_LIST"].([]any)[0].(map[string]any)["k"]; got != "v" {
		t.Fatalf("snapshot backend nested env mutated via getter: got %v", got)
	}
}
