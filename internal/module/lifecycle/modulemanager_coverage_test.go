// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func TestBootstrapMetaTablesSkipsWhenBaseTablesExist(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
}

func TestEnsureMetaTablesPreservesBootstrapError(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	seedDuplicateLiveModelsForTest(t, db)

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	first := manager.ensureMetaTables()
	if first == nil || !strings.Contains(first.Error(), "ensure effective app/name unique index") {
		t.Fatalf("ensureMetaTables() first error = %v, want unique index failure", first)
	}
	second := manager.ensureMetaTables()
	if second == nil || !strings.Contains(second.Error(), "ensure effective app/name unique index") {
		t.Fatalf("ensureMetaTables() second error = %v, want preserved bootstrap failure", second)
	}
	if first.Error() != second.Error() {
		t.Fatalf("preserved error changed: first=%v second=%v", first, second)
	}
}

func TestBootstrapMetaTablesEnsuresUniqueIndexWhenModelExists(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
	if !db.Migrator().HasIndex(&meta.Model{}, "uidx_meta_model_app_name_alive") {
		t.Fatal("expected live unique index after early-path bootstrap")
	}
}

func TestBootstrapMetaTablesEnsureUniqueIndexErrorWhenModelExists(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	seedDuplicateLiveModelsForTest(t, db)

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "ensure effective app/name unique index") {
		t.Fatalf("bootstrapMetaTables() error = %v, want unique index failure", err)
	}
}

func TestBootstrapMetaTablesFullPathEnsuresUniqueIndex(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
	if !db.Migrator().HasIndex(&meta.Model{}, "uidx_meta_model_app_name_alive") {
		t.Fatal("expected live unique index after full bootstrap")
	}
}

func TestBootstrapMetaTablesSkipsUniqueIndexWhenModelMissing(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	// Skip catalog tables so meta_model stays absent; index ensure must no-op.
	manager.entities = nil

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v, want nil when meta_model is absent", err)
	}
}

func TestBootstrapMetaTablesPropagatesUniqueIndexError(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatalf("migrate meta_model: %v", err)
	}
	seedDuplicateLiveModelsForTest(t, db)

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "ensure effective app/name unique index") {
		t.Fatalf("bootstrapMetaTables() error = %v, want unique index failure", err)
	}
}

func TestPeekLoadMigrateNilGuardsAndPreloads(t *testing.T) {
	t.Parallel()
	var nilMgr *ModuleManager
	if _, err := nilMgr.peekModuleRecord(context.Background(), "x"); err == nil {
		t.Fatal("nil peekModuleRecord want error")
	}
	if _, err := nilMgr.loadModuleRecord("x"); err == nil {
		t.Fatal("nil loadModuleRecord want error")
	}
	db := newModuleSourceSnapshotSyncDB(t)
	scopeNoSession := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	scopeNoSession.session = nil
	bare := NewModuleManager(scopeNoSession, nil)
	bare.bootstrapOnce.Do(func() {})
	if _, err := bare.peekModuleRecord(nil, "missing-local-module"); err != nil {
		_ = err // nil ctx normalize + origin miss
	}
	if err := bare.migrateBaseModule(); err == nil || !strings.Contains(err.Error(), "session is unavailable") {
		t.Fatalf("migrateBaseModule nil session = %v", err)
	}

	db2 := newModuleSourceSnapshotSyncDB(t)
	if err := db2.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("catalog migrate: %v", err)
	}
	core := &meta.Module{Name: "core", Version: "1.0.0", Status: meta.Installed}
	auth := &meta.Module{Name: "auth", Version: "1.0.0", Status: meta.Installed, Dependencies: []*meta.Module{core}}
	if err := db2.Create(core).Error; err != nil {
		t.Fatalf("seed core: %v", err)
	}
	if err := db2.Create(auth).Error; err != nil {
		t.Fatalf("seed auth: %v", err)
	}
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db2)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	if _, err := manager.peekModuleRecord(nil, "missing-local-module"); err != nil {
		_ = err // origin miss is fine; covers nil-ctx normalize
	}
	loaded, err := manager.loadModuleRecord("auth")
	if err != nil {
		t.Fatalf("loadModuleRecord(auth): %v", err)
	}
	if loaded == nil || len(loaded.Dependencies) == 0 {
		t.Fatalf("expected Dependencies preload, got %#v", loaded)
	}
	if _, err := manager.loadModuleRecord("core"); err != nil {
		t.Fatalf("loadModuleRecord(core): %v", err)
	}
}

