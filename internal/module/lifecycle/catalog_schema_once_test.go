// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"strings"
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

func TestEnsureCatalogSchemaMigrated_StaleMemoWithoutOpsTableRemigrates(t *testing.T) {
	// Package-level migrateCatalogEntities / memo map; not safe with t.Parallel.
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	resetCatalogSchemaMigrateForTest()
	t.Cleanup(resetCatalogSchemaMigrateForTest)
	// Simulate recycled *sql.DB pointer: memo says done but ops table is absent.
	rememberCatalogSchemaMigrated(db, nil)
	if db.Migrator().HasTable(&modmeta.ModuleOperation{}) {
		t.Fatal("fixture must not include meta_module_operation")
	}

	var calls int
	prev := migrateCatalogEntities
	migrateCatalogEntities = func(gdb *gorm.DB, entities ...any) error {
		calls++
		return prev(gdb, entities...)
	}
	t.Cleanup(func() { migrateCatalogEntities = prev })

	if err := ensureCatalogSchemaMigrated(db); err != nil {
		t.Fatalf("ensureCatalogSchemaMigrated: %v", err)
	}
	if calls != 1 {
		t.Fatalf("stale memo remigrate calls=%d, want 1", calls)
	}
	if !db.Migrator().HasTable(&modmeta.ModuleOperation{}) {
		t.Fatal("expected meta_module_operation after remigrate")
	}
}

func TestRememberCatalogSchemaMigrated_IgnoresFailure(t *testing.T) {
	db := newModuleSourceSnapshotSyncDB(t)
	resetCatalogSchemaMigrateForTest()
	t.Cleanup(resetCatalogSchemaMigrateForTest)

	rememberCatalogSchemaMigrated(db, gorm.ErrInvalidDB)
	st := catalogSchemaStateFor(db)
	st.mu.Lock()
	done := st.done
	st.mu.Unlock()
	if done {
		t.Fatal("failure must not set memo done")
	}
}

func TestCatalogSchemaKeyAndEnsureNilDB(t *testing.T) {
	if got := catalogSchemaKey(nil); got != "nil" {
		t.Fatalf("catalogSchemaKey(nil)=%q, want nil", got)
	}
	if err := ensureCatalogSchemaMigrated(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("ensureCatalogSchemaMigrated(nil)=%v, want db is nil", err)
	}

	// gorm.DB without an opened pool falls back to the gorm:%p key.
	gdb := &gorm.DB{}
	key := catalogSchemaKey(gdb)
	if !strings.HasPrefix(key, "gorm:") {
		t.Fatalf("catalogSchemaKey(empty gorm)=%q, want gorm: prefix", key)
	}
	// Config present but no ConnPool: db.DB() errors → same gorm:%p fallback.
	gdb2 := &gorm.DB{Config: &gorm.Config{}}
	key2 := catalogSchemaKey(gdb2)
	if !strings.HasPrefix(key2, "gorm:") {
		t.Fatalf("catalogSchemaKey(config-only)=%q, want gorm: prefix", key2)
	}
}
