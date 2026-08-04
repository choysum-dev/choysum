// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestExpandModelsAlongExtends_NilDB(t *testing.T) {
	if err := ExpandModelsAlongExtends(nil, []*Model{{Name: "X"}}); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}
}

func TestExpandModelsAlongExtends_SkipsNilAndEmptyPath(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	child := &Model{Name: "Child", Path: "/child.ts", Fields: []*Field{{Name: "A"}}}
	emptyPath := &Model{Name: "NoPath", Path: "  ", Fields: []*Field{{Name: "B"}}}
	if err := ExpandModelsAlongExtends(db, []*Model{nil, child, emptyPath}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(child.Fields) != 1 || child.Fields[0].Name != "A" {
		t.Fatalf("child fields %#v", child.Fields)
	}
	if len(emptyPath.Fields) != 1 || emptyPath.Fields[0].Name != "B" {
		t.Fatalf("emptyPath fields %#v", emptyPath.Fields)
	}
}

func TestExpandModelsAlongExtends_CacheHitViaTwoChildren(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	parentPath := "/parent.ts"
	parent := &RawModel{
		BaseModel: BaseModel{Id: sql.NullString{String: "raw-p", Valid: true}},
		Name:      "Parent",
		Path:      parentPath,
		ModuleId:  sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "pf", Valid: true}},
		Name:      "Shared",
		ModelId:   parent.Id,
	}).Error; err != nil {
		t.Fatalf("create parent field: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawService{
		BaseModel: BaseModel{Id: sql.NullString{String: "ps", Valid: true}},
		Name:      "ParentSvc",
		ModelId:   parent.Id,
	}).Error; err != nil {
		t.Fatalf("create parent service: %v", err)
	}

	childA := &Model{Name: "A", Path: "/a.ts", Extends: parentPath, Fields: []*Field{{Name: "AField"}}}
	childB := &Model{Name: "B", Path: "/b.ts", Extends: parentPath, Fields: []*Field{{Name: "BField"}}}
	if err := ExpandModelsAlongExtends(db, []*Model{childA, childB}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, m := range []*Model{childA, childB} {
		names := map[string]bool{}
		for _, f := range m.Fields {
			names[f.Name] = true
		}
		if !names["Shared"] {
			t.Fatalf("%s missing Shared field: %#v", m.Path, m.Fields)
		}
		svcNames := map[string]bool{}
		for _, s := range m.Services {
			svcNames[s.Name] = true
		}
		if !svcNames["ParentSvc"] {
			t.Fatalf("%s missing ParentSvc: %#v", m.Path, m.Services)
		}
	}
}

func TestExpandModelsAlongExtends_CircularExtends(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	a := &Model{Name: "A", Path: "/a.ts", Extends: "/b.ts"}
	b := &Model{Name: "B", Path: "/b.ts", Extends: "/a.ts"}
	if err := ExpandModelsAlongExtends(db, []*Model{a, b}); err == nil || !strings.Contains(err.Error(), "circular extends") {
		t.Fatalf("expected circular error, got %v", err)
	}
}

func TestExpandModelsAlongExtends_MissingParentInDB(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	child := &Model{
		Name:    "Child",
		Path:    "/child.ts",
		Extends: "/missing/parent.ts",
		Fields:  []*Field{{Name: "OnlyChild"}},
	}
	if err := ExpandModelsAlongExtends(db, []*Model{child}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(child.Fields) != 1 || child.Fields[0].Name != "OnlyChild" {
		t.Fatalf("expected child-only fields, got %#v", child.Fields)
	}
}

func TestResolveExtendsModel_DBError(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := db.Migrator().DropTable(&RawModel{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	child := &Model{Name: "Child", Path: "/c.ts", Extends: "/parent.ts", Fields: []*Field{{Name: "X"}}}
	if err := ExpandModelsAlongExtends(db, []*Model{child}); err == nil || !strings.Contains(err.Error(), "load raw parent") {
		t.Fatalf("expected load raw parent error, got %v", err)
	}
}

func TestExpandModelsAlongExtends_PrefersSameApplicationParent(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	parentPath := "/shared/base.ts"
	// Newer foreign-app parent with a distinctive field would win if path-only.
	foreign := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "foreign-parent", Valid: true}},
		Name:        "Base",
		Path:        parentPath,
		Application: "other",
		ModuleId:    sql.NullString{String: "mod-other", Valid: true},
	}
	home := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "home-parent", Valid: true}},
		Name:        "Base",
		Path:        parentPath,
		Application: "home",
		ModuleId:    sql.NullString{String: "mod-home", Valid: true},
	}
	for _, row := range []*RawModel{home, foreign} {
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(row).Error; err != nil {
			t.Fatalf("create parent: %v", err)
		}
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "ff", Valid: true}},
		Name:      "ForeignOnly",
		ModelId:   foreign.Id,
	}).Error; err != nil {
		t.Fatalf("create foreign field: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "hf", Valid: true}},
		Name:      "HomeOnly",
		ModelId:   home.Id,
	}).Error; err != nil {
		t.Fatalf("create home field: %v", err)
	}
	child := &Model{
		Name:        "Child",
		Path:        "/home/child.ts",
		Application: "home",
		Extends:     parentPath,
		Fields:      []*Field{{Name: "ChildField"}},
	}
	if err := ExpandModelsAlongExtends(db, []*Model{child}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	names := map[string]bool{}
	for _, f := range child.Fields {
		if f != nil {
			names[f.Name] = true
		}
	}
	if !names["HomeOnly"] || names["ForeignOnly"] || !names["ChildField"] {
		t.Fatalf("expected home parent fields, got %#v", names)
	}
}

