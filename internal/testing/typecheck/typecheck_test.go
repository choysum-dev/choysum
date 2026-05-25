// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypecheckApp_MissingVueTsc(t *testing.T) {
	ctx := context.Background()

	repoRoot := t.TempDir()
	addonsPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(addonsPath, "auth", "service"), 0o755); err != nil {
		t.Fatalf("mkdir auth service: %v", err)
	}
	if err := os.WriteFile(filepath.Join(addonsPath, "auth", "service", "index.ts"), []byte("export const auth = 1\n"), 0o644); err != nil {
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

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	opts := RunOptions{
		AddonsPath: addonsPath,
		RepoRoot:   repoRoot,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}

	err := TypecheckApp(ctx, opts, "auth")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "vue-tsc") {
		t.Fatalf("expected error to mention vue-tsc, got: %s", msg)
	}
	if !strings.Contains(msg, "npm install") {
		t.Fatalf("expected error to mention npm install, got: %s", msg)
	}
}
