// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestBootstrapMetaTables_NilSession(t *testing.T) {
	t.Parallel()
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), nil)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "session is unavailable") {
		t.Fatalf("bootstrapMetaTables() error = %v, want session is unavailable", err)
	}
	err = manager.bootstrapMetaTables(nil)
	if err == nil || !strings.Contains(err.Error(), "session is unavailable") {
		t.Fatalf("bootstrapMetaTables(nil) error = %v, want session is unavailable", err)
	}
}

func TestBootstrapMetaTables_OpsCutoverError(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err = manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "ops schema cutover") {
		t.Fatalf("bootstrapMetaTables() error = %v, want ops schema cutover", err)
	}
}

func TestBootstrapMetaTables_OpsEntitiesMigrateError(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	// Block catalog creation so OpsEntities AutoMigrate fails on the early path.
	if err := db.Exec(`CREATE VIEW meta_module_catalog AS SELECT 1 AS id`).Error; err != nil {
		t.Fatalf("create blocking view: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "auto migrate ops entities") {
		t.Fatalf("bootstrapMetaTables() error = %v, want auto migrate ops entities", err)
	}
}
