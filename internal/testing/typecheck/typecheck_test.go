// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTypecheckApp_AllowsNpxWithoutGlobalVueTsc(t *testing.T) {
	ctx := context.Background()

	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(modulesPath, "auth", "service"), 0o755); err != nil {
		t.Fatalf("mkdir auth service: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "auth", "service", "index.ts"), []byte("export const auth = 1\n"), 0o644); err != nil {
		t.Fatalf("write auth service ts: %v", err)
	}

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	npxPath := filepath.Join(binDir, "npx")
	if err := os.WriteFile(npxPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake npx: %v", err)
	}

	t.Setenv("PATH", binDir)
	if err := os.MkdirAll(filepath.Join(repoRoot, "node_modules", "vue-tsc"), 0o755); err != nil {
		t.Fatalf("mkdir local vue-tsc module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write local vue-tsc package.json: %v", err)
	}

	opts := RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}

	err := TypecheckApp(ctx, opts, "auth")
	if err != nil {
		t.Fatalf("expected success with npx-resolved vue-tsc, got: %v", err)
	}
}
