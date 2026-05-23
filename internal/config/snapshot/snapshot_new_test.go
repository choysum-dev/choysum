// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package snapshot

import "testing"

func TestConfigSnapshotNewDeepCopiesNestedStructures(t *testing.T) {
	source := snapshotFixtureConfig()
	snapshot := New(source)
	if snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}

	source.Auth.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0] = "write"
	source.Server.Security.CSP.Development.ScriptSrc[0] = "'unsafe-inline'"
	source.Task.Retention.TaskJob.Overrides["auth.User"].SucceededDays = 365
	source.FrontendEnv["NESTED"].(map[string]any)["k"] = "changed"
	source.BackendEnv["NESTED_LIST"].([]any)[0].(map[string]any)["k"] = "changed"

	if got := snapshot.Auth.GrpcEntryPolicy["auth.User/Login"].RecordRuleAllow[0].Ops[0]; got != "read" {
		t.Fatalf("snapshot auth grpcEntryPolicy mutated: got %q", got)
	}
	if got := snapshot.Server.Security.CSP.Development.ScriptSrc[0]; got != "'self'" {
		t.Fatalf("snapshot CSP development directive mutated: got %q", got)
	}
	if got := snapshot.Task.Retention.TaskJob.Overrides["auth.User"].SucceededDays; got != 1 {
		t.Fatalf("snapshot task retention override mutated: got %d", got)
	}
	if got := snapshot.FrontendEnv["NESTED"].(map[string]any)["k"]; got != "v" {
		t.Fatalf("snapshot frontend env nested map mutated: got %v", got)
	}
	if got := snapshot.BackendEnv["NESTED_LIST"].([]any)[0].(map[string]any)["k"]; got != "v" {
		t.Fatalf("snapshot backend env nested list/map mutated: got %v", got)
	}
}
