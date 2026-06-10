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

func startStaticIndexServer(t *testing.T, payload map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	return httptest.NewServer(mux)
}

func TestCatalogInfoFromStaticIndex_ResolvesNPMPackageSource(t *testing.T) {
	t.Parallel()

	server := startStaticIndexServer(t, map[string]any{
		"modules": map[string]any{
			"sale": map[string]any{
				"moduleId":      "sale",
				"latestVersion": "v0.1.0",
				"package":       "@acme/choysum-sale",
				"versions": map[string]any{
					"v0.1.0": map[string]any{
						"source": map[string]any{
							"type":      "npm",
							"registry":  "https://registry.npmjs.org",
							"package":   "@acme/choysum-sale",
							"integrity": "sha512-sale",
						},
					},
				},
			},
		},
	})
	defer server.Close()

	catalog := NewCatalog(nil)
	item, err := catalog.Info(context.Background(), server.URL+"/v1/index.json", "sale")
	if err != nil {
		t.Fatalf("Catalog.Info() error = %v", err)
	}
	if item == nil {
		t.Fatal("Catalog.Info() returned nil")
	}
	if got := item.ResolvedNPMPackage(); got != "@acme/choysum-sale" {
		t.Fatalf("ResolvedNPMPackage() = %q, want %q", got, "@acme/choysum-sale")
	}
	if got := item.ResolvedNPMRegistry(""); got != "https://registry.npmjs.org" {
		t.Fatalf("ResolvedNPMRegistry() = %q, want %q", got, "https://registry.npmjs.org")
	}
	if item.Source == nil || item.Source.Integrity != "sha512-sale" {
		t.Fatalf("expected source integrity from index, got %#v", item.Source)
	}
}

func TestCatalogInfoFromStaticIndex_PrefersVersionSourcePackage(t *testing.T) {
	t.Parallel()

	server := startStaticIndexServer(t, map[string]any{
		"modules": map[string]any{
			"auth": map[string]any{
				"moduleId":      "auth",
				"latestVersion": "v1.0.0",
				"package":       "@choysum/module-auth",
				"versions": map[string]any{
					"v1.0.0": map[string]any{
						"source": map[string]any{
							"package": "@acme/choysum-auth",
						},
					},
				},
			},
		},
	})
	defer server.Close()

	catalog := NewCatalog(nil)
	item, err := catalog.Info(context.Background(), server.URL+"/v1/index.json", "auth")
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

func TestCatalogInfoFromStaticIndex_MergesVersionSourceIntoModuleSource(t *testing.T) {
	t.Parallel()

	server := startStaticIndexServer(t, map[string]any{
		"modules": map[string]any{
			"auth": map[string]any{
				"moduleId":      "auth",
				"latestVersion": "v1.0.0",
				"package":       "@choysum/module-auth",
				"source": map[string]any{
					"type":     "npm",
					"registry": "https://registry.acme.dev",
					"package":  "@choysum/module-auth",
				},
				"versions": map[string]any{
					"v1.0.0": map[string]any{
						"source": map[string]any{
							"package":   "@acme/choysum-auth",
							"integrity": "sha512-auth",
						},
					},
				},
			},
		},
	})
	defer server.Close()

	catalog := NewCatalog(nil)
	item, err := catalog.Info(context.Background(), server.URL+"/v1/index.json", "auth")
	if err != nil {
		t.Fatalf("Catalog.Info() error = %v", err)
	}
	if item == nil {
		t.Fatal("Catalog.Info() returned nil")
	}
	if got := item.ResolvedNPMPackage(); got != "@acme/choysum-auth" {
		t.Fatalf("ResolvedNPMPackage() = %q, want %q", got, "@acme/choysum-auth")
	}
	if got := item.ResolvedNPMRegistry(""); got != "https://registry.acme.dev" {
		t.Fatalf("ResolvedNPMRegistry() = %q, want %q", got, "https://registry.acme.dev")
	}
	if item.Source == nil {
		t.Fatal("expected merged source, got nil")
	}
	if item.Source.Type != "npm" || item.Source.Registry != "https://registry.acme.dev" || item.Source.Package != "@acme/choysum-auth" || item.Source.Integrity != "sha512-auth" {
		t.Fatalf("unexpected merged source: %#v", item.Source)
	}
}

func TestCatalogListFromStaticIndex_FiltersAndSorts(t *testing.T) {
	t.Parallel()

	server := startStaticIndexServer(t, map[string]any{
		"modules": map[string]any{
			"partner": map[string]any{
				"moduleId":      " partner ",
				"latestVersion": " v0.9.0 ",
				"description":   " partner module ",
				"package":       "@acme/choysum-partner",
				"versions": map[string]any{
					" v0.9.0 ": map[string]any{},
				},
			},
			"auth": map[string]any{
				"moduleId":      " auth ",
				"latestVersion": " v1.2.3 ",
				"description":   " auth module ",
				"package":       "@acme/choysum-auth",
				"versions": map[string]any{
					" v1.0.0 ": map[string]any{},
					" v1.2.3 ": map[string]any{},
				},
			},
		},
	})
	defer server.Close()

	catalog := NewCatalog(nil)
	items, err := catalog.List(context.Background(), server.URL+"/v1/index.json", "au")
	if err != nil {
		t.Fatalf("Catalog.List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Catalog.List() len = %d, want 1", len(items))
	}
	if items[0].Name != "auth" || items[0].LatestVersion != "v1.2.3" {
		t.Fatalf("unexpected normalized item: %#v", items[0])
	}
	if len(items[0].Versions) != 2 || items[0].Versions[0] != "v1.0.0" || items[0].Versions[1] != "v1.2.3" {
		t.Fatalf("unexpected versions: %#v", items[0].Versions)
	}
}

func TestCatalogInfoFromStaticIndex_NotFound(t *testing.T) {
	t.Parallel()

	server := startStaticIndexServer(t, map[string]any{"modules": map[string]any{"auth": map[string]any{"moduleId": "auth"}}})
	defer server.Close()

	catalog := NewCatalog(nil)
	_, err := catalog.Info(context.Background(), server.URL+"/v1/index.json", "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
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
