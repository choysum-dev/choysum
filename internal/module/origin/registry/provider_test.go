// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type providerTestScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *providerTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *providerTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *providerTestScope) Session() *scope.Session { return &scope.Session{} }
func (e *providerTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *providerTestScope) Context() context.Context { return e.ctx }
func (e *providerTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *providerTestScope) Config() *config.Config { return e.cfg }

func (e *providerTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReadCloser struct {
	err error
}

func (r errReadCloser) Read([]byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return 0, io.EOF
}

func (r errReadCloser) Close() error {
	return nil
}

func httpResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func npmSHA512Integrity(data []byte) string {
	h := sha512.New()
	h.Write(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func buildPackageJSON(t *testing.T, moduleName, version string, entryPoints map[string]string) string {
	t.Helper()
	choysum := map[string]any{
		"moduleName":  moduleName,
		"application": moduleName,
	}
	if len(entryPoints) > 0 {
		choysum["entryPoints"] = entryPoints
	}
	payload := map[string]any{
		"name":        "@choysum-dev/" + moduleName,
		"version":     version,
		"description": "test module",
		"license":     "Apache-2.0",
		"author": map[string]any{
			"name": "test",
		},
		"choysum": choysum,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal package.json: %v", err)
	}
	return string(raw)
}

func buildMetadata(t *testing.T, distTags map[string]string, versions map[string]any) []byte {
	t.Helper()
	payload := map[string]any{
		"dist-tags": distTags,
		"versions":  versions,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal npm metadata: %v", err)
	}
	return raw
}

func TestProviderPeekManifestFromNPMMetadata(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.2.3.tgz"
	metadata := buildMetadata(t, map[string]string{"latest": "1.2.3"}, map[string]any{
		"1.2.3": map[string]any{
			"name":        "@choysum-dev/auth",
			"version":     "1.2.3",
			"description": "auth module",
			"license":     "Apache-2.0",
			"author": map[string]any{
				"name": "test",
			},
			"choysum": map[string]any{
				"moduleName":  "auth",
				"application": "auth",
				"entryPoints": map[string]any{"service": "./service/main.ts"},
			},
			"dist": map[string]any{"tarball": tarballURL, "integrity": "sha512-auth-v123"},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "v1.2.3")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil {
		t.Fatal("PeekManifest() returned nil module")
	}
	if mod.Name != "auth" || mod.Version != "v1.2.3" || mod.ServiceEntryPoint != "./service/main.ts" {
		t.Fatalf("unexpected module: %#v", mod)
	}
	if mod.Tarball != tarballURL || mod.Integrity != "sha512-auth-v123" {
		t.Fatalf("unexpected distribution metadata: %#v", mod)
	}
	if mod.Path != "" {
		t.Fatalf("peek should not materialize module path, got %q", mod.Path)
	}
}

func TestProviderPeekManifestUsesConfiguredDefaultNPMRegistryURL(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath, NPMRegistryURL: "https://registry.npmmirror.com/"}}

	metadataURL := "https://registry.npmmirror.com/@choysum-dev%2Fauth"
	metadata := buildMetadata(t, map[string]string{"latest": "1.0.0"}, map[string]any{
		"1.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "1.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmmirror.com/@choysum-dev/auth/-/auth-1.0.0.tgz"},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "", "auth", "@choysum-dev/auth", "latest")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil || mod.Name != "auth" || mod.Version != "v1.0.0" {
		t.Fatalf("unexpected module: %#v", mod)
	}
}

func TestProviderPeekManifestResolvesLatestDistTag(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	metadata := buildMetadata(t, map[string]string{"latest": "2.0.0"}, map[string]any{
		"1.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "1.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.0.tgz"},
		},
		"2.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "2.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum-dev/auth/-/auth-2.0.0.tgz"},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil {
		t.Fatal("PeekManifest() returned nil module")
	}
	if mod.Version != "v2.0.0" {
		t.Fatalf("unexpected resolved version: %q", mod.Version)
	}
}

func TestProviderFetchMaterializesModuleToModules(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-2.0.0.tgz"
	tgz := buildTarGz(t, map[string]string{
		"package/package.json":    buildPackageJSON(t, "auth", "2.0.0", map[string]string{"service": "./service/main.ts", "web": "./web/index.ts"}),
		"package/service/main.ts": "export const main = true;",
	})
	integrity := npmSHA512Integrity(tgz)
	metadata := buildMetadata(t, map[string]string{"latest": "2.0.0"}, map[string]any{
		"2.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "2.0.0",
			"author":  map[string]any{"name": "test"},
			"choysum": map[string]any{
				"moduleName":  "auth",
				"application": "auth",
				"entryPoints": map[string]any{"service": "./service/main.ts", "web": "./web/index.ts"},
			},
			"dist": map[string]any{"tarball": tarballURL, "integrity": integrity},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case metadataURL:
			return httpResponse(http.StatusOK, metadata), nil
		case tarballURL:
			return httpResponse(http.StatusOK, tgz), nil
		default:
			t.Fatalf("unexpected request url: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "v2.0.0")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if mod == nil {
		t.Fatal("Fetch() returned nil module")
	}
	if mod.Name != "auth" || mod.Path != filepath.Join(modulesPath, "auth") {
		t.Fatalf("unexpected fetched module: %#v", mod)
	}
	if mod.Tarball != tarballURL || mod.Integrity != integrity {
		t.Fatalf("unexpected fetched distribution metadata: %#v", mod)
	}
	if _, err := os.Stat(filepath.Join(modulesPath, "auth", "service", "main.ts")); err != nil {
		t.Fatalf("materialized module file missing: %v", err)
	}
}

func TestProviderFetchAcceptsAlternativeIntegrityDigests(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-2.1.0.tgz"
	tgz := buildTarGz(t, map[string]string{
		"package/package.json": buildPackageJSON(t, "auth", "2.1.0", map[string]string{"service": "./service/main.ts"}),
	})
	integrity := npmSHA512Integrity([]byte("different-content")) + " " + npmSHA512Integrity(tgz)
	metadata := buildMetadata(t, map[string]string{"latest": "2.1.0"}, map[string]any{
		"2.1.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "2.1.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth", "entryPoints": map[string]any{"service": "./service/main.ts"}},
			"dist":    map[string]any{"tarball": tarballURL, "integrity": integrity},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case metadataURL:
			return httpResponse(http.StatusOK, metadata), nil
		case tarballURL:
			return httpResponse(http.StatusOK, tgz), nil
		default:
			t.Fatalf("unexpected request url: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "v2.1.0")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if mod == nil || mod.Name != "auth" || mod.Version != "v2.1.0" {
		t.Fatalf("unexpected module: %#v", mod)
	}
}

func TestProviderPeekManifestConcurrentRequests(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	metadata := buildMetadata(t, map[string]string{"latest": "3.0.0"}, map[string]any{
		"3.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "3.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum-dev/auth/-/auth-3.0.0.tgz"},
		},
	})

	var metadataHits int64
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			return nil, fmt.Errorf("unexpected request url: %s", req.URL.String())
		}
		atomic.AddInt64(&metadataHits, 1)
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	const workers = 8
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mod, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest")
			if err != nil {
				errCh <- err
				return
			}
			if mod == nil || mod.Name != "auth" || mod.Version != "v3.0.0" {
				errCh <- fmt.Errorf("unexpected module: %#v", mod)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent PeekManifest failed: %v", err)
		}
	}
	if atomic.LoadInt64(&metadataHits) != workers {
		t.Fatalf("unexpected metadata request count: got=%d want=%d", metadataHits, workers)
	}
}

func TestProviderFetchErrorScenarios(t *testing.T) {
	t.Parallel()

	t.Run("missing tarball url in npm metadata", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.0"}, map[string]any{
			"1.0.0": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.0",
				"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
				"dist":    map[string]any{},
			},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != metadataURL {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}
			return httpResponse(http.StatusOK, metadata), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "no tarball url found in npm metadata") {
			t.Fatalf("expected missing tarball url error, got %v", err)
		}
	})

	t.Run("tarball without package.json", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.1.tgz"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.1"}, map[string]any{
			"1.0.1": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.1",
				"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
				"dist":    map[string]any{"tarball": tarballURL},
			},
		})
		tgz := buildTarGz(t, map[string]string{"package/README.md": "no package"})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case metadataURL:
				return httpResponse(http.StatusOK, metadata), nil
			case tarballURL:
				return httpResponse(http.StatusOK, tgz), nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "package.json not found in tarball") {
			t.Fatalf("expected package.json not found error, got %v", err)
		}
	})

	t.Run("oversized tarball fails fetch with explicit limit error", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.6.tgz"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.6"}, map[string]any{
			"1.0.6": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.6",
				"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
				"dist":    map[string]any{"tarball": tarballURL},
			},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case metadataURL:
				return httpResponse(http.StatusOK, metadata), nil
			case tarballURL:
				return &http.Response{
					StatusCode:    http.StatusOK,
					Status:        "200 OK",
					Header:        make(http.Header),
					Body:          io.NopCloser(bytes.NewReader([]byte("small-body"))),
					ContentLength: maxTarballSizeBytes + 1,
				}, nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "tarball size exceeds maximum limit") {
			t.Fatalf("expected explicit tarball size limit error, got %v", err)
		}
	})

	t.Run("malformed integrity metadata fails fetch", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.4.tgz"
		tgz := buildTarGz(t, map[string]string{
			"package/package.json": buildPackageJSON(t, "auth", "1.0.4", nil),
		})
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.4"}, map[string]any{
			"1.0.4": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.4",
				"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
				"dist":    map[string]any{"tarball": tarballURL, "integrity": "sha512-not-base64@@"},
			},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case metadataURL:
				return httpResponse(http.StatusOK, metadata), nil
			case tarballURL:
				return httpResponse(http.StatusOK, tgz), nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "tarball integrity check failed") {
			t.Fatalf("expected malformed integrity error, got %v", err)
		}
	})

	t.Run("integrity mismatch fails fetch", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.5.tgz"
		tgz := buildTarGz(t, map[string]string{
			"package/package.json": buildPackageJSON(t, "auth", "1.0.5", nil),
		})
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.5"}, map[string]any{
			"1.0.5": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.5",
				"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
				"dist":    map[string]any{"tarball": tarballURL, "integrity": npmSHA512Integrity([]byte("different-content"))},
			},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case metadataURL:
				return httpResponse(http.StatusOK, metadata), nil
			case tarballURL:
				return httpResponse(http.StatusOK, tgz), nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "integrity mismatch") {
			t.Fatalf("expected integrity mismatch error, got %v", err)
		}
	})

	t.Run("unsafe tar path", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.2.tgz"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.2"}, map[string]any{
			"1.0.2": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.2",
				"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
				"dist":    map[string]any{"tarball": tarballURL},
			},
		})
		tgz := buildTarGz(t, map[string]string{"../package/package.json": buildPackageJSON(t, "auth", "1.0.2", nil)})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case metadataURL:
				return httpResponse(http.StatusOK, metadata), nil
			case tarballURL:
				return httpResponse(http.StatusOK, tgz), nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil {
			t.Fatalf("expected unsafe tar path error")
		}
	})

	t.Run("package moduleName mismatch", func(t *testing.T) {
		modulesPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.3"}, map[string]any{
			"1.0.3": map[string]any{
				"name":    "@choysum-dev/auth",
				"version": "1.0.3",
				"choysum": map[string]any{"moduleName": "sale", "application": "sale"},
				"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.3.tgz"},
			},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != metadataURL {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}
			return httpResponse(http.StatusOK, metadata), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "does not match requested module") {
			t.Fatalf("expected moduleName mismatch error, got %v", err)
		}
	})
}

