// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/rs/xid"
)

func TestListDeclarations_FiltersAndPreload(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	falseVal := false
	mod := xid.New().String()
	if err := PersistModelTreeAsRaw(db, &Model{
		Name: "AppSetting", Path: "/a/virt", Application: "crm", Abstract: false, AutoMigrate: &falseVal,
		ModuleId: sql.NullString{String: mod, Valid: true},
		Fields:   []*Field{{Name: "Key"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := PersistModelTreeAsRaw(db, &Model{
		Name: "AppSetting", Path: "/a/hand", Application: "crm", Abstract: false,
		ModuleId: sql.NullString{String: mod, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := PersistModelTreeAsRaw(db, &Model{
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

func TestUpsertDeclaration_CreateThenIdempotentServices(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	src := NewAbstractReadonlyDeclaration("I18n", "go://i18n/auth", "auth", sql.NullString{}, []string{"GetTranslations", "SearchTerms"})
	if err := UpsertDeclaration(db, src); err != nil {
		t.Fatal(err)
	}
	src.Services = append(src.Services, &Service{Name: "UpdateTerm", IsStatic: true, AccessibilityModifier: "public"})
	mod := sql.NullString{String: xid.New().String(), Valid: true}
	src.ModuleId = mod
	if err := UpsertDeclaration(db, src); err != nil {
		t.Fatal(err)
	}

	listed, err := ListDeclarations(db, DeclarationQuery{
		Application: "auth", Name: "I18n", Path: "go://i18n/auth", PreloadTree: true,
	})
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: n=%d err=%v", len(listed), err)
	}
	if !listed[0].ModuleId.Valid || listed[0].ModuleId.String != mod.String {
		t.Fatalf("module_id not updated: %+v", listed[0].ModuleId)
	}
	if len(listed[0].Services) != 3 {
		t.Fatalf("services=%d want 3", len(listed[0].Services))
	}
}

func TestPreferDeclarationTip_PromotesCanonical(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	older := time.Now().UTC().Add(-time.Hour)
	newer := older.Add(time.Minute)
	ext := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: newer, UpdatedAt: newer},
		Name:        "I18n",
		Path:        "/ext",
		Application: "auth",
		Abstract:    true,
	}
	canon := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: older, UpdatedAt: older},
		Name:        "I18n",
		Path:        "go://i18n/auth",
		Application: "auth",
		Abstract:    true,
	}
	for _, row := range []*RawModel{ext, canon} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := PreferDeclarationTip(db, "auth", "I18n", "go://i18n/auth"); err != nil {
		t.Fatal(err)
	}
	var tip RawModel
	if err := db.Where("path = ?", "go://i18n/auth").Take(&tip).Error; err != nil {
		t.Fatal(err)
	}
	if !tip.CreatedAt.After(ext.CreatedAt) || !tip.UpdatedAt.After(ext.UpdatedAt) {
		t.Fatalf("tip not promoted: tip=%v ext=%v", tip.CreatedAt, ext.CreatedAt)
	}
}

func TestDeleteDeclarationTrees_Cascade(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	id := xid.New().String()
	if err := PersistModelTreeAsRaw(db, &Model{
		BaseModel: BaseModel{Id: sql.NullString{String: id, Valid: true}},
		Name:      "X", Path: "/x", Application: "a",
		Fields:   []*Field{{Name: "F"}},
		Services: []*Service{{Name: "S", AccessibilityModifier: "public"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteDeclarationTrees(db, []string{id, id, " ", ""}); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Unscoped().Model(&RawModel{}).Where("id = ?", id).Count(&n).Error; err != nil || n != 0 {
		t.Fatalf("left=%d err=%v", n, err)
	}
	if err := DeleteDeclarationTrees(nil, []string{"x"}); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}
	if err := UpsertDeclaration(nil, &Model{}); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("upsert nil: %v", err)
	}
	if !HasDeclarationCatalog(db) || !HasEffectiveCatalog(db) {
		t.Fatal("catalog helpers")
	}
	if HasDeclarationCatalog(nil) || HasEffectiveCatalog(nil) {
		t.Fatal("nil catalog")
	}
}