func TestMergeFieldsForSchema_Coverage(t *testing.T) {
	parentPath := "/parent.ts"
	childPath := "/child.ts"

	// nil/empty name fields skipped
	got, err := mergeFieldsForSchema(
		[]*Field{nil, {Name: ""}, {Name: "P1"}},
		[]*Field{nil, {Name: ""}, {Name: "C1"}},
		parentPath, childPath,
	)
	if err != nil || len(got) != 2 {
		t.Fatalf("skip empty: got %#v err=%v", got, err)
	}

	// duplicate parent names skipped
	got, err = mergeFieldsForSchema(
		[]*Field{{Name: "Dup"}, {Name: "Dup"}},
		nil, parentPath, childPath,
	)
	if err != nil || len(got) != 1 {
		t.Fatalf("dup parent: got %#v err=%v", got, err)
	}

	// OriginModelPath already set on parent
	got, err = mergeFieldsForSchema(
		[]*Field{{Name: "F", OriginModelPath: "custom-origin"}},
		nil, parentPath, childPath,
	)
	if err != nil || got[0].OriginModelPath != "custom-origin" {
		t.Fatalf("origin kept: got %#v err=%v", got, err)
	}

	// child override replaces parent field
	got, err = mergeFieldsForSchema(
		[]*Field{{Name: "F", FieldType: "varchar"}},
		[]*Field{{Name: "F", FieldType: "int"}},
		parentPath, childPath,
	)
	if err != nil || got[0].FieldType != "int" || got[0].OriginModelPath != childPath {
		t.Fatalf("child override: got %#v err=%v", got, err)
	}

	// selectionAdd without parent → error
	childAdd := &Field{Name: "Kind"}
	_ = childAdd.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Kind",
		Structural: FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	})
	if _, err := mergeFieldsForSchema(nil, []*Field{childAdd}, parentPath, childPath); err == nil || !strings.Contains(err.Error(), "selectionAdd") {
		t.Fatalf("expected selectionAdd error, got %v", err)
	}

	// ResolveSelectionFieldConflict error: dynamic parent + selectionAdd child
	base := &Field{Name: "Status", FieldType: "selection", SelectionKind: "dynamic", SelectionMethod: "Opts"}
	_ = base.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Status",
		Structural: FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			SelectionKind:   "dynamic",
			SelectionMethod: "Opts",
		},
	})
	childConflict := &Field{Name: "Status"}
	_ = childConflict.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Status",
		Structural: FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []FieldSelectionItem{{Value: "x", Label: "X"}},
		},
	})
	if _, err := mergeFieldsForSchema([]*Field{base}, []*Field{childConflict}, parentPath, childPath); err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestMergeServicesForSchema_Coverage(t *testing.T) {
	parentPath := "/parent.ts"
	childPath := "/child.ts"

	got := mergeServicesForSchema(
		[]*Service{nil, {Name: ""}, {Name: "PSvc", OriginModelPath: "kept-origin"}},
		[]*Service{nil, {Name: ""}, {Name: "CSvc"}},
		parentPath, childPath,
	)
	if len(got) != 2 {
		t.Fatalf("expected 2 services, got %#v", got)
	}
	if got[0].OriginModelPath != "kept-origin" || got[0].Name != "PSvc" {
		t.Fatalf("parent svc: %#v", got[0])
	}
	if got[1].OriginModelPath != childPath || got[1].Name != "CSvc" {
		t.Fatalf("child svc: %#v", got[1])
	}

	// duplicate parent skipped
	got = mergeServicesForSchema(
		[]*Service{{Name: "Dup"}, {Name: "Dup"}},
		nil, parentPath, childPath,
	)
	if len(got) != 1 {
		t.Fatalf("dup parent: %#v", got)
	}

	// child override
	got = mergeServicesForSchema(
		[]*Service{{Name: "S", TsTypeAnnotation: "old"}},
		[]*Service{{Name: "S", TsTypeAnnotation: "new"}},
		parentPath, childPath,
	)
	if len(got) != 1 || got[0].TsTypeAnnotation != "new" || got[0].OriginModelPath != childPath {
		t.Fatalf("child override: %#v", got)
	}
}

