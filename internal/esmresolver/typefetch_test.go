// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestFetchTypeDefinition_CacheHitURLDerivedFile(t *testing.T) {
	typesURLPath := "/types/pkg@1.0.0/index.d.ts"
	getCalls := 0

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+typesURLPath)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			getCalls++
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	cacheFile := typeCachePathForURL(typesDir, srv.URL+typesURLPath)
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte("export declare const cached: boolean;"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.FromCache {
		t.Fatal("expected URL-derived cache hit")
	}
	if result.CachedPath != cacheFile {
		t.Fatalf("cached path = %s, want %s", result.CachedPath, cacheFile)
	}
	if getCalls != 0 {
		t.Fatalf("expected no GET request when URL-derived cache exists, got %d", getCalls)
	}
}

func TestFetchTypeDefinition_RefreshesCorruptedLocalSpecifierCache(t *testing.T) {
	typesURLPath := "/types/pkg@1.0.0/index.d.ts"
	subURLPath := "/types/pkg@1.0.0/sub.d.ts"
	indexGetCalls := 0

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+typesURLPath)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case typesURLPath:
				indexGetCalls++
				fmt.Fprint(w, `export * from "./sub.d.ts";`)
				return
			case subURLPath:
				fmt.Fprint(w, `export declare const sub: string;`)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	cacheFile := typeCachePathForURL(typesDir, srv.URL+typesURLPath)
	missingLocal := "./esm.sh_pkg@1.0.0_es_sub.d.ts.d.ts"
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte(fmt.Sprintf(`export * from "%s";`, missingLocal)), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FromCache {
		t.Fatal("expected corrupted cache refresh to mark result as fetched")
	}
	if indexGetCalls == 0 {
		t.Fatal("expected GET request for index.d.ts when local specifier target is missing")
	}

	if _, err := os.Stat(typeCachePathForURL(typesDir, srv.URL+subURLPath)); err != nil {
		t.Fatalf("expected refreshed fetch to restore sub type cache file: %v", err)
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
			if strings.HasSuffix(r.URL.Path, "index.d.ts") {
				fmt.Fprint(w, `export * from "./sub.d.ts";`)
				return
			}
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

	depPath := typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/sub.d.ts")
	if _, err := os.Stat(depPath); err != nil {
		t.Fatalf("expected transitive type file at %s: %v", depPath, err)
	}

	// Verify file content.
	data, err := os.ReadFile(result.CachedPath)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := string(data)
	if !strings.Contains(rewritten, filepath.Base(depPath)) {
		t.Fatalf("expected rewritten import to local cache file, got: %q", rewritten)
	}
}

func TestFetchTypeDefinition_CircularImports_NoDeadlock(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+"/types/pkg@1.0.0/a.d.ts")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case "/types/pkg@1.0.0/a.d.ts":
				fmt.Fprint(w, `export * from "./b.d.ts";`)
				return
			case "/types/pkg@1.0.0/b.d.ts":
				fmt.Fprint(w, `export * from "./a.d.ts";`)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	errCh := make(chan error, 1)
	go func() {
		_, _, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("FetchTypeDefinition should not deadlock on circular imports: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FetchTypeDefinition timed out, possible recursive deadlock on circular imports")
	}

	if _, err := os.Stat(typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/a.d.ts")); err != nil {
		t.Fatalf("expected cached a.d.ts file: %v", err)
	}
	if _, err := os.Stat(typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/b.d.ts")); err != nil {
		t.Fatalf("expected cached b.d.ts file: %v", err)
	}
}

