// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestSyncRegistryModuleSourceSnapshot_CoverageBranches(t *testing.T) {
	t.Run("lease renew warn", func(t *testing.T) {
		db := newModuleSourceSnapshotSyncDB(t)
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)

		// Hold the catalog response until renew has fired so the lease heartbeat runs.
		releaseFetch := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			<-releaseFetch
			_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
		}))
		defer server.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

		locker := &renewCountingLocker{renewErr: errors.New("renew boom")}
		ctx := withModuleIndexLockTTL(context.Background(), 5*time.Millisecond)
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = SyncRegistryModuleSourceSnapshot(ctx, runtimeScope, func(scope.Scope) statepkg.Locker {
				return locker
			})
		}()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && locker.renewed.Load() == 0 {
			time.Sleep(2 * time.Millisecond)
		}
		close(releaseFetch)
		<-done
		if locker.renewed.Load() == 0 {
			t.Fatal("expected renew")
		}
	})

	t.Run("nil session", func(t *testing.T) {
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), nil)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
		}))
		defer server.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

		_, err := SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err == nil || err.Error() != "module source snapshot session is unavailable" {
			t.Fatalf("error = %v, want session unavailable", err)
		}
	})

	t.Run("marshal failed", func(t *testing.T) {
		db := newModuleSourceSnapshotSyncDB(t)
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
		}))
		defer server.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

		ctx := withRegistryModuleJSONMarshal(context.Background(), func(any) ([]byte, error) {
			return nil, errors.New("marshal boom")
		})
		stats, err := SyncRegistryModuleSourceSnapshot(ctx, runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err != nil {
			t.Fatalf("sync error = %v", err)
		}
		if stats.Failed == 0 {
			t.Fatalf("expected marshal failure stats=%+v", stats)
		}
	})

	t.Run("reconcile find and update warns", func(t *testing.T) {
		db := newModuleSourceSnapshotSyncDB(t)
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"2.0.0"}}}`)
		}))
		defer server.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

		if err := db.Create(&modmeta.ModuleSourceSnapshot{
			ModuleName: "orphan",
			SourceType: "registry",
			SourceRef:  "@legacy/orphan",
			Available:  true,
		}).Error; err != nil {
			t.Fatalf("seed orphan: %v", err)
		}
		if err := db.Exec(`
CREATE TRIGGER block_registry_updates
BEFORE UPDATE ON meta_module_source_snapshot
BEGIN
  SELECT RAISE(ABORT, 'update blocked');
END;`).Error; err != nil {
			t.Fatalf("create trigger: %v", err)
		}

		_, err := SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err != nil {
			t.Fatalf("sync error = %v", err)
		}
		var orphan modmeta.ModuleSourceSnapshot
		if err := db.Where("module_name = ? AND source_type = ?", "orphan", "registry").Take(&orphan).Error; err != nil {
			t.Fatalf("load orphan: %v", err)
		}
		if !orphan.Available {
			t.Fatalf("orphan Available = false, want true after blocked reconcile update")
		}
	})

	t.Run("reconcile find failed", func(t *testing.T) {
		db := newModuleSourceSnapshotSyncDB(t)
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
		}))
		defer server.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

		ctx := withRegistryModuleJSONMarshal(context.Background(), func(any) ([]byte, error) {
			return nil, errors.New("marshal boom")
		})
		if err := db.Exec(`DROP TABLE meta_module_source_snapshot`).Error; err != nil {
			t.Fatalf("drop table: %v", err)
		}

		stats, err := SyncRegistryModuleSourceSnapshot(ctx, runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err != nil {
			t.Fatalf("sync error = %v", err)
		}
		if stats.Failed == 0 {
			t.Fatalf("expected failed stats=%+v", stats)
		}
	})

	t.Run("reconcile row key update failed", func(t *testing.T) {
		db := newModuleSourceSnapshotSyncDB(t)
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"modules":{"auth":{"name":"auth","package":"@choysum-dev/auth","latestVersion":"1.0.0"}}}`)
		}))
		defer server.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = server.URL

		if err := db.Exec(`INSERT INTO meta_module_source_snapshot
			(module_name, source_type, source_ref, available) VALUES ('orphan', 'registry', '@legacy/orphan', 1)`).Error; err != nil {
			t.Fatalf("seed orphan without id: %v", err)
		}
		if err := db.Exec(`
CREATE TRIGGER block_registry_key_updates
BEFORE UPDATE ON meta_module_source_snapshot
BEGIN
  SELECT RAISE(ABORT, 'update blocked');
END;`).Error; err != nil {
			t.Fatalf("create trigger: %v", err)
		}

		_, err := SyncRegistryModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err != nil {
			t.Fatalf("sync error = %v", err)
		}
		var orphanAvailable bool
		if err := db.Model(&modmeta.ModuleSourceSnapshot{}).
			Select("available").
			Where("module_name = ? AND source_type = ? AND source_ref = ?", "orphan", "registry", "@legacy/orphan").
			Scan(&orphanAvailable).Error; err != nil {
			t.Fatalf("load orphan available: %v", err)
		}
		if !orphanAvailable {
			t.Fatalf("orphan Available = false, want true after blocked key update")
		}
	})
}
