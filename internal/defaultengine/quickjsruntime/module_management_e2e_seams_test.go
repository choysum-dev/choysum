// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"fmt"
	"testing"
)

type backendEnvStringer string

func (s backendEnvStringer) String() string { return string(s) }

func TestModuleManagementE2ESeamsFromBackendEnv(t *testing.T) {
	t.Parallel()

	if got := ModuleManagementE2ESeamsFromBackendEnv(nil); len(got) != 0 {
		t.Fatalf("nil env opts = %d, want 0", len(got))
	}
	if got := ModuleManagementE2ESeamsFromBackendEnv(map[string]any{}); len(got) != 0 {
		t.Fatalf("empty env opts = %d, want 0", len(got))
	}
	if got := ModuleManagementE2ESeamsFromBackendEnv(map[string]any{
		e2eForceLockConflictEnv: "true",
		e2eForceResultStatusEnv: "FAILED",
		e2eForceReloadFailedEnv: "true",
	}); len(got) != 0 {
		t.Fatalf("FORCE_* without harness marker opts = %d, want 0", len(got))
	}

	lockOnly := ModuleManagementE2ESeamsFromBackendEnv(map[string]any{
		e2eHarnessMarkerEnv:     "true",
		e2eForceLockConflictEnv: "true",
	})
	if len(lockOnly) != 1 {
		t.Fatalf("lock-conflict opts = %d, want 1", len(lockOnly))
	}
	cfg := resolveModuleManagementConfig(lockOnly...)
	if cfg.applyHook == nil || cfg.runHook != nil || cfg.forceReloadFailed {
		t.Fatalf("lock-conflict cfg = %#v", cfg)
	}

	resultFailed := ModuleManagementE2ESeamsFromBackendEnv(map[string]any{
		e2eHarnessMarkerEnv:     true,
		e2eForceResultStatusEnv: "FAILED",
	})
	if len(resultFailed) != 1 {
		t.Fatalf("result-failed opts = %d, want 1", len(resultFailed))
	}
	cfg = resolveModuleManagementConfig(resultFailed...)
	if cfg.runHook == nil || cfg.applyHook != nil || cfg.forceReloadFailed {
		t.Fatalf("result-failed cfg = %#v", cfg)
	}

	reloadFailed := ModuleManagementE2ESeamsFromBackendEnv(map[string]any{
		e2eHarnessMarkerEnv:     "yes",
		e2eForceReloadFailedEnv: 1,
	})
	if len(reloadFailed) != 1 {
		t.Fatalf("reload-failed opts = %d, want 1", len(reloadFailed))
	}
	cfg = resolveModuleManagementConfig(reloadFailed...)
	if !cfg.forceReloadFailed || cfg.applyHook != nil || cfg.runHook != nil {
		t.Fatalf("reload-failed cfg = %#v", cfg)
	}

	all := ModuleManagementE2ESeamsFromBackendEnv(map[string]any{
		e2eHarnessMarkerEnv:     "on",
		e2eForceLockConflictEnv: "yes",
		e2eForceResultStatusEnv: "failed",
		e2eForceReloadFailedEnv: "on",
	})
	if len(all) != 3 {
		t.Fatalf("combined opts = %d, want 3", len(all))
	}
}

func TestBackendEnvTextAndTruthy(t *testing.T) {
	t.Parallel()

	if got := backendEnvText("x"); got != "x" {
		t.Fatalf("string = %q", got)
	}
	if got := backendEnvText(backendEnvStringer("s")); got != "s" {
		t.Fatalf("stringer = %q", got)
	}
	if got := backendEnvText(true); got != "true" {
		t.Fatalf("bool true = %q", got)
	}
	if got := backendEnvText(false); got != "false" {
		t.Fatalf("bool false = %q", got)
	}
	if got := backendEnvText(nil); got != "" {
		t.Fatalf("nil = %q", got)
	}
	if got := backendEnvText(42); got != "42" {
		t.Fatalf("default = %q", got)
	}
	if !isTruthyBackendEnv("TRUE") || !isTruthyBackendEnv(1) || !isTruthyBackendEnv(backendEnvStringer("on")) {
		t.Fatal("expected truthy values")
	}
	if isTruthyBackendEnv("") || isTruthyBackendEnv(false) || isTruthyBackendEnv(fmt.Sprint(0)) || isTruthyBackendEnv(nil) {
		t.Fatal("expected falsy values")
	}
}