func TestProviderUsesExplicitNPMPackageName(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@acme%2Fchoysum-sale"
	metadata := buildMetadata(t, map[string]string{"latest": "0.1.0"}, map[string]any{
		"0.1.0": map[string]any{
			"name":    "@acme/choysum-sale",
			"version": "0.1.0",
			"choysum": map[string]any{"moduleName": "sale", "application": "sale"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@acme/choysum-sale/-/choysum-sale-0.1.0.tgz"},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "sale", "@acme/choysum-sale", "latest")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil || mod.Name != "sale" || mod.Version != "v0.1.0" {
		t.Fatalf("unexpected module: %#v", mod)
	}
}

func TestProviderRejectsNilRuntimeScope(t *testing.T) {
	t.Parallel()

	var nilProvider *SourceRegistryProvider
	if got := nilProvider.defaultRegistryURL(); got != config.DefaultNPMRegistryURL {
		t.Fatalf("defaultRegistryURL(nil) = %q, want %q", got, config.DefaultNPMRegistryURL)
	}

	if _, err := nilProvider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "registry provider env is nil") {
		t.Fatalf("expected nil provider guard error from PeekManifest, got %v", err)
	}
	if _, err := nilProvider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "registry provider env is nil") {
		t.Fatalf("expected nil provider guard error from Fetch, got %v", err)
	}

	emptyProvider := &SourceRegistryProvider{}
	if _, err := emptyProvider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "registry provider env is nil") {
		t.Fatalf("expected empty provider guard error from PeekManifest, got %v", err)
	}
}

