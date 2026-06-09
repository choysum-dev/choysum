// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

type catalogFakeProvider struct {
	peekFn  func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error)
	fetchFn func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error)
}

func (p *catalogFakeProvider) PeekManifest(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
	if p.peekFn != nil {
		return p.peekFn(ctx, registryURL, moduleName, packageName, version)
	}
	if p.fetchFn != nil {
		return p.fetchFn(ctx, registryURL, moduleName, packageName, version)
	}
	return nil, nil
}

func (p *catalogFakeProvider) Fetch(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
	if p.fetchFn != nil {
		return p.fetchFn(ctx, registryURL, moduleName, packageName, version)
	}
	return nil, nil
}

func TestCatalogInfoFromRemoteAPI_ResolvesNPMPackageSource(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/modules/sale", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":          "sale",
			"latestVersion": "v0.1.0",
			"npmPackage":    "@acme/choysum-sale",
			"source": map[string]any{
				"type":     "npm",
				"registry": "https://registry.npmjs.org",
				"package":  "@acme/choysum-sale",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	catalog := NewCatalog(nil)
	item, err := catalog.Info(context.Background(), server.URL, "sale")
	if err != nil {
		t.Fatalf("Catalog.Info() error = %v", err)
	}
	if item == nil {
		t.Fatal("Catalog.Info() returned nil")
	}
	if got := item.ResolvedNPMPackage(); got != "@acme/choysum-sale" {
		t.Fatalf("ResolvedNPMPackage() = %q, want %q", got, "@acme/choysum-sale")
	}
	if got := item.ResolvedNPMRegistry(server.URL); got != "https://registry.npmjs.org" {
		t.Fatalf("ResolvedNPMRegistry() = %q, want %q", got, "https://registry.npmjs.org")
	}
}

func TestCatalogInfoFromRemoteAPI_PrefersSourcePackage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/modules/auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":          "auth",
			"latestVersion": "v1.0.0",
			"npmPackage":    "@choysum/module-auth",
			"source": map[string]any{
				"package": "@acme/choysum-auth",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	catalog := NewCatalog(nil)
	item, err := catalog.Info(context.Background(), server.URL, "auth")
	if err != nil {
		t.Fatalf("Catalog.Info() error = %v", err)
	}
	if item == nil {
		t.Fatal("Catalog.Info() returned nil")
	}
	if got := item.ResolvedNPMPackage(); got != "@acme/choysum-auth" {
		t.Fatalf("ResolvedNPMPackage() = %q, want %q", got, "@acme/choysum-auth")
	}
}

func TestCatalogListFromRemoteAPI_SupportsEnvelopeAndArray(t *testing.T) {
	t.Parallel()

	t.Run("envelope payload", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/modules", func(w http.ResponseWriter, r *http.Request) {
			if gotQ := r.URL.Query().Get("q"); gotQ != "au" {
				t.Fatalf("query parameter q = %q, want %q", gotQ, "au")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modules": []map[string]any{{
					"name":          " auth ",
					"latestVersion": " v1.2.3 ",
					"source": map[string]any{
						"package": " @acme/choysum-auth ",
					},
				}},
			})
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		catalog := NewCatalog(nil)
		items, err := catalog.List(context.Background(), server.URL, "au")
		if err != nil {
			t.Fatalf("Catalog.List() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Catalog.List() len = %d, want 1", len(items))
		}
		if items[0].Name != "auth" || items[0].LatestVersion != "v1.2.3" {
			t.Fatalf("unexpected normalized item: %#v", items[0])
		}
		if got := items[0].ResolvedNPMPackage(); got != "@acme/choysum-auth" {
			t.Fatalf("ResolvedNPMPackage() = %q, want %q", got, "@acme/choysum-auth")
		}
	})

	t.Run("array payload", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/modules", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"name":"sale","latestVersion":"v0.1.0"}]`))
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		catalog := NewCatalog(nil)
		items, err := catalog.List(context.Background(), server.URL, "")
		if err != nil {
			t.Fatalf("Catalog.List() error = %v", err)
		}
		if len(items) != 1 || items[0].Name != "sale" {
			t.Fatalf("unexpected array payload result: %#v", items)
		}
	})
}

func TestCatalogGitHubListAndInfo(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api.github.com/repos/acme/registry/contents/modules":
			return httpResponse(http.StatusOK, []byte(`[
				{"name":"sale","type":"dir"},
				{"name":"auth","type":"dir"},
				{"name":"README.md","type":"file"}
			]`)), nil
		case "https://api.github.com/repos/acme/registry/contents/modules/auth":
			return httpResponse(http.StatusOK, []byte(`[
				{"name":"v1.0.0","type":"dir"},
				{"name":"v2.0.0","type":"dir"}
			]`)), nil
		case "https://api.github.com/repos/acme/registry/contents/modules/sale":
			return httpResponse(http.StatusOK, []byte(`[
				{"name":"0.9.0","type":"dir"},
				{"name":"v1.1.0","type":"dir"}
			]`)), nil
		default:
			return httpResponse(http.StatusNotFound, []byte(`{"message":"not found"}`)), nil
		}
	})

	catalog := NewCatalog(nil, WithCatalogHTTPClient(&http.Client{Transport: transport}))
	items, err := catalog.List(context.Background(), "https://github.com/acme/registry", "")
	if err != nil {
		t.Fatalf("Catalog.List(github) error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Catalog.List(github) len = %d, want 2", len(items))
	}
	if items[0].Name != "auth" || items[1].Name != "sale" {
		t.Fatalf("expected sorted github modules [auth sale], got %#v", items)
	}
	if items[0].LatestVersion != "v2.0.0" {
		t.Fatalf("unexpected auth latest version: %#v", items[0])
	}

	peekCalls := 0
	provider := &catalogFakeProvider{peekFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
		peekCalls++
		if registryURL != "https://github.com/acme/registry" || moduleName != "auth" || packageName != "auth" || version != "v2.0.0" {
			t.Fatalf("unexpected provider peek args: registry=%s module=%s package=%s version=%s", registryURL, moduleName, packageName, version)
		}
		return &meta.IrModule{Description: "  Auth module from provider  "}, nil
	}}

	catalog = NewCatalog(nil, WithCatalogHTTPClient(&http.Client{Transport: transport}), WithCatalogProvider(provider))
	item, err := catalog.Info(context.Background(), "https://github.com/acme/registry", "auth")
	if err != nil {
		t.Fatalf("Catalog.Info(github) error = %v", err)
	}
	if item == nil || item.Name != "auth" || item.LatestVersion != "v2.0.0" {
		t.Fatalf("unexpected github info item: %#v", item)
	}
	if item.Description != "Auth module from provider" {
		t.Fatalf("expected trimmed provider description, got %q", item.Description)
	}
	if peekCalls != 1 {
		t.Fatalf("provider peek calls = %d, want 1", peekCalls)
	}
}

