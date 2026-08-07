// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func TestListDeclarations_FiltersAndPreload(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	falseVal := false
	mod := xid.New().String()
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		Name: "AppSetting", Path: "/a/virt", Application: "crm", Abstract: false, AutoMigrate: &falseVal,
		ModuleId: sql.NullString{String: mod, Valid: true},
		Fields:   []*pkgmeta.Field{{Name: "Key"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		Name: "AppSetting", Path: "/a/hand", Application: "crm", Abstract: false,
		ModuleId: sql.NullString{String: mod, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		Name: "Other", Path: "/o", Application: "crm", Abstract: true,
	}); err != nil {
		t.Fatal(err)
	}

	absFalse := false
	got, err := ListDeclarations(db, DeclarationQuery{
		Application: "crm", Name: "AppSetting", Abstract: &absFalse, PreloadTree: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	var sawField bool
	for _, m := range got {
		if m.Path == "/a/virt" && len(m.Fields) == 1 && m.Fields[0].Name == "Key" {
			sawField = true
		}
	}
	if !sawField {
		t.Fatalf("expected preloaded field on virt: %+v", got)
	}

	byMod, err := ListDeclarations(db, DeclarationQuery{ModuleID: mod})
	if err != nil || len(byMod) != 2 {
		t.Fatalf("by module: n=%d err=%v", len(byMod), err)
	}
}

func TestDeleteDeclarationTrees_Cascade(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	id := xid.New().String()
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: id, Valid: true}},
		Name:      "X", Path: "/x", Application: "a",
		Fields:   []*pkgmeta.Field{{Name: "F"}},
		Services: []*pkgmeta.Service{{Name: "S", AccessibilityModifier: "public"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteDeclarationTrees(db, []string{id, id, " ", ""}); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Unscoped().Model(&rawModel{}).Where("id = ?", id).Count(&n).Error; err != nil || n != 0 {
		t.Fatalf("left=%d err=%v", n, err)
	}
	for name, dest := range map[string]any{"field": &rawField{}, "service": &rawService{}} {
		var c int64
		if err := db.Unscoped().Model(dest).Where("model_id = ?", id).Count(&c).Error; err != nil || c != 0 {
			t.Fatalf("%s rows left=%d err=%v", name, c, err)
		}
	}
	if err := DeleteDeclarationTrees(nil, []string{"x"}); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}
}

func TestReplaceModuleDeclarations(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}

	modID := "mod-1"
	tree := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-partner", Valid: true}},
		Name:        "Partner",
		Path:        "/m/partner/service/models/partner.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	keys, err := ReplaceModuleDeclarations(db, modID, []*pkgmeta.Model{tree})
	if err != nil {
		t.Fatalf("ReplaceModuleDeclarations: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("expected keys")
	}
	if err := FlushEffective(db, keys); err != nil {
		t.Fatalf("flush partner: %v", err)
	}
	var n int64
	if err := db.Model(&pkgmeta.Model{}).Where("application = ? AND name = ?", "partner", "Partner").Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("effective Partner count=%d err=%v", n, err)
	}
}

func TestReplaceModuleDeclarations_RollsBackOnPersistFailure(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}

	modID := "mod-rollback"
	prevPath := "/m/partner/service/models/old.ts"
	prev := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-old", Valid: true}},
		Name:        "OldPartner",
		Path:        prevPath,
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	if err := persistModelTreeAsRaw(db, prev); err != nil {
		t.Fatalf("seed previous declaration: %v", err)
	}

	_, err := ReplaceModuleDeclarations(db, modID, []*pkgmeta.Model{
		{
			BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "dup-id", Valid: true}},
			Name:        "First",
			Path:        "/m/partner/service/models/first.ts",
			Application: "partner",
		},
		{
			BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "dup-id", Valid: true}},
			Name:        "Second",
			Path:        "/m/partner/service/models/second.ts",
			Application: "partner",
		},
	})
	if err == nil {
		t.Fatal("expected persist failure for duplicate id")
	}

	decls, listErr := ListDeclarations(db, DeclarationQuery{ModuleID: modID})
	if listErr != nil {
		t.Fatalf("list after failed replace: %v", listErr)
	}
	if len(decls) != 1 || decls[0] == nil || decls[0].Path != prevPath || decls[0].Name != "OldPartner" {
		t.Fatalf("expected previous declaration retained, got %#v", decls)
	}
}

func TestRemoveModuleDeclarations(t *testing.T) {
	db := openDeclarationTestDB(t, "remove-decls")
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	modID := "mod-remove"
	if _, err := ReplaceModuleDeclarations(db, modID, []*pkgmeta.Model{{
		Name: "Partner", Path: "/models/partner", Application: "partner",
		ModuleId: sql.NullString{String: modID, Valid: true},
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	keys, err := RemoveModuleDeclarations(db, modID)
	if err != nil {
		t.Fatalf("RemoveModuleDeclarations: %v", err)
	}
	if len(keys) != 1 || keys[0].Application != "partner" || keys[0].Name != "Partner" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
	remaining, err := ListDeclarations(db, DeclarationQuery{ModuleID: modID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no declarations, got %#v", remaining)
	}
	// Empty / nil guards
	if keys, err := RemoveModuleDeclarations(nil, modID); err == nil || keys != nil {
		t.Fatalf("expected nil db error, got keys=%v err=%v", keys, err)
	}
	if keys, err := RemoveModuleDeclarations(db, ""); err != nil || keys != nil {
		t.Fatalf("empty moduleID: keys=%v err=%v", keys, err)
	}
}

func TestRemoveModuleDeclarations_IncludesLegacyEffectiveKeys(t *testing.T) {
	db := openDeclarationTestDB(t, "remove-eff-only")
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	modID := "mod-eff-only"
	eff := pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "eff-legacy", Valid: true}},
		Name:        "LegacyShell",
		Path:        "/legacy.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&eff).Error; err != nil {
		t.Fatalf("seed effective: %v", err)
	}
	keys, err := RemoveModuleDeclarations(db, modID)
	if err != nil {
		t.Fatalf("RemoveModuleDeclarations: %v", err)
	}
	found := false
	for _, k := range keys {
		if k.Application == "partner" && k.Name == "LegacyShell" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected legacy effective key, got %#v", keys)
	}
}
