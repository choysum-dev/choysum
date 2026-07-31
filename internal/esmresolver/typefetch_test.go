// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	logutil "github.com/choysum-dev/choysum/internal/logger"
)

type typeFetchErrorRoundTripper struct{}

func (typeFetchErrorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network blocked in test")
}

func captureTypeFetchStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return string(data)
}

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
	if result.FromCache {
		t.Fatal("expected download, got cache hit")
	}
	if result.Package != "pkg" || result.Version != "1.0.0" {
		t.Fatalf("result = %+v", result)
	}

	depPath := typeCachePathForURL(typesDir, srv.URL+"/types/pkg@1.0.0/sub.d.ts")
	if len(transitive) == 0 {
		t.Fatal("expected transitive results to include imported type files")
	}
	hasDepResult := false
	for _, tr := range transitive {
		if tr.CachedPath == depPath {
			hasDepResult = true
			break
		}
	}
	if !hasDepResult {
		t.Fatalf("expected transitive results to include %s, got %+v", depPath, transitive)
	}
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

func TestFetchTypeDefinition_RequestTimeoutStartsAfterSlotAcquired(t *testing.T) {
	oldTimeout := typeFetchRequestTimeout
	typeFetchRequestTimeout = 300 * time.Millisecond
	defer func() { typeFetchRequestTimeout = oldTimeout }()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-typescript-types", srv.URL+"/types/pkg@1.0.0/index.d.ts")
			w.WriteHeader(http.StatusOK)
			return
		}
		fmt.Fprint(w, "export declare const x: number;")
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	state := newTypeFetchState(1)

	// Occupy the only request slot long enough to exceed request timeout if
	// timeout starts before waiting for the slot.
	state.requestSem <- struct{}{}
	go func() {
		time.Sleep(500 * time.Millisecond)
		<-state.requestSem
	}()

	result, _, err := fetchTypeDefinitionWithState(context.Background(), nil, srv.URL, typesDir, "pkg", "1.0.0", state)
	if err != nil {
		t.Fatalf("fetchTypeDefinitionWithState failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestFetchTypeDefinition_TransitiveChildRetryRecoversInSameRun(t *testing.T) {
	typesURLPath := "/types/pkg@1.0.0/index.d.ts"
	subURLPath := "/types/pkg@1.0.0/sub.d.ts"
	subCalls := 0

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
				fmt.Fprint(w, `export * from "./sub.d.ts";`)
				return
			case subURLPath:
				subCalls++
				if subCalls == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				fmt.Fprint(w, "export declare const sub: string;")
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	result, _, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
	if err != nil {
		t.Fatalf("FetchTypeDefinition failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FromCache {
		t.Fatal("expected fetched result, got cache hit")
	}
	if subCalls != 2 {
		t.Fatalf("expected sub.d.ts to be retried once (2 calls total), got %d", subCalls)
	}

	depPath := typeCachePathForURL(typesDir, srv.URL+subURLPath)
	if _, err := os.Stat(depPath); err != nil {
		t.Fatalf("expected retried transitive type file at %s: %v", depPath, err)
	}
}

func TestFetchTypeDefinition_TransitiveChildRetryExhaustedWarns(t *testing.T) {
	typesURLPath := "/types/pkg@1.0.0/index.d.ts"
	subURLPath := "/types/pkg@1.0.0/sub.d.ts"

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
				fmt.Fprint(w, `export * from "./sub.d.ts";`)
				return
			case subURLPath:
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	output := captureTypeFetchStderr(t, func() {
		result, _, err := FetchTypeDefinition(nil, srv.URL, typesDir, "pkg", "1.0.0")
		if err != nil {
			t.Fatalf("FetchTypeDefinition failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	if !strings.Contains(output, "[esm-type-fetch] warn: pkg transitive fetch had 1/1 failures") {
		t.Fatalf("expected transitive failure warning in stderr, got %q", output)
	}
	if !strings.Contains(output, "example:") || !strings.Contains(output, subURLPath) {
		t.Fatalf("expected warning to include failing URL sample, got %q", output)
	}

	depPath := typeCachePathForURL(typesDir, srv.URL+subURLPath)
	if _, err := os.Stat(depPath); !os.IsNotExist(err) {
		t.Fatalf("expected transitive child cache file to be absent after exhausted retries, err=%v", err)
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
		if strings.HasSuffix(r.URL.Path, "/index.d.ts") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
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

func TestFetchTypeDefinition_NoTypesHeaderFallsBackToIndexDTS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/index.d.ts") {
			_, _ = w.Write([]byte("export declare const process: any;"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")

	result, transitive, err := FetchTypeDefinition(nil, server.URL, typesDir, "@types/node", "25.9.4")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(transitive) != 0 {
		t.Fatalf("expected no transitive fetch results, got %d", len(transitive))
	}
	if _, statErr := os.Stat(result.CachedPath); statErr != nil {
		t.Fatalf("expected cached file to be written: %v", statErr)
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

func TestBeginTypeFetchTransitiveProgress_FallbackLogs(t *testing.T) {
	output := captureTypeFetchStderr(t, func() {
		updateProgress, stopProgress := beginTypeFetchTransitiveProgress(context.Background(), "pkg", 400, true)
		updateProgress(199)
		updateProgress(200)
		updateProgress(400)
		stopProgress()
	})

	for _, want := range []string{
		"[esm-type-fetch] info: pkg has 400 transitive type imports",
		"[esm-type-fetch] info: pkg transitive progress 200/400",
		"[esm-type-fetch] info: pkg transitive progress 400/400",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected stderr to contain %q, got %q", want, output)
		}
	}
}

func TestBeginTypeFetchTransitiveProgress_WithContextTickerSuppressesFallbackOutput(t *testing.T) {
	ticker := logutil.NewProgressTicker(nil, logutil.ProgressTickerOptions{})
	defer ticker.Stop()
	ctx := logutil.WithProgressTicker(context.Background(), ticker)

	output := captureTypeFetchStderr(t, func() {
		updateProgress, stopProgress := beginTypeFetchTransitiveProgress(ctx, "pkg", 400, true)
		updateProgress(200)
		stopProgress()
	})

	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected no fallback stderr output with context ticker, got %q", output)
	}
}

func TestFetchTypesForModuleWithState_WarnOutputWithoutVerboseStartDone(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "warnmod")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	pkg := &PackageJSON{Dependencies: map[string]string{"dep": "1.0.0"}}
	data, _ := json.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), data, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	ticker := logutil.NewProgressTicker(nil, logutil.ProgressTickerOptions{})
	defer ticker.Stop()
	ctx := logutil.WithProgressTicker(context.Background(), ticker)
	client := &http.Client{Transport: typeFetchErrorRoundTripper{}}

	var (
		results []TypeFetchResult
		runErr  error
	)
	output := captureTypeFetchStderr(t, func() {
		results, runErr = fetchTypesForModuleWithState(ctx, client, "https://esm.sh", filepath.Join(dir, "types"), moduleDir, nil)
	})

	if runErr != nil {
		t.Fatalf("fetchTypesForModuleWithState error = %v", runErr)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results on network-blocked dependency fetch, got %d", len(results))
	}
	if !strings.Contains(output, "[esm-type-fetch] warn: [warnmod] dep@1.0.0") {
		t.Fatalf("expected warn output to include module-scoped package message, got %q", output)
	}
	if strings.Contains(output, ") start ") || strings.Contains(output, ") done ") {
		t.Fatalf("expected start/done verbose lines to be suppressed, got %q", output)
	}
}

func TestFetchTypesForModuleWithState_CachedDependency(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "cachedmod")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	pkg := &PackageJSON{Dependencies: map[string]string{"dep": "1.0.0"}}
	data, _ := json.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), data, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	typesDir := filepath.Join(dir, "types")
	cacheFile := filepath.Join(typesDir, "dep@1.0.0.d.ts")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("export declare const x: number;"), 0o644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	ticker := logutil.NewProgressTicker(nil, logutil.ProgressTickerOptions{})
	defer ticker.Stop()
	ctx := logutil.WithProgressTicker(context.Background(), ticker)

	results, err := fetchTypesForModuleWithState(ctx, nil, "https://esm.sh", typesDir, moduleDir, nil)
	if err != nil {
		t.Fatalf("fetchTypesForModuleWithState error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 cached result, got %d", len(results))
	}
	if !results[0].FromCache {
		t.Fatalf("expected cached result, got %#v", results[0])
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

	// Change into the temp directory so that filepath.Abs inside
	// UpdateTsconfigPaths resolves relative CachedPath values
	// against the correct base.  t.Chdir is test-scoped and
	// automatically restores the working directory on cleanup.
	t.Chdir(dir)

	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	typesDir := filepath.Join(".", "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatalf("mkdir types: %v", err)
	}
	if err := os.WriteFile(filepath.Join(typesDir, "vue.d.ts"), []byte("// types"), 0o644); err != nil {
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
	if !strings.Contains(content, `"types/vue.d.ts"`) {
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
	if !strings.Contains(content, `"compilerOptions"`) {
		t.Fatalf("created tsconfig should include compilerOptions: %s", content)
	}
}

func TestEnsureTsconfigCompilerTypeRoots(t *testing.T) {
	dir := t.TempDir()
	modulesDir := filepath.Join(dir, "modules")
	tsconfigPath := filepath.Join(modulesDir, "tsconfig.json")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("mkdir modules dir: %v", err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"types":["node"]}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}

	typesDir := filepath.Join(dir, ".choysum", "pkg", "types")
	cachedPath := filepath.Join(typesDir, "@types", "node@26.1.1.d.ts")
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0o755); err != nil {
		t.Fatalf("mkdir cached path dir: %v", err)
	}
	if err := os.WriteFile(cachedPath, []byte("declare var process: any;"), 0o644); err != nil {
		t.Fatalf("write cached type file: %v", err)
	}

	links := []CompilerTypeRootLink{{TypeName: "node", CachedPath: cachedPath}}
	if err := EnsureTsconfigCompilerTypeRoots(tsconfigPath, typesDir, links); err != nil {
		t.Fatalf("EnsureTsconfigCompilerTypeRoots failed: %v", err)
	}

	tsconfigData, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatalf("read tsconfig: %v", err)
	}
	tsconfigContent := string(tsconfigData)
	if !strings.Contains(tsconfigContent, `"typeRoots"`) {
		t.Fatalf("tsconfig missing typeRoots after bridge generation: %s", tsconfigContent)
	}
	if !strings.Contains(tsconfigContent, `"../.choysum/pkg/types/typeRoots"`) {
		t.Fatalf("tsconfig missing expected relative typeRoots path: %s", tsconfigContent)
	}

	bridgePath := filepath.Join(typesDir, "typeRoots", "node", "index.d.ts")
	bridgeData, err := os.ReadFile(bridgePath)
	if err != nil {
		t.Fatalf("read bridge file: %v", err)
	}
	bridgeContent := string(bridgeData)
	if !strings.Contains(bridgeContent, `compilerOptions.types="node"`) {
		t.Fatalf("bridge file missing type annotation context: %s", bridgeContent)
	}
	if !strings.Contains(bridgeContent, `node@26.1.1.d.ts`) {
		t.Fatalf("bridge file missing cached declaration reference: %s", bridgeContent)
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

func TestRewriteTypeImportSpecifiers_BridgesVueImportToBare(t *testing.T) {
	typesDir := t.TempDir()
	cacheFile := filepath.Join(typesDir, "root.d.ts")

	content := `import { ComputedRef } from "./vue.d.mts";`
	rewritten := rewriteTypeImportSpecifiers(content, cacheFile, typesDir, []resolvedTypeImport{
		{Original: "./vue.d.mts", ResolvedURL: "https://esm.sh/vue@3.5.30/dist/vue.d.mts"},
	})

	if !strings.Contains(rewritten, `"vue"`) || strings.Contains(rewritten, `"./vue.d.mts"`) {
		t.Fatalf("expected vue import to bridge to bare specifier, got: %q", rewritten)
	}
}

func TestRewriteTypeModuleAugmentationSpecifiers(t *testing.T) {
	content := `
declare module 'https://esm.sh/pinia@3.0.4/dist/pinia.d.ts' {
  interface DefineStoreOptionsBase<S, Store> {
    persist?: boolean
  }
}

declare module 'https://esm.sh/moment@2.30.1/ts3.1-typings/moment.d.ts' {
	interface Moment {
		tz(): string | undefined
	}
}

declare module 'https://esm.sh/not-bridged@1.0.0/index.d.ts' {
  interface X {}
}

declare module 'https://mirror.example.com/vue@3.5.35/dist/vue.d.mts' {
	interface Y {}
}
`
	rewritten := rewriteTypeModuleAugmentationSpecifiers(content)

	if !strings.Contains(rewritten, `declare module 'pinia'`) {
		t.Fatalf("expected pinia module augmentation to be bridged, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, `declare module 'moment'`) {
		t.Fatalf("expected moment module augmentation to be bridged, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, `declare module 'https://esm.sh/not-bridged@1.0.0/index.d.ts'`) {
		t.Fatalf("expected non-bridged module augmentation to remain unchanged, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, `declare module 'vue'`) {
		t.Fatalf("expected custom-host vue augmentation to be bridged, got: %q", rewritten)
	}
}

func TestRewriteLocalCachedBridgeSpecifiers(t *testing.T) {
	content := `import { Ref } from "./esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts";
import { defineStore } from "./esm.sh_pinia@3.0.4_dist_pinia.d.ts.d.ts";
import { X } from "./esm.sh_other@1.0.0_dist_index.d.ts.d.ts";
import { Ref2 } from "./mirror.example.com_vue@3.5.35_dist_vue.d.mts.d.ts";`

	rewritten := rewriteLocalCachedBridgeSpecifiers(content)
	if !strings.Contains(rewritten, `"vue"`) || strings.Contains(rewritten, `"./esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts"`) {
		t.Fatalf("expected vue local cache import to bridge, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, `"pinia"`) || strings.Contains(rewritten, `"./esm.sh_pinia@3.0.4_dist_pinia.d.ts.d.ts"`) {
		t.Fatalf("expected pinia local cache import to bridge, got: %q", rewritten)
	}
	if !strings.Contains(rewritten, `from "./esm.sh_other@1.0.0_dist_index.d.ts.d.ts"`) {
		t.Fatalf("expected non-bridge local cache import to remain unchanged, got: %q", rewritten)
	}
	if strings.Contains(rewritten, `"./mirror.example.com_vue@3.5.35_dist_vue.d.mts.d.ts"`) || strings.Count(rewritten, `"vue"`) != 2 {
		t.Fatalf("expected custom-host local cache import to bridge, got: %q", rewritten)
	}
}

func TestBridgedBareSpecifierForLocalCacheSpecifier_ScopedPackage(t *testing.T) {
	const scopedPkg = "@scope/pkg"
	typeModuleBridgePackages[scopedPkg] = struct{}{}
	defer delete(typeModuleBridgePackages, scopedPkg)

	got := bridgedBareSpecifierForLocalCacheSpecifier("./esm.sh_@scope_pkg@1.2.3_dist_index.d.ts.d.ts")
	if got != scopedPkg {
		t.Fatalf("bridgedBareSpecifierForLocalCacheSpecifier() = %q, want %q", got, scopedPkg)
	}
}

func TestBridgedBareSpecifierForLocalCacheSpecifier_BuildPrefix(t *testing.T) {
	got := bridgedBareSpecifierForLocalCacheSpecifier("./esm.sh_v135_vue@3.5.35_dist_vue.d.mts.d.ts")
	if got != "vue" {
		t.Fatalf("bridgedBareSpecifierForLocalCacheSpecifier() = %q, want %q", got, "vue")
	}
}

func TestNormalizeBridgeCachedTypeChildren_RewritesChildAugmentation(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root.d.ts")
	child := filepath.Join(dir, "esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts")

	if err := os.WriteFile(root, []byte(`export * from "./esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts";`), 0o644); err != nil {
		t.Fatalf("write root: %v", err)
	}
	if err := os.WriteFile(child, []byte(`declare module 'https://esm.sh/vue@3.5.35/dist/vue.d.mts' { interface X {} }`), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	if err := normalizeBridgeCachedTypeChildren(dir, root, []string{"./esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts"}); err != nil {
		t.Fatalf("normalizeBridgeCachedTypeChildren: %v", err)
	}

	data, err := os.ReadFile(child)
	if err != nil {
		t.Fatalf("read child: %v", err)
	}
	if !strings.Contains(string(data), `declare module 'vue'`) {
		t.Fatalf("expected child augmentation to be bridged to bare vue, got: %q", string(data))
	}
}

func TestNormalizeBridgeCachedTypeChildren_ConcurrentSharedChild(t *testing.T) {
	dir := t.TempDir()
	rootA := filepath.Join(dir, "root-a.d.ts")
	rootB := filepath.Join(dir, "root-b.d.ts")
	child := filepath.Join(dir, "esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts")

	if err := os.WriteFile(rootA, []byte(`export * from "./esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts";`), 0o644); err != nil {
		t.Fatalf("write rootA: %v", err)
	}
	if err := os.WriteFile(rootB, []byte(`export * from "./esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts";`), 0o644); err != nil {
		t.Fatalf("write rootB: %v", err)
	}
	if err := os.WriteFile(child, []byte(`declare module 'https://mirror.example.com/vue@3.5.35/dist/vue.d.mts' { interface X {} }`), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- normalizeBridgeCachedTypeChildren(dir, rootA, []string{"./esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts"})
	}()
	go func() {
		errCh <- normalizeBridgeCachedTypeChildren(dir, rootB, []string{"./esm.sh_vue-router@5.1.0_dist_index-BQLwgiyK.d.ts.d.ts"})
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("normalizeBridgeCachedTypeChildren concurrent run failed: %v", err)
		}
	}

	data, err := os.ReadFile(child)
	if err != nil {
		t.Fatalf("read child: %v", err)
	}
	if !strings.Contains(string(data), `declare module 'vue'`) {
		t.Fatalf("expected child augmentation to be bridged to bare vue, got: %q", string(data))
	}
}

func TestEsmTypeURLBarePackage(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
	}{
		{rawURL: "https://esm.sh/vue@3.5.35/dist/vue.d.mts", want: "vue"},
		{rawURL: "https://esm.sh/v135/vue@3.5.35/dist/vue.d.mts", want: "vue"},
		{rawURL: "https://esm.sh/pinia@3.0.4/dist/pinia.d.ts", want: "pinia"},
		{rawURL: "https://esm.sh/v135/@scope/pkg@1.2.3/dist/index.d.ts", want: "@scope/pkg"},
		{rawURL: "https://esm.sh/@scope/pkg@1.2.3/dist/index.d.ts", want: "@scope/pkg"},
		{rawURL: "https://mirror.example.com/vue@3.5.35/dist/vue.d.mts", want: "vue"},
		{rawURL: "https://mirror.example.com/v135/vue@3.5.35/dist/vue.d.mts", want: "vue"},
		{rawURL: "https://mirror.example.com/@scope/pkg@1.2.3/dist/index.d.ts", want: "@scope/pkg"},
		{rawURL: "https://mirror.example.com/not-bridged@1.0.0/index.d.ts", want: "not-bridged"},
		{rawURL: "./local/path.d.ts", want: ""},
	}

	for _, tt := range tests {
		if got := esmTypeURLBarePackage(tt.rawURL); got != tt.want {
			t.Fatalf("esmTypeURLBarePackage(%q) = %q, want %q", tt.rawURL, got, tt.want)
		}
	}
}

func TestIsLocalCachedTypeSpecifier(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "./esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts", want: true},
		{path: "./esm.sh_v135_vue@3.5.35_dist_vue.d.mts.d.ts", want: true},
		{path: "./mirror.example.com_vue@3.5.35_dist_vue.d.mts.d.ts", want: true},
		{path: "./mirror.example.com_v135_vue@3.5.35_dist_vue.d.mts.d.ts", want: true},
		{path: "../esm.sh_kysely@0.29.2_dist_index.d.ts.d.ts", want: true},
		{path: "./runtime-dom.d.ts", want: false},
		{path: "./local_vue@3.5.35_dist_vue.d.mts.d.ts", want: false},
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
	if !hasMissingLocalCachedImports(dir, cacheFile, imports) {
		t.Fatal("expected missing relative import to be detected")
	}

	missingFile := filepath.Join(dir, "MissingIcon.d.ts")
	if err := os.WriteFile(missingFile, []byte("export {};"), 0644); err != nil {
		t.Fatal(err)
	}
	if hasMissingLocalCachedImports(dir, cacheFile, imports) {
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
		{
			// ".." remains literal after flattening separators; must not collide with "__".
			"https://esm.sh/foo/../bar.d.ts",
			"esm.sh_foo_.._bar.d.ts.d.ts",
		},
	}
	for _, tt := range tests {
		got := typeCachePathForURL(dir, tt.url)
		if filepath.Base(got) != tt.want {
			t.Fatalf("typeCachePathForURL(%q) base = %q, want %q", tt.url, filepath.Base(got), tt.want)
		}
	}

	// Distinct URLs that only differ by ".." vs "__" must not share a cache file.
	dotdot := typeCachePathForURL(dir, "https://esm.sh/foo/../bar.d.ts")
	dunder := typeCachePathForURL(dir, "https://esm.sh/foo/__/bar.d.ts")
	if filepath.Base(dotdot) == filepath.Base(dunder) {
		t.Fatalf("cache collision: %q and %q map to the same file", "foo/../bar", "foo/__/bar")
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

	if err := writeTypeCacheFile(dir, cacheFile, []byte("export {};")); err != nil {
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

func TestWithRequestSlotContext_CanceledWhileWaiting(t *testing.T) {
	state := newTypeFetchState(1)
	state.requestSem <- struct{}{}
	defer func() { <-state.requestSem }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	err := state.withRequestSlotContext(ctx, func() error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
	if called {
		t.Fatal("expected run func not to be called when context canceled before slot acquisition")
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
	if err := writeTypeCacheFile(dir, cacheFile, []byte("export {};")); err == nil {
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
	if hasMissingLocalCachedImports(dir, cacheFile, nil) {
		t.Fatal("expected no missing imports for nil/empty import list")
	}
	if hasMissingLocalCachedImports(dir, cacheFile, []string{}) {
		t.Fatal("expected no missing imports for empty import list")
	}
}

func TestHasMissingLocalCachedImports_BareImportSkipped(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "bare.d.ts")
	os.WriteFile(cacheFile, []byte(`import "vue";`), 0644)
	// Bare imports are not local-cached type specifiers; should be skipped.
	if hasMissingLocalCachedImports(dir, cacheFile, []string{"vue"}) {
		t.Fatal("expected bare import to be skipped in integrity check")
	}
}

func TestHasMissingLocalCachedImports_PathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "sub", "root.d.ts")
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	os.WriteFile(cacheFile, []byte(`import "../escape.d.ts";`), 0644)
	// The candidate would resolve outside baseDir, so should be skipped.
	if hasMissingLocalCachedImports(dir, cacheFile, []string{"../escape.d.ts"}) {
		t.Fatal("expected path traversal to be blocked in integrity check")
	}
}

func TestResolveAndValidateTypeCachePath_RejectsEscape(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := resolveAndValidateTypeCachePath(typesDir, filepath.Join(typesDir, "..", "escape.d.ts"))
	if err == nil {
		t.Fatal("expected escape path to be rejected")
	}
}

func TestWriteTypeCacheFile_RejectsPathOutsideTypesDir(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(dir, "outside.d.ts")
	if err := writeTypeCacheFile(typesDir, outside, []byte("export {};")); err == nil {
		t.Fatal("expected writeTypeCacheFile to reject paths outside typesDir")
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

// ---- normalizeCompilerTypeRootName tests ----

func TestNormalizeCompilerTypeRootName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "whitespace only", input: "  ", expected: ""},
		{name: "bare package", input: "node", expected: "node"},
		{name: "bare package with whitespace", input: "  node  ", expected: "node"},
		{name: "@types/ prefix", input: "@types/node", expected: "node"},
		{name: "@types/ prefix with whitespace", input: " @types/ node ", expected: "node"},
		{name: "non-scoped subpath", input: "vitest/globals", expected: "vitest"},
		{name: "scoped package", input: "@scope/pkg", expected: "scope__pkg"},
		{name: "scoped package with @types/", input: "@types/@scope/pkg", expected: "scope__pkg"},
		{name: "scoped no subpath", input: "@scope", expected: "scope"},
		{name: "scoped package whitespace", input: " @scope / pkg ", expected: "scope__pkg"},
		{name: "dot segment rejected", input: ".", expected: ""},
		{name: "double-dot segment rejected", input: "..", expected: ""},
		{name: "path traversal rejected", input: "../../etc/passwd", expected: ""},
		{name: "backslash rejected", input: "foo\\bar", expected: ""},
		{name: "scoped unsafe scope", input: "@../pkg", expected: ""},
		{name: "scoped unsafe pkg", input: "@scope/..", expected: ""},
		{name: "scoped backslash in scope", input: "@scope\\x/pkg", expected: ""},
		{name: "forward slash in bare segment", input: "a/b", expected: "a"},
		{name: "bare @ symbol", input: "@", expected: ""},
		{name: "scoped with nested slash in pkg", input: "@scope/pkg/sub", expected: "scope__pkg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCompilerTypeRootName(tt.input)
			if got != tt.expected {
				t.Fatalf("normalizeCompilerTypeRootName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---- EnsureTsconfigCompilerTypeRoots missing tsconfig test ----

func TestEnsureTsconfigCompilerTypeRoots_MissingTsconfig(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, ".choysum", "pkg", "types")
	tsconfigPath := filepath.Join(dir, "nonexistent", "tsconfig.json")

	cachedPath := filepath.Join(typesDir, "@types", "node@26.1.1.d.ts")
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0o755); err != nil {
		t.Fatalf("mkdir cached path dir: %v", err)
	}
	if err := os.WriteFile(cachedPath, []byte("declare var process: any;"), 0o644); err != nil {
		t.Fatalf("write cached type file: %v", err)
	}

	links := []CompilerTypeRootLink{{TypeName: "node", CachedPath: cachedPath}}
	// Should succeed even though tsconfig does not exist — the IsNotExist
	// path allows the function to create a fresh tsconfig with typeRoots.
	if err := EnsureTsconfigCompilerTypeRoots(tsconfigPath, typesDir, links); err != nil {
		t.Fatalf("EnsureTsconfigCompilerTypeRoots should tolerate missing tsconfig: %v", err)
	}

	// Verify the generated tsconfig contains typeRoots.
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatalf("read generated tsconfig: %v", err)
	}
	if !strings.Contains(string(data), `"typeRoots"`) {
		t.Fatalf("generated tsconfig missing typeRoots: %s", string(data))
	}
}

func TestWriteTypeCacheFile_PathGuardBranches(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("empty types dir", func(t *testing.T) {
		err := writeTypeCacheFile("", filepath.Join(typesDir, "x.d.ts"), []byte("export {};"))
		if err == nil || !strings.Contains(err.Error(), "types dir is empty") {
			t.Fatalf("expected empty types dir error, got %v", err)
		}
	})

	t.Run("types dir abs error", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(string) (string, error) { return "", errors.New("abs types boom") }
		err := writeTypeCacheFile(typesDir, filepath.Join(typesDir, "x.d.ts"), []byte("export {};"))
		if err == nil || !strings.Contains(err.Error(), "absolute types dir") {
			t.Fatalf("expected absolute types dir error, got %v", err)
		}
	})

	t.Run("cache abs error", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(p string) (string, error) {
			if strings.Contains(p, ".d.ts") {
				return "", errors.New("abs cache boom")
			}
			return filepath.Abs(p)
		}
		err := writeTypeCacheFile(typesDir, filepath.Join(typesDir, "x.d.ts"), []byte("export {};"))
		if err == nil || !strings.Contains(err.Error(), "absolute target path") {
			t.Fatalf("expected absolute target path error, got %v", err)
		}
	})
}

func TestResolveAndValidateTypeCachePath_GuardBranches(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveAndValidateTypeCachePath("", filepath.Join(typesDir, "a.d.ts")); err == nil {
		t.Fatal("expected empty types dir error")
	}
	if _, err := resolveAndValidateTypeCachePath(typesDir, "  "); err == nil {
		t.Fatal("expected empty target path error")
	}

	t.Run("types abs error", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(string) (string, error) { return "", errors.New("abs types boom") }
		if _, err := resolveAndValidateTypeCachePath(typesDir, filepath.Join(typesDir, "a.d.ts")); err == nil || !strings.Contains(err.Error(), "absolute types dir") {
			t.Fatalf("expected absolute types dir error, got %v", err)
		}
	})

	t.Run("target abs error", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(p string) (string, error) {
			if strings.Contains(p, "a.d.ts") {
				return "", errors.New("abs target boom")
			}
			return filepath.Abs(p)
		}
		if _, err := resolveAndValidateTypeCachePath(typesDir, filepath.Join(typesDir, "a.d.ts")); err == nil || !strings.Contains(err.Error(), "absolute target path") {
			t.Fatalf("expected absolute target path error, got %v", err)
		}
	})

	got, err := resolveAndValidateTypeCachePath(typesDir, filepath.Join(typesDir, "ok.d.ts"))
	if err != nil {
		t.Fatalf("expected valid path, got %v", err)
	}
	if filepath.Base(got) != "ok.d.ts" {
		t.Fatalf("unexpected validated path %q", got)
	}
}

func TestFetchTypeRecursive_PathGuardBranches(t *testing.T) {
	typesDir := t.TempDir()
	state := newTypeFetchState(defaultTypeFetchParallelism)

	t.Run("empty types dir", func(t *testing.T) {
		_, _, err := fetchTypeRecursive(context.Background(), http.DefaultClient, "", "https://example.com/pkg.d.ts", "pkg", "1.0.0", state, nil)
		if err == nil || !strings.Contains(err.Error(), "types dir is empty") {
			t.Fatalf("expected empty types dir error, got %v", err)
		}
	})

	t.Run("types abs error", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(string) (string, error) { return "", errors.New("abs types boom") }
		_, _, err := fetchTypeRecursive(context.Background(), http.DefaultClient, typesDir, "https://example.com/pkg.d.ts", "pkg", "1.0.0", state, nil)
		if err == nil || !strings.Contains(err.Error(), "absolute types dir") {
			t.Fatalf("expected absolute types dir error, got %v", err)
		}
	})

	t.Run("cache abs error", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(p string) (string, error) {
			if strings.Contains(p, ".d.ts") {
				return "", errors.New("abs cache boom")
			}
			return filepath.Abs(p)
		}
		_, _, err := fetchTypeRecursive(context.Background(), http.DefaultClient, typesDir, "https://example.com/pkg.d.ts", "pkg", "1.0.0", state, nil)
		if err == nil || !strings.Contains(err.Error(), "absolute target path") {
			t.Fatalf("expected absolute target path error, got %v", err)
		}
	})

	t.Run("hasprefix escape", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(p string) (string, error) {
			if strings.Contains(p, ".d.ts") {
				return filepath.Join(t.TempDir(), "outside.d.ts"), nil
			}
			return filepath.Abs(p)
		}
		_, _, err := fetchTypeRecursive(context.Background(), http.DefaultClient, typesDir, "https://example.com/pkg.d.ts", "pkg", "1.0.0", state, nil)
		if err == nil || !strings.Contains(err.Error(), "escapes types dir") {
			t.Fatalf("expected escapes types dir error, got %v", err)
		}
	})
}

func TestHasMissingLocalCachedImports_AbsAndEscapeBranches(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "root.d.ts")
	if err := os.WriteFile(cacheFile, []byte("export {};"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("types abs error continues", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(string) (string, error) { return "", errors.New("abs boom") }
		if hasMissingLocalCachedImports(dir, cacheFile, []string{"./missing.d.ts"}) {
			t.Fatal("expected abs error to skip candidate without reporting missing")
		}
	})

	t.Run("candidate abs error continues", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(p string) (string, error) {
			if strings.Contains(p, "missing.d.ts") {
				return "", errors.New("abs candidate boom")
			}
			return filepath.Abs(p)
		}
		if hasMissingLocalCachedImports(dir, cacheFile, []string{"./missing.d.ts"}) {
			t.Fatal("expected candidate abs error to skip without reporting missing")
		}
	})

	t.Run("escape continues", func(t *testing.T) {
		old := filepathAbs
		t.Cleanup(func() { filepathAbs = old })
		filepathAbs = func(p string) (string, error) {
			if strings.Contains(p, "missing.d.ts") {
				return filepath.Join(t.TempDir(), "outside.d.ts"), nil
			}
			return filepath.Abs(p)
		}
		if hasMissingLocalCachedImports(dir, cacheFile, []string{"./missing.d.ts"}) {
			t.Fatal("expected escaped candidate to be skipped")
		}
	})
}

func TestEnsureTsconfigCompilerTypeRoots_SkipsOutsideCachedPath(t *testing.T) {
	dir := t.TempDir()
	modulesDir := filepath.Join(dir, "modules")
	tsconfigPath := filepath.Join(modulesDir, "tsconfig.json")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"types":["node"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "outside.d.ts")
	if err := os.WriteFile(outside, []byte("declare const x: 1;"), 0o644); err != nil {
		t.Fatal(err)
	}

	links := []CompilerTypeRootLink{{TypeName: "node", CachedPath: outside}}
	if err := EnsureTsconfigCompilerTypeRoots(tsconfigPath, typesDir, links); err != nil {
		t.Fatalf("EnsureTsconfigCompilerTypeRoots failed: %v", err)
	}
	bridgePath := filepath.Join(typesDir, "typeRoots", "node", "index.d.ts")
	if _, err := os.Stat(bridgePath); !os.IsNotExist(err) {
		t.Fatalf("expected outside cached path to be skipped, bridge err=%v", err)
	}
}

func TestEnsureTsconfigCompilerTypeRoots_RelativeCachedPathAbsError(t *testing.T) {
	dir := t.TempDir()
	modulesDir := filepath.Join(dir, "modules")
	tsconfigPath := filepath.Join(modulesDir, "tsconfig.json")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"types":["node"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	old := filepathAbs
	t.Cleanup(func() { filepathAbs = old })
	filepathAbs = func(p string) (string, error) {
		// Absolute typesDir Abs succeeds; relative CachedPath Abs fails.
		if filepath.IsAbs(p) {
			return filepath.Abs(p)
		}
		return "", errors.New("abs relative boom")
	}
	links := []CompilerTypeRootLink{{TypeName: "node", CachedPath: "relative-node.d.ts"}}
	if err := EnsureTsconfigCompilerTypeRoots(tsconfigPath, typesDir, links); err != nil {
		t.Fatalf("expected relative abs failure to skip link, got %v", err)
	}
	bridgePath := filepath.Join(typesDir, "typeRoots", "node", "index.d.ts")
	if _, err := os.Stat(bridgePath); !os.IsNotExist(err) {
		t.Fatalf("expected relative abs failure to skip bridge, err=%v", err)
	}
}

func TestEnsureTsconfigCompilerTypeRoots_AbsError(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "modules", "tsconfig.json")
	if err := os.MkdirAll(filepath.Dir(tsconfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	old := filepathAbs
	t.Cleanup(func() { filepathAbs = old })
	filepathAbs = func(string) (string, error) { return "", errors.New("abs boom") }
	links := []CompilerTypeRootLink{{TypeName: "node", CachedPath: filepath.Join(dir, "types", "x.d.ts")}}
	if err := EnsureTsconfigCompilerTypeRoots(tsconfigPath, filepath.Join(dir, "types"), links); err == nil || !strings.Contains(err.Error(), "absolute types dir") {
		t.Fatalf("expected absolute types dir error, got %v", err)
	}
}