func TestProviderFetchRequiresModulesPath(t *testing.T) {
	t.Parallel()

	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: ""}}
	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.0.tgz"
	metadata := buildMetadata(t, map[string]string{"latest": "1.0.0"}, map[string]any{
		"1.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "1.0.0",
			"author":  map[string]any{"name": "test"},
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": tarballURL},
		},
	})
	tgz := buildTarGz(t, map[string]string{
		"package/package.json": buildPackageJSON(t, "auth", "1.0.0", nil),
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case metadataURL:
			return httpResponse(http.StatusOK, metadata), nil
		case tarballURL:
			return httpResponse(http.StatusOK, tgz), nil
		default:
			t.Fatalf("unexpected request url: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "modules path is empty") {
		t.Fatalf("expected modules path required error, got %v", err)
	}
}

func TestRegistryHostForLogFallsBackToDefaultAndTarball(t *testing.T) {
	t.Parallel()

	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), NPMRegistryURL: "https://registry.npmmirror.com/"}}
	provider := NewProvider(runtimeScope)

	if got := provider.registryHostForLog("https://registry.npmjs.org", "https://registry.npmmirror.com/@choysum-dev/auth/-/auth-1.0.0.tgz"); got != "registry.npmjs.org" {
		t.Fatalf("registryHostForLog(explicit) = %q, want registry.npmjs.org", got)
	}
	if got := provider.registryHostForLog("", "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.0.tgz"); got != "registry.npmmirror.com" {
		t.Fatalf("registryHostForLog(default) = %q, want registry.npmmirror.com", got)
	}

	runtimeScope.cfg.NPMRegistryURL = ""
	if got := provider.registryHostForLog("", "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.0.tgz"); got != "registry.npmjs.org" {
		t.Fatalf("registryHostForLog(tarball fallback) = %q, want registry.npmjs.org", got)
	}
}

