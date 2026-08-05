// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestPickAppSettingOwnerModule_LastEligible(t *testing.T) {
	mods := []*meta.Module{
		nil,
		{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip_path", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip_entry", Path: "/virtual/modules/x", ApplicationStr: "partner", ServiceEntryPoint: ""},
		{Name: "partner_bank", Path: "/virtual/modules/partner_bank", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}
	owner := pickAppSettingOwnerModule("partner", mods)
	if owner == nil || owner.Name != "partner_bank" {
		t.Fatalf("expected last eligible owner partner_bank, got %#v", owner)
	}
	if pickAppSettingOwnerModule("core", mods) != nil {
		t.Fatal("core must not pick an owner")
	}
	if pickAppSettingOwnerModule("", mods) != nil {
		t.Fatal("empty app must return nil")
	}
	if pickAppSettingOwnerModule("partner", nil) != nil {
		t.Fatal("empty mods must return nil")
	}
	if pickAppSettingOwnerModule("partner", []*meta.Module{
		{Name: "no", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}) != nil {
		t.Fatal("no eligible module must return nil")
	}
}

type stubBundleC2Injector struct {
	err error
	n   int
	got []*meta.Module
}

func (s *stubBundleC2Injector) BundleInjectAppModels(mods []*meta.Module) error {
	s.n++
	s.got = append([]*meta.Module(nil), mods...)
	return s.err
}

func TestEnsureBundleC2VirtualImports_BundleInject(t *testing.T) {
	owners := []*meta.Module{{Name: "crm_partner", Path: "/m", ApplicationStr: "crm", ServiceEntryPoint: "service/main.ts"}}
	asOwners := []*meta.Module{
		{Name: "crm_partner", Path: "/m", ApplicationStr: "crm", ServiceEntryPoint: "service/main.ts"},
		{Name: "crm_extra", Path: "/m2", ApplicationStr: "crm", ServiceEntryPoint: "service/main.ts"},
	}

	okStub := &stubBundleC2Injector{}
	if err := ensureBundleC2VirtualImports(okStub, owners, asOwners); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if okStub.n != 1 {
		t.Fatalf("expected BundleInjectAppModels once, got %d", okStub.n)
	}
	if len(okStub.got) != 2 {
		t.Fatalf("expected merged unique owners, got %#v", okStub.got)
	}

	fail := &stubBundleC2Injector{err: errors.New("boom")}
	err := ensureBundleC2VirtualImports(fail, owners, owners)
	if err == nil || !strings.Contains(err.Error(), "inject app models for bundles") {
		t.Fatalf("expected wrap, got %v", err)
	}

	if err := ensureBundleC2VirtualImports(struct{}{}, owners, owners); err != nil {
		t.Fatalf("unexpected error for bare builder: %v", err)
	}
}

func TestBuildBackendBundlesToDir_AppSettingOwnerAndEnsureError(t *testing.T) {
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
