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
	// Block catalog creation so CatalogEntities AutoMigrate fails on the early path.
	if err := db.Exec(`CREATE VIEW meta_module_catalog AS SELECT 1 AS id`).Error; err != nil {
		t.Fatalf("create blocking view: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "auto migrate catalog entities") {
		t.Fatalf("bootstrapMetaTables() error = %v, want auto migrate catalog entities", err)
	}
}

func TestBootstrapMetaTables_AddsModelOrderByOnExistingInstall(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	// Pre-OrderBy shape: baseReady tables + catalog tables without order_by.
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE meta_model (id char(20) PRIMARY KEY, name varchar(255) NOT NULL, path varchar(512) NOT NULL, application varchar(255))`,
		`CREATE TABLE meta_raw_model (id char(20) PRIMARY KEY, name varchar(255) NOT NULL, path varchar(512) NOT NULL, application varchar(255), module_id char(20))`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create pre-order_by table: %v", err)
		}
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
	if !db.Migrator().HasColumn(&meta.Model{}, "order_by") {
		t.Fatal("expected meta_model.order_by after bootstrap on existing install")
	}
	if !db.Migrator().HasColumn("meta_raw_model", "order_by") {
		t.Fatal("expected meta_raw_model.order_by after bootstrap on existing install")
	}
}
