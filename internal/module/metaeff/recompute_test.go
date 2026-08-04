// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metaeff_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/metaeff"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openMetaeffTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "metaeff.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	return db
}

// Partner + bank + commercial §4-style fixture: union fields, stable effective id.
func TestRecomputeKeys_PartnerBankCommercialFixture(t *testing.T) {
	db := openMetaeffTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	skip := db.Session(&gorm.Session{SkipHooks: true})

	partner := &meta.RawModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "raw-partner", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner/partner.ts",
		ModuleId:    sql.NullString{String: "mod-partner", Valid: true},
	}
	bank := &meta.RawModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "raw-bank", Valid: true}, CreatedAt: ts.Add(time.Hour), UpdatedAt: ts.Add(time.Hour)},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner_bank/partner.ts",
		Extends:     "/partner/partner.ts",
		ModuleId:    sql.NullString{String: "mod-bank", Valid: true},
	}
	commercial := &meta.RawModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "raw-commercial", Valid: true}, CreatedAt: ts.Add(2 * time.Hour), UpdatedAt: ts.Add(2 * time.Hour)},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner_commercial/partner.ts",
		Extends:     "/partner/partner.ts",
		ModuleId:    sql.NullString{String: "mod-commercial", Valid: true},
	}
	for _, row := range []*meta.RawModel{partner, bank, commercial} {
		if err := skip.Create(row).Error; err != nil {
			t.Fatalf("create raw: %v", err)
		}
	}
	for _, f := range []struct {
		id, name, model string
	}{
		{"f-name", "Name", "raw-partner"},
		{"f-bank", "BankAccounts", "raw-bank"},
		{"f-vat", "Vat", "raw-commercial"},
	} {
		if err := skip.Create(&meta.RawField{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: f.id, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      f.name,
			ModelId:   sql.NullString{String: f.model, Valid: true},
		}).Error; err != nil {
			t.Fatalf("create field %s: %v", f.name, err)
		}
	}

	key := metaeff.LogicalKey{Application: "partner", Name: "Partner"}
	if err := metaeff.RecomputeKeys(db, []metaeff.LogicalKey{key, key}); err != nil {
		t.Fatalf("triangle recompute: %v", err)
	}
	var first meta.Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "partner", "Partner").Take(&first).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if len(first.Fields) != 3 {
		t.Fatalf("want 3 union fields, got %d", len(first.Fields))
	}
	firstID := first.Id.String

	// Uninstall commercial: field Vat gone, id stable.
	if err := db.Unscoped().Where("id = ?", commercial.Id.String).Delete(&meta.RawModel{}).Error; err != nil {
		t.Fatalf("delete commercial raw: %v", err)
	}
	_ = db.Unscoped().Where("model_id = ?", commercial.Id.String).Delete(&meta.RawField{})
	if err := metaeff.RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("recompute after commercial: %v", err)
	}
	var afterCommercial meta.Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "partner", "Partner").Take(&afterCommercial).Error; err != nil {
		t.Fatalf("load after commercial: %v", err)
	}
	if afterCommercial.Id.String != firstID {
		t.Fatalf("effective id changed: %s -> %s", firstID, afterCommercial.Id.String)
	}
	if len(afterCommercial.Fields) != 2 {
		t.Fatalf("want 2 fields after commercial uninstall, got %d", len(afterCommercial.Fields))
	}

	// Uninstall bank.
	if err := db.Unscoped().Where("id = ?", bank.Id.String).Delete(&meta.RawModel{}).Error; err != nil {
		t.Fatalf("delete bank raw: %v", err)
	}
	_ = db.Unscoped().Where("model_id = ?", bank.Id.String).Delete(&meta.RawField{})
	if err := metaeff.RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("recompute after bank: %v", err)
	}
	var afterBank meta.Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "partner", "Partner").Take(&afterBank).Error; err != nil {
		t.Fatalf("load after bank: %v", err)
	}
	if afterBank.Id.String != firstID || len(afterBank.Fields) != 1 {
		t.Fatalf("after bank: id=%s fields=%d", afterBank.Id.String, len(afterBank.Fields))
	}

	// Full uninstall: effective gone.
	if err := db.Unscoped().Where("id = ?", partner.Id.String).Delete(&meta.RawModel{}).Error; err != nil {
		t.Fatalf("delete partner raw: %v", err)
	}
	_ = db.Unscoped().Where("model_id = ?", partner.Id.String).Delete(&meta.RawField{})
	if err := metaeff.RecomputeEffective(db, "partner", "Partner"); err != nil {
		t.Fatalf("recompute after full: %v", err)
	}
	var left int64
	if err := db.Model(&meta.Model{}).Where("application = ? AND name = ?", "partner", "Partner").Count(&left).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatalf("want no effective after full uninstall, got %d", left)
	}
}
