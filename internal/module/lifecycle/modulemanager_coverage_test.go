// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"strings"
	"testing"

	leasemodel "github.com/choysum-dev/choysum/internal/state/lease/model"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestBootstrapMetaTablesSkipsWhenBaseTablesExist(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&meta.Module{}, &leasemodel.LockLease{}); err != nil {
		t.Fatalf("auto migrate base tables: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := manager.bootstrapMetaTables(runtimeScope); err != nil {
		t.Fatalf("bootstrapMetaTables() error = %v", err)
	}
}

func TestModuleManagerListInstalledNonWebApps(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
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
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
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
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
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
