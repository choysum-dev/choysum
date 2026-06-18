// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	pkg := &PackageJSON{
		Dependencies:     map[string]string{"vue": "^3.4.0", "pinia": "~2.1.0"},
		PeerDependencies: map[string]string{"typescript": "^5.0.0"},
	}
	data, _ := json.Marshal(pkg)
	os.WriteFile(pkgPath, data, 0644)

	got, err := ReadPackageJSON(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Dependencies) != 2 {
		t.Fatalf("deps = %d, want 2", len(got.Dependencies))
	}
	if len(got.PeerDependencies) != 1 {
		t.Fatalf("peer deps = %d, want 1", len(got.PeerDependencies))
	}
}

func TestCollectDependencies(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies:     map[string]string{"a": "1.0.0"},
		PeerDependencies: map[string]string{"b": "2.0.0", "a": "0.9.0"},
	}
	deps := pkg.CollectDependencies()
	if len(deps) != 2 {
		t.Fatalf("len = %d, want 2", len(deps))
	}
	if deps["a"] != "1.0.0" {
		t.Fatalf("a = %s, want 1.0.0 (dependencies overrides peer)", deps["a"])
	}
	if deps["b"] != "2.0.0" {
		t.Fatalf("b = %s, want 2.0.0", deps["b"])
	}
}

func TestCollectDependencies_Empty(t *testing.T) {
	pkg := &PackageJSON{}
	deps := pkg.CollectDependencies()
	if len(deps) != 0 {
		t.Fatalf("expected 0 deps, got %d", len(deps))
	}
}

func TestFetchTypeDefinition_CacheHit(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	// Pre-populate cache.
	cacheFile := filepath.Join(typesDir, "testpkg@1.0.0.d.ts")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte("export declare const x: number;"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := FetchTypeDefinition(nil, "https://esm.sh", typesDir, "testpkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.FromCache {
		t.Fatal("expected cache hit")
	}
	if result.Package != "testpkg" || result.Version != "1.0.0" {
		t.Fatalf("result = %+v", result)
	}
}

func TestFetchTypeDefinition_DiscoverAndDownload(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+"/types/pkg@1.0.0/index.d.ts")
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".d.ts") {
			fmt.Fprint(w, "export declare const y: string;")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	result, transitive, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
	if err != nil {
		t.Fatalf("FetchTypeDefinition failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	_ = transitive
	if result.FromCache {
		t.Fatal("expected download, got cache hit")
	}
	if result.Package != "pkg" || result.Version != "1.0.0" {
		t.Fatalf("result = %+v", result)
	}

	// Verify file content.
	data, err := os.ReadFile(result.CachedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "export declare const y: string;" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFetchTypeDefinition_NoTypesHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	_, _, err := FetchTypeDefinition(nil, server.URL, typesDir, "noplace", "1.0.0")
	if err == nil {
		t.Fatal("expected error for missing x-typescript-types header")
	}
}

func TestFetchTypesForModule(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			pkgName := "unknown"
			if strings.Contains(r.URL.Path, "vue") {
				pkgName = "vue"
			} else if strings.Contains(r.URL.Path, "pinia") {
				pkgName = "pinia"
			}
			w.Header().Set("x-typescript-types", srv.URL+"/types/"+pkgName+"@1.0.0/index.d.ts")
			w.WriteHeader(http.StatusOK)
			return
		}
		fmt.Fprint(w, "export declare const z: number;")
	}))
	defer srv.Close()

	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "testmod")
	os.MkdirAll(moduleDir, 0755)

	pkg := &PackageJSON{
		Dependencies: map[string]string{"vue": "^3.0.0", "pinia": "^2.0.0"},
	}
	data, _ := json.Marshal(pkg)
	os.WriteFile(filepath.Join(moduleDir, "package.json"), data, 0644)

	typesDir := filepath.Join(dir, "types")
	results, err := FetchTypesForModule(nil, srv.URL, typesDir, moduleDir)
	if err != nil {
		t.Fatalf("FetchTypesForModule failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.FromCache {
			t.Fatalf("expected download for %s@%s", r.Package, r.Version)
		}
	}
}

func TestFetchTypesForModule_NoDeps(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "nodeps")
	os.MkdirAll(moduleDir, 0755)

	pkg := &PackageJSON{}
	data, _ := json.Marshal(pkg)
	os.WriteFile(filepath.Join(moduleDir, "package.json"), data, 0644)

	typesDir := filepath.Join(dir, "types")
	results, err := FetchTypesForModule(nil, "https://esm.sh", typesDir, moduleDir)
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil results for no deps, got %d", len(results))
	}
}

func TestUpdateTsconfigPaths(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")

	// Write initial tsconfig.
	initial := `{
  "compilerOptions": {
    "target": "ES2020",
    "paths": {
      "@/*": ["./*"]
    }
  }
}`
	if err := os.WriteFile(tsconfigPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	results := []TypeFetchResult{
		{Package: "vue", Version: "3.4.29", CachedPath: filepath.Join(dir, "types", "vue@3.4.29.d.ts")},
	}

	// Create the cached file so the relative path can be computed.
	os.MkdirAll(filepath.Join(dir, "types"), 0755)
	os.WriteFile(filepath.Join(dir, "types", "vue@3.4.29.d.ts"), []byte("// types"), 0644)

	if err := UpdateTsconfigPaths(tsconfigPath, results); err != nil {
		t.Fatalf("UpdateTsconfigPaths failed: %v", err)
	}

	// Read back and verify.
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"vue"`) {
		t.Fatalf("tsconfig missing vue path: %s", content)
	}
	if !strings.Contains(content, "types/vue@3.4.29.d.ts") {
		t.Fatalf("tsconfig missing types path: %s", content)
	}
	if !strings.Contains(content, `"@/*"`) {
		t.Fatalf("tsconfig lost existing @/* path: %s", content)
	}
}

func TestUpdateTsconfigPaths_EmptyResults(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{"@/*":["./*"]}}}`), 0644)

	if err := UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("UpdateTsconfigPaths with nil results should be no-op: %v", err)
	}
}

func TestParseDTSImports(t *testing.T) {
	content := `
import { Foo } from './foo';
import type { Bar } from "../bar";
import("dynamic");
/// <reference path="./types.d.ts" />
/// <reference types="node" />
export declare const x: number;
`
	paths := parseDTSImports(content)
	if len(paths) != 5 {
		t.Fatalf("expected 5 imports, got %d: %v", len(paths), paths)
	}
}

func TestParseDTSImports_Empty(t *testing.T) {
	paths := parseDTSImports("export declare const x: number;")
	if len(paths) != 0 {
		t.Fatalf("expected 0 imports, got %d", len(paths))
	}
}

func TestResolveTypeImport_Relative(t *testing.T) {
	base := "https://esm.sh/vue@3.4.29/dist/vue.d.ts"
	tests := []struct{ imp, want string }{
		{"./foo", "https://esm.sh/vue@3.4.29/dist/foo"},
		{"../bar", "https://esm.sh/vue@3.4.29/bar"},
		{"../../baz", "https://esm.sh/baz"},
		{"https://other.com/types.d.ts", "https://other.com/types.d.ts"},
	}
	for _, tt := range tests {
		got, err := resolveTypeImport(base, tt.imp)
		if err != nil {
			t.Fatalf("resolveTypeImport(%q, %q) error: %v", base, tt.imp, err)
		}
		if got != tt.want {
			t.Fatalf("resolveTypeImport(%q, %q) = %q, want %q", base, tt.imp, got, tt.want)
		}
	}
}
