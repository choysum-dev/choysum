// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openRecomputeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "recompute.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	return db
}

func TestRecomputeEffective_CreatesPreservesAndDeletes(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	base := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "raw-base", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner/partner.ts",
		ModuleId:    sql.NullString{String: "mod-base", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(base).Error; err != nil {
		t.Fatalf("create base: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "f-name", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Name",
		ModelId:   base.Id,
	}).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}

	if err := RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("first recompute: %v", err)
	}
	var first Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "partner", "Partner").Take(&first).Error; err != nil {
		t.Fatalf("load first: %v", err)
	}
	if first.Id.String == "" || len(first.Fields) != 1 {
		t.Fatalf("unexpected first effective: %#v", first)
	}
	firstID := first.Id.String

	bank := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "raw-bank", Valid: true}, CreatedAt: ts.Add(time.Hour), UpdatedAt: ts.Add(time.Hour)},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner_bank/partner.ts",
		Extends:     "/partner/partner.ts",
		ModuleId:    sql.NullString{String: "mod-bank", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(bank).Error; err != nil {
		t.Fatalf("create bank: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "f-bank", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "BankAccounts",
		ModelId:   bank.Id,
	}).Error; err != nil {
		t.Fatalf("create bank field: %v", err)
	}

	if err := RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("second recompute: %v", err)
	}
	var second Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "partner", "Partner").Take(&second).Error; err != nil {
		t.Fatalf("load second: %v", err)
	}
	if second.Id.String != firstID {
		t.Fatalf("effective id changed: %q → %q", firstID, second.Id.String)
	}
	names := map[string]bool{}
	for _, f := range second.Fields {
		if f != nil {
			names[f.Name] = true
		}
	}
	if !names["Name"] || !names["BankAccounts"] {
		t.Fatalf("expected union fields, got %#v", names)
	}

	if err := db.Unscoped().Where("id = ?", bank.Id.String).Delete(&RawModel{}).Error; err != nil {
		t.Fatalf("delete bank raw: %v", err)
	}
	_ = db.Unscoped().Where("model_id = ?", bank.Id.String).Delete(&RawField{}).Error

	if err := RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("third recompute: %v", err)
	}
	var third Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "partner", "Partner").Take(&third).Error; err != nil {
		t.Fatalf("load third: %v", err)
	}
	if third.Id.String != firstID {
		t.Fatalf("id changed after bank uninstall: %q → %q", firstID, third.Id.String)
	}
	if len(third.Fields) != 1 || third.Fields[0].Name != "Name" {
		t.Fatalf("expected only Name after bank gone, got %#v", third.Fields)
	}

	if err := db.Unscoped().Where("id = ?", base.Id.String).Delete(&RawModel{}).Error; err != nil {
		t.Fatalf("delete base raw: %v", err)
	}
	_ = db.Unscoped().Where("model_id = ?", base.Id.String).Delete(&RawField{}).Error
	if err := RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("final recompute: %v", err)
	}
	var count int64
	if err := db.Model(&Model{}).Where("application = ? AND name = ?", "partner", "Partner").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected effective deleted, count=%d", count)
	}
}

func TestRecomputeEffective_PreservesServiceIDsByName(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	raw := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "raw-i18n", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "I18n",
		Application: "base",
		Path:        "go://i18n/base",
		Abstract:    true,
		ModuleId:    sql.NullString{String: "mod-base", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	for _, name := range []string{"SearchTerms", "UpdateTerm"} {
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawService{
			BaseModel:             BaseModel{Id: sql.NullString{String: "rs-" + name, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:                  name,
			AccessibilityModifier: "public",
			IsStatic:              true,
			ModelId:               raw.Id,
		}).Error; err != nil {
			t.Fatalf("create raw service %s: %v", name, err)
		}
	}

	if err := RecomputeEffective(db, "base", "I18n"); err != nil {
		t.Fatalf("first recompute: %v", err)
	}
	var first Model
	if err := db.Preload("Services").Where("application = ? AND name = ?", "base", "I18n").Take(&first).Error; err != nil {
		t.Fatalf("load first: %v", err)
	}
	firstByName := map[string]string{}
	for _, s := range first.Services {
		if s != nil {
			firstByName[s.Name] = s.Id.String
		}
	}
	if firstByName["SearchTerms"] == "" || firstByName["UpdateTerm"] == "" {
		t.Fatalf("missing services: %#v", firstByName)
	}

	if err := RecomputeEffective(db, "base", "I18n"); err != nil {
		t.Fatalf("second recompute: %v", err)
	}
	var second Model
	if err := db.Preload("Services").Where("application = ? AND name = ?", "base", "I18n").Take(&second).Error; err != nil {
		t.Fatalf("load second: %v", err)
	}
	if second.Id.String != first.Id.String {
		t.Fatalf("model id changed: %q → %q", first.Id.String, second.Id.String)
	}
	for _, s := range second.Services {
		if s == nil {
			continue
		}
		if got, want := s.Id.String, firstByName[s.Name]; got != want {
			t.Fatalf("service %s id rotated: %q → %q", s.Name, want, got)
		}
	}
}

func TestRecomputeKeys_Dedupes(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "r1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "m", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := RecomputeKeys(db, []LogicalKey{
		{Application: "a", Name: "X"},
		{Application: "a", Name: "X"},
		{Application: " ", Name: "X"},
	}); err != nil {
		t.Fatalf("RecomputeKeys: %v", err)
	}
	var count int64
	if err := db.Model(&Model{}).Where("name = ?", "X").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
}
