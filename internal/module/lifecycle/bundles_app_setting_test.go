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
	fdErr error
	asErr error
	fdN   int
	asN   int
}

func (s *stubBundleC2Injector) EnsureFieldDefaultVirtualImports([]*meta.Module) error {
	s.fdN++
	return s.fdErr
}

func (s *stubBundleC2Injector) EnsureAppSettingVirtualImports([]*meta.Module) error {
	s.asN++
	return s.asErr
}

func TestEnsureBundleC2VirtualImports_AppSettingError(t *testing.T) {
	owners := []*meta.Module{{Name: "crm_partner", Path: "/m", ApplicationStr: "crm", ServiceEntryPoint: "service/main.ts"}}

	okStub := &stubBundleC2Injector{}
	if err := ensureBundleC2VirtualImports(okStub, owners, owners); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if okStub.fdN != 1 || okStub.asN != 1 {
		t.Fatalf("expected both Ensures called once, got fd=%d as=%d", okStub.fdN, okStub.asN)
	}

	asFail := &stubBundleC2Injector{asErr: errors.New("as boom")}
	err := ensureBundleC2VirtualImports(asFail, owners, owners)
	if err == nil || !strings.Contains(err.Error(), "inject AppSetting virtual imports") {
		t.Fatalf("expected AppSetting Ensure wrap, got %v", err)
	}
	if asFail.fdN != 1 || asFail.asN != 1 {
		t.Fatalf("FieldDefault should succeed before AppSetting fails: fd=%d as=%d", asFail.fdN, asFail.asN)
	}

	fdFail := &stubBundleC2Injector{fdErr: errors.New("fd boom")}
	err = ensureBundleC2VirtualImports(fdFail, owners, owners)
	if err == nil || !strings.Contains(err.Error(), "inject FieldDefault virtual imports") {
		t.Fatalf("expected FieldDefault Ensure wrap, got %v", err)
	}
	if fdFail.asN != 0 {
		t.Fatal("AppSetting Ensure must not run after FieldDefault failure")
	}

	// Builder without Ensure methods is a no-op.
	if err := ensureBundleC2VirtualImports(struct{}{}, owners, owners); err != nil {
		t.Fatalf("unexpected error for bare builder: %v", err)
	}

	fdOnly := &fieldDefaultOnlyBundleInjector{}
	if err := ensureBundleC2VirtualImports(fdOnly, owners, owners); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fdOnly.n != 1 {
		t.Fatalf("expected FieldDefault-only Ensure once, got %d", fdOnly.n)
	}

	asOnly := &appSettingOnlyBundleInjector{}
	if err := ensureBundleC2VirtualImports(asOnly, owners, owners); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asOnly.n != 1 {
		t.Fatalf("expected AppSetting-only Ensure once, got %d", asOnly.n)
	}
}

type fieldDefaultOnlyBundleInjector struct {
	n int
}

func (s *fieldDefaultOnlyBundleInjector) EnsureFieldDefaultVirtualImports([]*meta.Module) error {
	s.n++
	return nil
}

type appSettingOnlyBundleInjector struct {
	n int
}

func (s *appSettingOnlyBundleInjector) EnsureAppSettingVirtualImports([]*meta.Module) error {
	s.n++
	return nil
}

func TestBuildBackendBundlesToDir_AppSettingOwnerAndEnsureError(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
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

	// Drop meta_raw_model so Ensure*VirtualImports fails while owners were collected.
	// FieldDefault Ensure runs first, so the surfaced error usually names FieldDefault;
	// AppSetting Ensure shares the same owner-collection + entry-write path.
	if err := db.Migrator().DropTable(&meta.RawModel{}); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	err := manager.buildBackendBundlesToDir(context.Background(), distBundlesDir, nil)
	if err == nil {
		t.Fatal("expected Ensure error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "inject FieldDefault virtual imports") &&
		!strings.Contains(msg, "inject AppSetting virtual imports") {
		t.Fatalf("expected Ensure virtual-imports error, got %v", err)
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