func TestProviderFetchCopyModuleToModulesFailure(t *testing.T) {
	t.Parallel()

	modulesPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath}}

	metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
	tarballURL := "https://registry.npmjs.org/@choysum-dev/auth/-/auth-1.0.0.tgz"
	metadata := buildMetadata(t, map[string]string{"latest": "1.0.0"}, map[string]any{
		"1.0.0": map[string]any{
			"name":    "@choysum-dev/auth",
			"version": "1.0.0",
			"author":  map[string]any{"name": "test"},
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": tarballURL},
		},
	})
	tgz := buildTarGz(t, map[string]string{
		"package/package.json": buildPackageJSON(t, "auth", "1.0.0", nil),
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case metadataURL:
			return httpResponse(http.StatusOK, metadata), nil
		case tarballURL:
			return httpResponse(http.StatusOK, tgz), nil
		default:
			t.Fatalf("unexpected request url: %s", req.URL.String())
			return nil, nil
		}
	})}

	readOnlyRoot := filepath.Join(modulesPath, "readonly")
	if err := os.MkdirAll(readOnlyRoot, 0o755); err != nil {
		t.Fatalf("create read-only modules root: %v", err)
	}
	if err := os.Chmod(readOnlyRoot, 0o500); err != nil {
		t.Fatalf("chmod read-only modules root: %v", err)
	}
	defer func() {
		_ = os.Chmod(readOnlyRoot, 0o755)
	}()
	runtimeScope.cfg.ModulesPath = readOnlyRoot

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "copy module to modules failed") {
		t.Fatalf("expected copy module to modules failure, got %v", err)
	}
}

