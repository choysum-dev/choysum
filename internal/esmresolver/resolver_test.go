// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// ---- helpers ----

func sha256HexBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func computeSha512(b []byte) string {
	h := sha512.Sum512(b)
	return hex.EncodeToString(h[:])
}

func writeCacheFile(dir, cacheKey, content string) (string, error) {
	cacheFile := filepath.Join(dir, cacheKey[:2], cacheKey[2:])
	integrityFile := cacheFile + ".integrity"
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(integrityFile, []byte(computeSha512([]byte(content))), 0644); err != nil {
		return "", err
	}
	return cacheFile, nil
}

func newTestResolver(dir string) *Resolver {
	return New(
		WithCacheDir(dir),
		WithTarget("es2020"),
		WithHTTPClient(&http.Client{Timeout: 5 * time.Second}),
	)
}

// ---- M7a: Cache hit → no HTTP request ----

func TestResolver_CacheHit_ReadCache(t *testing.T) {
	dir := t.TempDir()
	r := newTestResolver(dir)

	content := "export const x = 1;"
	cacheKey := sha256Hex("https://esm.sh/test-pkg@1.0.0")
	cacheFile, err := writeCacheFile(r.codeCacheDir(), cacheKey, content)
	if err != nil {
		t.Fatal(err)
	}

	// Read should return cached content.
	got, ok := r.readCache(cacheFile)
	if !ok {
		t.Fatal("readCache returned false for valid cache")
	}
	if got != content {
		t.Fatalf("readCache = %q, want %q", got, content)
	}
}

func TestResolver_CacheHit_OnLoad(t *testing.T) {
	dir := t.TempDir()
	r := newTestResolver(dir)

	// Write a cache entry for a known URL.
	content := "export const y = 2;"
	url := "https://esm.sh/hit-pkg@1.0.0?target=es2020"
	cacheKey := sha256Hex(url)
	if _, err := writeCacheFile(r.codeCacheDir(), cacheKey, content); err != nil {
		t.Fatal(err)
	}

	// Simulate OnLoad with the URL as path.
	plugin := r.Plugin()
	// We need to invoke OnLoad indirectly. Create a minimal esbuild build
	// that triggers OnResolve → OnLoad.
	// For simplicity, test that readCache works (cache hit path).
	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])
	got, ok := r.readCache(cacheFile)
	if !ok {
		t.Fatal("cache should be present after writeCacheFile")
	}
	if got != content {
		t.Fatalf("content = %q, want %q", got, content)
	}
	_ = plugin // plugin tested via integration below
}

// ---- M7b: Cache miss → HTTP download → write → return ----

func TestResolver_CacheMiss_Download(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "export const z = 3;")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	url := server.URL + "/test-pkg@1.0.0?target=es2020"
	content, err := r.downloadWithRetry(url)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if content != "export const z = 3;" {
		t.Fatalf("content = %q, want %q", content, "export const z = 3;")
	}

	// Manually write to cache (simulating OnLoad behavior).
	cacheKey := sha256Hex(url)
	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])
	if err := r.writeCache(cacheFile, []byte(content)); err != nil {
		t.Fatalf("writeCache failed: %v", err)
	}
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatal("expected cache file to exist after download")
	}

	// Read back from cache.
	got, ok := r.readCache(cacheFile)
	if !ok {
		t.Fatal("readCache failed after download")
	}
	if got != content {
		t.Fatalf("cached = %q, want %q", got, content)
	}
}

// ---- M7c: HTTP 4xx/5xx → retry with backoff ----

func TestResolver_Download_RetryOn500(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "ok after retry")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	// Use downloadWithRetry directly.
	content, err := r.downloadWithRetry(server.URL + "/pkg?target=es2020")
	if err != nil {
		t.Fatalf("downloadWithRetry failed: %v", err)
	}
	if content != "ok after retry" {
		t.Fatalf("content = %q", content)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestResolver_Download_NoRetryOn404(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	_, err := r.downloadWithRetry(server.URL + "/nonexistent?target=es2020")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt for 4xx, got %d", attempts)
	}
}

func TestResolver_Download_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	_, err := r.downloadWithRetry(server.URL + "/pkg?target=es2020")
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
}

// ---- M7d: Integrity check fails → delete dirty cache → error ----

