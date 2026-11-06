// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestMaterializeCatalog_MergesLocalRegistryAndInstalled(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core",
		SourceType: originLocal,
		SourceRef:  "local",
		Available:  true,
		Version:    nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatalf("local snapshot: %v", err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core",
		SourceType: originRegistry,
		SourceRef:  "registry",
		Available:  true,
		Version:    nullString("1.1.0"),
	}).Error; err != nil {
		t.Fatalf("registry snapshot: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name:           "core",
		Version:        "0.9.0",
		Status:         pkgmeta.Installed,
		ApplicationStr: "core",
	}).Error; err != nil {
		t.Fatalf("installed: %v", err)
	}

	if err := o.MaterializeCatalog(context.Background(), []string{"core"}); err != nil {
		t.Fatalf("MaterializeCatalog: %v", err)
	}

	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "core").Take(&row).Error; err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if row.LocalVersion.String != "1.0.0" {
		t.Fatalf("LocalVersion = %q", row.LocalVersion.String)
	}
	if row.RegistryVersion.String != "1.1.0" {
		t.Fatalf("RegistryVersion = %q", row.RegistryVersion.String)
	}
	if !row.Installed || row.InstalledVersion.String != "0.9.0" {
		t.Fatalf("Installed=%v version=%q", row.Installed, row.InstalledVersion.String)
	}
	if !row.UpdateAvailable {
		t.Fatal("expected UpdateAvailable (local 1.0.0 > installed 0.9.0)")
	}
	if !row.RegistryNewerThanLocal {
		t.Fatal("expected RegistryNewerThanLocal (1.1.0 > 1.0.0)")
	}
	if !row.Available {
		t.Fatal("expected Available")
	}
}

func TestMaterializeCatalog_UpdateAvailableUsesRegistryWhenNoLocal(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "auth",
		SourceType: originRegistry,
		SourceRef:  "registry",
		Available:  true,
		Version:    nullString("2.0.0"),
	}).Error; err != nil {
		t.Fatalf("registry snapshot: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name:           "auth",
		Version:        "1.0.0",
		Status:         pkgmeta.Installed,
		ApplicationStr: "auth",
	}).Error; err != nil {
		t.Fatalf("installed: %v", err)
	}

	if err := o.MaterializeCatalog(context.Background(), nil); err != nil {
		t.Fatalf("MaterializeCatalog: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "auth").Take(&row).Error; err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if !row.UpdateAvailable {
		t.Fatal("expected UpdateAvailable from registry when local absent")
	}
	if row.RegistryNewerThanLocal {
		t.Fatal("RegistryNewerThanLocal should be false without local")
	}
}
