// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func TestReadOnlyResolver_WarmPathSkipsAutoMigrate(t *testing.T) {
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	if err := db.Create(&meta.Module{Name: "core", Version: "1.0.0", Status: meta.Installed}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
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
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	ro := NewReadOnlyResolver(manager)
	if _, err := ro.Peek(context.Background(), "missing-local"); err != nil {
		// Peek may fail if origin has no module; that is fine as long as no migrate.
		_ = err
	}
	if _, err := ro.Load("core"); err != nil {
		t.Fatalf("Load(core): %v", err)
	}
	if calls != 0 {
		t.Fatalf("warm ReadOnlyResolver AutoMigrate calls=%d, want 0", calls)
	}
}

func TestReadOnlyResolver_ColdStartMigratesOnce(t *testing.T) {
	db := newModuleSourceSnapshotSyncDB(t)
	// Empty DB: no meta_module table.
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
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	ro := NewReadOnlyResolver(manager)
	if _, err := ro.Load("core"); err != nil {
		t.Fatalf("cold Load: %v", err)
	}
	if calls != 1 {
		t.Fatalf("cold ReadOnlyResolver AutoMigrate calls=%d, want 1", calls)
	}
	if _, err := ro.Load("core"); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if calls != 1 {
		t.Fatalf("second Load remigrated: calls=%d, want 1", calls)
	}
}

func TestReadOnlyResolver_PartialSchemaMigratesOpsTable(t *testing.T) {
	// Mirrors CLI e2e seedModuleStatusForCLI: only meta_module exists.
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("seed meta_module: %v", err)
	}
	if db.Migrator().HasTable(&modmeta.ModuleOperation{}) {
		t.Fatal("fixture must not include meta_module_operation")
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
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	ro := NewReadOnlyResolver(manager)
	if _, err := ro.Load("missing"); err != nil {
		t.Fatalf("partial-schema Load: %v", err)
	}
	if calls != 1 {
		t.Fatalf("partial-schema AutoMigrate calls=%d, want 1", calls)
	}
	if !db.Migrator().HasTable(&modmeta.ModuleOperation{}) {
		t.Fatal("expected meta_module_operation after cold migrate")
	}
}

func TestAsReadOnlyResolver(t *testing.T) {
	if AsReadOnlyResolver(nil) != nil {
		t.Fatal("nil in => nil out")
	}
	if NewReadOnlyResolver(nil) != nil {
		t.Fatal("NewReadOnlyResolver(nil) => nil")
	}
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	ro := AsReadOnlyResolver(manager)
	if _, ok := ro.(*ReadOnlyResolver); !ok {
		t.Fatalf("got %T, want *ReadOnlyResolver", ro)
	}
	if AsReadOnlyResolver(ro) != ro {
		t.Fatal("idempotent wrap")
	}
	passthrough := &stubPlanResolver{}
	if AsReadOnlyResolver(passthrough) != passthrough {
		t.Fatal("non-manager resolver must pass through")
	}

	var nilRO *ReadOnlyResolver
	if _, err := nilRO.Load("x"); err == nil {
		t.Fatal("nil ReadOnlyResolver.Load want error")
	}
	if _, err := nilRO.Peek(context.Background(), "x"); err == nil {
		t.Fatal("nil ReadOnlyResolver.Peek want error")
	}
	bare := &ReadOnlyResolver{m: &ModuleManager{}}
	if err := bare.ensureMetaModuleTable(); err == nil || !strings.Contains(err.Error(), "session is unavailable") {
		t.Fatalf("ensureMetaModuleTable empty manager = %v", err)
	}

	resetCatalogSchemaMigrateForTest()
	t.Cleanup(resetCatalogSchemaMigrateForTest)
	prev := migrateCatalogEntities
	migrateCatalogEntities = func(*gorm.DB, ...any) error {
		return errors.New("migrate boom")
	}
	t.Cleanup(func() { migrateCatalogEntities = prev })
	emptyDB := newModuleSourceSnapshotSyncDB(t)
	failing := NewReadOnlyResolver(NewModuleManager(newModuleSourceSnapshotSyncScope(t.TempDir(), emptyDB), nil))
	failing.m.bootstrapOnce.Do(func() {})
	if _, err := failing.Load("core"); err == nil || !strings.Contains(err.Error(), "migrate boom") {
		t.Fatalf("Load cold migrate failure = %v", err)
	}
	if _, err := failing.Peek(context.Background(), "core"); err == nil || !strings.Contains(err.Error(), "migrate boom") {
		t.Fatalf("Peek cold migrate failure = %v", err)
	}
}

type stubPlanResolver struct{}

func (stubPlanResolver) Load(string) (*meta.Module, error) { return nil, nil }
func (stubPlanResolver) Peek(context.Context, string) (*meta.Module, error) {
	return nil, nil
}

func TestLoad_DoesNotMigrateBaseModule(t *testing.T) {
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	resetCatalogSchemaMigrateForTest()
	t.Cleanup(resetCatalogSchemaMigrateForTest)

	// Mark catalog already migrated so ensureMetaTables baseReady is a memo hit
	// after we force bootstrapOnce to run ensure (or skip bootstrap).
	rememberCatalogSchemaMigrated(db, nil)

	var calls int
	prev := migrateCatalogEntities
	migrateCatalogEntities = func(gdb *gorm.DB, entities ...any) error {
		calls++
		return prev(gdb, entities...)
	}
	t.Cleanup(func() { migrateCatalogEntities = prev })

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	// Allow ensureMetaTables to run; baseReady + memo => 0 AutoMigrate.
	if _, err := manager.Load("core"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if calls != 0 {
		t.Fatalf("Load(core) AutoMigrate calls=%d, want 0 (no migrateBaseModule)", calls)
	}
}
