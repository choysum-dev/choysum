// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"testing"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

func TestFillCatalogFromManifest_ReadsChoysumNestedFields(t *testing.T) {
	t.Parallel()
	row := &modmeta.ModuleCatalog{}
	fillCatalogFromManifest(row, datatypes.JSON(`{
		"description": "Partner commercial extension",
		"author": "Choysum Team",
		"license": "Apache-2.0",
		"choysum": {
			"moduleName": "partner_commercial",
			"application": "partner",
			"category": "Partner",
			"depends": ["partner", "base", "auth"]
		}
	}`))
	if row.Application != "partner" {
		t.Fatalf("Application = %q, want partner from choysum.application", row.Application)
	}
	if row.Category != "Partner" {
		t.Fatalf("Category = %q, want Partner from choysum.category", row.Category)
	}
	if string(row.DependsJson) == "" {
		t.Fatal("expected DependsJson from choysum.depends")
	}
	if row.Description != "Partner commercial extension" {
		t.Fatalf("Description = %q", row.Description)
	}
}

func TestMaterializeCatalog_MergesLocalRegistryAndInstalled(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core",
		SourceType: originLocal,
		SourceRef:  "local",
		Available:  true,
		// Internal local sync stores v-prefixed SemVer; catalog UI must show npm form.
		Version: nullString("v1.0.0"),
	}).Error; err != nil {
		t.Fatalf("local snapshot: %v", err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core",
		SourceType: originRegistry,
		SourceRef:  "@choysum-dev/core",
		Available:  true,
		Version:    nullString("v1.1.0"),
		ManifestJson: datatypes.JSON(`{
			"package": "@choysum-dev/core",
			"versions": {
				"1.1.0": {"tarball": "https://registry.npmjs.org/@choysum-dev/core/-/core-1.1.0.tgz"}
			}
		}`),
	}).Error; err != nil {
		t.Fatalf("registry snapshot: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name:           "core",
		Version:        "v0.9.0",
		Status:         pkgmeta.Installed,
		ApplicationStr: "core",
	}).Error; err != nil {
		t.Fatalf("installed: %v", err)
	}
	installAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	upgradeAt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := db.Create(&modmeta.ModuleOperation{
		Action:     ActionInstall,
		ModuleName: "core",
		Status:     StatusSucceeded,
		FinishedAt: &installAt,
	}).Error; err != nil {
		t.Fatalf("install op: %v", err)
	}
	if err := db.Create(&modmeta.ModuleOperation{
		Action:     ActionUpgrade,
		ModuleName: "core",
		Status:     StatusSucceeded,
		FinishedAt: &upgradeAt,
	}).Error; err != nil {
		t.Fatalf("upgrade op: %v", err)
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
	if row.RegistryPath.String != "@choysum-dev/core" {
		t.Fatalf("RegistryPath = %q", row.RegistryPath.String)
	}
	if row.RegistryTarball.String != "https://registry.npmjs.org/@choysum-dev/core/-/core-1.1.0.tgz" {
		t.Fatalf("RegistryTarball = %q", row.RegistryTarball.String)
	}
	if !row.Installed || row.InstalledVersion.String != "0.9.0" {
		t.Fatalf("Installed=%v version=%q", row.Installed, row.InstalledVersion.String)
	}
	if row.LastInstalledOrUpgradedAt == nil || !row.LastInstalledOrUpgradedAt.Equal(upgradeAt) {
		t.Fatalf("LastInstalledOrUpgradedAt = %v, want upgrade time %v", row.LastInstalledOrUpgradedAt, upgradeAt)
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
