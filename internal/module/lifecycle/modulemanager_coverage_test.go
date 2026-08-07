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
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
}

func TestBootstrapMetaTablesEnsuresUniqueIndexWhenModelExists(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
	if !db.Migrator().HasIndex(&meta.Model{}, "uidx_meta_model_app_name_alive") {
		t.Fatal("expected live unique index after early-path bootstrap")
	}
}

func TestBootstrapMetaTablesEnsureUniqueIndexErrorWhenModelExists(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	seedDuplicateLiveModelsForTest(t, db)

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "ensure effective app/name unique index") {
		t.Fatalf("bootstrapMetaTables() error = %v, want unique index failure", err)
	}
}

func TestBootstrapMetaTablesFullPathEnsuresUniqueIndex(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
	if !db.Migrator().HasIndex(&meta.Model{}, "uidx_meta_model_app_name_alive") {
		t.Fatal("expected live unique index after full bootstrap")
	}
}

func TestBootstrapMetaTablesFullPathEnsureUniqueIndexError(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	// Skip catalog tables so Ensure runs against a missing meta_model.
	manager.entities = nil

	err := manager.bootstrapMetaTables(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "ensure effective app/name unique index") {
		t.Fatalf("bootstrapMetaTables() error = %v, want unique index failure", err)
	}
}

func TestModuleManagerMigrateBaseModuleEnsureUniqueIndexError(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
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

func TestModuleManagerListInstalledNonWebApps(t *testing.T) {
	db := newModuleIndexSyncDB(t)
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

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	apps, err := manager.listInstalledNonWebApps(context.Background())
	if err != nil {
		t.Fatalf("listInstalledNonWebApps() error = %v", err)
	}
	if len(apps) != 1 || apps[0] != "crm" {
		t.Fatalf("listInstalledNonWebApps() = %#v, want [crm]", apps)
	}

	apps, err = manager.listInstalledNonWebApps(nil)
	if err != nil {
		t.Fatalf("listInstalledNonWebApps(nil ctx) error = %v", err)
	}
	if len(apps) != 1 || apps[0] != "crm" {
		t.Fatalf("listInstalledNonWebApps(nil ctx) = %#v, want [crm]", apps)
	}
}

func TestModuleManagerListInstalledNonWebAppsQueryError(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}
	if err := db.Migrator().DropTable(&meta.Module{}); err != nil {
		t.Fatalf("drop meta_module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	_, err := manager.listInstalledNonWebApps(context.Background())
	if err == nil || !strings.Contains(err.Error(), "list installed apps") {
		t.Fatalf("listInstalledNonWebApps() error = %v, want list installed apps failure", err)
	}
}

func TestModuleOpCtxBinderDelegatesToCtxMethods(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
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
