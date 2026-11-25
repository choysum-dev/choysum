// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func TestEnsureCatalogSchemaMigratedOncePerSQLDB(t *testing.T) {
	// Package-level migrateCatalogEntities / memo map; not safe with t.Parallel.
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	resetCatalogSchemaMigrateForTest()
	t.Cleanup(resetCatalogSchemaMigrateForTest)

	var calls int
	prev := migrateCatalogEntities
	migrateCatalogEntities = func(gdb *gorm.DB, entities ...any) error {
		calls++
		return prev(gdb, entities...)
	}
	t.Cleanup(func() { migrateCatalogEntities = prev })

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	first := NewModuleManager(runtimeScope, nil)
	if err := first.ensureMetaTables(); err != nil {
		t.Fatalf("first ensureMetaTables: %v", err)
	}
	if calls != 1 {
		t.Fatalf("first ensure MetaTables migrate calls=%d, want 1", calls)
	}

	second := NewModuleManager(runtimeScope, nil)
	if err := second.ensureMetaTables(); err != nil {
		t.Fatalf("second ensureMetaTables: %v", err)
	}
	if calls != 1 {
		t.Fatalf("second ModuleManager remigrated: calls=%d, want 1", calls)
	}

	if err := second.migrateBaseModule(); err != nil {
		t.Fatalf("migrateBaseModule after ensure: %v", err)
	}
	if calls != 1 {
		t.Fatalf("migrateBaseModule remigrated: calls=%d, want 1", calls)
	}
}

func TestEnsureCatalogSchemaMigratedDistinctPoolsIndependent(t *testing.T) {
	// Package-level migrateCatalogEntities / memo map; not safe with t.Parallel.
	resetCatalogSchemaMigrateForTest()
	t.Cleanup(resetCatalogSchemaMigrateForTest)

	var calls int
	prev := migrateCatalogEntities
	migrateCatalogEntities = func(gdb *gorm.DB, entities ...any) error {
		calls++
		return prev(gdb, entities...)
	}
	t.Cleanup(func() { migrateCatalogEntities = prev })

	for i := 0; i < 2; i++ {
		db := newModuleSourceSnapshotSyncDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
			t.Fatalf("auto migrate base tables: %v", err)
		}
		runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
		manager := NewModuleManager(runtimeScope, nil)
		if err := manager.ensureMetaTables(); err != nil {
			t.Fatalf("ensureMetaTables pool %d: %v", i, err)
		}
	}
	if calls != 2 {
		t.Fatalf("distinct sql.DB pools migrate calls=%d, want 2", calls)
	}
}
