// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestResolveI18nModules_AllSkipsNoiseDirs(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"auth", "node_modules", "dist", "build", "coverage", "tmp"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "auth", "package.json"), []byte(`{"name":"auth"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	// Noise dir with package.json must still be skipped by name.
	if err := os.WriteFile(filepath.Join(root, "node_modules", "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write node_modules package.json: %v", err)
	}
	// Non-module sibling without package.json must be ignored.
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}

	got, err := resolveI18nModules(root, true, nil)
	if err != nil {
		t.Fatalf("resolveI18nModules: %v", err)
	}
	if !slices.Equal(got, []string{"auth"}) {
		t.Fatalf("got %#v, want [auth]", got)
	}
}