func TestResolver_IntegrityCheck_Fails(t *testing.T) {
	dir := t.TempDir()
	r := newTestResolver(dir)

	content := "export const w = 4;"
	cacheKey := sha256Hex("https://esm.sh/int-pkg@1.0.0")
	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])

	// Write the cache file manually.
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Write WRONG integrity.
	integrityFile := cacheFile + ".integrity"
	if err := os.WriteFile(integrityFile, []byte("badhash"), 0644); err != nil {
		t.Fatal(err)
	}

	// readCache should detect mismatch and clean up.
	got, ok := r.readCache(cacheFile)
	if ok {
		t.Fatal("expected readCache to fail with bad integrity")
	}
	if got != "" {
		t.Fatalf("expected empty string on integrity fail, got %q", got)
	}

	// Verify cache file was deleted.
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatal("expected cache file to be deleted after integrity failure")
	}
	if _, err := os.Stat(integrityFile); !os.IsNotExist(err) {
		t.Fatal("expected integrity file to be deleted after integrity failure")
	}
}

// ---- M7e: Concurrent same-key → singleflight ----

func TestResolver_Singleflight_Concurrency(t *testing.T) {
	reqCount := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		// Small delay to make concurrency visible.
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(w, "concurrent ok")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	url := server.URL + "/sf-pkg@1.0.0?target=es2020"

	// Use singleflight directly to test deduplication.
	cacheKey := sha256Hex(url)
	type result struct {
		content string
		err     error
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err, _ := r.singleflight.Do(cacheKey, func() (any, error) {
				content, dlErr := r.downloadWithRetry(url)
				if dlErr != nil {
					return result{}, dlErr
				}
				return result{content: content}, nil
			})
			if err != nil {
				errors <- err
				return
			}
			_ = v.(result).content
		}()
	}
	wg.Wait()
	close(errors)

	for err := range errors {
		t.Fatalf("concurrent download failed: %v", err)
	}

	if reqCount != 1 {
		t.Fatalf("expected 1 HTTP request with singleflight, got %d", reqCount)
	}
}

// ---- M7f: Cache corrupted → integrity mismatch → re-download ----

func TestResolver_CorruptedCache_ReDownload(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprint(w, "fresh content")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	url := server.URL + "/corrupt-pkg@1.0.0?target=es2020"
	cacheKey := sha256Hex(url)
	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])

	// Write corrupted cache.
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte("corrupted!"), 0644); err != nil {
		t.Fatal(err)
	}
	integrityFile := cacheFile + ".integrity"
	if err := os.WriteFile(integrityFile, []byte(computeSha512([]byte("original"))), 0644); err != nil {
		t.Fatal(err)
	}

	// First read: integrity mismatch → miss.
	got, ok := r.readCache(cacheFile)
	if ok {
		t.Fatal("expected cache miss on corrupted content")
	}
	if got != "" {
		t.Fatal("expected empty content for corrupted cache")
	}

	// Download should fetch fresh content.
	content, err := r.downloadWithRetry(url)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if content != "fresh content" {
		t.Fatalf("content = %q, want %q", content, "fresh content")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 download call, got %d", callCount)
	}

	// Write to cache (simulating OnLoad).
	if err := r.writeCache(cacheFile, []byte(content)); err != nil {
		t.Fatalf("writeCache failed: %v", err)
	}

	// Cache should now contain fresh content.
	got2, ok2 := r.readCache(cacheFile)
	if !ok2 {
		t.Fatal("expected cache hit after re-download")
	}
	if got2 != "fresh content" {
		t.Fatalf("cached = %q, want %q", got2, "fresh content")
	}
}

// ---- M7g: Offline mode + cache miss → error with recovery guidance ----

func TestResolver_Offline_CacheMiss(t *testing.T) {
	dir := t.TempDir()
	r := New(
		WithCacheDir(dir),
		WithTarget("es2020"),
		WithOffline(true),
	)

	url := "https://esm.sh/offline-miss@1.0.0?target=es2020"
	cacheKey := sha256Hex(url)
	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])

	// Ensure no cache.
	os.Remove(cacheFile)

	_, ok := r.readCache(cacheFile)
	if ok {
		t.Fatal("expected cache miss")
	}

	// Verify formatError produces recovery guidance.
	err := r.formatError("cache miss (offline)", "offline-miss@1.0.0", url,
		"run 'choysum install' with network access to populate the cache")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "[esm-resolver] cache miss (offline)") {
		t.Fatalf("error missing prefix: %s", errStr)
	}
	if !strings.Contains(errStr, "choysum install") {
		t.Fatalf("error missing recovery command: %s", errStr)
	}
}

