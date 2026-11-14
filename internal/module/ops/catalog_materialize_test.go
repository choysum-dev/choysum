// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"strings"
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

func TestProjectCatalogInstalled_HealsStaleFlag(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := db.Create(&pkgmeta.Module{
		Name: "document", Version: "v1.2.3", Status: pkgmeta.Installed, ApplicationStr: "document",
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName:   "document",
		Available:    true,
		Installed:    false,
		LocalVersion: nullString("1.2.3"),
		LocalPath:    nullString("/modules/document"),
		HasDemo:      true,
		DemoLoaded:   true,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	// Uninstalled catalog row should be cleared.
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName:       "partner",
		Available:        true,
		Installed:        true,
		InstalledVersion: nullString("9.9.9"),
		LocalVersion:     nullString("9.9.9"),
	}).Error; err != nil {
		t.Fatalf("seed partner catalog: %v", err)
	}

	if err := o.ProjectCatalogInstalled(context.Background()); err != nil {
		t.Fatalf("ProjectCatalogInstalled: %v", err)
	}

	var doc modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "document").Take(&doc).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if !doc.Installed || doc.InstalledVersion.String != "1.2.3" {
		t.Fatalf("document installed=%v version=%q", doc.Installed, doc.InstalledVersion.String)
	}
	if doc.LocalVersion.String != "1.2.3" || doc.LocalPath.String != "/modules/document" {
		t.Fatalf("snapshot fields changed: local=%q path=%q", doc.LocalVersion.String, doc.LocalPath.String)
	}
	if !doc.DemoLoaded {
		t.Fatal("expected DemoLoaded preserved when version unchanged and HasDemo")
	}

	var partner modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "partner").Take(&partner).Error; err != nil {
		t.Fatalf("load partner: %v", err)
	}
	if partner.Installed || partner.InstalledVersion.Valid {
		t.Fatalf("partner installed=%v version=%q want cleared", partner.Installed, partner.InstalledVersion.String)
	}

	// Installed module without catalog row gets a minimal row.
	if err := db.Create(&pkgmeta.Module{
		Name: "task", Version: "v0.1.0", Status: pkgmeta.Installed, ApplicationStr: "task", Summary: "Tasks",
	}).Error; err != nil {
		t.Fatalf("seed task module: %v", err)
	}
	if err := o.ProjectCatalogInstalled(context.Background()); err != nil {
		t.Fatalf("ProjectCatalogInstalled (task): %v", err)
	}
	var task modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "task").Take(&task).Error; err != nil {
		t.Fatalf("load task catalog: %v", err)
	}
	if !task.Installed || task.InstalledVersion.String != "0.1.0" || task.Summary != "Tasks" {
		t.Fatalf("task catalog installed=%v version=%q summary=%q", task.Installed, task.InstalledVersion.String, task.Summary)
	}
}

