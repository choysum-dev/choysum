// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestEnsureDualStoreTables_CreatesRawAndEffective(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	m := db.Migrator()
	for _, table := range []string{
		"meta_model", "meta_field", "meta_service",
		"meta_raw_model", "meta_raw_field", "meta_raw_service",
		"meta_raw_decorator", "meta_raw_argument",
		"meta_raw_parameter", "meta_raw_type_parameter",
	} {
		if !m.HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}
}

func TestMigrateIMDCatalogToDualStore_CollapsesSameNameToEffective(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := db.AutoMigrate(append(DualStoreEffectiveEntities(), &Module{})...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	basePath := "/partner/partner.ts"
	bankPath := "/partner_bank/partner.ts"
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	base := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-base", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        basePath,
		ModelTable:  "partner_partner",
		ModuleId:    sql.NullString{String: "mod-base", Valid: true},
		Fields: []*Field{{
			BaseModel: BaseModel{Id: sql.NullString{String: "f-name", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "Name",
			Decorators: []*Decorator{{
				BaseModel: BaseModel{Id: sql.NullString{String: "d-name", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:      "Field",
				Arguments: []*Argument{{
					BaseModel: BaseModel{Id: sql.NullString{String: "a-name", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Type:      "object",
					Value:     `{"type":"varchar"}`,
				}},
			}},
		}},
	}
	bank := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-bank", Valid: true}, CreatedAt: ts.Add(time.Hour), UpdatedAt: ts.Add(time.Hour)},
		Name:        "Partner",
		Application: "partner",
		Path:        bankPath,
		Extends:     basePath,
		ModelTable:  "partner_partner",
		ModuleId:    sql.NullString{String: "mod-bank", Valid: true},
		Fields:      []*Field{{BaseModel: BaseModel{Id: sql.NullString{String: "f-bank", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "BankAccounts"}},
	}
	for _, m := range []*Model{base, bank} {
		fields := m.Fields
		m.Fields = nil
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(m).Error; err != nil {
			t.Fatalf("create model: %v", err)
		}
		for _, f := range fields {
			decs := f.Decorators
			f.Decorators = nil
			f.ModelId = m.Id
			if err := db.Session(&gorm.Session{SkipHooks: true}).Create(f).Error; err != nil {
				t.Fatalf("create field: %v", err)
			}
			for _, d := range decs {
				args := d.Arguments
				d.Arguments = nil
				d.FieldId = f.Id
				if err := db.Session(&gorm.Session{SkipHooks: true}).Create(d).Error; err != nil {
					t.Fatalf("create decorator: %v", err)
				}
				for _, a := range args {
					a.DecoratorId = d.Id
					if err := db.Session(&gorm.Session{SkipHooks: true}).Create(a).Error; err != nil {
						t.Fatalf("create argument: %v", err)
					}
				}
			}
		}
	}

	if err := MigrateIMDCatalogToDualStore(db); err != nil {
		t.Fatalf("MigrateIMDCatalogToDualStore: %v", err)
	}

	var rawCount int64
	if err := db.Model(&RawModel{}).Count(&rawCount).Error; err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if rawCount != 2 {
		t.Fatalf("raw models = %d, want 2", rawCount)
	}

	var eff []*Model
	if err := db.Preload("Fields.Decorators.Arguments").Find(&eff).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if len(eff) != 1 {
		t.Fatalf("effective models = %d, want 1", len(eff))
	}
	if eff[0].Id.String != "id-bank" {
		t.Fatalf("effective id = %q, want tip id-bank", eff[0].Id.String)
	}
	if eff[0].ModuleId.Valid {
		t.Fatalf("effective ModuleId should be empty, got %#v", eff[0].ModuleId)
	}
	names := map[string]bool{}
	for _, f := range eff[0].Fields {
		if f != nil {
			names[f.Name] = true
		}
	}
	if !names["Name"] || !names["BankAccounts"] {
		t.Fatalf("effective fields %#v", names)
	}

	var nameField *Field
	for _, f := range eff[0].Fields {
		if f != nil && f.Name == "Name" {
			nameField = f
			break
		}
	}
	if nameField == nil || len(nameField.Decorators) != 1 || nameField.Decorators[0].Name != "Field" {
		t.Fatalf("expected Name field decorator preserved, got %#v", nameField)
	}
	if len(nameField.Decorators[0].Arguments) != 1 {
		t.Fatalf("expected decorator argument preserved, got %#v", nameField.Decorators[0].Arguments)
	}
	arg := nameField.Decorators[0].Arguments[0]
	if arg == nil || arg.Type != "object" || arg.Value != `{"type":"varchar"}` {
		t.Fatalf("expected decorator argument payload preserved, got %#v", arg)
	}

	// Soft-deleted effective tip must not block a new live row (partial unique index).
	if err := db.Delete(eff[0]).Error; err != nil {
		t.Fatalf("soft-delete effective: %v", err)
	}
	replacement := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-relive", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner/partner2.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(replacement).Error; err != nil {
		t.Fatalf("create live row after soft-delete: %v", err)
	}

	// Unique (application, name) still rejects a second live row.
	dup := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-dup", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/other/partner.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(dup).Error; err == nil {
		t.Fatal("expected unique (application, name) violation for second live row")
	}
}

func TestMigrateIMDCatalogToDualStore_RejectsMissingModuleId(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	src := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-nomod", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(src).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := MigrateIMDCatalogToDualStore(db); err == nil || !strings.Contains(err.Error(), "missing module_id") {
		t.Fatalf("expected missing module_id error, got %v", err)
	}
}

func TestMigrateIMDCatalogToDualStore_RefusesNonEmptyRaw(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	raw := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "r1", Valid: true}},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod-x", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := MigrateIMDCatalogToDualStore(db); err == nil {
		t.Fatal("expected refuse when raw non-empty")
	}
}

func TestMigrateIMDCatalogToDualStore_EmptySourcesIsNoOp(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	kept := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-soft", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Kept",
		Application: "a",
		Path:        "/kept.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(kept).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Delete(kept).Error; err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	if err := MigrateIMDCatalogToDualStore(db); err != nil {
		t.Fatalf("MigrateIMDCatalogToDualStore: %v", err)
	}
	var count int64
	if err := db.Unscoped().Model(&Model{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("empty migrate wiped effective catalog, Unscoped count=%d", count)
	}
}

func TestRecomputeAllEffectiveFromRaw_OmitsThisParameter(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "r-svc", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	svc := &RawService{
		BaseModel: BaseModel{Id: sql.NullString{String: "s1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Create",
		ModelId:   raw.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svc).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	for i, name := range []string{"this", "vals"} {
		p := &RawParameter{
			BaseModel: BaseModel{Id: sql.NullString{String: fmt.Sprintf("p%d", i), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      name,
			ServiceId: svc.Id,
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(p).Error; err != nil {
			t.Fatalf("create param %s: %v", name, err)
		}
	}
	if err := RecomputeAllEffectiveFromRaw(db); err != nil {
		t.Fatalf("RecomputeAllEffectiveFromRaw: %v", err)
	}
	var eff []*Model
	if err := db.Preload("Services.Parameters").Find(&eff).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if len(eff) != 1 || len(eff[0].Services) != 1 {
		t.Fatalf("expected one effective model/service, got %#v", eff)
	}
	params := eff[0].Services[0].Parameters
	if len(params) != 1 || params[0].Name != "vals" {
		t.Fatalf("expected only vals on effective service, got %#v", params)
	}
	var rawParams []*RawParameter
	if err := db.Where("service_id = ?", svc.Id).Find(&rawParams).Error; err != nil {
		t.Fatalf("load raw params: %v", err)
	}
	names := map[string]bool{}
	for _, p := range rawParams {
		if p != nil {
			names[p.Name] = true
		}
	}
	if len(rawParams) != 2 || !names["this"] || !names["vals"] {
		t.Fatalf("expected raw service to retain this and vals, got %#v", rawParams)
	}
}