func TestProviderHelperNormalizationBranches(t *testing.T) {
	t.Parallel()

	moduleName, version, err := normalizeModuleNameVersion(" auth ", "")
	if err != nil {
		t.Fatalf("normalizeModuleNameVersion() error = %v", err)
	}
	if moduleName != "auth" || version != "latest" {
		t.Fatalf("unexpected normalizeModuleNameVersion result: module=%q version=%q", moduleName, version)
	}

	if _, _, err := normalizeModuleNameVersion("   ", "v1.0.0"); err == nil {
		t.Fatal("expected empty module name error")
	}

	if pkg, err := normalizePackageName("auth", ""); err != nil || pkg != "auth" {
		t.Fatalf("normalizePackageName fallback = %q err=%v, want auth nil", pkg, err)
	}
	if _, err := normalizePackageName("", "@acme/choysum-auth"); err == nil {
		t.Fatal("expected empty module name error from normalizePackageName")
	}

	base, err := normalizeDefaultRegistryMetadataBaseURL("https://registry.npmjs.org/")
	if err != nil {
		t.Fatalf("normalizeDefaultRegistryMetadataBaseURL() error = %v", err)
	}
	if base != "https://registry.npmjs.org" {
		t.Fatalf("default registry base = %q, want %q", base, "https://registry.npmjs.org")
	}
	if _, err := normalizeDefaultRegistryMetadataBaseURL("ftp://registry.npmjs.org"); err == nil {
		t.Fatal("expected unsupported default registry scheme error")
	}
	if _, err := normalizeDefaultRegistryMetadataBaseURL("https://"); err == nil {
		t.Fatal("expected unsupported default registry host error")
	}

	fallback, err := normalizeRegistryMetadataBaseURL("", "https://registry.npmjs.org/")
	if err != nil {
		t.Fatalf("normalizeRegistryMetadataBaseURL(empty) error = %v", err)
	}
	if fallback != "https://registry.npmjs.org" {
		t.Fatalf("fallback registry base = %q, want %q", fallback, "https://registry.npmjs.org")
	}

	assertLegacyRegistrySourceRejected := func(registryURL string) {
		t.Helper()
		_, err := normalizeRegistryMetadataBaseURL(registryURL, "https://registry.npmjs.org")
		if err == nil {
			t.Fatalf("normalizeRegistryMetadataBaseURL(%s): expected legacy source rejection", registryURL)
		}
		if !strings.Contains(err.Error(), "no longer supported") {
			t.Fatalf("normalizeRegistryMetadataBaseURL(%s): expected no longer supported guidance, got %v", registryURL, err)
		}
		if !strings.Contains(err.Error(), "module_catalog_index_url") {
			t.Fatalf("normalizeRegistryMetadataBaseURL(%s): expected module_catalog_index_url guidance, got %v", registryURL, err)
		}
	}

	assertLegacyRegistrySourceRejected("https://github.com/acme/catalog")
	assertLegacyRegistrySourceRejected("https://www.github.com/acme/catalog")
	assertLegacyRegistrySourceRejected("https://catalog.acme.dev/api/v1/modules")
	assertLegacyRegistrySourceRejected("https://catalog.acme.dev/api/modules")
	assertLegacyRegistrySourceRejected("https://index.acme.dev/v1/index.json")

	custom, err := normalizeRegistryMetadataBaseURL("https://registry.acme.dev/custom/", "https://registry.npmjs.org")
	if err != nil {
		t.Fatalf("normalizeRegistryMetadataBaseURL(custom) error = %v", err)
	}
	if custom != "https://registry.acme.dev/custom" {
		t.Fatalf("custom registry base = %q, want %q", custom, "https://registry.acme.dev/custom")
	}

	artifactory, err := normalizeRegistryMetadataBaseURL("https://acme.jfrog.io/artifactory/api/npm/npm-virtual/", "https://registry.npmjs.org")
	if err != nil {
		t.Fatalf("normalizeRegistryMetadataBaseURL(artifactory) error = %v", err)
	}
	if artifactory != "https://acme.jfrog.io/artifactory/api/npm/npm-virtual" {
		t.Fatalf("artifactory registry base = %q, want %q", artifactory, "https://acme.jfrog.io/artifactory/api/npm/npm-virtual")
	}

	githubPackages, err := normalizeRegistryMetadataBaseURL("https://npm.pkg.github.com", "https://registry.npmjs.org")
	if err != nil {
		t.Fatalf("normalizeRegistryMetadataBaseURL(npm.pkg.github.com) error = %v", err)
	}
	if githubPackages != "https://npm.pkg.github.com" {
		t.Fatalf("github packages registry base = %q, want %q", githubPackages, "https://npm.pkg.github.com")
	}

	metadataURL, pkgName, err := registryPackageMetadataURL("https://registry.acme.dev", "auth", "@acme/choysum-auth", config.DefaultNPMRegistryURL)
	if err != nil {
		t.Fatalf("registryPackageMetadataURL() error = %v", err)
	}
	if pkgName != "@acme/choysum-auth" {
		t.Fatalf("resolved package name = %q, want %q", pkgName, "@acme/choysum-auth")
	}
	if metadataURL != "https://registry.acme.dev/@acme%2Fchoysum-auth" {
		t.Fatalf("metadata URL = %q, want %q", metadataURL, "https://registry.acme.dev/@acme%2Fchoysum-auth")
	}

	if _, _, err := registryPackageMetadataURL("https://registry.acme.dev", "", "", config.DefaultNPMRegistryURL); err == nil {
		t.Fatal("expected empty module name error from registryPackageMetadataURL")
	}
}

