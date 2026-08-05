// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"database/sql"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func persistDeclaration(t *testing.T, db *gorm.DB, m *meta.Model) {
	t.Helper()
	if err := meta.PersistModelTreeAsRaw(db, m); err != nil {
		t.Fatalf("persist declaration: %v", err)
	}
}

func seedVirtualDeclarationTree(
	t *testing.T,
	db *gorm.DB,
	name, path, virtID, fieldID, svcID, decModel, decField, decSvc, argID, tpID, pID string,
) {
	t.Helper()
	m := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: virtID, Valid: true}},
		Name:        name,
		Path:        path,
		Application: "partner",
		Fields: []*meta.Field{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: fieldID, Valid: true}},
			Name:      "Model",
			Decorators: []*meta.Decorator{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: decField, Valid: true}},
				Name:      "Field",
			}},
		}},
		Services: []*meta.Service{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: svcID, Valid: true}},
			Name:      "Get",
			Decorators: []*meta.Decorator{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: decSvc, Valid: true}},
				Name:      "Service",
			}},
			TypeParameters: []*meta.TypeParameter{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: tpID, Valid: true}},
				Name:      "T",
			}},
			Parameters: []*meta.Parameter{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: pID, Valid: true}},
				Name:      "x",
			}},
		}},
		Decorators: []*meta.Decorator{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: decModel, Valid: true}},
			Name:      "Model",
			Arguments: []*meta.Argument{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: argID, Valid: true}},
			}},
		}},
	}
	persistDeclaration(t, db, m)
}

func seedVirtualDeclarationErrorBranchTree(
	t *testing.T,
	db *gorm.DB,
	name, path, virtID, fieldID, svcID, decID, argID, tpID, pID string,
) {
	t.Helper()
	m := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: virtID, Valid: true}},
		Name:        name,
		Path:        path,
		Application: "partner",
		Fields: []*meta.Field{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: fieldID, Valid: true}},
			Name:      "Model",
		}},
		Services: []*meta.Service{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: svcID, Valid: true}},
			Name:      "Get",
			TypeParameters: []*meta.TypeParameter{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: tpID, Valid: true}},
				Name:      "T",
			}},
			Parameters: []*meta.Parameter{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: pID, Valid: true}},
				Name:      "x",
			}},
		}},
		Decorators: []*meta.Decorator{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: decID, Valid: true}},
			Name:      "Model",
			Arguments: []*meta.Argument{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: argID, Valid: true}},
			}},
		}},
	}
	persistDeclaration(t, db, m)
}

func assertRawModelAbsent(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	left, err := meta.CountRawModelsByID(db, id, true)
	if err != nil || left != 0 {
		t.Fatalf("raw model id=%q left=%d err=%v", id, left, err)
	}
}
