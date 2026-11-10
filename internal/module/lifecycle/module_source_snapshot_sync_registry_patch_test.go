// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestResolveRegistryIndexURLAndVersionHelpers(t *testing.T) {
	t.Parallel()
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), nil)
	runtimeScope.cfg.ModuleCatalogIndexURL = "https://custom.example/index.json"
	if got := resolveRegistryIndexURL(runtimeScope); got != "https://custom.example/index.json" {
		t.Fatalf("custom url = %q", got)
	}
	if got := resolveRegistryIndexURL(newModuleSourceSnapshotSyncScope(t.TempDir(), nil)); got != config.DefaultModuleCatalogIndexURL {
		t.Fatalf("default url = %q", got)
	}

	if got := normalizeCatalogIndexVersion(""); got != "" {
		t.Fatalf("empty version = %q", got)
	}
	if got := normalizeCatalogIndexVersion("not-a-semver"); got != "vnot-a-semver" {
		t.Fatalf("non-semver = %q", got)
	}
	if got := normalizeCatalogIndexVersion("valready"); got != "valready" {
		t.Fatalf("v-prefixed invalid = %q", got)
	}

	got := resolveCatalogIndexLatestVersion(catalogIndexModule{
		Versions: map[string]catalogIndexEntry{
			"1.0.0": {},
			"not-semver": {},
			"":         {},
		},
	})
	if got != "v1.0.0" {
		t.Fatalf("semver pick = %q", got)
	}
	got = resolveCatalogIndexLatestVersion(catalogIndexModule{
		Versions: map[string]catalogIndexEntry{"zzz": {}, "aaa": {}},
	})
	if got != "vzzz" {
		t.Fatalf("lex fallback = %q", got)
	}
}

func TestFetchCatalogIndexWithClient_Errors(t *testing.T) {
	t.Parallel()
	_, err := fetchCatalogIndexWithClient(context.Background(), "http://[::1", &http.Client{})
	if err == nil || !strings.Contains(err.Error(), "create registry index request") {
		t.Fatalf("bad url: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()
	_, err = fetchCatalogIndexWithClient(context.Background(), server.URL, server.Client())
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status") {
		t.Fatalf("bad status: %v", err)
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{`)
	}))
	defer badJSON.Close()
	_, err = fetchCatalogIndexWithClient(context.Background(), badJSON.URL, badJSON.Client())
	if err == nil || !strings.Contains(err.Error(), "decode registry index") {
		t.Fatalf("decode: %v", err)
	}
}

func TestSyncRegistryModuleSourceSnapshot_EarlyPaths(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"modules":{}}`)
	}))
	defer okServer.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = okServer.URL
	if _, err := SyncRegistryModuleSourceSnapshot(nil, runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	}); err != nil {
		t.Fatalf("nil ctx should default: %v", err)
	}

	_, err := SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, nil)
	if err == nil || !strings.Contains(err.Error(), "locker factory is nil") {
		t.Fatalf("nil locker: %v", err)
	}

	busy := &moduleIndexSyncTestLocker{acquireErr: statepkg.ErrLeaseBusy}
	_, err = SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker { return busy })
	if err == nil || !strings.Contains(err.Error(), "LEASE_CONFLICT") {
		t.Fatalf("lease busy: %v", err)
	}

	failAcquire := &moduleIndexSyncTestLocker{acquireErr: errors.New("acquire boom")}
	_, err = SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker { return failAcquire })
	if err == nil || !strings.Contains(err.Error(), "acquire boom") {
		t.Fatalf("acquire error: %v", err)
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"modules":{}}`)
	}))
	defer empty.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = empty.URL
	stats, err := SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil || stats.Total != 0 {
		t.Fatalf("empty index: stats=%+v err=%v", stats, err)
	}

	skipNames := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"modules":{"":{"name":""}}}`)
	}))
	defer skipNames.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = skipNames.URL
	stats, err = SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil || stats.Total != 0 {
		t.Fatalf("empty module name: stats=%+v err=%v", stats, err)
	}
}

func TestSyncRegistryModuleSourceSnapshot_PackageDefaultsToModuleName(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"modules":{"baremod":{"latestVersion":"1.0.0"}}}`)
	}))
	defer server.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = server.URL
	stats, err := SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if stats.Total != 1 || stats.Success != 1 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestFetchCatalogIndexWithClient_DecodeSetsEmptyModules(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	idx, err := fetchCatalogIndexWithClient(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if idx.Modules == nil || len(idx.Modules) != 0 {
		t.Fatalf("modules = %#v", idx.Modules)
	}
}

func TestFetchCatalogIndexWithClient_ReadBodyError(t *testing.T) {
	t.Parallel()
	client := &http.Client{
		Transport: catalogRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(errReader{}),
			}, nil
		}),
	}
	_, err := fetchCatalogIndexWithClient(context.Background(), "http://example.invalid", client)
	if err == nil || !strings.Contains(err.Error(), "read registry index body") {
		t.Fatalf("read body err: %v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (errReader) Close() error             { return nil }

func TestSyncRegistryModuleSourceSnapshot_CancelledContext(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"modules":{`)
		<-release
	}))
	defer server.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := SyncRegistryModuleSourceSnapshot(ctx, runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled ctx: %v", err)
	}
	close(release)
}