func TestProviderAdditionalHelperBranches(t *testing.T) {
	t.Parallel()

	if got, err := normalizeDefaultRegistryMetadataBaseURL("   "); err != nil || got != config.DefaultNPMRegistryURL {
		t.Fatalf("normalizeDefaultRegistryMetadataBaseURL(blank) = %q err=%v, want %q nil", got, err, config.DefaultNPMRegistryURL)
	}
	if _, err := normalizeDefaultRegistryMetadataBaseURL("https://%zz"); err == nil || !strings.Contains(err.Error(), "invalid default npm registry url") {
		t.Fatalf("expected invalid default registry URL parse error, got %v", err)
	}
	if _, err := normalizeRegistryMetadataBaseURL("https://registry.npmjs.org", "https://%zz"); err == nil || !strings.Contains(err.Error(), "invalid default npm registry url") {
		t.Fatalf("expected default registry normalization error, got %v", err)
	}
	if _, err := normalizeRegistryMetadataBaseURL("https://%zz", config.DefaultNPMRegistryURL); err == nil || !strings.Contains(err.Error(), "invalid registry url") {
		t.Fatalf("expected invalid registry URL parse error, got %v", err)
	}
	if _, err := normalizeRegistryMetadataBaseURL("https://", config.DefaultNPMRegistryURL); err == nil || !strings.Contains(err.Error(), "unsupported registry url") {
		t.Fatalf("expected unsupported registry host error, got %v", err)
	}
	if _, _, err := registryPackageMetadataURL("https://registry.npmjs.org", "auth", "", "https://%zz"); err == nil || !strings.Contains(err.Error(), "invalid default npm registry url") {
		t.Fatalf("expected registryPackageMetadataURL default registry error, got %v", err)
	}

	rawLatest := json.RawMessage(`{"name":"pkg","version":"1.0.0"}`)
	metadata := &npmPackageMetadata{
		DistTags: map[string]string{"latest": "1.0.0"},
		Versions: map[string]json.RawMessage{"1.0.0": rawLatest},
	}
	if gotVersion, gotRaw, err := resolveNPMVersion(metadata, "   "); err != nil || gotVersion != "1.0.0" || string(gotRaw) != string(rawLatest) {
		t.Fatalf("resolveNPMVersion(blank requested version) got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	normalized, err := normalizePackageAuthor([]byte(`{"name":"pkg","author":123}`))
	if err != nil {
		t.Fatalf("normalizePackageAuthor(non-string author) error = %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(normalized, &payload); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}
	if _, ok := payload["author"]; ok {
		t.Fatalf("expected non-string author to be removed, got payload=%#v", payload)
	}

	if _, err := parseModuleFromPackageJSON([]byte(`{"name":`), "auth", filepath.Join(t.TempDir(), "auth")); err == nil || !strings.Contains(err.Error(), "decode package.json payload") {
		t.Fatalf("expected parseModuleFromPackageJSON decode error, got %v", err)
	}
	if _, _, err := extractTarballURL(json.RawMessage(`{"dist":`)); err == nil || !strings.Contains(err.Error(), "decode npm version dist") {
		t.Fatalf("expected extractTarballURL decode error, got %v", err)
	}

	if scorePackageJSONPath("/tmp/modules/auth/package.json", "auth") <= scorePackageJSONPath("package/package.json", "auth") {
		t.Fatal("expected /tmp/modules/auth/package.json to receive higher score than package/package.json")
	}
}

func TestProviderInspectRegistryPackageErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("invalid default registry configured in runtime scope", func(t *testing.T) {
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), NPMRegistryURL: "https://%zz"}}
		provider := NewProvider(runtimeScope)

		if _, _, err := provider.fetchPackageMetadata(context.Background(), "", "auth", ""); err == nil || !strings.Contains(err.Error(), "invalid default npm registry url") {
			t.Fatalf("expected invalid default registry URL error, got %v", err)
		}
	})

	t.Run("metadata load failure bubbles up from inspect", func(t *testing.T) {
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir()}}
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))

		if _, err := provider.inspectRegistryPackage(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "get npm metadata") {
			t.Fatalf("expected inspectRegistryPackage metadata error, got %v", err)
		}
	})

	t.Run("version resolution failure is wrapped by inspect", func(t *testing.T) {
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir()}}
		metadataURL := "https://registry.npmjs.org/@choysum-dev%2Fauth"
		metadata := buildMetadata(t, map[string]string{}, map[string]any{
			"1.0.0": map[string]any{"name": "@choysum-dev/auth", "version": "1.0.0"},
			"2.0.0": map[string]any{"name": "@choysum-dev/auth", "version": "2.0.0"},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != metadataURL {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}
			return httpResponse(http.StatusOK, metadata), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))

		if _, err := provider.inspectRegistryPackage(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth", "latest"); err == nil || !strings.Contains(err.Error(), "inspect package") || !strings.Contains(err.Error(), "missing latest dist-tag") {
			t.Fatalf("expected inspectRegistryPackage version resolution error, got %v", err)
		}
	})
}

