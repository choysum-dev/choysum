// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
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
		"name":        "@choysum/addon-" + moduleName,
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

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
	tarballURL := "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-1.2.3.tgz"
	metadata := buildMetadata(t, map[string]string{"latest": "1.2.3"}, map[string]any{
		"1.2.3": map[string]any{
			"name":        "@choysum/addon-auth",
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
			"dist": map[string]any{"tarball": tarballURL},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://github.com/acme/registry", "auth", "@choysum/addon-auth", "v1.2.3")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil {
		t.Fatal("PeekManifest() returned nil module")
	}
	if mod.Name != "auth" || mod.Version != "v1.2.3" || mod.ServiceEntryPoint != "./service/main.ts" {
		t.Fatalf("unexpected module: %#v", mod)
	}
	if mod.Path != "" {
		t.Fatalf("peek should not materialize module path, got %q", mod.Path)
	}
}

func TestProviderPeekManifestUsesConfiguredDefaultNPMRegistryURL(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath, NPMRegistryURL: "https://registry.npmmirror.com/"}}

	metadataURL := "https://registry.npmmirror.com/@choysum%2Faddon-auth"
	metadata := buildMetadata(t, map[string]string{"latest": "1.0.0"}, map[string]any{
		"1.0.0": map[string]any{
			"name":    "@choysum/addon-auth",
			"version": "1.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmmirror.com/@choysum/addon-auth/-/addon-auth-1.0.0.tgz"},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "", "auth", "@choysum/addon-auth", "latest")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil || mod.Name != "auth" || mod.Version != "v1.0.0" {
		t.Fatalf("unexpected module: %#v", mod)
	}
}

func TestProviderPeekManifestResolvesLatestDistTag(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
	metadata := buildMetadata(t, map[string]string{"latest": "2.0.0"}, map[string]any{
		"1.0.0": map[string]any{
			"name":    "@choysum/addon-auth",
			"version": "1.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-1.0.0.tgz"},
		},
		"2.0.0": map[string]any{
			"name":    "@choysum/addon-auth",
			"version": "2.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-2.0.0.tgz"},
		},
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != metadataURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return httpResponse(http.StatusOK, metadata), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "latest")
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

func TestProviderFetchMaterializesModuleToAddons(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
	tarballURL := "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-2.0.0.tgz"
	metadata := buildMetadata(t, map[string]string{"latest": "2.0.0"}, map[string]any{
		"2.0.0": map[string]any{
			"name":    "@choysum/addon-auth",
			"version": "2.0.0",
			"author":  map[string]any{"name": "test"},
			"choysum": map[string]any{
				"moduleName":  "auth",
				"application": "auth",
				"entryPoints": map[string]any{"service": "./service/main.ts", "web": "./web/index.ts"},
			},
			"dist": map[string]any{"tarball": tarballURL},
		},
	})
	tgz := buildTarGz(t, map[string]string{
		"package/package.json":    buildPackageJSON(t, "auth", "2.0.0", map[string]string{"service": "./service/main.ts", "web": "./web/index.ts"}),
		"package/service/main.ts": "export const main = true;",
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
	mod, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "v2.0.0")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if mod == nil {
		t.Fatal("Fetch() returned nil module")
	}
	if mod.Name != "auth" || mod.Path != filepath.Join(addonsPath, "auth") {
		t.Fatalf("unexpected fetched module: %#v", mod)
	}
	if _, err := os.Stat(filepath.Join(addonsPath, "auth", "service", "main.ts")); err != nil {
		t.Fatalf("materialized module file missing: %v", err)
	}
}

func TestProviderPeekManifestConcurrentRequests(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
	metadata := buildMetadata(t, map[string]string{"latest": "3.0.0"}, map[string]any{
		"3.0.0": map[string]any{
			"name":    "@choysum/addon-auth",
			"version": "3.0.0",
			"choysum": map[string]any{"moduleName": "auth", "application": "auth"},
			"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-3.0.0.tgz"},
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
			mod, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "latest")
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
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.0"}, map[string]any{
			"1.0.0": map[string]any{
				"name":    "@choysum/addon-auth",
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
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "latest"); err == nil || !strings.Contains(err.Error(), "no tarball url found in npm metadata") {
			t.Fatalf("expected missing tarball url error, got %v", err)
		}
	})

	t.Run("tarball without package.json", func(t *testing.T) {
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
		tarballURL := "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-1.0.1.tgz"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.1"}, map[string]any{
			"1.0.1": map[string]any{
				"name":    "@choysum/addon-auth",
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
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "latest"); err == nil || !strings.Contains(err.Error(), "package.json not found in tarball") {
			t.Fatalf("expected package.json not found error, got %v", err)
		}
	})

	t.Run("unsafe tar path", func(t *testing.T) {
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
		tarballURL := "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-1.0.2.tgz"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.2"}, map[string]any{
			"1.0.2": map[string]any{
				"name":    "@choysum/addon-auth",
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
		if _, err := provider.Fetch(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "latest"); err == nil {
			t.Fatalf("expected unsafe tar path error")
		}
	})

	t.Run("package moduleName mismatch", func(t *testing.T) {
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		metadataURL := "https://registry.npmjs.org/@choysum%2Faddon-auth"
		metadata := buildMetadata(t, map[string]string{"latest": "1.0.3"}, map[string]any{
			"1.0.3": map[string]any{
				"name":    "@choysum/addon-auth",
				"version": "1.0.3",
				"choysum": map[string]any{"moduleName": "sale", "application": "sale"},
				"dist":    map[string]any{"tarball": "https://registry.npmjs.org/@choysum/addon-auth/-/addon-auth-1.0.3.tgz"},
			},
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != metadataURL {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}
			return httpResponse(http.StatusOK, metadata), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.PeekManifest(context.Background(), "https://registry.npmjs.org", "auth", "@choysum/addon-auth", "latest"); err == nil || !strings.Contains(err.Error(), "does not match requested module") {
			t.Fatalf("expected moduleName mismatch error, got %v", err)
		}
	})
}

func TestProviderUsesExplicitNPMPackageName(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

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
