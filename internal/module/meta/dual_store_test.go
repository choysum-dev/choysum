// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDualStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "dual-store.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestPersistModelTreeAsRaw_AssignsIDsWhenMissing(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	// Parser-shaped trees often have empty primary keys; SkipHooks Create must still mint ids
	// so Fields preload by model_id for schema migrator.
	src := &pkgmeta.Model{
		Name:        "Address",
		Application: "base",
		Path:        "/base/service/models/address.ts",
		ModelTable:  "base_address",
		ModuleId:    sql.NullString{String: "mod-base", Valid: true},
		Fields: []*pkgmeta.Field{{
			Name:         "Street",
			ResolvedSpec: `{"fieldName":"Street","structural":{"name":"Street","fieldType":"varchar"},"migration":{"shouldCreateColumn":true,"resolvedColumnType":"varchar"}}`,
			Decorators: []*pkgmeta.Decorator{{
				Name: "Field",
				Arguments: []*pkgmeta.Argument{{
					Type:  "object",
					Value: `{"type":"varchar"}`,
				}},
			}},
		}},
		Services: []*pkgmeta.Service{{
			Name: "Search",
			Parameters: []*pkgmeta.Parameter{
				{Name: "this"},
				{Name: "domain"},
			},
		}},
	}
	if err := persistModelTreeAsRaw(db, src); err != nil {
		t.Fatalf("persistModelTreeAsRaw: %v", err)
	}

	var raw rawModel
	if err := db.Preload("Fields").Preload("Fields.Decorators").Preload("Fields.Decorators.Arguments").
		Preload("Services").Preload("Services.Parameters").
		Where("path = ?", src.Path).Take(&raw).Error; err != nil {
		t.Fatalf("load raw: %v", err)
	}
	if strings.TrimSpace(raw.Id.String) == "" {
		t.Fatal("expected non-empty raw model id")
	}
	if len(raw.Fields) != 1 || strings.TrimSpace(raw.Fields[0].Id.String) == "" {
		t.Fatalf("expected linked field with id, got %#v", raw.Fields)
	}
	if raw.Fields[0].ModelId.String != raw.Id.String {
		t.Fatalf("field.model_id=%q want %q", raw.Fields[0].ModelId.String, raw.Id.String)
	}
	if raw.Fields[0].ResolvedSpec == "" {
		t.Fatal("expected ResolvedSpec preserved")
	}
	if len(raw.Fields[0].Decorators) != 1 || len(raw.Fields[0].Decorators[0].Arguments) != 1 {
		t.Fatalf("expected decorator tree, got %#v", raw.Fields[0].Decorators)
	}
	if len(raw.Services) != 1 || len(raw.Services[0].Parameters) != 2 {
		t.Fatalf("unexpected services: %#v", raw.Services)
	}
}

func TestEnsureDualStoreTables_EffectiveAppNameUniqueIndex(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	live := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "id-live", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner/partner.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(live).Error; err != nil {
		t.Fatalf("create live: %v", err)
	}

	// Soft-deleted effective tip must not block a new live row (partial unique index).
	if err := db.Delete(live).Error; err != nil {
		t.Fatalf("soft-delete effective: %v", err)
	}
	replacement := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "id-relive", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner/partner2.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(replacement).Error; err != nil {
		t.Fatalf("create live row after soft-delete: %v", err)
	}

	// Unique (application, name) still rejects a second live row.
	dup := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "id-dup", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/other/partner.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(dup).Error; err == nil {
		t.Fatal("expected unique (application, name) violation for second live row")
	}
}
