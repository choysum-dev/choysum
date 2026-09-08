// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyChoysumFEUnitTsconfigPaths(t *testing.T) {
	root := t.TempDir()
	modulesDir := filepath.Join(root, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testUtils := filepath.Join(root, "pkg", "jsengine", "scripts", "choysummount", "choysummount.d.ts")
	pageMount := filepath.Join(root, "internal", "testing", "frontend", "testdata", "stubs", "page_mount.d.ts")
	for _, p := range []string{testUtils, pageMount} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("export {};\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	paths := map[string]interface{}{"@/*": []string{"./*"}}
	applied, err := applyChoysumFEUnitTsconfigPaths(modulesDir, paths)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if applied != 2 {
		t.Fatalf("applied = %d, want 2", applied)
	}
	if !tsconfigPathMappingEquals(paths["@choysum/test-utils"], []string{"../pkg/jsengine/scripts/choysummount/choysummount.d.ts"}) {
		t.Fatalf("test-utils path = %#v", paths["@choysum/test-utils"])
	}
	if !tsconfigPathMappingEquals(paths["@choysum/page-mount"], []string{"../internal/testing/frontend/testdata/stubs/page_mount.d.ts"}) {
		t.Fatalf("page-mount path = %#v", paths["@choysum/page-mount"])
	}

	applied, err = applyChoysumFEUnitTsconfigPaths(modulesDir, paths)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if applied != 0 {
		t.Fatalf("second apply = %d, want 0", applied)
	}
}

func TestUpdateTsconfigPaths_WritesChoysumFEUnitPaths(t *testing.T) {
	root := t.TempDir()
	modulesDir := filepath.Join(root, "modules")
	tsconfigPath := filepath.Join(modulesDir, "tsconfig.json")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	testUtils := filepath.Join(root, "pkg", "jsengine", "scripts", "choysummount", "choysummount.d.ts")
	pageMount := filepath.Join(root, "internal", "testing", "frontend", "testdata", "stubs", "page_mount.d.ts")
	for _, p := range []string{testUtils, pageMount} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("export {};\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("UpdateTsconfigPaths: %v", err)
	}

	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, needle := range []string{`"@choysum/test-utils"`, `"@choysum/page-mount"`, "choysummount.d.ts", "page_mount.d.ts"} {
		if !strings.Contains(content, needle) {
			t.Fatalf("tsconfig missing %q: %s", needle, content)
		}
	}
}

func TestChoysumFEUnitTsconfigPathEntries_RelError(t *testing.T) {
	root := t.TempDir()
	modulesDir := filepath.Join(root, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "pkg", "jsengine", "scripts", "choysummount", "choysummount.d.ts")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := filepathRel
	t.Cleanup(func() { filepathRel = old })
	filepathRel = func(string, string) (string, error) { return "", errors.New("rel boom") }

	if _, err := choysumFEUnitTsconfigPathEntries(modulesDir); err == nil {
		t.Fatal("expected filepathRel error from choysumFEUnitTsconfigPathEntries")
	}
	if _, err := applyChoysumFEUnitTsconfigPaths(modulesDir, map[string]interface{}{}); err == nil {
		t.Fatal("expected filepathRel error from applyChoysumFEUnitTsconfigPaths")
	}
}

func TestUpdateTsconfigPaths_ChoysumFEUnitPathsError(t *testing.T) {
	root := t.TempDir()
	modulesDir := filepath.Join(root, "modules")
	tsconfigPath := filepath.Join(modulesDir, "tsconfig.json")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "pkg", "jsengine", "scripts", "choysummount", "choysummount.d.ts")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := filepathRel
	t.Cleanup(func() { filepathRel = old })
	filepathRel = func(string, string) (string, error) { return "", errors.New("rel boom") }

	err := UpdateTsconfigPaths(tsconfigPath, nil)
	if err == nil || !strings.Contains(err.Error(), "choysum FE unit paths") {
		t.Fatalf("expected FE unit paths error, got %v", err)
	}
}

func TestUpdateTsconfigPaths_ChoysumFEUnitPathsAbsFallback(t *testing.T) {
	root := t.TempDir()
	modulesDir := filepath.Join(root, "modules")
	tsconfigPath := filepath.Join(modulesDir, "tsconfig.json")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	testUtils := filepath.Join(root, "pkg", "jsengine", "scripts", "choysummount", "choysummount.d.ts")
	pageMount := filepath.Join(root, "internal", "testing", "frontend", "testdata", "stubs", "page_mount.d.ts")
	for _, p := range []string{testUtils, pageMount} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("export {};\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	old := filepathAbs
	t.Cleanup(func() { filepathAbs = old })
	filepathAbs = func(string) (string, error) { return "", errors.New("abs boom") }

	if err := UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("UpdateTsconfigPaths with abs fallback: %v", err)
	}
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"@choysum/test-utils"`) {
		t.Fatalf("expected FE unit paths after abs fallback: %s", data)
	}
}
