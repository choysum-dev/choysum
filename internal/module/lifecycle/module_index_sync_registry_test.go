// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type catalogRoundTripFunc func(*http.Request) (*http.Response, error)

func (f catalogRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchCatalogIndexUsesDefaultURLWhenEmpty(t *testing.T) {
	requestedURL := ""
	testClient := &http.Client{
		Transport: catalogRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"modules":{"auth":{"name":"auth"}}}`)),
			}, nil
		}),
	}

	index, err := fetchCatalogIndexWithClient(context.Background(), "", testClient)
	if err != nil {
		t.Fatalf("fetchCatalogIndex() error = %v", err)
	}
	if requestedURL != config.DefaultModuleCatalogIndexURL {
		t.Fatalf("request URL = %q, want %q", requestedURL, config.DefaultModuleCatalogIndexURL)
	}
	if _, ok := index.Modules["auth"]; !ok {
		t.Fatalf("expected auth module in decoded index, got %#v", index.Modules)
	}
}

func TestSyncRegistryModuleIndexUpsertsVersionAndReconcilesByOriginRef(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/index.json" {
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "modules": {
    "auth": {
      "name": "auth",
      "package": "@choysum-dev/auth",
      "latestVersion": "2.0.0",
      "versions": {
        "2.0.0": {"tarball": "https://registry.example.dev/auth-2.0.0.tgz"}
      }
    }
  }
}`)
	}))
	defer testServer.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = testServer.URL + "/v1/index.json"

	if err := db.Create(&modmeta.ModuleIndex{ModuleName: "auth", OriginType: "registry", OriginRef: "@legacy/choysum-auth", Available: true, Version: nullString("1.0.0")}).Error; err != nil {
		t.Fatalf("seed legacy auth row: %v", err)
	}
	if err := db.Create(&modmeta.ModuleIndex{ModuleName: "auth", OriginType: "registry", OriginRef: "@choysum-dev/auth", Available: false, Version: nullString("0.9.0")}).Error; err != nil {
		t.Fatalf("seed current auth row: %v", err)
	}
	if err := db.Create(&modmeta.ModuleIndex{ModuleName: "orphan", OriginType: "registry", OriginRef: "@legacy/orphan", Available: true, Version: nullString("0.1.0")}).Error; err != nil {
		t.Fatalf("seed orphan row: %v", err)
	}

	stats, err := SyncRegistryModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("SyncRegistryModuleIndex() error = %v", err)
	}
	if stats.Total != 1 || stats.Success != 1 || stats.Failed != 0 {
		t.Fatalf("unexpected stats = %#v", stats)
	}

	var current modmeta.ModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "auth", "registry", "@choysum-dev/auth").Take(&current).Error; err != nil {
		t.Fatalf("load current auth row: %v", err)
	}
	if !current.Available {
		t.Fatal("expected current auth row to be available")
	}
	if !current.Version.Valid || current.Version.String != "2.0.0" {
		t.Fatalf("current version = %#v, want 2.0.0", current.Version)
	}

	var legacy modmeta.ModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "auth", "registry", "@legacy/choysum-auth").Take(&legacy).Error; err != nil {
		t.Fatalf("load legacy auth row: %v", err)
	}
	if legacy.Available {
		t.Fatal("expected legacy auth row to be marked unavailable")
	}

	var orphan modmeta.ModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "orphan", "registry", "@legacy/orphan").Take(&orphan).Error; err != nil {
		t.Fatalf("load orphan row: %v", err)
	}
	if orphan.Available {
		t.Fatal("expected orphan row to be marked unavailable")
	}
}

func TestSyncRegistryModuleIndexFallsBackToRowUpsertsAfterBatchFailure(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "modules": {
    "auth": {"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"},
    "task": {"name":"task","package":"@choysum-dev/task","latestVersion":"1.0.0"}
  }
}`)
	}))
	defer testServer.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = testServer.URL + "/v1/index.json"

	if err := db.Exec(`
CREATE TRIGGER block_registry_batch_insert
BEFORE INSERT ON meta_module_index
WHEN (SELECT COUNT(*) FROM meta_module_index WHERE origin_type = 'registry') >= 1
BEGIN
  SELECT RAISE(ABORT, 'batch blocked');
END`).Error; err != nil {
		t.Fatalf("create batch insert trigger: %v", err)
	}

	stats, err := SyncRegistryModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("SyncRegistryModuleIndex() error = %v", err)
	}
	if stats.Total != 2 || stats.Success < 1 || stats.Failed < 1 {
		t.Fatalf("unexpected stats = %#v", stats)
	}
}

func TestSyncRegistryModuleIndexReconcilesOrphanWithoutID(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
	}))
	defer testServer.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = testServer.URL + "/v1/index.json"

	if err := db.Exec(`
INSERT INTO meta_module_index (module_name, origin_type, origin_ref, available, created_at, updated_at)
VALUES ('orphan', 'registry', '@legacy/orphan', 1, datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("seed orphan without id: %v", err)
	}

	stats, err := SyncRegistryModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("SyncRegistryModuleIndex() error = %v", err)
	}
	if stats.Total != 1 || stats.Success != 1 {
		t.Fatalf("unexpected stats = %#v", stats)
	}

	var orphan modmeta.ModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "orphan", "registry", "@legacy/orphan").Take(&orphan).Error; err != nil {
		t.Fatalf("load orphan row: %v", err)
	}
	if orphan.Available {
		t.Fatal("expected orphan without id to be marked unavailable")
	}
}

func TestSyncRegistryModuleIndexWarnsWhenBatchSyncTimestampUpdateFails(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
	}))
	defer testServer.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = testServer.URL + "/v1/index.json"

	if err := db.Exec(`
CREATE TRIGGER block_registry_batch_sync_ts
BEFORE UPDATE OF last_batch_sync_at ON meta_module_index
BEGIN
  SELECT RAISE(ABORT, 'batch sync timestamp blocked');
END`).Error; err != nil {
		t.Fatalf("create batch sync timestamp trigger: %v", err)
	}

	stats, err := SyncRegistryModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("SyncRegistryModuleIndex() error = %v", err)
	}
	if stats.Total != 1 || stats.Success != 1 || stats.Failed != 0 {
		t.Fatalf("unexpected stats = %#v", stats)
	}
}
