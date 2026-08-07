// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestMergeSameNameModelsByExtensionChain_EmptyAndNilGuards(t *testing.T) {
	merged, err := MergeSameNameModelsByExtensionChain(nil)
	if err != nil || merged != nil {
		t.Fatalf("empty nil slice: got %#v %v", merged, err)
	}
	merged, err = MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{})
	if err != nil || merged != nil {
		t.Fatalf("empty slice: got %#v %v", merged, err)
	}
	merged, err = MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{nil, nil})
	if err != nil || merged != nil {
		t.Fatalf("nil models only: got %#v %v", merged, err)
	}
}

func TestMergeSameNameModelsByExtensionChain_EmptyPathAndNilChildren(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	m := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "m", Valid: true}},
		Name:      "X",
		Path:      "",
		Fields:    []*pkgmeta.Field{nil, {Name: ""}, {Name: "Ok"}},
		Services:  []*pkgmeta.Service{nil, {Name: ""}, {Name: "Svc"}},
	}
	merged, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{m})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merged == nil || len(merged.Fields) != 1 || merged.Fields[0].Name != "Ok" {
		t.Fatalf("fields %#v", merged.Fields)
	}
	if len(merged.Services) != 1 || merged.Services[0].Name != "Svc" {
		t.Fatalf("services %#v", merged.Services)
	}
}

func TestMergeSameNameModelsByExtensionChain_ServiceLastWriteWins(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	base := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "a", Valid: true}},
		Name:      "X",
		Path:      "/a.ts",
		Services:  []*pkgmeta.Service{{Name: "Create", TsTypeAnnotation: "base"}},
	}
	ext := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: ts.Add(time.Hour), Id: sql.NullString{String: "b", Valid: true}},
		Name:      "X",
		Path:      "/b.ts",
		Extends:   "/a.ts",
		Services:  []*pkgmeta.Service{{Name: "Create", TsTypeAnnotation: "ext"}},
	}
	merged, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{base, ext})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(merged.Services) != 1 || merged.Services[0].TsTypeAnnotation != "ext" {
		t.Fatalf("expected ext service win, got %#v", merged.Services)
	}
}

func TestMergeSameNameModelsByExtensionChain_FieldConflictError(t *testing.T) {
	baseField := &pkgmeta.Field{Name: "Kind", FieldType: "selection"}
	if err := baseField.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:      "Kind",
			FieldType: "selection",
			Selection: []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	}); err != nil {
		t.Fatalf("base spec: %v", err)
	}
	extField := &pkgmeta.Field{Name: "Kind", FieldType: "selection"}
	if err := extField.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "b", Label: "B"}},
			Selection:       []pkgmeta.FieldSelectionItem{{Value: "c", Label: "C"}},
		},
	}); err != nil {
		t.Fatalf("ext spec: %v", err)
	}
	base := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "X",
		Path:      "/base.ts",
		Fields:    []*pkgmeta.Field{baseField},
	}
	ext := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "ext", Valid: true}},
		Name:      "X",
		Path:      "/ext.ts",
		Extends:   "/base.ts",
		Fields:    []*pkgmeta.Field{extField},
	}
	_, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{base, ext})
	if err == nil || !strings.Contains(err.Error(), "cannot combine selection and selectionAdd") {
		t.Fatalf("expected selection+selectionAdd conflict, got %v", err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SameUpdatedAtIdTieBreak(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	a := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "id-a", Valid: true}},
		Name:      "X",
		Path:      "/a.ts",
		Fields:    []*pkgmeta.Field{{Name: "F", TsTypeAnnotation: "a"}},
	}
	b := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "id-b", Valid: true}},
		Name:      "X",
		Path:      "/b.ts",
		Fields:    []*pkgmeta.Field{{Name: "F", TsTypeAnnotation: "b"}},
	}
	merged, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{b, a})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	// Same depth + same UpdatedAt → lower Id first, higher Id last (canonical / field win).
	if merged.Id.String != "id-b" {
		t.Fatalf("canonical id = %q, want id-b", merged.Id.String)
	}
	if merged.Fields[0].TsTypeAnnotation != "b" {
		t.Fatalf("field win = %q, want b", merged.Fields[0].TsTypeAnnotation)
	}
}

func TestRawModelsAsModels_SkipsNilAndMapsTrees(t *testing.T) {
	raws := []*rawModel{
		nil,
		{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "r1", Valid: true}},
			Name:      "X",
			Path:      "/x.ts",
			Fields: []*rawField{
				nil,
				{
					Name: "Name",
					Decorators: []*rawDecorator{
						nil,
						{Name: "Field", Arguments: []*rawArgument{nil, {Type: "object", Value: "{}"}}},
					},
				},
			},
			Services: []*rawService{
				nil,
				{
					Name: "Create",
					Parameters: []*rawParameter{
						nil,
						{Name: "this"},
						{Name: "vals"},
					},
					TypeParameters: []*rawTypeParameter{
						nil,
						{Name: "T", ModuleSpecPath: "/m", ReferenceIdent: "T"},
					},
					Decorators: []*rawDecorator{
						nil,
						{Name: "Rpc"},
					},
				},
			},
			Decorators: []*rawDecorator{
				nil,
				{Name: "Model", Arguments: []*rawArgument{{Type: "string", Value: `"X"`}}},
			},
		},
	}
	models := rawModelsAsModels(raws)
	if len(models) != 1 {
		t.Fatalf("models = %d", len(models))
	}
	m := models[0]
	if len(m.Fields) != 1 || len(m.Fields[0].Decorators) != 1 || len(m.Fields[0].Decorators[0].Arguments) != 1 {
		t.Fatalf("fields %#v", m.Fields)
	}
	if len(m.Services) != 1 {
		t.Fatalf("services %#v", m.Services)
	}
	svc := m.Services[0]
	if len(svc.Parameters) != 1 || svc.Parameters[0].Name != "vals" {
		t.Fatalf("params %#v", svc.Parameters)
	}
	if len(svc.TypeParameters) != 1 || svc.TypeParameters[0].Name != "T" {
		t.Fatalf("type params %#v", svc.TypeParameters)
	}
	if len(svc.Decorators) != 1 || svc.Decorators[0].Name != "Rpc" {
		t.Fatalf("svc decorators %#v", svc.Decorators)
	}
	if len(m.Decorators) != 1 || m.Decorators[0].Name != "Model" {
		t.Fatalf("model decorators %#v", m.Decorators)
	}
}