func TestFetchTypeDefinition_CircularSiblingImports_NoDeadlock(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+"/types/pkg@1.0.0/index.d.ts")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case "/types/pkg@1.0.0/index.d.ts":
				fmt.Fprint(w, "export * from \"./a.d.ts\";\nexport * from \"./b.d.ts\";")
				return
			case "/types/pkg@1.0.0/a.d.ts":
				fmt.Fprint(w, `export * from "./b.d.ts";`)
				return
			case "/types/pkg@1.0.0/b.d.ts":
				fmt.Fprint(w, `export * from "./a.d.ts";`)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	errCh := make(chan error, 1)
	go func() {
		_, _, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("FetchTypeDefinition should not deadlock on sibling circular imports: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FetchTypeDefinition timed out, possible deadlock on sibling circular imports")
	}

	for _, path := range []string{
		typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/index.d.ts"),
		typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/a.d.ts"),
		typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/b.d.ts"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected cached file %s: %v", path, err)
		}
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

func TestFetchTypesForModule_SkipsLocalProtocols(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "localdeps")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}

	pkg := &PackageJSON{
		Dependencies: map[string]string{
			"workspace-pkg": "workspace:*",
			"file-pkg":      "file:../shared/pkg",
			"link-pkg":      "link:../linked/pkg",
		},
	}
	data, _ := json.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), data, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	typesDir := filepath.Join(dir, "types")
	results, err := FetchTypesForModule(nil, "https://esm.sh", typesDir, moduleDir)
	if err != nil {
		t.Fatalf("FetchTypesForModule failed: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for local protocol deps, got %d", len(results))
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

func TestUpdateTsconfigPaths_RelativeCachedPath(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join("types"), 0o755); err != nil {
		t.Fatalf("mkdir types: %v", err)
	}
	if err := os.WriteFile(filepath.Join("types", "vue.d.ts"), []byte("// types"), 0o644); err != nil {
		t.Fatalf("write vue types: %v", err)
	}

	results := []TypeFetchResult{{Package: "vue", Version: "3.4.29", CachedPath: filepath.Join("types", "vue.d.ts")}}
	if err := UpdateTsconfigPaths(tsconfigPath, results); err != nil {
		t.Fatalf("UpdateTsconfigPaths failed: %v", err)
	}

	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatalf("read tsconfig: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"vue"`) {
		t.Fatalf("tsconfig missing vue path: %s", content)
	}
	if !strings.Contains(content, "types/vue.d.ts") {
		t.Fatalf("tsconfig missing relative cached path mapping: %s", content)
	}
}

