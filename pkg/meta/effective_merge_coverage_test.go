// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestMergeSameNameModelsByExtensionChain_EmptyAndNilGuards(t *testing.T) {
	merged, err := MergeSameNameModelsByExtensionChain(nil)
	if err != nil || merged != nil {
		t.Fatalf("empty nil slice: got %#v %v", merged, err)
	}
	merged, err = MergeSameNameModelsByExtensionChain([]*Model{})
	if err != nil || merged != nil {
		t.Fatalf("empty slice: got %#v %v", merged, err)
	}
	merged, err = MergeSameNameModelsByExtensionChain([]*Model{nil, nil})
	if err != nil || merged != nil {
		t.Fatalf("nil models only: got %#v %v", merged, err)
	}
}

func TestMergeSameNameModelsByExtensionChain_EmptyPathAndNilChildren(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	m := &Model{
		BaseModel: BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "m", Valid: true}},
		Name:      "X",
		Path:      "",
		Fields:    []*Field{nil, {Name: ""}, {Name: "Ok"}},
		Services:  []*Service{nil, {Name: ""}, {Name: "Svc"}},
	}
	merged, err := MergeSameNameModelsByExtensionChain([]*Model{m})
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
	base := &Model{
		BaseModel: BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "a", Valid: true}},
		Name:      "X",
		Path:      "/a.ts",
		Services:  []*Service{{Name: "Create", TsTypeAnnotation: "base"}},
	}
	ext := &Model{
		BaseModel: BaseModel{UpdatedAt: ts.Add(time.Hour), Id: sql.NullString{String: "b", Valid: true}},
		Name:      "X",
		Path:      "/b.ts",
		Extends:   "/a.ts",
		Services:  []*Service{{Name: "Create", TsTypeAnnotation: "ext"}},
	}
	merged, err := MergeSameNameModelsByExtensionChain([]*Model{base, ext})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(merged.Services) != 1 || merged.Services[0].TsTypeAnnotation != "ext" {
		t.Fatalf("expected ext service win, got %#v", merged.Services)
	}
}

func TestMergeSameNameModelsByExtensionChain_FieldConflictError(t *testing.T) {
	baseField := &Field{Name: "Kind", FieldType: "selection"}
	if err := baseField.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Kind",
		Structural: FieldStructuralSpec{
			Name:      "Kind",
			FieldType: "selection",
			Selection: []FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	}); err != nil {
		t.Fatalf("base spec: %v", err)
	}
	extField := &Field{Name: "Kind", FieldType: "selection"}
	if err := extField.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Kind",
		Structural: FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []FieldSelectionItem{{Value: "b", Label: "B"}},
			Selection:       []FieldSelectionItem{{Value: "c", Label: "C"}},
		},
	}); err != nil {
		t.Fatalf("ext spec: %v", err)
	}
	base := &Model{
		BaseModel: BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "X",
		Path:      "/base.ts",
		Fields:    []*Field{baseField},
	}
	ext := &Model{
		BaseModel: BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "ext", Valid: true}},
		Name:      "X",
		Path:      "/ext.ts",
		Extends:   "/base.ts",
		Fields:    []*Field{extField},
	}
	_, err := MergeSameNameModelsByExtensionChain([]*Model{base, ext})
	if err == nil || !strings.Contains(err.Error(), "cannot combine selection and selectionAdd") {
		t.Fatalf("expected selection+selectionAdd conflict, got %v", err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SameUpdatedAtIdTieBreak(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	a := &Model{
		BaseModel: BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "id-a", Valid: true}},
		Name:      "X",
		Path:      "/a.ts",
		Fields:    []*Field{{Name: "F", TsTypeAnnotation: "a"}},
	}
	b := &Model{
		BaseModel: BaseModel{UpdatedAt: ts, Id: sql.NullString{String: "id-b", Valid: true}},
		Name:      "X",
		Path:      "/b.ts",
		Fields:    []*Field{{Name: "F", TsTypeAnnotation: "b"}},
	}
	merged, err := MergeSameNameModelsByExtensionChain([]*Model{b, a})
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
	raws := []*RawModel{
		nil,
		{
			BaseModel: BaseModel{Id: sql.NullString{String: "r1", Valid: true}},
			Name:      "X",
			Path:      "/x.ts",
			Fields: []*RawField{
				nil,
				{
					Name: "Name",
					Decorators: []*RawDecorator{
						nil,
						{Name: "Field", Arguments: []*RawArgument{nil, {Type: "object", Value: "{}"}}},
					},
				},
			},
			Services: []*RawService{
				nil,
				{
					Name: "Create",
					Parameters: []*RawParameter{
						nil,
						{Name: "this"},
						{Name: "vals"},
					},
					TypeParameters: []*RawTypeParameter{
						nil,
						{Name: "T", ModuleSpecPath: "/m", ReferenceIdent: "T"},
					},
					Decorators: []*RawDecorator{
						nil,
						{Name: "Rpc"},
					},
				},
			},
			Decorators: []*RawDecorator{
				nil,
				{Name: "Model", Arguments: []*RawArgument{{Type: "string", Value: `"X"`}}},
			},
		},
	}
	models := RawModelsAsModels(raws)
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