func TestResolver_Offline_CacheHit(t *testing.T) {
	dir := t.TempDir()
	r := New(
		WithCacheDir(dir),
		WithTarget("es2020"),
		WithOffline(true),
	)

	content := "offline content"
	url := "https://esm.sh/offline-hit@1.0.0?target=es2020"
	cacheKey := sha256Hex(url)
	if _, err := writeCacheFile(r.codeCacheDir(), cacheKey, content); err != nil {
		t.Fatal(err)
	}

	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])
	got, ok := r.readCache(cacheFile)
	if !ok {
		t.Fatal("expected cache hit in offline mode")
	}
	if got != content {
		t.Fatalf("content = %q, want %q", got, content)
	}
}

// ---- M7h: Atomic write → no partial data ----

func TestResolver_WriteCache_Atomic(t *testing.T) {
	dir := t.TempDir()
	r := newTestResolver(dir)

	cacheKey := sha256Hex("https://esm.sh/atomic@1.0.0")
	cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])

	content := "atomic write content"
	if err := r.writeCache(cacheFile, []byte(content)); err != nil {
		t.Fatalf("writeCache failed: %v", err)
	}

	// Verify tmp file was cleaned up.
	if _, err := os.Stat(cacheFile + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("tmp file should not exist after successful write")
	}

	// Verify content.
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("content = %q, want %q", string(data), content)
	}

	// Verify integrity file.
	integrityFile := cacheFile + ".integrity"
	integrityData, err := os.ReadFile(integrityFile)
	if err != nil {
		t.Fatal(err)
	}
	expectedHash := computeSha512([]byte(content))
	if strings.TrimSpace(string(integrityData)) != expectedHash {
		t.Fatalf("integrity = %q, want %q", string(integrityData), expectedHash)
	}
}

// ---- Plugin integration tests ----

func TestResolver_Plugin_OnResolve_OnLoad_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// esm.sh returns ESM content for the requested package.
		fmt.Fprintf(w, "export const answer = 42;")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)
	plugin := r.Plugin()

	// Build a simple esbuild bundle that imports a bare specifier.
	// The OnResolve should map it to the test server, OnLoad should fetch it.
	entry := filepath.Join(dir, "entry.ts")
	if err := os.WriteFile(entry, []byte(`import { answer } from "test-lib"; console.log(answer);`), 0644); err != nil {
		t.Fatal(err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:      true,
		Write:       false,
		Plugins:     []api.Plugin{plugin},
		Platform:    api.PlatformBrowser,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("build failed: %v", result.Errors)
	}
	if len(result.OutputFiles) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(result.OutputFiles))
	}
	output := string(result.OutputFiles[0].Contents)
	if !strings.Contains(output, "answer") {
		t.Fatalf("output missing expected content: %s", output)
	}
}

func TestResolver_Plugin_CSS_External(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, ".css") {
			fmt.Fprint(w, "body { color: red; }")
			return
		}
		fmt.Fprint(w, "export const x = 1;")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	// Download a CSS URL.
	cssURL := server.URL + "/style.css?target=es2020"
	content, err := r.download(cssURL)
	if err != nil {
		t.Fatalf("download CSS failed: %v", err)
	}
	if content != "body { color: red; }" {
		t.Fatalf("CSS content = %q", content)
	}

	// Verify loaderForURL returns CSS loader.
	if got := loaderForURL(cssURL); got != api.LoaderCSS {
		t.Fatalf("loaderForURL(.css) = %v, want LoaderCSS", got)
	}
}

func TestResolver_Plugin_TS_File_Loader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "export const x: number = 1;")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
	)

	tsURL := server.URL + "/types.ts?target=es2020"
	content, err := r.download(tsURL)
	if err != nil {
		t.Fatalf("download TS failed: %v", err)
	}
	if content != "export const x: number = 1;" {
		t.Fatalf("TS content = %q", content)
	}

	if got := loaderForURL(tsURL); got != api.LoaderTS {
		t.Fatalf("loaderForURL(.ts) = %v, want LoaderTS", got)
	}
}

// ---- isCSSURL tests ----

func TestIsCSSURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://esm.sh/style.css", true},
		{"https://esm.sh/style.css?target=es2020", true},
		{"https://esm.sh/style.CSS", true},
		{"https://esm.sh/style.js", false},
		{"https://esm.sh/style.mjs", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isCSSURL(tt.url); got != tt.want {
			t.Fatalf("isCSSURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

// ---- isFragmentOnly tests ----

func TestIsFragmentOnly(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"#icon", true},
		{"#", false},
		{"", false},
		{"./file.js", false},
		{"http://example.com#frag", false},
	}
	for _, tt := range tests {
		if got := isFragmentOnly(tt.path); got != tt.want {
			t.Fatalf("isFragmentOnly(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// ---- isLocalFilesystemPath tests ----

func TestIsLocalFilesystemPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/Users/wangbuke/test", true},
		{"/home/user/project", true},
		{"/var/log/app.log", true},
		{"/tmp/foo", true},
		{"/etc/config", true},
		{"/usr/local/bin", true},
		{"/opt/app", true},
		{"/proc/cpuinfo", true},
		{"/node/process.mjs", false},        // esm.sh Node.js polyfill
		{"/node/buffer.mjs", false},         // esm.sh Node.js polyfill
		{"/pkg@ver/deno/foo.mjs", false},    // versioned esm.sh path
		{"/stable/pkg/dist/foo.mjs", false}, // esm.sh target prefix
		{"/v135/pkg/foo.mjs", false},        // esm.sh version prefix
		{"https://esm.sh/pkg", false},       // not absolute
		{"", false},
	}
	for _, tt := range tests {
		if got := isLocalFilesystemPath(tt.path); got != tt.want {
			t.Fatalf("isLocalFilesystemPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// ---- loaderForURL tests ----

func TestLoaderForURL(t *testing.T) {
	tests := []struct {
		url  string
		want api.Loader
	}{
		{"https://esm.sh/pkg.js", api.LoaderJS},
		{"https://esm.sh/pkg.mjs", api.LoaderJS},
		{"https://esm.sh/pkg.css", api.LoaderCSS},
		{"https://esm.sh/pkg.css?target=es2020", api.LoaderCSS},
		{"https://esm.sh/pkg.ts", api.LoaderTS},
		{"https://esm.sh/pkg.tsx", api.LoaderTS},
		{"https://esm.sh/pkg.mts", api.LoaderTS},
		{"https://esm.sh/pkg.ts?target=es2020", api.LoaderTS},
		{"https://esm.sh/pkg", api.LoaderJS},
	}
	for _, tt := range tests {
		if got := loaderForURL(tt.url); got != tt.want {
			t.Fatalf("loaderForURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

// ---- formatError tests ----

func TestFormatError(t *testing.T) {
	r := New(WithTarget("es2020"))
	err := r.formatError("download failed", "test-pkg@1.0.0", "https://esm.sh/test-pkg@1.0.0", "http 404: not found")
	if err == nil {
		t.Fatal("expected error")
	}
	s := err.Error()
	for _, want := range []string{"[esm-resolver]", "download failed", "test-pkg@1.0.0", "https://esm.sh/test-pkg@1.0.0", "http 404: not found"} {
		if !strings.Contains(s, want) {
			t.Fatalf("error missing %q: %s", want, s)
		}
	}
}

// ---- stripNamespace tests ----

func TestStripNamespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"choysum-esm:https://esm.sh/pkg", "https://esm.sh/pkg"},
		{"https://esm.sh/pkg", "https://esm.sh/pkg"},
		{"", ""},
		{"no-namespace", "no-namespace"},
	}
	for _, tt := range tests {
		if got := stripNamespace(tt.input); got != tt.want {
			t.Fatalf("stripNamespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---- extractPkgFromURL tests ----

func TestExtractPkgFromURL(t *testing.T) {
	tests := []struct {
		url      string
		upstream string
		want     string
	}{
		{"https://esm.sh/kysely@0.27.6?target=es2020", "https://esm.sh", "kysely@0.27.6"},
		{"https://esm.sh/@scope/pkg@1.0.0?target=es2020", "https://esm.sh", "@scope/pkg@1.0.0"},
		{"https://other.com/pkg", "https://esm.sh", ""},
		{"", "https://esm.sh", ""},
	}
	for _, tt := range tests {
		if got := extractPkgFromURL(tt.url, tt.upstream); got != tt.want {
			t.Fatalf("extractPkgFromURL(%q, %q) = %q, want %q", tt.url, tt.upstream, got, tt.want)
		}
	}
}

// ---- isBareImport tests ----

func TestIsBareImport(t *testing.T) {
	// Use a Resolver instance to access the closure via Plugin's internal logic.
	// For now, test the logic directly by creating a minimal resolver and
	// verifying through integration.
	r := New(WithTarget("es2020"))
	_ = r

	// The isBareImport closure is private, so we test indirectly through
	// the plugin's OnResolve behavior with a build.
	t.Run("bare_import_resolved", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "export const x = 1;")
		}))
		defer server.Close()

		dir := t.TempDir()
		r := New(WithUpstream(server.URL), WithCacheDir(dir), WithTarget("es2020"))
		entry := filepath.Join(dir, "entry.ts")
		if err := os.WriteFile(entry, []byte(`import { x } from "bare-lib";`), 0644); err != nil {
			t.Fatal(err)
		}
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{entry},
			Bundle:      true,
			Write:       false,
			Plugins:     []api.Plugin{r.Plugin()},
			Platform:    api.PlatformBrowser,
		})
		if len(result.Errors) > 0 {
			t.Fatalf("bare import should resolve: %v", result.Errors)
		}
	})

	t.Run("relative_import_untouched", func(t *testing.T) {
		dir := t.TempDir()
		r := New(WithCacheDir(dir), WithTarget("es2020"))
		// Create a local module.
		lib := filepath.Join(dir, "lib.ts")
		if err := os.WriteFile(lib, []byte(`export const y = 2;`), 0644); err != nil {
			t.Fatal(err)
		}
		entry := filepath.Join(dir, "entry.ts")
		if err := os.WriteFile(entry, []byte(`import { y } from "./lib";`), 0644); err != nil {
			t.Fatal(err)
		}
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{entry},
			Bundle:      true,
			Write:       false,
			Plugins:     []api.Plugin{r.Plugin()},
			Platform:    api.PlatformBrowser,
		})
		if len(result.Errors) > 0 {
			t.Fatalf("relative import should not be intercepted: %v", result.Errors)
		}
	})
}

// ---- Metrics tests ----

func TestMetrics_Snapshot(t *testing.T) {
	m := &Metrics{}
	m.CacheHit.Store(10)
	m.CacheMiss.Store(5)
	m.Downloads.Store(3)
	m.Errors.Store(1)
	m.DownloadDurationMs.Store(1500)

	hit, miss, downloads, errors, downloadMs := m.Snapshot()
	if hit != 10 || miss != 5 || downloads != 3 || errors != 1 || downloadMs != 1500 {
		t.Fatalf("snapshot = (%d,%d,%d,%d,%d), want (10,5,3,1,1500)", hit, miss, downloads, errors, downloadMs)
	}
}

func TestMetrics_Concurrency(t *testing.T) {
	m := &Metrics{}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			m.CacheHit.Add(1)
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 500; i++ {
			m.CacheMiss.Add(1)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	if m.CacheHit.Load() != 1000 {
		t.Fatalf("CacheHit = %d, want 1000", m.CacheHit.Load())
	}
	if m.CacheMiss.Load() != 500 {
		t.Fatalf("CacheMiss = %d, want 500", m.CacheMiss.Load())
	}
}

func TestResolver_WithLogger(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "export const x = 1;")
	}))
	defer server.Close()

	dir := t.TempDir()
	r := New(
		WithUpstream(server.URL),
		WithCacheDir(dir),
		WithTarget("es2020"),
		WithLogger(logger),
	)

	entry := filepath.Join(dir, "entry.ts")
	if err := os.WriteFile(entry, []byte(`import { x } from "test-pkg";`), 0644); err != nil {
		t.Fatal(err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:      true,
		Write:       false,
		Plugins:     []api.Plugin{r.Plugin()},
		Platform:    api.PlatformBrowser,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("build failed: %v", result.Errors)
	}

	// Verify metrics were logged.
	output := buf.String()
	if !strings.Contains(output, "esm resolver metrics") {
		t.Fatalf("expected metrics log, got: %s", output)
	}
	if !strings.Contains(output, "cache_miss") {
		t.Fatalf("expected cache_miss in metrics: %s", output)
	}
}
