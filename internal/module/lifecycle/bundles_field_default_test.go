// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestPickFieldDefaultOwnerModule_LastEligible(t *testing.T) {
	mods := []*meta.Module{
		nil,
		{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip_path", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip_entry", Path: "/virtual/modules/x", ApplicationStr: "partner", ServiceEntryPoint: ""},
		{Name: "partner_bank", Path: "/virtual/modules/partner_bank", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}
	owner := pickFieldDefaultOwnerModule("partner", mods)
	if owner == nil || owner.Name != "partner_bank" {
		t.Fatalf("expected last eligible owner partner_bank, got %#v", owner)
	}
	if pickFieldDefaultOwnerModule("core", mods) != nil {
		t.Fatal("core must not pick an owner")
	}
	if pickFieldDefaultOwnerModule("", mods) != nil {
		t.Fatal("empty app must return nil")
	}
	if pickFieldDefaultOwnerModule("partner", nil) != nil {
		t.Fatal("empty mods must return nil")
	}
	if pickFieldDefaultOwnerModule("partner", []*meta.Module{
		{Name: "no", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}) != nil {
		t.Fatal("no eligible module must return nil")
	}
}

func TestBuildBackendBundlesToDir_FieldDefaultOwnerAndEnsureError(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
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
	if err := meta.DropRawModelTable(db); err != nil {
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
