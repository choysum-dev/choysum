// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestEnsureAbstractModelAndReplaceModuleDeclarations(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "facade.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}

	if err := EnsureAbstractModel(nil, AbstractModelSpec{}); err == nil {
		t.Fatal("expected nil db error")
	}
	if err := EnsureAbstractModel(db, AbstractModelSpec{Name: "I18n"}); err == nil {
		t.Fatal("expected missing fields error")
	}

	if err := EnsureAbstractModel(db, AbstractModelSpec{
		Name:         "I18n",
		Path:         "go://i18n/auth",
		Application:  "auth",
		ServiceNames: []string{"GetTranslations"},
	}); err != nil {
		t.Fatalf("EnsureAbstractModel: %v", err)
	}
	if err := FlushEffective(db, []LogicalKey{{Application: "auth", Name: "I18n"}}); err != nil {
		t.Fatalf("FlushEffective: %v", err)
	}
	var n int64
	if err := db.Model(&Model{}).Where("application = ? AND name = ?", "auth", "I18n").Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("effective I18n count=%d err=%v", n, err)
	}

	modID := "mod-1"
	tree := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "raw-partner", Valid: true}},
		Name:        "Partner",
		Path:        "/m/partner/service/models/partner.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	keys, err := ReplaceModuleDeclarations(db, modID, []*Model{tree})
	if err != nil {
		t.Fatalf("ReplaceModuleDeclarations: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("expected keys")
	}
	if err := FlushEffective(db, keys); err != nil {
		t.Fatalf("flush partner: %v", err)
	}
	if err := db.Model(&Model{}).Where("application = ? AND name = ?", "partner", "Partner").Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("effective Partner count=%d err=%v", n, err)
	}
}

func TestReplaceModuleDeclarations_RollsBackOnPersistFailure(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "facade-rollback.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}

	modID := "mod-rollback"
	prevPath := "/m/partner/service/models/old.ts"
	prev := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "raw-old", Valid: true}},
		Name:        "OldPartner",
		Path:        prevPath,
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	if err := PersistModelTreeAsRaw(db, prev); err != nil {
		t.Fatalf("seed previous declaration: %v", err)
	}

	_, err = ReplaceModuleDeclarations(db, modID, []*Model{
		{
			BaseModel:   BaseModel{Id: sql.NullString{String: "dup-id", Valid: true}},
			Name:        "First",
			Path:        "/m/partner/service/models/first.ts",
			Application: "partner",
		},
		{
			BaseModel:   BaseModel{Id: sql.NullString{String: "dup-id", Valid: true}},
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