func TestProviderResolveNPMVersionBranches(t *testing.T) {
	t.Parallel()

	rawV100 := json.RawMessage(`{"name":"pkg","version":"1.0.0"}`)
	rawV200 := json.RawMessage(`{"name":"pkg","version":"2.0.0"}`)
	rawV300rc := json.RawMessage(`{"name":"pkg","version":"3.0.0-rc.1"}`)

	metadata := &npmPackageMetadata{
		DistTags: map[string]string{"latest": "v2.0.0", "rc": "v3.0.0-rc.1"},
		Versions: map[string]json.RawMessage{
			"1.0.0":      rawV100,
			"2.0.0":      rawV200,
			"3.0.0-rc.1": rawV300rc,
		},
	}

	if gotVersion, gotRaw, err := resolveNPMVersion(metadata, "1.0.0"); err != nil || gotVersion != "1.0.0" || string(gotRaw) != string(rawV100) {
		t.Fatalf("direct version resolve got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	if gotVersion, gotRaw, err := resolveNPMVersion(metadata, "v1.0.0"); err != nil || gotVersion != "1.0.0" || string(gotRaw) != string(rawV100) {
		t.Fatalf("v-prefixed version resolve got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	if gotVersion, gotRaw, err := resolveNPMVersion(metadata, "rc"); err != nil || gotVersion != "3.0.0-rc.1" || string(gotRaw) != string(rawV300rc) {
		t.Fatalf("dist-tag resolve got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	if gotVersion, gotRaw, err := resolveNPMVersion(metadata, "latest"); err != nil || gotVersion != "2.0.0" || string(gotRaw) != string(rawV200) {
		t.Fatalf("latest dist-tag v-prefixed resolve got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	metadataLatestPlain := &npmPackageMetadata{
		DistTags: map[string]string{"latest": "2.0.0"},
		Versions: map[string]json.RawMessage{"2.0.0": rawV200},
	}
	if gotVersion, gotRaw, err := resolveNPMVersion(metadataLatestPlain, "latest"); err != nil || gotVersion != "2.0.0" || string(gotRaw) != string(rawV200) {
		t.Fatalf("latest dist-tag plain resolve got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	singleVersion := &npmPackageMetadata{Versions: map[string]json.RawMessage{"0.9.0": rawV100}}
	if gotVersion, gotRaw, err := resolveNPMVersion(singleVersion, "latest"); err != nil || gotVersion != "0.9.0" || string(gotRaw) != string(rawV100) {
		t.Fatalf("single-version latest fallback got version=%q raw=%s err=%v", gotVersion, string(gotRaw), err)
	}

	multiWithoutLatest := &npmPackageMetadata{Versions: map[string]json.RawMessage{"1.0.0": rawV100, "2.0.0": rawV200}}
	if _, _, err := resolveNPMVersion(multiWithoutLatest, "latest"); err == nil || !strings.Contains(err.Error(), "missing latest dist-tag") {
		t.Fatalf("expected latest dist-tag missing error, got %v", err)
	}

	if _, _, err := resolveNPMVersion(metadata, "9.9.9"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected version not found error, got %v", err)
	}

	if _, _, err := resolveNPMVersion(nil, "latest"); err == nil || !strings.Contains(err.Error(), "metadata is nil") {
		t.Fatalf("expected nil metadata error, got %v", err)
	}
}

func TestProviderNormalizePackageAuthorAndParseModule(t *testing.T) {
	t.Parallel()

	decodeAuthor := func(raw []byte) any {
		t.Helper()
		payload := map[string]any{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("unmarshal normalized payload: %v", err)
		}
		return payload["author"]
	}

	trimmedString, err := normalizePackageAuthor([]byte(`{"name":"@acme/choysum-auth","author":"  Alice  "}`))
	if err != nil {
		t.Fatalf("normalizePackageAuthor(string) error = %v", err)
	}
	if author := decodeAuthor(trimmedString); author != "Alice" {
		t.Fatalf("normalized string author = %#v, want %q", author, "Alice")
	}

	trimmedMap, err := normalizePackageAuthor([]byte(`{"name":"@acme/choysum-auth","author":{"name":"  Bob  ","email":"bob@example.com"}}`))
	if err != nil {
		t.Fatalf("normalizePackageAuthor(map) error = %v", err)
	}
	if author := decodeAuthor(trimmedMap); author != "Bob" {
		t.Fatalf("normalized map author = %#v, want %q", author, "Bob")
	}

	removedInvalidAuthor, err := normalizePackageAuthor([]byte(`{"name":"@acme/choysum-auth","author":{"email":"bob@example.com"}}`))
	if err != nil {
		t.Fatalf("normalizePackageAuthor(invalid map) error = %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(removedInvalidAuthor, &payload); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}
	if _, ok := payload["author"]; ok {
		t.Fatalf("expected invalid author field to be removed, got payload=%#v", payload)
	}

	if _, err := normalizePackageAuthor([]byte(`{"author":`)); err == nil {
		t.Fatal("expected normalizePackageAuthor decode error")
	}

	modulePath := filepath.Join(t.TempDir(), "auth")
	raw := []byte(buildPackageJSON(t, "auth", "1.0.0", map[string]string{"service": "./service/main.ts"}))
	module, err := parseModuleFromPackageJSON(raw, "auth", modulePath)
	if err != nil {
		t.Fatalf("parseModuleFromPackageJSON(success) error = %v", err)
	}
	if module == nil || module.Name != "auth" || module.Path != modulePath {
		t.Fatalf("unexpected parsed module: %#v", module)
	}

	mismatchRaw := []byte(buildPackageJSON(t, "sale", "1.0.0", nil))
	if _, err := parseModuleFromPackageJSON(mismatchRaw, "auth", modulePath); err == nil || !strings.Contains(err.Error(), "does not match requested module") {
		t.Fatalf("expected module mismatch error, got %v", err)
	}

	invalidContractRaw := []byte(`{"name":"@acme/choysum-auth","version":"1.0.0","choysum":{"moduleName":"","application":"auth"}}`)
	if _, err := parseModuleFromPackageJSON(invalidContractRaw, "auth", modulePath); err == nil || !strings.Contains(err.Error(), "parse package.json") {
		t.Fatalf("expected parse package.json contract error, got %v", err)
	}
}

func TestProviderPathAndTarballHelpers(t *testing.T) {
	t.Parallel()

	if !isUnsafeTarPath("") {
		t.Fatal("empty tar path should be unsafe")
	}
	if !isUnsafeTarPath("../escape") {
		t.Fatal("parent traversal tar path should be unsafe")
	}
	if isUnsafeTarPath("package/package.json") {
		t.Fatal("relative package path should be safe")
	}

	baseDir := t.TempDir()
	joined, err := safeJoin(baseDir, "package/package.json")
	if err != nil {
		t.Fatalf("safeJoin(valid) error = %v", err)
	}
	if joined != filepath.Join(baseDir, "package", "package.json") {
		t.Fatalf("safeJoin(valid) = %q, want %q", joined, filepath.Join(baseDir, "package", "package.json"))
	}
	if _, err := safeJoin(baseDir, "../outside"); err == nil {
		t.Fatal("expected safeJoin traversal error")
	}

	if err := validateTarballURL("https://registry.npmjs.org/pkg/-/pkg-1.0.0.tgz"); err != nil {
		t.Fatalf("validateTarballURL(valid) error = %v", err)
	}
	if err := validateTarballURL("ftp://registry.npmjs.org/pkg/-/pkg-1.0.0.tgz"); err == nil {
		t.Fatal("expected invalid tarball scheme error")
	}
	if err := validateTarballURL("https://registry.npmjs.org/pkg/-/pkg-1.0.0.zip"); err == nil {
		t.Fatal("expected invalid tarball file type error")
	}

	if scorePackageJSONPath("modules/auth/package.json", "auth") <= scorePackageJSONPath("package/package.json", "auth") {
		t.Fatal("expected modules/auth/package.json to score higher than package/package.json")
	}

	rootDir := t.TempDir()
	for _, rel := range []string{"package/package.json", "modules/auth/package.json", "auth/package.json"} {
		abs := filepath.Join(rootDir, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(`{"name":"pkg"}`), 0o644); err != nil {
			t.Fatalf("write %q: %v", rel, err)
		}
	}

	bestPath, err := findBestPackageJSONPath(rootDir, "auth")
	if err != nil {
		t.Fatalf("findBestPackageJSONPath() error = %v", err)
	}
	expectedPath := filepath.Join(rootDir, "modules", "auth", "package.json")
	if filepath.Clean(bestPath) != filepath.Clean(expectedPath) {
		t.Fatalf("best package path = %q, want %q", bestPath, expectedPath)
	}

	if _, err := findBestPackageJSONPath(t.TempDir(), "auth"); err == nil || !strings.Contains(err.Error(), "package.json not found") {
		t.Fatalf("expected package.json not found error, got %v", err)
	}
}

func TestProviderFetchPackageMetadataErrors(t *testing.T) {
	t.Parallel()

	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir()}}

	t.Run("invalid registry url", func(t *testing.T) {
		provider := NewProvider(runtimeScope)
		if _, _, err := provider.fetchPackageMetadata(context.Background(), "ftp://registry.acme.dev", "auth", ""); err == nil {
			t.Fatal("expected invalid registry URL error")
		}
	})

	t.Run("metadata request non-200", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return httpResponse(http.StatusBadGateway, []byte(`{"error":"upstream"}`)), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, _, err := provider.fetchPackageMetadata(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth"); err == nil || !strings.Contains(err.Error(), "status code") {
			t.Fatalf("expected metadata non-200 error, got %v", err)
		}
	})

	t.Run("metadata request transport error", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, _, err := provider.fetchPackageMetadata(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth"); err == nil || !strings.Contains(err.Error(), "get npm metadata") {
			t.Fatalf("expected metadata transport error, got %v", err)
		}
	})

	t.Run("metadata response body read error", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       errReadCloser{err: errors.New("read failed")},
			}, nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, _, err := provider.fetchPackageMetadata(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth"); err == nil || !strings.Contains(err.Error(), "read npm metadata") {
			t.Fatalf("expected metadata read error, got %v", err)
		}
	})

	t.Run("invalid metadata payload", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return httpResponse(http.StatusOK, []byte(`{"dist-tags":`)), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, _, err := provider.fetchPackageMetadata(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth"); err == nil || !strings.Contains(err.Error(), "decoding npm metadata") {
			t.Fatalf("expected metadata decode error, got %v", err)
		}
	})

	t.Run("metadata has no versions", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return httpResponse(http.StatusOK, []byte(`{"dist-tags":{"latest":"1.0.0"},"versions":{}}`)), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, _, err := provider.fetchPackageMetadata(context.Background(), "https://registry.npmjs.org", "auth", "@choysum-dev/auth"); err == nil || !strings.Contains(err.Error(), "no versions") {
			t.Fatalf("expected metadata no versions error, got %v", err)
		}
	})
}
