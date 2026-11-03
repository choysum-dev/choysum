// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func TestBootstrapMetaTablesSkipsWhenBaseTablesExist(t *testing.T) {
	t.Parallel()
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

func TestRetireMetaSettingTableNilGuards(t *testing.T) {
	t.Parallel()
	var nilManager *ModuleManager
	if err := nilManager.retireMetaSettingTable(); err != nil {
		t.Fatalf("nil receiver: %v", err)
	}
	manager := &ModuleManager{}
	if err := manager.retireMetaSettingTable(); err != nil {
		t.Fatalf("nil runtimeScope: %v", err)
	}
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), nil)
	manager = NewModuleManager(runtimeScope, nil)
	if err := manager.retireMetaSettingTable(); err != nil {
		t.Fatalf("nil session DB: %v", err)
	}
}

func TestEnsureMetaTablesRetireError(t *testing.T) {
	t.Parallel()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`).Error; err != nil {
		t.Fatalf("create meta_setting: %v", err)
	}
	forced := errors.New("forced drop failure")
	registerFailDropMetaSetting(t, db, forced)

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	err := manager.ensureMetaTables()
	if err == nil || !strings.Contains(err.Error(), "drop retired meta_setting") {
		t.Fatalf("ensureMetaTables() error = %v, want retire drop failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("ensureMetaTables() error = %v, want forced cause", err)
	}
}

func TestEnsureMetaTablesPreservesBootstrapError(t *testing.T) {
	t.Parallel()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	seedDuplicateLiveModelsForTest(t, db)

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
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

func TestEnsureMetaTablesRetriesDeferredRetireAfterAuthAppears(t *testing.T) {
	t.Parallel()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create meta_setting: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", "system.init.done", "true", time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("seed legacy marker: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	if err := manager.ensureMetaTables(); err != nil {
		t.Fatalf("ensureMetaTables() before auth: %v", err)
	}
	if !db.Migrator().HasTable("meta_setting") {
		t.Fatal("expected meta_setting kept until auth_app_setting exists")
	}

	if err := db.Exec(`CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create auth_app_setting: %v", err)
	}
	if err := manager.ensureMetaTables(); err != nil {
		t.Fatalf("ensureMetaTables() after auth: %v", err)
	}
	if db.Migrator().HasTable("meta_setting") {
		t.Fatal("expected meta_setting dropped after deferred retire")
	}
	var count int64
	if err := db.Table("auth_app_setting").Where(map[string]any{"key": "setup.completed_at"}).Count(&count).Error; err != nil {
		t.Fatalf("count setup.completed_at: %v", err)
	}
	if count != 1 {
		t.Fatalf("setup.completed_at rows=%d, want 1", count)
	}
}

func TestBootstrapMetaTablesEnsuresUniqueIndexWhenModelExists(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func registerFailDropMetaSetting(t *testing.T, db *gorm.DB, fail error) {
	t.Helper()
	const name = "fail_drop_meta_setting"
	if err := db.Callback().Raw().Before("gorm:raw").Register(name, func(tx *gorm.DB) {
		sqlText := ""
		if tx.Statement != nil {
			sqlText = strings.ToUpper(tx.Statement.SQL.String())
		}
		if strings.Contains(sqlText, "DROP") && strings.Contains(sqlText, "META_SETTING") {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatalf("register drop callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Raw().Remove(name) })
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
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}
	if err := db.Migrator().DropTable(&meta.Module{}); err != nil {
		t.Fatalf("drop meta_module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	_, err := manager.listInstalledApps(context.Background())
	if err == nil || !strings.Contains(err.Error(), "list installed apps") {
		t.Fatalf("listInstalledApps() error = %v, want list installed apps failure", err)
	}
}

func TestModuleOpCtxBinderDelegatesToCtxMethods(t *testing.T) {
	t.Parallel()
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
