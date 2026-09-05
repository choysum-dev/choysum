// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"testing"
)

func TestBuildCompilerOptions_PathsAlias(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": { "@/*": ["./*"], "@demo/*": ["./demo/*"] }
  }
}
`)
	opts, err := BuildCompilerOptions(modules, dir)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Paths == nil {
		t.Fatal("expected Paths")
	}
	targets, ok := opts.Paths.Get("@demo/*")
	if !ok || len(targets) != 1 {
		t.Fatalf("expected @demo/* path, got %v ok=%v", targets, ok)
	}
	want := filepath.ToSlash(filepath.Join(modules, "demo", "*"))
	if targets[0] != want {
		t.Fatalf("path target = %q, want %q", targets[0], want)
	}
}

func TestBuildCompilerOptions_JSONC(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  // comment
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@demo/*": ["./demo/*"],
    },
  },
}
`)
	opts, err := BuildCompilerOptions(modules, dir)
	if err != nil {
		t.Fatal(err)
	}
	targets, ok := opts.Paths.Get("@demo/*")
	if !ok || len(targets) != 1 {
		t.Fatalf("expected @demo/* from JSONC tsconfig, got %v ok=%v", targets, ok)
	}
}
