// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
