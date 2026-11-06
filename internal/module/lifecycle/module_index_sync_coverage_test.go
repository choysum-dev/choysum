// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestWithModuleIndexWriteRetryNilArgs(t *testing.T) {
	t.Parallel()
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), nil)
	if err := withModuleIndexWriteRetry(context.Background(), runtimeScope, nil, func(*scope.Session) error { return nil }); err == nil || err.Error() != "module source snapshot session is unavailable" {
		t.Fatalf("nil session error = %v", err)
	}
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope = newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	if err := withModuleIndexWriteRetry(context.Background(), runtimeScope, runtimeScope.session, nil); err == nil || err.Error() != "module source snapshot write operation is required" {
		t.Fatalf("nil op error = %v", err)
	}
}

type renewCountingLocker struct {
	renewErr error
	renewed  atomic.Int32
}

func (l *renewCountingLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (l *renewCountingLocker) Renew(context.Context, string, string, time.Duration) error {
	l.renewed.Add(1)
	return l.renewErr
}
func (l *renewCountingLocker) Release(context.Context, string, string) error { return nil }

func TestSyncLocalModuleSourceSnapshot_LeaseRenewWarn(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(modulesPath, db)

	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("mod%d", i)
		writePackageJSON(t, modulesPath, name,
			fmt.Sprintf(`{"name":"@acme/choysum-%s","version":"0.1.0","choysum":{"moduleName":"%s","application":"%s"}}`, name, name, name))
	}

	locker := &renewCountingLocker{renewErr: errors.New("renew boom")}
	ctx := withModuleIndexLockTTL(context.Background(), 5*time.Millisecond)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = SyncLocalModuleSourceSnapshot(ctx, runtimeScope, func(scope.Scope) statepkg.Locker {
			return locker
		})
	}()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && locker.renewed.Load() == 0 {
		time.Sleep(2 * time.Millisecond)
	}
	<-done
	if locker.renewed.Load() == 0 {
		t.Fatal("expected lease renew attempt")
	}
}

func TestSyncLocalModuleSourceSnapshot_UpsertAndReconcileWarns(t *testing.T) {
	t.Parallel()
	modulesPath := t.TempDir()
	writePackageJSON(t, modulesPath, "partner", `{"name":"@acme/choysum-partner","version":"0.2.0","choysum":{"moduleName":"partner","application":"partner"}}`)
	if err := os.MkdirAll(filepath.Join(modulesPath, "bad"), 0o755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "bad", "package.json"), []byte(`{`), 0o644); err != nil {
		t.Fatalf("write bad package.json: %v", err)
	}

	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "orphan",
		SourceType: "local",
		SourceRef:  "local",
		Available:  true,
	}).Error; err != nil {
		t.Fatalf("seed orphan: %v", err)
	}
	if err := db.Exec(`
CREATE TRIGGER block_source_snapshot_writes
BEFORE INSERT ON meta_module_source_snapshot
BEGIN
  SELECT RAISE(ABORT, 'insert blocked');
END;
CREATE TRIGGER block_source_snapshot_updates
BEFORE UPDATE ON meta_module_source_snapshot
BEGIN
  SELECT RAISE(ABORT, 'update blocked');
END;`).Error; err != nil {
		t.Fatalf("create triggers: %v", err)
	}
	runtimeScope := newModuleSourceSnapshotSyncScope(modulesPath, db)

	stats, err := SyncLocalModuleSourceSnapshot(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("SyncLocalModuleSourceSnapshot() error = %v", err)
	}
	if stats.Failed == 0 {
		t.Fatalf("expected failed upserts, stats=%+v", stats)
	}
}