func TestCatalogHelpersAndOptions(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{}
	customProvider := &catalogFakeProvider{}
	catalog := NewCatalog(nil, WithCatalogHTTPClient(customClient), WithCatalogProvider(customProvider))
	if catalog.client != customClient {
		t.Fatal("expected custom HTTP client option to be applied")
	}
	if catalog.provider != customProvider {
		t.Fatal("expected custom provider option to be applied")
	}

	if got := pickLatestVersion([]string{"v1.0.0", "v2.0.0", "v1.5.0"}); got != "v2.0.0" {
		t.Fatalf("pickLatestVersion(semver) = %q, want %q", got, "v2.0.0")
	}
	if got := pickLatestVersion([]string{"alpha", "beta"}); got != "beta" {
		t.Fatalf("pickLatestVersion(non-semver fallback) = %q, want %q", got, "beta")
	}

	if normalized, ok := normalizeSemVer("1.2.3"); !ok || normalized != "v1.2.3" {
		t.Fatalf("normalizeSemVer(%q) = (%q,%v), want (v1.2.3,true)", "1.2.3", normalized, ok)
	}
	if _, ok := normalizeSemVer("invalid"); ok {
		t.Fatal("normalizeSemVer(invalid) should fail")
	}

	if !isGitHubRegistryURL("https://github.com/acme/registry") {
		t.Fatal("expected github URL to be recognized")
	}
	if isGitHubRegistryURL("http://github.com/acme/registry") {
		t.Fatal("expected non-https github URL to be rejected")
	}

	owner, repo, err := parseGitHubOwnerRepo("https://github.com/acme/registry")
	if err != nil || owner != "acme" || repo != "registry" {
		t.Fatalf("parseGitHubOwnerRepo(valid) = (%q,%q,%v), want (acme,registry,nil)", owner, repo, err)
	}
	if _, _, err := parseGitHubOwnerRepo("https://github.com/acme"); err == nil {
		t.Fatal("expected invalid github registry URL error")
	}

	mod := &CatalogModule{
		Name:          " auth ",
		LatestVersion: "",
		Description:   "  auth module  ",
		Versions:      []string{" v1.2.0 ", "", "v1.0.0"},
		NPMPackage:    " @acme/choysum-auth ",
		Source:        &CatalogSource{},
	}
	normalizeCatalogModule(mod)
	if mod.Name != "auth" || mod.Description != "auth module" || mod.NPMPackage != "@acme/choysum-auth" {
		t.Fatalf("normalizeCatalogModule() unexpected basic normalization: %#v", mod)
	}
	if mod.LatestVersion != "v1.2.0" {
		t.Fatalf("normalizeCatalogModule() latest version = %q, want %q", mod.LatestVersion, "v1.2.0")
	}
	if len(mod.Versions) != 2 || mod.Versions[0] != "v1.0.0" || mod.Versions[1] != "v1.2.0" {
		t.Fatalf("normalizeCatalogModule() versions = %#v, want [v1.0.0 v1.2.0]", mod.Versions)
	}
	if mod.Source != nil {
		t.Fatalf("expected empty source to be removed, got %#v", mod.Source)
	}
	if got := (&CatalogModule{NPMPackage: "  @acme/choysum-auth  "}).ResolvedNPMPackage(); got != "@acme/choysum-auth" {
		t.Fatalf("ResolvedNPMPackage(default fallback) = %q, want %q", got, "@acme/choysum-auth")
	}

	if got := (&CatalogModule{NPMPackage: "@acme/choysum-auth"}).ResolvedNPMRegistry("  https://registry.npmjs.org  "); got != "https://registry.npmjs.org" {
		t.Fatalf("ResolvedNPMRegistry(default fallback) = %q, want %q", got, "https://registry.npmjs.org")
	}
	if got := (&CatalogModule{Source: &CatalogSource{Registry: "  https://registry.acme.dev  "}}).ResolvedNPMRegistry("https://registry.npmjs.org"); got != "https://registry.acme.dev" {
		t.Fatalf("ResolvedNPMRegistry(source override) = %q, want %q", got, "https://registry.acme.dev")
	}

	notFoundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer notFoundServer.Close()
	catalog = NewCatalog(nil)
	if _, err := catalog.fetchJSON(context.Background(), notFoundServer.URL); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected fetchJSON not found error, got %v", err)
	}

	badStatusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer badStatusServer.Close()
	if _, err := catalog.fetchJSON(context.Background(), badStatusServer.URL); err == nil || !strings.Contains(err.Error(), "unexpected HTTP status") {
		t.Fatalf("expected fetchJSON bad status error, got %v", err)
	}
}