func TestModuleManagerMigrateBaseModuleEnsureUniqueIndexError(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	if err := manager.migrateBaseModule(); err != nil {
		t.Fatalf("migrateBaseModule() error = %v", err)
	}
	if err := db.Exec(`DROP INDEX IF EXISTS "uidx_meta_model_app_name_alive"`).Error; err != nil {
		t.Fatalf("drop unique index: %v", err)
	}
	seedDuplicateLiveModelsForTest(t, db)

	err := manager.migrateBaseModule()
	if err == nil || !strings.Contains(err.Error(), "ensure effective app/name unique index") {
		t.Fatalf("migrateBaseModule() error = %v, want unique index failure", err)
	}
}

// seedDuplicateLiveModelsForTest inserts two live meta_model rows that share
// (application, name) so EnsureEffectiveAppNameUniqueIndex must fail.
func seedDuplicateLiveModelsForTest(t *testing.T, db *gorm.DB) {
	t.Helper()
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	for _, id := range []string{"dup-a", "dup-b"} {
		row := &meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: id, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "Partner",
			Application: "partner",
			Path:        "/" + id + ".ts",
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(row).Error; err != nil {
			t.Fatalf("seed duplicate %s: %v", id, err)
		}
	}
}

func TestModuleManagerListInstalledApps(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}
	for _, row := range []meta.Module{
		{Name: "crm_core", ApplicationStr: "crm", Status: meta.Installed},
		{Name: "web", ApplicationStr: "web", Status: meta.Installed},
		{Name: "blank_app", ApplicationStr: "  ", Status: meta.Installed},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed module %q: %v", row.Name, err)
		}
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	apps, err := manager.listInstalledApps(context.Background())
	if err != nil {
		t.Fatalf("listInstalledApps() error = %v", err)
	}
	if len(apps) != 2 || apps[0] != "crm" || apps[1] != "web" {
		t.Fatalf("listInstalledApps() = %#v, want [crm web]", apps)
	}

	apps, err = manager.listInstalledApps(nil)
	if err != nil {
		t.Fatalf("listInstalledApps(nil ctx) error = %v", err)
	}
	if len(apps) != 2 || apps[0] != "crm" || apps[1] != "web" {
		t.Fatalf("listInstalledApps(nil ctx) = %#v, want [crm web]", apps)
	}
}

func TestModuleManagerListInstalledAppsQueryError(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}
	if err := db.Migrator().DropTable(&meta.Module{}); err != nil {
		t.Fatalf("drop meta_module: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	_, err := manager.listInstalledApps(context.Background())
	if err == nil || !strings.Contains(err.Error(), "list installed apps") {
		t.Fatalf("listInstalledApps() error = %v, want list installed apps failure", err)
	}
}

func TestModuleOpCtxBinderDelegatesToCtxMethods(t *testing.T) {
	t.Parallel()
	db := newModuleSourceSnapshotSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}

	runtimeScope := newModuleSourceSnapshotSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	opCtx := newOpContext()
	binder := moduleOpCtxBinder{m: manager, opCtx: opCtx}

	mod := &meta.Module{Name: "demo", Status: meta.Installed, Version: "1.0.0"}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	opCtx.markInstallDone(mod.Name)
	opCtx.markUninstallDone(mod.Name)
	opCtx.markUpgradeDone(mod.Name)

	if err := binder.install(mod); err != nil {
		t.Fatalf("binder.install() error = %v, want nil for done module", err)
	}
	if err := binder.uninstall(mod); err != nil {
		t.Fatalf("binder.uninstall() error = %v, want nil for done module", err)
	}
	if err := binder.upgrade(mod); err != nil {
		t.Fatalf("binder.upgrade() error = %v, want nil for done module", err)
	}
}
