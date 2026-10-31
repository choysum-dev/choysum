// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestRunReadyForInstalledModulesSkipsWhenNoReadyHooks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "index.ts"), "export const x = 1\n")
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), nil, RunReadyOptions{
		Modules: []meta.Module{{Name: "noop", Path: root, ApplicationStr: "noop"}},
	})
	if err != nil {
		t.Fatalf("expected nil with nil executor when no ready hooks, got %v", err)
	}
}

func TestRunReadyForInstalledModulesFiltersByApplication(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), `
export class ReadyHooks {
  @HookReady()
  static async onReady(): Promise<void> {}
}
`)
	other := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(other, "service", "hook", "ready.ts"), `
export class ReadyHooks {
  @HookReady()
  static async onReady(): Promise<void> {}
}
`)
	// Nil executor: modules that declare ready would call NewRunner then RunPhase and fail
	// if selected. Filter to empty application set → no candidates → nil.
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), nil, RunReadyOptions{
		ApplicationNames: []string{"only-missing"},
		Modules: []meta.Module{
			{Name: "doc", Path: root, ApplicationStr: "document"},
			{Name: "other", Path: other, ApplicationStr: "meta"},
		},
	})
	if err != nil {
		t.Fatalf("expected no candidates, got %v", err)
	}
}