func TestCloneFieldShallow_Coverage(t *testing.T) {
	if cloneFieldShallow(nil) != nil {
		t.Fatal("nil src should return nil")
	}

	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	src := &Field{
		BaseModel: BaseModel{Id: sql.NullString{String: "f1", Valid: true}, CreatedAt: ts},
		Name:      "Name",
		ModelId:   sql.NullString{String: "mid", Valid: true},
		Decorators: []*Decorator{
			nil,
			{Name: "EmptyDec"},
			{
				Name: "FullDec",
				Arguments: []*Argument{
					nil,
					{Type: "string", Value: `"x"`},
				},
			},
		},
	}
	dst := cloneFieldShallow(src)
	if dst == nil || dst.Name != "Name" {
		t.Fatalf("clone: %#v", dst)
	}
	if dst.Id.Valid || dst.ModelId.Valid || dst.Model != nil {
		t.Fatalf("cleared refs: Id=%#v ModelId=%#v", dst.Id, dst.ModelId)
	}
	if len(dst.Decorators) != 2 {
		t.Fatalf("decorators: %#v", dst.Decorators)
	}
	if len(dst.Decorators[1].Arguments) != 1 {
		t.Fatalf("arguments: %#v", dst.Decorators[1].Arguments)
	}

	emptyDec := &Field{Name: "NoDec", Decorators: []*Decorator{}}
	dstEmpty := cloneFieldShallow(emptyDec)
	if dstEmpty.Decorators != nil {
		t.Fatalf("empty decorators should be nil, got %#v", dstEmpty.Decorators)
	}
}

func TestCloneServiceShallow_Coverage(t *testing.T) {
	if cloneServiceShallow(nil) != nil {
		t.Fatal("nil src should return nil")
	}

	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	src := &Service{
		BaseModel: BaseModel{Id: sql.NullString{String: "s1", Valid: true}, CreatedAt: ts},
		Name:      "Create",
		ModelId:   sql.NullString{String: "mid", Valid: true},
		Parameters: []*Parameter{
			nil,
			{Name: "vals"},
		},
		TypeParameters: []*TypeParameter{
			nil,
			{Name: "T"},
		},
		Decorators: []*Decorator{
			nil,
			{Name: "NoArgs"},
			{
				Name: "WithArgs",
				Arguments: []*Argument{
					nil,
					{Type: "string", Value: `"y"`},
				},
			},
		},
	}
	dst := cloneServiceShallow(src)
	if dst == nil || dst.Name != "Create" {
		t.Fatalf("clone: %#v", dst)
	}
	if dst.Id.Valid || dst.ModelId.Valid {
		t.Fatalf("cleared refs: Id=%#v ModelId=%#v", dst.Id, dst.ModelId)
	}
	if len(dst.Parameters) != 1 || len(dst.TypeParameters) != 1 {
		t.Fatalf("params/tps: params=%#v tps=%#v", dst.Parameters, dst.TypeParameters)
	}
	if len(dst.Decorators) != 2 {
		t.Fatalf("decorators: %#v", dst.Decorators)
	}
	if len(dst.Decorators[1].Arguments) != 1 {
		t.Fatalf("arguments: %#v", dst.Decorators[1].Arguments)
	}

	empty := &Service{Name: "Bare"}
	dstEmpty := cloneServiceShallow(empty)
	if dstEmpty.Parameters != nil || dstEmpty.TypeParameters != nil || dstEmpty.Decorators != nil {
		t.Fatalf("empty slices should be nil: %#v", dstEmpty)
	}
}