func TestUpdateTsconfigPaths_CreatesTsconfigWhenMissing(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "modules", "tsconfig.json")

	if err := UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("UpdateTsconfigPaths should create tsconfig when missing: %v", err)
	}

	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatalf("expected tsconfig to be created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"@/*"`) {
		t.Fatalf("created tsconfig should include default @/* path: %s", content)
	}
	if !strings.Contains(content, `"exclude"`) {
		t.Fatalf("created tsconfig should include exclude section: %s", content)
	}
}

func TestParseDTSImports(t *testing.T) {
	content := `
import { Foo } from './foo';
import type { Bar } from "../bar";
export * from "./baz";
declare namespace Nested {
  export type Dynamic = import("./nested").Dynamic;
}
export type ViaTypeof = typeof import("./typeof-nested");
/// <reference path="./types.d.ts" />
/// <reference types="node" />
export declare const x: number;
`
	paths := parseDTSImports(content)
	if len(paths) != 7 {
		t.Fatalf("expected 7 imports, got %d: %v", len(paths), paths)
	}
	if !contains(paths, "./foo") || !contains(paths, "../bar") || !contains(paths, "./baz") ||
		!contains(paths, "./nested") || !contains(paths, "./typeof-nested") ||
		!contains(paths, "./types.d.ts") || !contains(paths, "node") {
		t.Fatalf("missing expected imports: %v", paths)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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

func TestResolveTypeImport_BareImportUnsupported(t *testing.T) {
	_, err := resolveTypeImport("https://esm.sh/pkg@1.0.0/index.d.ts", "node")
	if err == nil {
		t.Fatal("expected error for bare type import")
	}
}

func TestRewriteTypeImportSpecifiers(t *testing.T) {
	typesDir := t.TempDir()
	cacheFile := filepath.Join(typesDir, "root.d.ts")

	urlA := "https://esm.sh/pkg@1.0.0/sub.d.ts"
	urlB := "https://esm.sh/pkg@1.0.0/types/ref.d.ts"
	content := `export * from "./sub.d.ts";
/// <reference path="./types/ref.d.ts" />`

	rewritten := rewriteTypeImportSpecifiers(content, cacheFile, typesDir, []resolvedTypeImport{
		{Original: "./sub.d.ts", ResolvedURL: urlA},
		{Original: "./types/ref.d.ts", ResolvedURL: urlB},
	})

	if strings.Contains(rewritten, `"./sub.d.ts"`) || strings.Contains(rewritten, `"./types/ref.d.ts"`) {
		t.Fatalf("expected original specifiers to be rewritten, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, filepath.Base(typeCachePathForURL(typesDir, urlA))) {
		t.Fatalf("missing rewritten target for sub.d.ts: %q", rewritten)
	}
	if !strings.Contains(rewritten, filepath.Base(typeCachePathForURL(typesDir, urlB))) {
		t.Fatalf("missing rewritten target for ref.d.ts: %q", rewritten)
	}
}

func TestRewriteTypeImportSpecifiers_DoesNotRewritePlainStrings(t *testing.T) {
	typesDir := t.TempDir()
	cacheFile := filepath.Join(typesDir, "root.d.ts")

	urlA := "https://esm.sh/pkg@1.0.0/sub.d.ts"
	content := `// keep this comment mentioning "./sub.d.ts"
const note = "./sub.d.ts";
export * from "./sub.d.ts";`

	rewritten := rewriteTypeImportSpecifiers(content, cacheFile, typesDir, []resolvedTypeImport{
		{Original: "./sub.d.ts", ResolvedURL: urlA},
	})

	if !strings.Contains(rewritten, `const note = "./sub.d.ts";`) {
		t.Fatalf("expected plain string literal to remain unchanged, got: %q", rewritten)
	}
	if strings.Contains(rewritten, `export * from "./sub.d.ts";`) {
		t.Fatalf("expected export specifier to be rewritten, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, filepath.Base(typeCachePathForURL(typesDir, urlA))) {
		t.Fatalf("missing rewritten target for sub.d.ts: %q", rewritten)
	}
}

func TestIsLocalCachedTypeSpecifier(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "./esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts", want: true},
		{path: "../esm.sh_kysely@0.29.2_dist_index.d.ts.d.ts", want: true},
		{path: "./runtime-dom.d.ts", want: false},
		{path: "https://esm.sh/vue@3.5.35/dist/vue.d.mts", want: false},
		{path: "node", want: false},
	}

	for _, tt := range tests {
		if got := isLocalCachedTypeSpecifier(tt.path); got != tt.want {
			t.Fatalf("isLocalCachedTypeSpecifier(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestHasMissingLocalCachedImports_MissingRelativeImport(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "root.d.ts")
	if err := os.WriteFile(cacheFile, []byte("export * from './MissingIcon.d.ts';"), 0644); err != nil {
		t.Fatal(err)
	}

	imports := []string{"./MissingIcon.d.ts"}
	if !hasMissingLocalCachedImports(cacheFile, imports) {
		t.Fatal("expected missing relative import to be detected")
	}

	missingFile := filepath.Join(dir, "MissingIcon.d.ts")
	if err := os.WriteFile(missingFile, []byte("export {};"), 0644); err != nil {
		t.Fatal(err)
	}
	if hasMissingLocalCachedImports(cacheFile, imports) {
		t.Fatal("expected existing relative import to pass cache integrity check")
	}
}

// ---- NewTypeFetchSession tests ----

func TestNewTypeFetchSession_Custom(t *testing.T) {
	s := NewTypeFetchSession(8)
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.state == nil {
		t.Fatal("expected non-nil state")
	}
}

func TestNewTypeFetchSession_Zero(t *testing.T) {
	s := NewTypeFetchSession(0)
	if s == nil || s.state == nil {
		t.Fatal("expected non-nil session with default parallelism")
	}
}

// ---- TypeFetchSession.FetchTypesForModule tests ----

func TestTypeFetchSession_FetchTypesForModule(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+"/types/testlib@1.0.0/index.d.ts")
			w.WriteHeader(http.StatusOK)
			return
		}
		fmt.Fprint(w, "export declare const sess: number;")
	}))
	defer srv.Close()

	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "mod")
	os.MkdirAll(moduleDir, 0755)
	os.WriteFile(filepath.Join(moduleDir, "package.json"),
		[]byte(`{"dependencies":{"testlib":"^1.0.0"}}`), 0644)

	typesDir := filepath.Join(dir, "types")
	session := NewTypeFetchSession(4)

	results, err := session.FetchTypesForModule(context.Background(), nil, srv.URL, typesDir, moduleDir)
	if err != nil {
		t.Fatalf("FetchTypesForModule via session failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestTypeFetchSession_Nil(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "mod")
	os.MkdirAll(moduleDir, 0755)
	os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(`{}`), 0644)

	var s *TypeFetchSession
	results, err := s.FetchTypesForModule(context.Background(), nil, "https://esm.sh", filepath.Join(dir, "types"), moduleDir)
	if err != nil {
		t.Fatalf("nil session should fall back to stateless fetch: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for no-deps module, got %d", len(results))
	}
}

func TestTypeFetchSession_FetchTypesForModule_ContextCanceled(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "mod")
	os.MkdirAll(moduleDir, 0755)
	os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(`{"dependencies":{"testlib":"^1.0.0"}}`), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewTypeFetchSession(1)
	_, err := s.FetchTypesForModule(ctx, nil, "https://esm.sh", filepath.Join(dir, "types"), moduleDir)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// ---- ReadPackageJSON error path tests ----

func TestReadPackageJSON_NotFound(t *testing.T) {
	_, err := ReadPackageJSON("/nonexistent/package.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "read package.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadPackageJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	os.WriteFile(path, []byte(`not json`), 0644)

	_, err := ReadPackageJSON(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse package.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- NewTypeFetchHTTPClient tests ----

func TestNewTypeFetchHTTPClient_DefaultTimeout(t *testing.T) {
	client := NewTypeFetchHTTPClient(0)
	if client == nil {
		t.Fatal("expected non-nil client with default timeout")
	}
	if client.Timeout != defaultTypeFetchRequestTimeout {
		t.Fatalf("Timeout = %v, want %v", client.Timeout, defaultTypeFetchRequestTimeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("Transport = %T, want *http.Transport", client.Transport)
	}
	if !transport.ForceAttemptHTTP2 {
		t.Fatal("expected ForceAttemptHTTP2 to be enabled")
	}
}

func TestNewTypeFetchHTTPClient_CustomTimeout(t *testing.T) {
	client := NewTypeFetchHTTPClient(5 * time.Second)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %v, want 5s", client.Timeout)
	}
}

// ---- typeCachePathForURL tests ----

func TestTypeCachePathForURL_Various(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		url  string
		want string
	}{
		{
			"https://esm.sh/vue@3.4.29/dist/vue.d.mts",
			"esm.sh_vue@3.4.29_dist_vue.d.mts.d.ts",
		},
		{
			"https://cdn.jsdelivr.net/npm/pkg@1.0.0/index.d.ts",
			"cdn.jsdelivr.net_npm_pkg@1.0.0_index.d.ts.d.ts",
		},
	}
	for _, tt := range tests {
		got := typeCachePathForURL(dir, tt.url)
		if filepath.Base(got) != tt.want {
			t.Fatalf("typeCachePathForURL(%q) base = %q, want %q", tt.url, filepath.Base(got), tt.want)
		}
	}
}

// ---- typePkgNameFromURL tests ----

func TestTypePkgNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://esm.sh/kysely@0.27.6/dist/index.d.ts", "kysely@0.27.6/dist/index.d.ts"},
		{"https://esm.sh/@scope/pkg@1.0.0/index.d.ts", "@scope/pkg@1.0.0/index.d.ts"},
		{"https://esm.sh/vue", "vue"},
		{"https://esm.sh/vue?target=es2020", "vue"},
	}
	for _, tt := range tests {
		got := typePkgNameFromURL(tt.url)
		if got != tt.want {
			t.Fatalf("typePkgNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// ---- FetchTypeDefinition HTTP error paths ----

func TestFetchTypeDefinition_HeadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	_, _, err := FetchTypeDefinition(nil, server.URL, typesDir, "pkg", "1.0.0")
	if err == nil {
		t.Fatal("expected error for non-200 HEAD")
	}
}

// ---- writeTypeCacheFile edge case tests ----

func TestWriteTypeCacheFile(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "nested", "deep", "types.d.ts")

	if err := writeTypeCacheFile(cacheFile, []byte("export {};")); err != nil {
		t.Fatalf("writeTypeCacheFile failed: %v", err)
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "export {};" {
		t.Fatalf("content = %q", string(data))
	}
}

// ---- downloadTypeContent error path test ----

func TestDownloadTypeContent_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewTypeFetchHTTPClient(5 * time.Second)
	state := newTypeFetchState(defaultTypeFetchParallelism)

	_, err := downloadTypeContent(context.Background(), client, server.URL, state)
	if err == nil {
		t.Fatal("expected error for 404 download")
	}
}

func TestDownloadTypeContent_ResponseTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("a"), int(maxTypeFetchDownloadBytes)+1))
	}))
	defer server.Close()

	client := NewTypeFetchHTTPClient(5 * time.Second)
	state := newTypeFetchState(defaultTypeFetchParallelism)

	_, err := downloadTypeContent(context.Background(), client, server.URL, state)
	if err == nil || !strings.Contains(err.Error(), "response exceeds") {
		t.Fatalf("expected response too large error, got %v", err)
	}
}

// ---- ensureModulesTsconfig edge cases ----

// ---- ensureModulesTsconfig edge cases ----

func TestEnsureModulesTsconfig_DirNotExist(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "nonexistent", "tsconfig.json")

	if err := ensureModulesTsconfig(tsconfigPath); err != nil {
		t.Fatalf("ensureModulesTsconfig should create missing dirs: %v", err)
	}

	if _, err := os.Stat(tsconfigPath); err != nil {
		t.Fatalf("tsconfig should exist: %v", err)
	}
}

func TestEnsureModulesTsconfig_Existing(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{}}`), 0644)

	if err := ensureModulesTsconfig(tsconfigPath); err != nil {
		t.Fatalf("ensureModulesTsconfig should be no-op for existing file: %v", err)
	}

	data, _ := os.ReadFile(tsconfigPath)
	if string(data) != `{"compilerOptions":{}}` {
		t.Fatal("existing tsconfig should not be modified")
	}
}

// ---- acquireVisit edge cases ----

func TestAcquireVisit_NilState(t *testing.T) {
	var s *typeFetchState
	shouldFetch, done := s.acquireVisit("https://example.com")
	if !shouldFetch {
		t.Fatal("expected true when state is nil (allow through)")
	}
	done(true)
}

func TestAcquireVisit_SkipsInFlightFetch(t *testing.T) {
	s := newTypeFetchState(defaultTypeFetchParallelism)
	shouldFetch, done := s.acquireVisit("https://example.com")
	if !shouldFetch {
		t.Fatal("expected first acquire to fetch")
	}

	// Second caller must return immediately (non-blocking) to avoid
	// deadlocks when concurrent sibling traversals share transitive deps.
	shouldFetch2, done2 := s.acquireVisit("https://example.com")
	if shouldFetch2 {
		t.Fatal("expected second acquire to skip in-flight fetch")
	}
	done2(true)

	done(true)

	// After the first fetch completes (success), subsequent calls
	// still skip because the URL stays in the visited set.
	shouldFetch3, done3 := s.acquireVisit("https://example.com")
	if shouldFetch3 {
		t.Fatal("expected third acquire to still skip after successful fetch")
	}
	done3(true)
}

func TestAcquireVisit_FailureAllowsRetry(t *testing.T) {
	s := newTypeFetchState(defaultTypeFetchParallelism)
	shouldFetch, done := s.acquireVisit("https://example.com")
	if !shouldFetch {
		t.Fatal("expected first acquire to fetch")
	}
	done(false)

	shouldFetchRetry, doneRetry := s.acquireVisit("https://example.com")
	if !shouldFetchRetry {
		t.Fatal("expected retry acquire to fetch after failure")
	}
	doneRetry(true)
}

func TestAcquireVisit_FailureAllowsRetryAfterSkip(t *testing.T) {
	s := newTypeFetchState(defaultTypeFetchParallelism)
	shouldFetch, done := s.acquireVisit("https://example.com")
	if !shouldFetch {
		t.Fatal("expected first acquire to fetch")
	}

	// With non-blocking acquire, the second caller skips immediately.
	shouldFetch2, done2 := s.acquireVisit("https://example.com")
	if shouldFetch2 {
		t.Fatal("expected second acquire to skip in-flight fetch")
	}
	done2(true)

	// First fetch fails → URL is removed from visited set.
	done(false)

	// Now a fresh caller can retry.
	shouldFetchRetry, doneRetry := s.acquireVisit("https://example.com")
	if !shouldFetchRetry {
		t.Fatal("expected retry acquire to fetch after failure")
	}
	doneRetry(true)
}

// ---- withRequestSlot edge cases ----

func TestWithRequestSlot_NilState(t *testing.T) {
	var s *typeFetchState
	called := false
	err := s.withRequestSlot(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected fn to be called when state is nil")
	}
}

// ---- newTypeFetchState edge cases ----

func TestNewTypeFetchState_ZeroParallelism(t *testing.T) {
	state := newTypeFetchState(0)
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	// Should use default parallelism.
	if state.requestSem == nil || cap(state.requestSem) != defaultTypeFetchParallelism {
		t.Fatalf("expected default parallelism = %d, got cap=%d", defaultTypeFetchParallelism, cap(state.requestSem))
	}
}

// ---- resolveTypeImport edge cases ----

func TestResolveTypeImport_InvalidBase(t *testing.T) {
	_, err := resolveTypeImport("://invalid", "./foo")
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

func TestResolveTypeImport_EmptyImportPath(t *testing.T) {
	_, err := resolveTypeImport("https://esm.sh/pkg/index.d.ts", "")
	if err == nil {
		t.Fatal("expected error for empty import path")
	}
}

// ---- writeTypeCacheFile rename failure test ----

func TestWriteTypeCacheFile_RenameFails_CleansUpTmp(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "types.d.ts")
	// Create a directory at the target path so Rename fails.
	if err := os.MkdirAll(cacheFile, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeTypeCacheFile(cacheFile, []byte("export {};")); err == nil {
		t.Fatal("expected rename error when target is a directory")
	}
	// Tmp file should be cleaned up.
	if _, err := os.Stat(cacheFile + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("expected tmp file to be cleaned up after failed rename")
	}
}

// ---- hasMissingLocalCachedImports edge cases ----

func TestHasMissingLocalCachedImports_EmptyImports(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "empty.d.ts")
	os.WriteFile(cacheFile, []byte("export {};"), 0644)
	if hasMissingLocalCachedImports(cacheFile, nil) {
		t.Fatal("expected no missing imports for nil/empty import list")
	}
	if hasMissingLocalCachedImports(cacheFile, []string{}) {
		t.Fatal("expected no missing imports for empty import list")
	}
}

func TestHasMissingLocalCachedImports_BareImportSkipped(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "bare.d.ts")
	os.WriteFile(cacheFile, []byte(`import "vue";`), 0644)
	// Bare imports are not local-cached type specifiers; should be skipped.
	if hasMissingLocalCachedImports(cacheFile, []string{"vue"}) {
		t.Fatal("expected bare import to be skipped in integrity check")
	}
}

func TestHasMissingLocalCachedImports_PathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "sub", "root.d.ts")
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	os.WriteFile(cacheFile, []byte(`import "../escape.d.ts";`), 0644)
	// The candidate would resolve outside baseDir, so should be skipped.
	if hasMissingLocalCachedImports(cacheFile, []string{"../escape.d.ts"}) {
		t.Fatal("expected path traversal to be blocked in integrity check")
	}
}

// ---- downloadTypeContent network error test ----

func TestDownloadTypeContent_NetworkError(t *testing.T) {
	client := NewTypeFetchHTTPClient(2 * time.Second)
	state := newTypeFetchState(defaultTypeFetchParallelism)
	_, err := downloadTypeContent(context.Background(), client, "http://127.0.0.1:1/nonexistent.d.ts", state)
	if err == nil {
		t.Fatal("expected network error for non-routable address")
	}
}

func TestDownloadTypeContent_InvalidURL(t *testing.T) {
	client := NewTypeFetchHTTPClient(5 * time.Second)
	state := newTypeFetchState(defaultTypeFetchParallelism)
	_, err := downloadTypeContent(context.Background(), client, "://invalid-url", state)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// ---- UpdateTsconfigPaths edge cases ----

func TestUpdateTsconfigPaths_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	os.WriteFile(tsconfigPath, []byte(`{invalid`), 0644)

	results := []TypeFetchResult{
		{Package: "vue", Version: "3.4.29", CachedPath: filepath.Join(dir, "types", "vue.d.ts")},
	}
	err := UpdateTsconfigPaths(tsconfigPath, results)
	if err == nil {
		t.Fatal("expected error for invalid JSON tsconfig")
	}
}

func TestUpdateTsconfigPaths_ReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping write test as root")
	}
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	// Make parent dir read-only so write fails.
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0755)

	err := UpdateTsconfigPaths(tsconfigPath, nil)
	if err == nil {
		t.Fatal("expected error when tsconfig dir is read-only")
	}
}

// ---- typeCachePathForURL edge cases ----

func TestTypeCachePathForURL_LongURL(t *testing.T) {
	dir := t.TempDir()
	longPath := strings.Repeat("x", 300)
	url := "https://esm.sh/" + longPath
	got := typeCachePathForURL(dir, url)
	base := filepath.Base(got)
	// safe name truncated to 200 + ".d.ts" = 205 chars.
	if len(base) > 205 {
		t.Fatalf("expected truncated filename, got len=%d: %q", len(base), base)
	}
	if !strings.HasSuffix(base, ".d.ts") {
		t.Fatalf("expected .d.ts suffix: %q", base)
	}
}

// ---- isLocalCachedTypeSpecifier edge cases ----

func TestIsLocalCachedTypeSpecifier_Empty(t *testing.T) {
	if isLocalCachedTypeSpecifier("") {
		t.Fatal("expected false for empty string")
	}
}

func TestIsLocalCachedTypeSpecifier_NonMatching(t *testing.T) {
	if isLocalCachedTypeSpecifier("./runtime-dom.d.ts") {
		t.Fatal("expected false for non-esm.sh_ prefix")
	}
}

// ---- fetchTypeRecursive read-only dir error path ----

func TestFetchTypeRecursive_WriteCacheFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping write test as root")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "export declare const x: number;")
	}))
	defer server.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	// Use an HTTP client that can reach the test server.
	client := NewTypeFetchHTTPClient(5 * time.Second)
	state := newTypeFetchState(defaultTypeFetchParallelism)

	// Write to a real readable directory... but the test is more about testing
	// that fetchTypeRecursive returns an error when cache writing fails.
	// We simulate by using a non-existent subdirectory in a read-only parent.
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(roDir, 0755)

	// typesDir inside read-only parent — MkdirAll will fail.
	typesDir = filepath.Join(roDir, "types")
	_, _, err := fetchTypeRecursive(context.Background(), client, typesDir, server.URL+"/test.d.ts", "testpkg", "1.0.0", state, nil)
	if err == nil {
		t.Fatal("expected error when cache write fails in read-only dir")
	}
}
