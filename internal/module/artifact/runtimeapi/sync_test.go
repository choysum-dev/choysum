// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtimeapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncMissingProtosRestoresFromGenerated(t *testing.T) {
	root := t.TempDir()
	distRoot := filepath.Join(root, "dist")
	generated := filepath.Join(root, "generated", "proto")
	_ = os.MkdirAll(distRoot, 0o755)

	for _, app := range []string{"auth", "base", "task"} {
		dir := filepath.Join(generated, app)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir generated: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, app+".proto"), []byte("body-"+app), 0o644); err != nil {
			t.Fatalf("write generated: %v", err)
		}
	}
	// base already present at runtime
	baseDst := filepath.Join(root, "api", "base", "proto")
	if err := os.MkdirAll(baseDst, 0o755); err != nil {
		t.Fatalf("mkdir base runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(baseDst, "base.proto"), []byte("keep-base"), 0o644); err != nil {
		t.Fatalf("write base runtime: %v", err)
	}

	synced, err := SyncMissingProtos(distRoot, []string{"auth", "base", "task", "web", ""})
	if err != nil {
		t.Fatalf("SyncMissingProtos: %v", err)
	}
	if len(synced) != 2 {
		t.Fatalf("synced = %v, want auth+task", synced)
	}

	authBody, err := os.ReadFile(filepath.Join(root, "api", "auth", "proto", "auth.proto"))
	if err != nil || string(authBody) != "body-auth" {
		t.Fatalf("auth proto = %q err=%v", authBody, err)
	}
	baseBody, err := os.ReadFile(filepath.Join(baseDst, "base.proto"))
	if err != nil || string(baseBody) != "keep-base" {
		t.Fatalf("base proto should be unchanged, got %q err=%v", baseBody, err)
	}
}

func TestSyncMissingProtosSkipsPathTraversalApps(t *testing.T) {
	root := t.TempDir()
	distRoot := filepath.Join(root, "dist")
	if err := os.MkdirAll(distRoot, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}

	synced, err := SyncMissingProtos(distRoot, []string{"../etc", "foo/bar", `foo\bar`, "okapp"})
	if err != nil {
		t.Fatalf("SyncMissingProtos: %v", err)
	}
	if len(synced) != 1 || synced[0] != "okapp" {
		t.Fatalf("synced=%v, want [okapp]", synced)
	}
	if _, err := os.Stat(filepath.Join(root, "etc")); !os.IsNotExist(err) {
		t.Fatalf("traversal app should not create paths outside generated/api layout, err=%v", err)
	}
}

func TestSyncMissingProtosCreatesEmptyDirWhenGeneratedMissing(t *testing.T) {
	root := t.TempDir()
	distRoot := filepath.Join(root, "dist")
	_ = os.MkdirAll(distRoot, 0o755)

	synced, err := SyncMissingProtos(distRoot, []string{"core"})
	if err != nil {
		t.Fatalf("SyncMissingProtos: %v", err)
	}
	if len(synced) != 1 || synced[0] != "core" {
		t.Fatalf("synced = %v", synced)
	}
	st, err := os.Stat(filepath.Join(root, "api", "core", "proto"))
	if err != nil || !st.IsDir() {
		t.Fatalf("expected empty core proto dir, err=%v", err)
	}
}
