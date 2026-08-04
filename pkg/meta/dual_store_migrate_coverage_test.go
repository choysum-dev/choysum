// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

type namedDialector struct {
	gorm.Dialector
	name string
}

func (d namedDialector) Name() string { return d.name }

func closeDualStoreDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestEnsureDualStoreTables_NilAndClosed(t *testing.T) {
	if err := EnsureDualStoreTables(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}
	db := openDualStoreTestDB(t)
	closeDualStoreDB(t, db)
	if err := EnsureDualStoreTables(db); err == nil {
		t.Fatal("expected AutoMigrate error on closed db")
	}
}

func TestMigrateIMDCatalogToDualStore_NilAndCountError(t *testing.T) {
	if err := MigrateIMDCatalogToDualStore(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	closeDualStoreDB(t, db)
	if err := MigrateIMDCatalogToDualStore(db); err == nil {
		t.Fatal("expected count/ensure error on closed db")
	}
}

func TestMigrateIMDCatalogToDualStore_DuplicateModulePath(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	for _, id := range []string{"a", "b"} {
		m := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: id, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "X",
			Application: "app",
			Path:        "/same.ts",
			ModuleId:    sql.NullString{String: "mod", Valid: true},
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(m).Error; err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	if err := MigrateIMDCatalogToDualStore(db); err == nil || !strings.Contains(err.Error(), "duplicate live") {
		t.Fatalf("expected duplicate path error, got %v", err)
	}
}

func TestMigrateIMDSources_NilEntriesAndWhitespaceModuleId(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	blank := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "blank", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "app",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "  ", Valid: true},
	}
	if err := migrateIMDSources(db, []*Model{nil, blank}); err == nil || !strings.Contains(err.Error(), "missing module_id") {
		t.Fatalf("expected missing module_id, got %v", err)
	}
}

