// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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

func TestProviderPeekManifestFromRegistryDirect(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v1.2.3/manifest.json"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != manifestURL {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		body := []byte(`{"application":"auth","version":"1.2.3","entryPoints":{"service":"service/main.ts"}}`)
		return httpResponse(http.StatusOK, body), nil
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://github.com/acme/registry", "auth", "v1.2.3")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil {
		t.Fatal("PeekManifest() returned nil module")
	}
	if mod.Name != "auth" || mod.Version != "v1.2.3" || mod.ServiceEntryPoint != "service/main.ts" {
		t.Fatalf("unexpected module: %#v", mod)
	}
	if mod.Path != "" {
		t.Fatalf("peek should not materialize module path, got %q", mod.Path)
	}
}

func TestProviderPeekManifestFromTarball(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v1.0.0/manifest.json"
	tarballURL := "https://github.com/acme/auth/archive/refs/tags/v1.0.0.tar.gz"
	tgz := buildTarGz(t, map[string]string{
		"auth-v1.0.0/addons/auth/manifest.json": `{"application":"auth","version":"v1.0.0","entryPoints":{"web":"web/app.ts"}}`,
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case manifestURL:
			return httpResponse(http.StatusOK, []byte(`{"tarball":"`+tarballURL+`"}`)), nil
		case tarballURL:
			return httpResponse(http.StatusOK, tgz), nil
		default:
			t.Fatalf("unexpected request url: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.PeekManifest(context.Background(), "https://github.com/acme/registry", "auth", "v1.0.0")
	if err != nil {
		t.Fatalf("PeekManifest() error = %v", err)
	}
	if mod == nil {
		t.Fatal("PeekManifest() returned nil module")
	}
	if mod.Name != "auth" || mod.Version != "v1.0.0" || mod.WebEntryPoint != "web/app.ts" {
		t.Fatalf("unexpected module from tarball: %#v", mod)
	}
}

func TestProviderFetchMaterializesModuleToAddons(t *testing.T) {
	t.Parallel()

	addonsPath := t.TempDir()
	runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v2.0.0/manifest.json"
	tarballURL := "https://github.com/acme/auth/archive/refs/tags/v2.0.0.tar.gz"
	tgz := buildTarGz(t, map[string]string{
		"auth-v2.0.0/addons/auth/manifest.json":   `{"application":"auth","version":"v2.0.0"}`,
		"auth-v2.0.0/addons/auth/service/main.ts": `export const main = true;`,
	})

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case manifestURL:
			return httpResponse(http.StatusOK, []byte(`{"tarball":"`+tarballURL+`"}`)), nil
		case tarballURL:
			return httpResponse(http.StatusOK, tgz), nil
		default:
			t.Fatalf("unexpected request url: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	mod, err := provider.Fetch(context.Background(), "https://github.com/acme/registry", "auth", "v2.0.0")
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

	manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v3.0.0/manifest.json"
	tarballURL := "https://github.com/acme/auth/archive/refs/tags/v3.0.0.tar.gz"
	tgz := buildTarGz(t, map[string]string{
		"auth-v3.0.0/addons/auth/manifest.json": `{"application":"auth","version":"v3.0.0"}`,
	})

	var manifestHits int64
	var tarballHits int64
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case manifestURL:
			atomic.AddInt64(&manifestHits, 1)
			return httpResponse(http.StatusOK, []byte(`{"tarball":"`+tarballURL+`"}`)), nil
		case tarballURL:
			atomic.AddInt64(&tarballHits, 1)
			return httpResponse(http.StatusOK, tgz), nil
		default:
			return nil, fmt.Errorf("unexpected request url: %s", req.URL.String())
		}
	})}

	provider := NewProvider(runtimeScope, WithHTTPClient(client))
	const workers = 8
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mod, err := provider.PeekManifest(context.Background(), "https://github.com/acme/registry", "auth", "v3.0.0")
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
	if atomic.LoadInt64(&manifestHits) != workers {
		t.Fatalf("unexpected manifest request count: got=%d want=%d", manifestHits, workers)
	}
	if atomic.LoadInt64(&tarballHits) != workers {
		t.Fatalf("unexpected tarball request count: got=%d want=%d", tarballHits, workers)
	}
}

func TestProviderFetchErrorScenarios(t *testing.T) {
	t.Parallel()

	t.Run("missing download url in registry manifest", func(t *testing.T) {
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v1.0.0/manifest.json"
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != manifestURL {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}
			return httpResponse(http.StatusOK, []byte(`{"name":"auth"}`)), nil
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://github.com/acme/registry", "auth", "v1.0.0"); err == nil {
			t.Fatalf("expected error for missing download url")
		}
	})

	t.Run("tarball without manifest", func(t *testing.T) {
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v1.0.1/manifest.json"
		tarballURL := "https://github.com/acme/auth/archive/refs/tags/v1.0.1.tar.gz"
		tgz := buildTarGz(t, map[string]string{
			"auth-v1.0.1/addons/auth/README.md": "no manifest",
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case manifestURL:
				return httpResponse(http.StatusOK, []byte(`{"tarball":"`+tarballURL+`"}`)), nil
			case tarballURL:
				return httpResponse(http.StatusOK, tgz), nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://github.com/acme/registry", "auth", "v1.0.1"); err == nil {
			t.Fatalf("expected error for tarball without manifest")
		}
	})

	t.Run("unsafe tar path", func(t *testing.T) {
		addonsPath := t.TempDir()
		runtimeScope := &providerTestScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}
		manifestURL := "https://raw.githubusercontent.com/acme/registry/main/addons/auth/v1.0.2/manifest.json"
		tarballURL := "https://github.com/acme/auth/archive/refs/tags/v1.0.2.tar.gz"
		tgz := buildTarGz(t, map[string]string{
			"../auth/manifest.json": `{"application":"auth","version":"v1.0.2"}`,
		})
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case manifestURL:
				return httpResponse(http.StatusOK, []byte(`{"tarball":"`+tarballURL+`"}`)), nil
			case tarballURL:
				return httpResponse(http.StatusOK, tgz), nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		})}
		provider := NewProvider(runtimeScope, WithHTTPClient(client))
		if _, err := provider.Fetch(context.Background(), "https://github.com/acme/registry", "auth", "v1.0.2"); err == nil {
			t.Fatalf("expected error for unsafe tar path")
		}
	})
}