func TestProjectCatalogInstalled_ToUpgradeStaysInstalled(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := db.Create(&pkgmeta.Module{
		Name: "web", Version: "v2.0.0", Status: pkgmeta.ToUpgrade, ApplicationStr: "web",
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName:       "web",
		Available:        true,
		Installed:        true,
		InstalledVersion: nullString("2.0.0"),
		LocalVersion:     nullString("2.0.0"),
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	if err := o.ProjectCatalogInstalled(context.Background()); err != nil {
		t.Fatalf("ProjectCatalogInstalled: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "web").Take(&row).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if !row.Installed || row.InstalledVersion.String != "2.0.0" {
		t.Fatalf("to-upgrade web installed=%v version=%q", row.Installed, row.InstalledVersion.String)
	}
}

func TestProjectCatalogInstalled_DemoStrPreservesLoaded(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "v1.0.0", Status: pkgmeta.Installed, ApplicationStr: "base",
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName:       "base",
		Available:        true,
		Installed:        true,
		InstalledVersion: nullString("1.0.0"),
		HasDemo:          false,
		DemoLoaded:       true,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	if err := o.ProjectCatalogInstalled(context.Background()); err != nil {
		t.Fatalf("ProjectCatalogInstalled: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&row).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if !row.DemoLoaded {
		t.Fatal("expected DemoLoaded preserved via installed DemoStr when catalog HasDemo is stale")
	}
}

func TestStampCatalogLifecycleAt_InstallUpgradeOnly(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "core", Available: true, Installed: true, InstalledVersion: nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := o.stampCatalogLifecycleAt(context.Background(), []string{"core", " ", "core"}, ActionUninstall); err != nil {
		t.Fatalf("uninstall stamp: %v", err)
	}
	var before modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "core").Take(&before).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if before.LastInstalledOrUpgradedAt != nil {
		t.Fatal("uninstall must not stamp lifecycle time")
	}
	if err := o.stampCatalogLifecycleAt(context.Background(), []string{"core"}, ActionUpgrade); err != nil {
		t.Fatalf("upgrade stamp: %v", err)
	}
	var after modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "core").Take(&after).Error; err != nil {
		t.Fatalf("load after: %v", err)
	}
	if after.LastInstalledOrUpgradedAt == nil {
		t.Fatal("expected last_installed_or_upgraded_at after upgrade stamp")
	}
}

func TestProjectCatalogInstalled_GuardsAndBranches(t *testing.T) {
	t.Parallel()
	if err := (*Ops)(nil).ProjectCatalogInstalled(context.Background()); err == nil || !strings.Contains(err.Error(), "nil host") {
		t.Fatalf("nil Project: %v", err)
	}
	if err := (*Ops)(nil).stampCatalogLifecycleAt(context.Background(), []string{"x"}, ActionInstall); err == nil || !strings.Contains(err.Error(), "nil host") {
		t.Fatalf("nil stamp: %v", err)
	}
	if err := (&Ops{}).ProjectCatalogInstalled(context.Background()); err == nil {
		t.Fatal("expected session error without scope")
	}
	if err := (&Ops{}).stampCatalogLifecycleAt(context.Background(), []string{"x"}, ActionInstall); err == nil {
		t.Fatal("expected stamp session error without scope")
	}
	if err := (&Ops{Scope: newOpsTestScope(t, newOpsTestDB(t))}).stampCatalogLifecycleAt(nil, []string{" ", ""}, ActionInstall); err != nil {
		t.Fatalf("empty names: %v", err)
	}

	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	// Empty module name on installed row is skipped; blank catalog ModuleName is skipped.
	if err := db.Create(&pkgmeta.Module{
		Name: " ", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "x",
	}).Error; err != nil {
		t.Fatalf("seed blank module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{ModuleName: " ", Available: true}).Error; err != nil {
		t.Fatalf("seed blank catalog: %v", err)
	}
	if err := o.ProjectCatalogInstalled(nil); err != nil {
		t.Fatalf("Project with nil ctx: %v", err)
	}

	// Version bump clears DemoLoaded even when HasDemo stays true.
	if err := db.Create(&pkgmeta.Module{
		Name: "auth", Version: "v2.0.0", Status: pkgmeta.Installed, ApplicationStr: "auth",
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed auth: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "auth", Available: true, Installed: true,
		InstalledVersion: nullString("1.0.0"), HasDemo: true, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed auth catalog: %v", err)
	}
	// No demo payload and HasDemo=false clears DemoLoaded.
	if err := db.Create(&pkgmeta.Module{
		Name: "core", Version: "v1.0.0", Status: pkgmeta.Installed, ApplicationStr: "core",
	}).Error; err != nil {
		t.Fatalf("seed core: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "core", Available: true, Installed: true,
		InstalledVersion: nullString("1.0.0"), HasDemo: false, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed core catalog: %v", err)
	}
	if err := o.ProjectCatalogInstalled(context.Background()); err != nil {
		t.Fatalf("Project: %v", err)
	}
	var auth, core modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "auth").Take(&auth).Error; err != nil {
		t.Fatalf("auth: %v", err)
	}
	if auth.DemoLoaded {
		t.Fatal("expected DemoLoaded cleared on version bump")
	}
	if err := db.Where("module_name = ?", "core").Take(&core).Error; err != nil {
		t.Fatalf("core: %v", err)
	}
	if core.DemoLoaded {
		t.Fatal("expected DemoLoaded cleared when HasDemo/DemoStr absent")
	}
}
