// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"path/filepath"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestExpandModelsAlongExtends_MergesParentFields(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "extends.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	parentPath := "/core/service/orm/model/model.ts"
	childPath := "/base/service/models/address.ts"
	parent := &rawModel{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "raw-parent", Valid: true}},
		Name:      "pkgmeta.BaseModel",
		Path:      parentPath,
		Abstract:  true,
		ModuleId:  sql.NullString{String: "mod-core", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawField{
		BaseModel:    pkgmeta.BaseModel{Id: sql.NullString{String: "f-id", Valid: true}},
		Name:         "Id",
		ResolvedSpec: `{"fieldName":"Id","structural":{"name":"Id","fieldType":"varchar"},"migration":{"shouldCreateColumn":true,"resolvedColumnType":"varchar"}}`,
		ModelId:      parent.Id,
	}).Error; err != nil {
		t.Fatalf("create parent field: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawService{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "s-create", Valid: true}},
		Name:      "Create",
		ModelId:   parent.Id,
	}).Error; err != nil {
		t.Fatalf("create parent service: %v", err)
	}

	child := &pkgmeta.Model{
		Name:       "Address",
		Path:       childPath,
		ModelTable: "base_address",
		Extends:    parentPath,
		Fields: []*pkgmeta.Field{{
			Name:         "Street1",
			ResolvedSpec: `{"fieldName":"Street1","structural":{"name":"Street1","fieldType":"varchar"},"migration":{"shouldCreateColumn":true,"resolvedColumnType":"varchar"}}`,
		}},
		Services: []*pkgmeta.Service{{Name: "Normalize"}},
	}
	if err := ExpandModelsAlongExtends(db, []*pkgmeta.Model{child}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(child.Fields) != 2 {
		t.Fatalf("want Id+Street1, got %#v", child.Fields)
	}
	names := map[string]bool{}
	for _, f := range child.Fields {
		names[f.Name] = true
	}
	if !names["Id"] || !names["Street1"] {
		t.Fatalf("unexpected field names: %#v", names)
	}
	svcNames := map[string]bool{}
	for _, s := range child.Services {
		svcNames[s.Name] = true
	}
	if !svcNames["Create"] || !svcNames["Normalize"] {
		t.Fatalf("unexpected services: %#v", svcNames)
	}
}
