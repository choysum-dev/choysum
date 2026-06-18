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

	result, err := FetchTypeDefinition(nil, "https://esm.sh", typesDir, "testpkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	result, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
	if err != nil {
		t.Fatalf("FetchTypeDefinition failed: %v", err)
	}
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

	_, err := FetchTypeDefinition(nil, server.URL, typesDir, "noplace", "1.0.0")
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