func TestExpandShapeAlongExtends_NilModel(t *testing.T) {
	shape, err := expandShapeAlongExtends(nil, nil, nil, nil, nil)
	if err != nil || shape == nil || len(shape.fields) != 0 {
		t.Fatalf("nil model: shape=%#v err=%v", shape, err)
	}
}

func TestExpandModelsAlongExtends_LocalParentOnly(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	parentPath := "/local/parent.ts"
	childPath := "/local/child.ts"
	parent := &Model{
		Name:     "Parent",
		Path:     parentPath,
		Fields:   []*Field{{Name: "Inherited"}},
		Services: []*Service{{Name: "InheritedSvc"}},
	}
	child := &Model{
		Name:    "Child",
		Path:    childPath,
		Extends: parentPath,
		Fields:  []*Field{{Name: "Own"}},
	}
	if err := ExpandModelsAlongExtends(db, []*Model{parent, child}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	names := map[string]bool{}
	for _, f := range child.Fields {
		names[f.Name] = true
	}
	if !names["Inherited"] || !names["Own"] {
		t.Fatalf("fields: %#v", child.Fields)
	}
}

func TestExpandModelsAlongExtends_MergeFieldsErrorPropagates(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	parentPath := "/err/parent.ts"
	childPath := "/err/child.ts"
	base := &Field{Name: "Status", FieldType: "selection", SelectionKind: "dynamic", SelectionMethod: "Opts"}
	_ = base.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Status",
		Structural: FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			SelectionKind:   "dynamic",
			SelectionMethod: "Opts",
		},
	})
	childField := &Field{Name: "Status"}
	_ = childField.SetResolvedSpec(&FieldResolvedSpec{
		FieldName: "Status",
		Structural: FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []FieldSelectionItem{{Value: "x", Label: "X"}},
		},
	})
	parent := &Model{Path: parentPath, Fields: []*Field{base}}
	child := &Model{Path: childPath, Extends: parentPath, Fields: []*Field{childField}}
	if err := ExpandModelsAlongExtends(db, []*Model{parent, child}); err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected merge error, got %v", err)
	}
}

func TestResolveExtendsModel_FullPreloadFromDB(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	parentPath := "/db/parent.ts"
	parent := &RawModel{
		BaseModel: BaseModel{Id: sql.NullString{String: "db-parent", Valid: true}},
		Name:      "DbParent",
		Path:      parentPath,
		ModuleId:  sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	field := &RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "dbf", Valid: true}},
		Name:      "F",
		ModelId:   parent.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	fieldDec := &RawDecorator{
		BaseModel: BaseModel{Id: sql.NullString{String: "dbfd", Valid: true}},
		Name:      "FieldDec",
		FieldId:   sql.NullString{String: field.Id.String, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldDec).Error; err != nil {
		t.Fatalf("create field dec: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawArgument{
		BaseModel:   BaseModel{Id: sql.NullString{String: "dbfa", Valid: true}},
		Type:        "string",
		Value:       `"x"`,
		DecoratorId: fieldDec.Id,
	}).Error; err != nil {
		t.Fatalf("create field arg: %v", err)
	}
	svc := &RawService{
		BaseModel: BaseModel{Id: sql.NullString{String: "dbs", Valid: true}},
		Name:      "Svc",
		ModelId:   parent.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svc).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	svcDec := &RawDecorator{
		BaseModel: BaseModel{Id: sql.NullString{String: "dbsd", Valid: true}},
		Name:      "SvcDec",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svcDec).Error; err != nil {
		t.Fatalf("create svc dec: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&RawArgument{
		BaseModel:   BaseModel{Id: sql.NullString{String: "dbsa", Valid: true}},
		Type:        "string",
		Value:       `"y"`,
		DecoratorId: svcDec.Id,
	}).Error; err != nil {
		t.Fatalf("create svc arg: %v", err)
	}

	child := &Model{Name: "Child", Path: "/db/child.ts", Extends: parentPath}
	if err := ExpandModelsAlongExtends(db, []*Model{child}); err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(child.Fields) != 1 || len(child.Services) != 1 {
		t.Fatalf("expected inherited shape, fields=%#v services=%#v", child.Fields, child.Services)
	}
}
