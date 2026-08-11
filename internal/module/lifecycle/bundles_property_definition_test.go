// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestPickPropertyDefinitionOwnerModule_LastEligible(t *testing.T) {
	mods := []*meta.Module{
		nil,
		{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip_path", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip_entry", Path: "/virtual/modules/x", ApplicationStr: "partner", ServiceEntryPoint: ""},
		{Name: "partner_bank", Path: "/virtual/modules/partner_bank", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}
	owner := pickPropertyDefinitionOwnerModule("partner", mods)
	if owner == nil || owner.Name != "partner_bank" {
		t.Fatalf("expected last eligible owner partner_bank, got %#v", owner)
	}
	if pickPropertyDefinitionOwnerModule("core", mods) != nil {
		t.Fatal("core must not pick an owner")
	}
	if pickPropertyDefinitionOwnerModule("", mods) != nil {
		t.Fatal("empty app must return nil")
	}
	if pickPropertyDefinitionOwnerModule("partner", nil) != nil {
		t.Fatal("empty mods must return nil")
	}
	if pickPropertyDefinitionOwnerModule("partner", []*meta.Module{
		{Name: "no", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}) != nil {
		t.Fatal("no eligible module must return nil")
	}
}

func TestBuildBackendBundlesToDir_PropertyDefinitionOwnerAndEnsureError(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	modPath := filepath.Join(modulesPath, "crm_partner")
	if err := os.MkdirAll(filepath.Join(modPath, "service"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := db.Create(&meta.Module{
		Name:              "crm_partner",
		ApplicationStr:    "crm",
		Status:            meta.Installed,
		ServiceEntryPoint: "service/main.ts",
		Path:              modPath,
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	distBundlesDir := t.TempDir()

	// Drop meta_raw_model so BundleInjectAppModels fails while owners were collected.
	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	err := manager.buildBackendBundlesToDir(context.Background(), distBundlesDir, nil)
	if err == nil || !strings.Contains(err.Error(), "inject app models for bundles") {
		t.Fatalf("expected BundleInject error, got %v", err)
	}

	entryFilePath := filepath.Join(distBundlesDir, "__choysum_bundles_entry.ts")
	entryRaw, readErr := os.ReadFile(entryFilePath)
	if readErr != nil {
		t.Fatalf("entry should be written before Ensure: %v", readErr)
	}
	if !strings.Contains(string(entryRaw), filepath.Join(modulesPath, "crm_partner", "service/main.ts")) {
		t.Fatalf("unexpected entry contents: %s", entryRaw)
	}
}

func TestBuildBackendAppToDir_C2OwnersAppendAndEnsureError(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	firstPath := filepath.Join(modulesPath, "crm_first")
	lastPath := filepath.Join(modulesPath, "crm_last")
	for _, modPath := range []string{firstPath, lastPath} {
		if err := os.MkdirAll(filepath.Join(modPath, "service"), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", modPath, err)
		}
	}
	if err := db.Create(&meta.Module{
		Name:              "crm_first",
		ApplicationStr:    "crm",
		Status:            meta.Installed,
		ServiceEntryPoint: "service/main.ts",
		Path:              firstPath,
	}).Error; err != nil {
		t.Fatalf("seed first module: %v", err)
	}
	if err := db.Create(&meta.Module{
		Name:              "crm_last",
		ApplicationStr:    "crm",
		Status:            meta.Installed,
		ServiceEntryPoint: "service/main.ts",
		Path:              lastPath,
	}).Error; err != nil {
		t.Fatalf("seed last module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	distAppDir := t.TempDir()

	// Mirror buildBackendAppToDir owner selection: last eligible module hosts C2 inject.
	var mods []meta.Module
	if err := db.Where("application_str = ? AND status = ?", "crm", meta.Installed).Order("id ASC").Find(&mods).Error; err != nil {
		t.Fatalf("load mods: %v", err)
	}
	ptrs := make([]*meta.Module, 0, len(mods))
	for i := range mods {
		ptrs = append(ptrs, &mods[i])
	}
	if owner := pickPropertyDefinitionOwnerModule("crm", ptrs); owner == nil || owner.Name != "crm_last" {
		t.Fatalf("expected PropertyDefinition owner crm_last, got %#v", owner)
	}
	if owner := pickFieldDefaultOwnerModule("crm", ptrs); owner == nil || owner.Name != "crm_last" {
		t.Fatalf("expected FieldDefault owner crm_last, got %#v", owner)
	}

	// Drop meta_raw_model after owners would be picked so FD/AS/PD append branches run.
	// (Inject fails in dbLoadModels before virtual sources are registered, so ownership
	// cannot be observed via RegisterVirtualSource under this failure mode.)
	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	err := manager.buildBackendAppToDir(context.Background(), "crm", distAppDir)
	if err == nil || !strings.Contains(err.Error(), "inject app models for bundles") {
		t.Fatalf("expected BundleInject error, got %v", err)
	}

	entryRaw, readErr := os.ReadFile(filepath.Join(distAppDir, "__choysum_app_entry.ts"))
	if readErr != nil {
		t.Fatalf("entry should be written before Ensure: %v", readErr)
	}
	entryText := string(entryRaw)
	if !strings.Contains(entryText, filepath.Join(modulesPath, "crm_first", "service/main.ts")) ||
		!strings.Contains(entryText, filepath.Join(modulesPath, "crm_last", "service/main.ts")) {
		t.Fatalf("unexpected entry contents: %s", entryText)
	}
}