func TestMigrateIMDCatalogToDualStore_FullTreeCopy(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := db.AutoMigrate(append(DualStoreEffectiveEntities(), DualStoreRawEntities()...)...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	src := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "m1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/partner.ts",
		ClassName:   "Partner",
		ModelTable:  "partner_partner",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	modelDec := &Decorator{
		BaseModel: BaseModel{Id: sql.NullString{String: "md", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Model",
	}
	modelArg := &Argument{
		BaseModel: BaseModel{Id: sql.NullString{String: "ma", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Type:      "string",
		Value:     `"Partner"`,
	}
	field := &Field{
		BaseModel: BaseModel{Id: sql.NullString{String: "f1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Name",
	}
	fieldDec := &Decorator{
		BaseModel: BaseModel{Id: sql.NullString{String: "fd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Field",
	}
	fieldArg := &Argument{
		BaseModel: BaseModel{Id: sql.NullString{String: "fa", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Type:      "object",
		Value:     `{}`,
	}
	svc := &Service{
		BaseModel: BaseModel{Id: sql.NullString{String: "s1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Create",
	}
	svcDec := &Decorator{
		BaseModel: BaseModel{Id: sql.NullString{String: "sd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Rpc",
	}
	params := []*Parameter{
		{BaseModel: BaseModel{Id: sql.NullString{String: "p-this", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "this"},
		{BaseModel: BaseModel{Id: sql.NullString{String: "p-vals", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "vals"},
	}
	tps := []*TypeParameter{
		{BaseModel: BaseModel{Id: sql.NullString{String: "tp1", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "T"},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(src).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	field.ModelId = src.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	fieldDec.FieldId = field.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldDec).Error; err != nil {
		t.Fatalf("create field decorator: %v", err)
	}
	fieldArg.DecoratorId = fieldDec.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldArg).Error; err != nil {
		t.Fatalf("create field arg: %v", err)
	}
	svc.ModelId = src.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svc).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	for _, p := range params {
		p.ServiceId = svc.Id
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(p).Error; err != nil {
			t.Fatalf("create param: %v", err)
		}
	}
	for _, tp := range tps {
		tp.ServiceId = svc.Id
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(tp).Error; err != nil {
			t.Fatalf("create type param: %v", err)
		}
	}
	svcDec.ServiceId = svc.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svcDec).Error; err != nil {
		t.Fatalf("create service decorator: %v", err)
	}
	modelDec.ModelId = src.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(modelDec).Error; err != nil {
		t.Fatalf("create model decorator: %v", err)
	}
	modelArg.DecoratorId = modelDec.Id
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(modelArg).Error; err != nil {
		t.Fatalf("create model arg: %v", err)
	}

	// Also exercise nil skips in copyModelTreeToRaw via direct call after migrate would work;
	// migrate path covers the live preload tree.
	if err := MigrateIMDCatalogToDualStore(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var eff []*Model
	if err := db.Preload("Decorators.Arguments").Preload("Fields.Decorators").Preload("Services.Parameters").Preload("Services.TypeParameters").Preload("Services.Decorators").Find(&eff).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if len(eff) != 1 {
		t.Fatalf("effective count=%d", len(eff))
	}
	if len(eff[0].Decorators) != 1 || len(eff[0].Decorators[0].Arguments) != 1 {
		t.Fatalf("model decorators %#v", eff[0].Decorators)
	}
	if len(eff[0].Fields) != 1 || eff[0].Fields[0].Name != "Name" {
		t.Fatalf("fields %#v", eff[0].Fields)
	}
	if len(eff[0].Fields[0].Decorators) != 1 || eff[0].Fields[0].Decorators[0].Name != "Field" {
		t.Fatalf("field decorators %#v", eff[0].Fields[0].Decorators)
	}
	if len(eff[0].Services) != 1 {
		t.Fatalf("services %#v", eff[0].Services)
	}
	effSvc := eff[0].Services[0]
	if len(effSvc.Parameters) != 1 || effSvc.Parameters[0].Name != "vals" {
		t.Fatalf("params %#v", effSvc.Parameters)
	}
	if len(effSvc.TypeParameters) != 1 || effSvc.TypeParameters[0].Name != "T" {
		t.Fatalf("type params %#v", effSvc.TypeParameters)
	}
	if len(effSvc.Decorators) != 1 {
		t.Fatalf("svc decorators %#v", effSvc.Decorators)
	}

	// Direct copy with nil children covers continue branches.
	db2 := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db2); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	srcNil := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "mn", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "NilKids",
		Application: "a",
		Path:        "/nil.ts",
		ModuleId:    sql.NullString{String: "modn", Valid: true},
		Fields:      []*Field{nil},
		Services: []*Service{nil, {
			BaseModel:      BaseModel{Id: sql.NullString{String: "sn", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:           "S",
			Parameters:     []*Parameter{nil},
			TypeParameters: []*TypeParameter{nil},
		}},
		Decorators: nil,
	}
	if err := copyModelTreeToRaw(db2, srcNil); err != nil {
		t.Fatalf("copy with nils: %v", err)
	}
}

func TestRecomputeAllEffectiveFromRaw_NilAndErrors(t *testing.T) {
	if err := RecomputeAllEffectiveFromRaw(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}

	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-older", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	newer := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "id-newer", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x2.ts",
		ModuleId:    sql.NullString{String: "mod2", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(newer).Error; err != nil {
		t.Fatalf("create newer: %v", err)
	}
	olderTime := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "zzz", Valid: true}, CreatedAt: ts.Add(-time.Hour), UpdatedAt: ts},
		Name:        "Y",
		Application: "b",
		Path:        "/y.ts",
		ModuleId:    sql.NullString{String: "mody", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(olderTime).Error; err != nil {
		t.Fatalf("create olderTime: %v", err)
	}
	younger := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "aaa", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Y",
		Application: "b",
		Path:        "/y2.ts",
		ModuleId:    sql.NullString{String: "mody2", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(younger).Error; err != nil {
		t.Fatalf("create younger: %v", err)
	}

	if err := RecomputeAllEffectiveFromRaw(db); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	// Equal CreatedAt → lexicographically greater Id wins ("id-older" > "id-newer").
	var x Model
	if err := db.Where("application = ? AND name = ?", "a", "X").First(&x).Error; err != nil {
		t.Fatalf("load X: %v", err)
	}
	if x.Id.String != "id-older" {
		t.Fatalf("X tip id=%q want id-older", x.Id.String)
	}
	var y Model
	if err := db.Where("application = ? AND name = ?", "b", "Y").First(&y).Error; err != nil {
		t.Fatalf("load Y: %v", err)
	}
	if y.Id.String != "aaa" {
		t.Fatalf("Y tip id=%q want aaa (newer CreatedAt)", y.Id.String)
	}
}

func TestRecomputeAllEffectiveFromRaw_MergeError(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "r1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	f := &RawField{
		BaseModel: BaseModel{Id: sql.NullString{String: "f1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Kind",
		FieldType: "selection",
		ModelId:   raw.Id,
	}
	spec := &FieldResolvedSpec{
		FieldName: "Kind",
		Structural: FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	}
	if err := f.SetResolvedSpec(spec); err != nil {
		t.Fatalf("spec: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(f).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	if err := RecomputeAllEffectiveFromRaw(db); err == nil || !strings.Contains(err.Error(), "selectionAdd") {
		t.Fatalf("expected merge error, got %v", err)
	}
}

func TestRecomputeAllEffectiveFromRaw_TxErrorOnClosed(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	closeDualStoreDB(t, db)
	if err := RecomputeAllEffectiveFromRaw(db); err == nil {
		t.Fatal("expected recompute error on closed db")
	}
}

func TestFindRawByIDAndRawIsNewerTip(t *testing.T) {
	if findRawByID([]*RawModel{nil}, "x") != nil {
		t.Fatal("expected nil")
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	a := &RawModel{BaseModel: BaseModel{Id: sql.NullString{String: "a", Valid: true}, CreatedAt: ts}}
	b := &RawModel{BaseModel: BaseModel{Id: sql.NullString{String: "b", Valid: true}, CreatedAt: ts}}
	if findRawByID([]*RawModel{nil, a}, "a") != a {
		t.Fatal("expected find a")
	}
	if !rawIsNewerTip(a, nil) {
		t.Fatal("nil previous")
	}
	if !rawIsNewerTip(b, a) { // equal time, higher id
		t.Fatal("expected b newer by id")
	}
	older := &RawModel{BaseModel: BaseModel{Id: sql.NullString{String: "z", Valid: true}, CreatedAt: ts.Add(-time.Hour)}}
	if rawIsNewerTip(older, a) {
		t.Fatal("older should not win")
	}
	newer := &RawModel{BaseModel: BaseModel{Id: sql.NullString{String: "c", Valid: true}, CreatedAt: ts.Add(time.Hour)}}
	if !rawIsNewerTip(newer, a) {
		t.Fatal("newer CreatedAt should win")
	}
}

func TestCopyModelTreeToRaw_CreateFailures(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	existing := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "existing", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(existing).Error; err != nil {
		t.Fatalf("seed raw: %v", err)
	}
	src := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "dup", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := copyModelTreeToRaw(db, src); err == nil {
		t.Fatal("expected unique violation creating raw model")
	}
}

func TestCopyModelTreeToRaw_ChildTableFailures(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		drop any
		src  *Model
	}{
		{
			name: "raw field",
			drop: &RawField{},
			src: &Model{
				BaseModel:   BaseModel{Id: sql.NullString{String: "m", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "X",
				Application: "a",
				Path:        "/x.ts",
				ModuleId:    sql.NullString{String: "mod", Valid: true},
				Fields: []*Field{{
					BaseModel: BaseModel{Id: sql.NullString{String: "f", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Name",
				}},
			},
		},
		{
			name: "raw service",
			drop: &RawService{},
			src: &Model{
				BaseModel:   BaseModel{Id: sql.NullString{String: "m2", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "Y",
				Application: "a",
				Path:        "/y.ts",
				ModuleId:    sql.NullString{String: "mod2", Valid: true},
				Services: []*Service{{
					BaseModel: BaseModel{Id: sql.NullString{String: "s", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Create",
				}},
			},
		},
		{
			name: "raw parameter",
			drop: &RawParameter{},
			src: &Model{
				BaseModel:   BaseModel{Id: sql.NullString{String: "m3", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "Z",
				Application: "a",
				Path:        "/z.ts",
				ModuleId:    sql.NullString{String: "mod3", Valid: true},
				Services: []*Service{{
					BaseModel:  BaseModel{Id: sql.NullString{String: "s3", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:       "Create",
					Parameters: []*Parameter{{BaseModel: BaseModel{Id: sql.NullString{String: "p", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "vals"}},
				}},
			},
		},
		{
			name: "raw type parameter",
			drop: &RawTypeParameter{},
			src: &Model{
				BaseModel:   BaseModel{Id: sql.NullString{String: "m4", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "W",
				Application: "a",
				Path:        "/w.ts",
				ModuleId:    sql.NullString{String: "mod4", Valid: true},
				Services: []*Service{{
					BaseModel:      BaseModel{Id: sql.NullString{String: "s4", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:           "Create",
					TypeParameters: []*TypeParameter{{BaseModel: BaseModel{Id: sql.NullString{String: "tp", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "T"}},
				}},
			},
		},
		{
			name: "raw decorator",
			drop: &RawDecorator{},
			src: &Model{
				BaseModel:   BaseModel{Id: sql.NullString{String: "m5", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "V",
				Application: "a",
				Path:        "/v.ts",
				ModuleId:    sql.NullString{String: "mod5", Valid: true},
				Decorators: []*Decorator{{
					BaseModel: BaseModel{Id: sql.NullString{String: "d5", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Model",
				}},
			},
		},
		{
			name: "raw argument",
			drop: &RawArgument{},
			src: &Model{
				BaseModel:   BaseModel{Id: sql.NullString{String: "m6", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "U",
				Application: "a",
				Path:        "/u.ts",
				ModuleId:    sql.NullString{String: "mod6", Valid: true},
				Decorators: []*Decorator{{
					BaseModel: BaseModel{Id: sql.NullString{String: "d6", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Model",
					Arguments: []*Argument{{BaseModel: BaseModel{Id: sql.NullString{String: "a6", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Type: "string", Value: `"U"`}},
				}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openDualStoreTestDB(t)
			if err := EnsureDualStoreTables(db); err != nil {
				t.Fatalf("ensure: %v", err)
			}
			if err := db.Migrator().DropTable(tc.drop); err != nil {
				t.Fatalf("drop %s: %v", tc.name, err)
			}
			if err := copyModelTreeToRaw(db, tc.src); err == nil {
				t.Fatalf("expected %s create failure", tc.name)
			}
		})
	}

	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure nil-decorator db: %v", err)
	}
	if err := copyDecoratorToRaw(db, nil, sql.NullString{}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("nil decorator: %v", err)
	}
}

func TestClearEffectiveShapeTrees_Error(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := db.Migrator().DropTable(&Argument{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := clearEffectiveShapeTrees(db); err == nil {
		t.Fatal("expected clear error")
	}
}

func TestPersistEffectiveProjection_FailuresAndNils(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	merged := &Model{
		BaseModel:   BaseModel{CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		Decorators:  []*Decorator{nil},
		Fields:      []*Field{nil, {Name: "Name", OriginModelPath: "kept"}},
		Services: []*Service{nil, {
			Name:           "Create",
			OriginModelPath: "svc-origin",
			Parameters:     []*Parameter{nil, {Name: "this"}, {Name: "vals"}},
			TypeParameters: []*TypeParameter{nil, {Name: "T"}},
			Decorators:     []*Decorator{{Name: "Rpc"}},
		}},
	}
	if err := persistEffectiveProjection(db, merged, "eff1"); err != nil {
		t.Fatalf("persist: %v", err)
	}

	// model create failure
	if err := persistEffectiveProjection(db, merged, "eff1"); err == nil {
		t.Fatal("expected duplicate effective id failure")
	}

	db2 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db2)
	_ = db2.Migrator().DropTable(&Field{})
	if err := persistEffectiveProjection(db2, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Fields: []*Field{{Name: "Name"}},
	}, "e2"); err == nil {
		t.Fatal("expected field create failure")
	}

	db3 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db3)
	_ = db3.Migrator().DropTable(&Service{})
	if err := persistEffectiveProjection(db3, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*Service{{Name: "Create"}},
	}, "e3"); err == nil {
		t.Fatal("expected service create failure")
	}

	db4 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db4)
	_ = db4.Migrator().DropTable(&Parameter{})
	if err := persistEffectiveProjection(db4, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*Service{{Name: "Create", Parameters: []*Parameter{{Name: "vals"}}}},
	}, "e4"); err == nil {
		t.Fatal("expected parameter create failure")
	}

	db5 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db5)
	_ = db5.Migrator().DropTable(&TypeParameter{})
	if err := persistEffectiveProjection(db5, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*Service{{Name: "Create", TypeParameters: []*TypeParameter{{Name: "T"}}}},
	}, "e5"); err == nil {
		t.Fatal("expected type parameter create failure")
	}

	db6 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db6)
	_ = db6.Migrator().DropTable(&Decorator{})
	if err := persistEffectiveProjection(db6, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Decorators: []*Decorator{{Name: "Model"}},
	}, "e6"); err == nil {
		t.Fatal("expected decorator create failure")
	}

	db7 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db7)
	_ = db7.Migrator().DropTable(&Argument{})
	if err := persistDecoratorTree(db7, &Decorator{
		Name: "Model", Arguments: []*Argument{nil, {Type: "string", Value: `"X"`}},
	}, sql.NullString{String: "mid", Valid: true}, sql.NullString{}, sql.NullString{}); err == nil {
		t.Fatal("expected argument create failure after dropping meta_argument")
	}

	if err := persistDecoratorTree(db7, nil, sql.NullString{}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("nil decorator: %v", err)
	}
}

func TestEnsureEffectiveAppNameUniqueIndex_Dialects(t *testing.T) {
	// namedDialector only overrides Name() for branch selection; SQL still runs on SQLite.
	// Each case opens a fresh DB so Dialector mutation does not leak across subtests.
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("sqlite index: %v", err)
	}
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "postgres"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("postgres index: %v", err)
	}

	db2 := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db2); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	db2.Dialector = namedDialector{Dialector: db2.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db2); err != nil {
		t.Fatalf("mysql first: %v", err)
	}
	// second call: HasIndex true → DropIndex → Create
	if err := ensureEffectiveAppNameUniqueIndex(db2); err != nil {
		t.Fatalf("mysql second: %v", err)
	}

	// CREATE failure when table is missing (first step creates temp index).
	_ = db2.Migrator().DropTable(&Model{})
	if err := ensureEffectiveAppNameUniqueIndex(db2); err == nil {
		t.Fatal("expected mysql create index failure")
	}

	// sqlite CREATE failure
	db3 := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db3); err != nil {
		t.Fatalf("ensure3: %v", err)
	}
	_ = db3.Migrator().DropTable(&Model{})
	if err := ensureEffectiveAppNameUniqueIndex(db3); err == nil || !strings.Contains(err.Error(), "create unique index") {
		t.Fatalf("expected sqlite create index failure, got %v", err)
	}

	// sqlite/postgres DDL failure must surface (closed DB; first step is CREATE temp).
	db4 := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db4); err != nil {
		t.Fatalf("ensure4: %v", err)
	}
	closeDualStoreDB(t, db4)
	if err := ensureEffectiveAppNameUniqueIndex(db4); err == nil || !strings.Contains(err.Error(), "create unique index") {
		t.Fatalf("expected create unique index error on closed db, got %v", err)
	}
}

func TestEnsureEffectiveAppNameUniqueIndex_MySQLDropError(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Break migrator by swapping dialector to one that reports HasIndex but fails DropIndex.
	// HasIndex(temp) is also true, so temp create is skipped; drop of final still fails.
	db.Dialector = failingDropDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop unique index") {
		t.Fatalf("expected drop error, got %v", err)
	}
}

func TestEnsurePartialAliveAppNameUniqueIndex_PreservesUniquenessOnCreateFinalFailure(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("initial index: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	soft := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "soft", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/p1.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(soft).Error; err != nil {
		t.Fatalf("create soft: %v", err)
	}
	if err := db.Delete(soft).Error; err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	prev := execDDL
	t.Cleanup(func() { execDDL = prev })
	execDDL = func(db *gorm.DB, sql string) error {
		// Allow temp create + drop of final; fail recreate of the final name.
		if strings.Contains(sql, "CREATE UNIQUE INDEX") &&
			strings.Contains(sql, effectiveAppNameUniqueIndex) &&
			!strings.Contains(sql, effectiveAppNameUniqueIndexTemp) {
			return errors.New("create final boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "create final boom") {
		t.Fatalf("expected create final failure, got %v", err)
	}
	// Temp partial index must still protect live uniqueness and allow soft-deleted reuse.
	live := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "live", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/p2.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(live).Error; err != nil {
		t.Fatalf("live row after soft-delete should succeed under temp partial index: %v", err)
	}
	dup := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "dup", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/p3.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(dup).Error; err == nil {
		t.Fatal("expected live (application, name) uniqueness still enforced")
	}
}

func TestEnsurePartialAliveAppNameUniqueIndex_DropErrors(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	prev := execDDL
	t.Cleanup(func() { execDDL = prev })

	execDDL = func(db *gorm.DB, sql string) error {
		if strings.HasPrefix(strings.TrimSpace(sql), "DROP INDEX") && strings.Contains(sql, effectiveAppNameUniqueIndex) && !strings.Contains(sql, "_new") {
			return errors.New("drop final boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop final boom") {
		t.Fatalf("expected drop final error, got %v", err)
	}

	execDDL = func(db *gorm.DB, sql string) error {
		if strings.HasPrefix(strings.TrimSpace(sql), "DROP INDEX") && strings.Contains(sql, effectiveAppNameUniqueIndexTemp) {
			return errors.New("drop temp boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop temp boom") {
		t.Fatalf("expected drop temp error, got %v", err)
	}
}

func TestEnsureFullAppNameUniqueIndex_CreateFinalAndDropTempErrors(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("initial: %v", err)
	}

	prev := execDDL
	t.Cleanup(func() { execDDL = prev })
	execDDL = func(db *gorm.DB, sql string) error {
		if strings.Contains(sql, "CREATE UNIQUE INDEX") &&
			strings.Contains(sql, effectiveAppNameUniqueIndex) &&
			!strings.Contains(sql, effectiveAppNameUniqueIndexTemp) {
			return errors.New("mysql create final boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "mysql create final boom") {
		t.Fatalf("expected mysql create final error, got %v", err)
	}

	execDDL = prev
	// Leave temp in place from the failed run, then force DropIndex(temp) to fail.
	db.Dialector = failingDropTempDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop boom temp") {
		t.Fatalf("expected drop temp error, got %v", err)
	}
}

type failingDropTempDialector struct {
	gorm.Dialector
	name string
}

func (d failingDropTempDialector) Name() string { return d.name }

func (d failingDropTempDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return failingDropTempMigrator{Migrator: d.Dialector.Migrator(db)}
}

type failingDropTempMigrator struct {
	gorm.Migrator
}

func (m failingDropTempMigrator) HasIndex(dst interface{}, name string) bool {
	return m.Migrator.HasIndex(dst, name)
}

func (m failingDropTempMigrator) DropIndex(dst interface{}, name string) error {
	if name == effectiveAppNameUniqueIndexTemp {
		return errors.New("drop boom temp")
	}
	return m.Migrator.DropIndex(dst, name)
}

type failingDropDialector struct {
	gorm.Dialector
	name string
}

func (d failingDropDialector) Name() string { return d.name }

func (d failingDropDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return failingDropMigrator{Migrator: d.Dialector.Migrator(db)}
}

type failingDropMigrator struct {
	gorm.Migrator
}

func (m failingDropMigrator) HasIndex(dst interface{}, name string) bool { return true }
func (m failingDropMigrator) DropIndex(dst interface{}, name string) error {
	return errors.New("drop boom")
}

func TestMigrateIMDCatalogToDualStore_FindError(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	prev := loadIMDModelsForMigrate
	t.Cleanup(func() { loadIMDModelsForMigrate = prev })
	loadIMDModelsForMigrate = func(*gorm.DB) ([]*Model, error) {
		return nil, errors.New("find boom")
	}
	if err := MigrateIMDCatalogToDualStore(db); err == nil || !strings.Contains(err.Error(), "load meta_model") {
		t.Fatalf("expected load meta_model error, got %v", err)
	}
}

func TestMigrateIMDCatalogToDualStore_CountError(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	prev := countMetaRawModels
	t.Cleanup(func() { countMetaRawModels = prev })
	countMetaRawModels = func(*gorm.DB) (int64, error) {
		return 0, errors.New("count boom")
	}
	if err := MigrateIMDCatalogToDualStore(db); err == nil || !strings.Contains(err.Error(), "count meta_raw_model") {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestRecomputeHooks_LoadClearPersistErrorsAndEmptyTip(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	prevLoad, prevClear, prevPersist := loadRawModelsForRecompute, clearEffectiveShapeTreesFn, persistEffectiveProjectionFn
	t.Cleanup(func() {
		loadRawModelsForRecompute = prevLoad
		clearEffectiveShapeTreesFn = prevClear
		persistEffectiveProjectionFn = prevPersist
	})

	loadRawModelsForRecompute = func(*gorm.DB) ([]*RawModel, error) {
		return nil, errors.New("raw find boom")
	}
	if err := RecomputeAllEffectiveFromRaw(db); err == nil || !strings.Contains(err.Error(), "load meta_raw_model") {
		t.Fatalf("expected load raw error, got %v", err)
	}

	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	loadRawModelsForRecompute = func(*gorm.DB) ([]*RawModel, error) {
		return []*RawModel{{
			BaseModel:   BaseModel{Id: sql.NullString{String: "r", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "X",
			Application: "a",
			Path:        "/x.ts",
		}}, nil
	}
	clearEffectiveShapeTreesFn = func(*gorm.DB) error { return errors.New("clear boom") }
	if err := RecomputeAllEffectiveFromRaw(db); err == nil || !strings.Contains(err.Error(), "clear boom") {
		t.Fatalf("expected clear error, got %v", err)
	}

	clearEffectiveShapeTreesFn = func(*gorm.DB) error { return nil }
	persistEffectiveProjectionFn = func(*gorm.DB, *Model, string) error { return errors.New("persist boom") }
	if err := RecomputeAllEffectiveFromRaw(db); err == nil || !strings.Contains(err.Error(), "persist effective") {
		t.Fatalf("expected persist error, got %v", err)
	}

	// Empty tip id → xid path.
	loadRawModelsForRecompute = func(*gorm.DB) ([]*RawModel, error) {
		return []*RawModel{{
			BaseModel:   BaseModel{Id: sql.NullString{String: "", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "EmptyTip",
			Application: "a",
			Path:        "/empty.ts",
		}}, nil
	}
	persistEffectiveProjectionFn = persistEffectiveProjection
	clearEffectiveShapeTreesFn = clearEffectiveShapeTrees
	if err := RecomputeAllEffectiveFromRaw(db); err != nil {
		t.Fatalf("empty tip recompute: %v", err)
	}
	var count int64
	if err := db.Model(&Model{}).Where("name = ?", "EmptyTip").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected EmptyTip row, got %d", count)
	}
}

func TestMigrateIMDSources_NilInCopyLoop(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	src := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "ok", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/ok.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := migrateIMDSources(db, []*Model{nil, src, nil}); err != nil {
		t.Fatalf("migrateIMDSources: %v", err)
	}
}

func TestCopyDecoratorToRaw_NilArgument(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	d := &Decorator{
		BaseModel: BaseModel{Id: sql.NullString{String: "d", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Model",
		Arguments: []*Argument{nil},
	}
	if err := copyDecoratorToRaw(db, d, sql.NullString{String: "m", Valid: true}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("copyDecoratorToRaw: %v", err)
	}
}

func TestCopyModelTreeToRaw_NestedDecoratorFailures(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	src := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "mf", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/xf.ts",
		ModuleId:    sql.NullString{String: "modf", Valid: true},
		Fields: []*Field{{
			BaseModel:  BaseModel{Id: sql.NullString{String: "ff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:       "Name",
			Decorators: []*Decorator{{BaseModel: BaseModel{Id: sql.NullString{String: "df", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "Field"}},
		}},
	}
	if err := db.Migrator().DropTable(&RawDecorator{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := copyModelTreeToRaw(db, src); err == nil {
		t.Fatal("expected field decorator create failure")
	}

	db2 := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db2); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	src2 := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "ms", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Y",
		Application: "a",
		Path:        "/ys.ts",
		ModuleId:    sql.NullString{String: "mods", Valid: true},
		Services: []*Service{{
			BaseModel:  BaseModel{Id: sql.NullString{String: "ss", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:       "Create",
			Decorators: []*Decorator{{BaseModel: BaseModel{Id: sql.NullString{String: "ds", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "Rpc"}},
		}},
	}
	if err := db2.Migrator().DropTable(&RawDecorator{}); err != nil {
		t.Fatalf("drop2: %v", err)
	}
	if err := copyModelTreeToRaw(db2, src2); err == nil {
		t.Fatal("expected service decorator create failure")
	}
}

func TestPersistEffectiveProjection_NestedDecoratorFailures(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	db := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db)
	_ = db.Migrator().DropTable(&Decorator{})
	if err := persistEffectiveProjection(db, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Fields: []*Field{{Name: "Name", Decorators: []*Decorator{{Name: "Field"}}}},
	}, "ef"); err == nil {
		t.Fatal("expected field decorator persist failure")
	}

	db2 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db2)
	_ = db2.Migrator().DropTable(&Decorator{})
	if err := persistEffectiveProjection(db2, &Model{
		BaseModel: BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*Service{{Name: "Create", Decorators: []*Decorator{{Name: "Rpc"}}}},
	}, "es"); err == nil {
		t.Fatal("expected service decorator persist failure")
	}

	db3 := openDualStoreTestDB(t)
	_ = EnsureDualStoreTables(db3)
	if err := persistDecoratorTree(db3, &Decorator{
		Name: "Model", Arguments: []*Argument{nil, {Type: "string", Value: `"X"`}},
	}, sql.NullString{String: "no-model", Valid: true}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("persistDecoratorTree with nil arg should succeed on sqlite: %v", err)
	}
	_ = db3.Migrator().DropTable(&Argument{})
	if err := persistDecoratorTree(db3, &Decorator{
		Name: "Model2", Arguments: []*Argument{{Type: "string", Value: `"Y"`}},
	}, sql.NullString{}, sql.NullString{}, sql.NullString{}); err == nil {
		t.Fatal("expected argument create failure")
	}
}

func TestMigrateIMDSources_CopyErrorPropagates(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	existing := &RawModel{
		BaseModel:   BaseModel{Id: sql.NullString{String: "ex", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(existing).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	src := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "src", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := migrateIMDSources(db, []*Model{src}); err == nil {
		t.Fatal("expected copy error")
	}
}