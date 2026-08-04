// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTipTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:meta_model_tip_" + xid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedModel(t *testing.T, db *gorm.DB, id, app, name string, createdAt time.Time) {
	t.Helper()
	m := &meta.Model{
		BaseModel: meta.BaseModel{
			Id:        sql.NullString{String: id, Valid: true},
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		Name:        name,
		Path:        "@/test/" + id + ".ts",
		Application: app,
		ModelTable:  "test_" + name,
	}
	if err := db.Create(m).Error; err != nil {
		t.Fatalf("create model %s: %v", id, err)
	}
}

func TestResolveMetaModelTip_PicksNewestCreatedAt(t *testing.T) {
	db := openTipTestDB(t)
	older := xid.New().String()
	newer := xid.New().String()
	seedModel(t, db, older, "partner", "Partner", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	seedModel(t, db, newer, "partner", "Partner", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	tip, err := meta.ResolveMetaModelTip(db, "partner", "Partner")
	if err != nil {
		t.Fatalf("ResolveMetaModelTip: %v", err)
	}
	if tip.Id.String != newer {
		t.Fatalf("expected tip %s, got %s", newer, tip.Id.String)
	}
}

func TestResolveMetaModelTip_TieBreaksByIdDesc(t *testing.T) {
	db := openTipTestDB(t)
	sameTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	lowID := "aaaaaaaaaaaaaaaaaaaa"
	highID := "zzzzzzzzzzzzzzzzzzzz"
	seedModel(t, db, lowID, "partner", "Partner", sameTime)
	seedModel(t, db, highID, "partner", "Partner", sameTime)

	tip, err := meta.ResolveMetaModelTip(db, "partner", "Partner")
	if err != nil {
		t.Fatalf("ResolveMetaModelTip: %v", err)
	}
	if tip.Id.String != highID {
		t.Fatalf("expected tip %s, got %s", highID, tip.Id.String)
	}
}

func TestResolveMetaModelTip_ExcludesIDsAndSoftDeleted(t *testing.T) {
	db := openTipTestDB(t)
	base := xid.New().String()
	ext := xid.New().String()
	seedModel(t, db, base, "partner", "Partner", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	seedModel(t, db, ext, "partner", "Partner", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	tip, err := meta.ResolveMetaModelTip(db, "partner", "Partner", ext)
	if err != nil {
		t.Fatalf("ResolveMetaModelTip exclude: %v", err)
	}
	if tip.Id.String != base {
		t.Fatalf("expected base tip %s, got %s", base, tip.Id.String)
	}

	if err := db.Where("id = ?", ext).Delete(&meta.Model{}).Error; err != nil {
		t.Fatalf("soft delete ext: %v", err)
	}
	tip, err = meta.ResolveMetaModelTip(db, "partner", "Partner")
	if err != nil {
		t.Fatalf("ResolveMetaModelTip after soft delete: %v", err)
	}
	if tip.Id.String != base {
		t.Fatalf("expected base after soft delete, got %s", tip.Id.String)
	}
}
