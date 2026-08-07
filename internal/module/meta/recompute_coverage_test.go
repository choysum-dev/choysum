// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func TestRecomputeKeys_NilAndFailure(t *testing.T) {
	if err := recomputeKeys(nil, []LogicalKey{{Application: "a", Name: "X"}}); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}

	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "r1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	closeDualStoreDB(t, db)
	if err := recomputeKeys(db, []LogicalKey{{Application: "a", Name: "X"}}); err == nil {
		t.Fatal("expected recomputeEffective failure on closed db")
	}
}

func TestRecomputeEffective_NilAndInvalid(t *testing.T) {
	if err := recomputeEffective(nil, "a", "X"); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}
	db := openRecomputeTestDB(t)
	if err := recomputeEffective(db, " ", "X"); err == nil || !strings.Contains(err.Error(), "non-empty application and name") {
		t.Fatalf("expected invalid key error, got %v", err)
	}
}

func TestRecomputeEffective_PostgresLockBranch(t *testing.T) {
	db := openRecomputeTestDB(t)
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "postgres"}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "pg-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "PgModel",
		Application: "pg",
		Path:        "/pg.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := recomputeEffective(db, "pg", "PgModel"); err != nil {
		t.Fatalf("recompute with postgres dialector: %v", err)
	}
}

func TestLockLogicalKey_PostgresNonNotFoundError(t *testing.T) {
	db := openRecomputeTestDB(t)
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "postgres"}
	if err := db.Migrator().DropTable(&pkgmeta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := lockLogicalKey(db, LogicalKey{Application: "a", Name: "X"})
	if err == nil || !strings.Contains(err.Error(), "lock effective row") {
		t.Fatalf("expected lock error, got %v", err)
	}
}

func TestRecomputeEffective_PreservesExistingEffectiveID(t *testing.T) {
	// Under the live (application, name) unique index only one effective row can exist;
	// recompute must reuse that id (EDS5).
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	existing := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "eff-keep", Valid: true}, CreatedAt: ts, UpdatedAt: ts.Add(time.Hour)},
		Name:        "TipPick",
		Application: "app",
		Path:        "/older.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(existing).Error; err != nil {
		t.Fatalf("create effective: %v", err)
	}
	rawTip := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-tip", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "TipPick",
		Application: "app",
		Path:        "/raw.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rawTip).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}

	if err := recomputeEffective(db, "app", "TipPick"); err != nil {
		t.Fatalf("recompute TipPick: %v", err)
	}
	var tipPick pkgmeta.Model
	if err := db.Where("application = ? AND name = ?", "app", "TipPick").Take(&tipPick).Error; err != nil {
		t.Fatalf("load TipPick: %v", err)
	}
	if tipPick.Id.String != "eff-keep" {
		t.Fatalf("TipPick id=%q want eff-keep", tipPick.Id.String)
	}
}

func TestRecomputeEffective_EffIDFromXidWhenNoExistingAndEmptyRawTip(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Fresh",
		Application: "fresh",
		Path:        "/fresh.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := recomputeEffective(db, "fresh", "Fresh"); err != nil {
		t.Fatalf("recompute: %v", err)
	}
	var eff pkgmeta.Model
	if err := db.Where("application = ? AND name = ?", "fresh", "Fresh").Take(&eff).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if strings.TrimSpace(eff.Id.String) == "" {
		t.Fatal("expected minted effective id")
	}
}

func TestLoadEffectiveServiceIDsByName_Coverage(t *testing.T) {
	db := openRecomputeTestDB(t)
	if got, err := loadEffectiveServiceIDsByName(db, ""); err != nil || got != nil {
		t.Fatalf("empty modelID: got=%#v err=%v", got, err)
	}
	if got, err := loadEffectiveServiceIDsByName(db, "  "); err != nil || got != nil {
		t.Fatalf("whitespace modelID: got=%#v err=%v", got, err)
	}

	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	eff := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "eff-svc", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "SvcModel",
		Application: "svc",
		Path:        "/svc.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	for _, s := range []*pkgmeta.Service{
		{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "s1", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "Good", ModelId: eff.Id},
		{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "s2", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "  ", ModelId: eff.Id},
		{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "NoID", ModelId: eff.Id},
	} {
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(s).Error; err != nil {
			t.Fatalf("create service: %v", err)
		}
	}
	got, err := loadEffectiveServiceIDsByName(db, eff.Id.String)
	if err != nil || len(got) != 1 || got["Good"] != "s1" {
		t.Fatalf("load services: got=%#v err=%v", got, err)
	}

	if err := db.Migrator().DropTable(&pkgmeta.Service{}); err != nil {
		t.Fatalf("drop service: %v", err)
	}
	if _, err := loadEffectiveServiceIDsByName(db, eff.Id.String); err == nil || !strings.Contains(err.Error(), "load prior effective services") {
		t.Fatalf("expected find error, got %v", err)
	}
}

func TestPickTipRaw_NilEntries(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	a := &rawModel{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "a", Valid: true}, CreatedAt: ts}}
	b := &rawModel{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "b", Valid: true}, CreatedAt: ts.Add(time.Hour)}}
	tip := pickTipRaw([]*rawModel{nil, a, nil, b})
	if tip == nil || tip.Id.String != "b" {
		t.Fatalf("pickTipRaw: got %#v", tip)
	}
}

func TestDeleteEffectiveModelTree_Coverage(t *testing.T) {
	if err := deleteEffectiveModelTree(nil, ""); err != nil {
		t.Fatalf("empty modelID no-op: %v", err)
	}
	db := openRecomputeTestDB(t)
	if err := deleteEffectiveModelTree(db, "  "); err != nil {
		t.Fatalf("whitespace no-op: %v", err)
	}

	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	eff := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "del-eff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Del",
		Application: "del",
		Path:        "/del.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	field := &pkgmeta.Field{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "df", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Name",
		ModelId:   eff.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	fieldDec := &pkgmeta.Decorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "dfd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Field",
		FieldId:   field.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldDec).Error; err != nil {
		t.Fatalf("create field dec: %v", err)
	}
	fieldArg := &pkgmeta.Argument{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "dfa", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Type:        "string",
		Value:       `"x"`,
		DecoratorId: fieldDec.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldArg).Error; err != nil {
		t.Fatalf("create field arg: %v", err)
	}
	svc := &pkgmeta.Service{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "ds", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Create",
		ModelId:   eff.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svc).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	param := &pkgmeta.Parameter{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "dp", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "vals",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(param).Error; err != nil {
		t.Fatalf("create param: %v", err)
	}
	tp := &pkgmeta.TypeParameter{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "dtp", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "T",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(tp).Error; err != nil {
		t.Fatalf("create tp: %v", err)
	}
	svcDec := &pkgmeta.Decorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "dsd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Rpc",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svcDec).Error; err != nil {
		t.Fatalf("create svc dec: %v", err)
	}
	modelDec := &pkgmeta.Decorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "dmd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Model",
		ModelId:   eff.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(modelDec).Error; err != nil {
		t.Fatalf("create model dec: %v", err)
	}

	if err := deleteEffectiveModelTree(db, eff.Id.String); err != nil {
		t.Fatalf("delete tree: %v", err)
	}
	var count int64
	if err := db.Model(&pkgmeta.Model{}).Where("id = ?", eff.Id.String).Count(&count).Error; err != nil {
		t.Fatalf("count model: %v", err)
	}
	if count != 0 {
		t.Fatalf("model still exists")
	}

	// Error paths by dropping tables mid-delete.
	dropCases := []struct {
		name string
		drop any
	}{
		{"services", &pkgmeta.Service{}},
		{"fields", &pkgmeta.Field{}},
		{"decorators", &pkgmeta.Decorator{}},
		{"arguments", &pkgmeta.Argument{}},
		{"type parameters", &pkgmeta.TypeParameter{}},
		{"parameters", &pkgmeta.Parameter{}},
		{"model", &pkgmeta.Model{}},
	}
	for _, tc := range dropCases {
		t.Run(tc.name, func(t *testing.T) {
			db2 := openRecomputeTestDB(t)
			eff2 := &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "e-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "E",
				Application: "e",
				Path:        "/e.ts",
			}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(eff2)
			f2 := &pkgmeta.Field{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "f-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "N", ModelId: eff2.Id}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(f2)
			s2 := &pkgmeta.Service{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "s-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "S", ModelId: eff2.Id}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(s2)
			d2 := &pkgmeta.Decorator{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "d-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "D", FieldId: f2.Id, ServiceId: s2.Id, ModelId: eff2.Id}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(d2)
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.Argument{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "a-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, DecoratorId: d2.Id})
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.Parameter{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "p-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "vals", ServiceId: s2.Id})
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.TypeParameter{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "tp-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "T", ServiceId: s2.Id})
			_ = db2.Migrator().DropTable(tc.drop)
			if err := deleteEffectiveModelTree(db2, eff2.Id.String); err == nil {
				t.Fatalf("expected delete error after dropping %s", tc.name)
			}
		})
	}
}

func TestPersistEffectiveProjection_PrivateHelper(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	merged := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts},
		Name:        "Wrap",
		Application: "wrap",
		Path:        "/wrap.ts",
		Fields:      []*pkgmeta.Field{{Name: "Name"}},
	}
	if err := persistEffectiveProjection(db, merged, "wrap-eff", nil); err != nil {
		t.Fatalf("persistEffectiveProjection: %v", err)
	}
	var count int64
	if err := db.Model(&pkgmeta.Model{}).Where("id = ?", "wrap-eff").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected effective row, count=%d", count)
	}
}

func TestDeleteRawModelsForModule_Coverage(t *testing.T) {
	if err := deleteRawModelsForModule(nil, ""); err != nil {
		t.Fatalf("nil db empty module: %v", err)
	}
	db := openRecomputeTestDB(t)
	if err := deleteRawModelsForModule(db, "  "); err != nil {
		t.Fatalf("whitespace module no-op: %v", err)
	}
	if err := deleteRawModelsForModule(db, "missing-mod"); err != nil {
		t.Fatalf("empty module: %v", err)
	}

	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	modID := "mod-del"
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-del", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "RawDel",
		Application: "raw",
		Path:        "/raw.ts",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	field := &rawField{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rf", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Name",
		ModelId:   raw.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	fieldDec := &rawDecorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rfd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Field",
		FieldId:   sql.NullString{String: field.Id.String, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldDec).Error; err != nil {
		t.Fatalf("create field dec: %v", err)
	}
	fieldArg := &rawArgument{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "rfa", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Type:        "string",
		Value:       `"x"`,
		DecoratorId: sql.NullString{String: fieldDec.Id.String, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(fieldArg).Error; err != nil {
		t.Fatalf("create field arg: %v", err)
	}
	svc := &rawService{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rs", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Create",
		ModelId:   raw.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svc).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	param := &rawParameter{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rp", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "vals",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(param).Error; err != nil {
		t.Fatalf("create param: %v", err)
	}
	tp := &rawTypeParameter{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rtp", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "T",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(tp).Error; err != nil {
		t.Fatalf("create tp: %v", err)
	}
	svcDec := &rawDecorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rsd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Rpc",
		ServiceId: svc.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svcDec).Error; err != nil {
		t.Fatalf("create svc dec: %v", err)
	}
	modelDec := &rawDecorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rmd", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Model",
		ModelId:   raw.Id,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(modelDec).Error; err != nil {
		t.Fatalf("create model dec: %v", err)
	}

	if err := deleteRawModelsForModule(db, modID); err != nil {
		t.Fatalf("delete raw module: %v", err)
	}
	var count int64
	if err := db.Model(&rawModel{}).Where("module_id = ?", modID).Count(&count).Error; err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if count != 0 {
		t.Fatalf("raw models remain: %d", count)
	}

	dropCases := []struct {
		name string
		drop any
	}{
		{"raw models pluck", &rawModel{}},
		{"raw services", &rawService{}},
		{"raw fields", &rawField{}},
		{"raw decorators", &rawDecorator{}},
		{"raw arguments", &rawArgument{}},
		{"raw type parameters", &rawTypeParameter{}},
		{"raw parameters", &rawParameter{}},
	}
	for _, tc := range dropCases {
		t.Run(tc.name, func(t *testing.T) {
			db2 := openRecomputeTestDB(t)
			r2 := &rawModel{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "r2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "R",
				Application: "r",
				Path:        "/r.ts",
				ModuleId:    sql.NullString{String: "m2", Valid: true},
			}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(r2)
			f2 := &rawField{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rf2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "N", ModelId: r2.Id}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(f2)
			s2 := &rawService{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rs2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "S", ModelId: r2.Id}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(s2)
			d2 := &rawDecorator{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rd2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "D", FieldId: sql.NullString{String: f2.Id.String, Valid: true}, ServiceId: s2.Id, ModelId: r2.Id}
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(d2)
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(&rawArgument{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "ra2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, DecoratorId: d2.Id})
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(&rawParameter{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rp2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "vals", ServiceId: s2.Id})
			_ = db2.Session(&gorm.Session{SkipHooks: true}).Create(&rawTypeParameter{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rtp2-" + tc.name, Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "T", ServiceId: s2.Id})
			// Drop before deleteRawModelsForModule so the first failing pluck/delete surfaces.
			_ = db2.Migrator().DropTable(tc.drop)
			if err := deleteRawModelsForModule(db2, "m2"); err == nil {
				t.Fatalf("expected delete error after dropping %s", tc.name)
			}
		})
	}
}

func TestRecomputeEffective_LoadAndExpandErrors(t *testing.T) {
	db := openRecomputeTestDB(t)
	if err := db.Migrator().DropTable(&rawModel{}); err != nil {
		t.Fatalf("drop raw: %v", err)
	}
	if err := recomputeEffective(db, "a", "X"); err == nil || !strings.Contains(err.Error(), "load meta_raw_model") {
		t.Fatalf("expected load raw error, got %v", err)
	}

	db2 := openRecomputeTestDB(t)
	if err := db2.Migrator().DropTable(&pkgmeta.Model{}); err != nil {
		t.Fatalf("drop model: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "r", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db2.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := recomputeEffective(db2, "a", "X"); err == nil || !strings.Contains(err.Error(), "load effective meta_model") {
		t.Fatalf("expected load effective error, got %v", err)
	}
}

func TestRecomputeEffective_DeleteExistingError(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	// No raws → delete existing effective trees
	eff := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "orphan", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Orphan",
		Application: "orph",
		Path:        "/orph.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
		t.Fatalf("create effective: %v", err)
	}
	if err := db.Migrator().DropTable(&pkgmeta.Field{}); err != nil {
		t.Fatalf("drop field: %v", err)
	}
	if err := recomputeEffective(db, "orph", "Orphan"); err == nil {
		t.Fatal("expected delete effective tree error")
	}
}

func TestRecomputeEffective_ExpandExtendsError(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	a := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "ra", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Circ",
		Application: "circ",
		Path:        "/a.ts",
		Extends:     "/b.ts",
		ModuleId:    sql.NullString{String: "m1", Valid: true},
	}
	b := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "rb", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Circ",
		Application: "circ",
		Path:        "/b.ts",
		Extends:     "/a.ts",
		ModuleId:    sql.NullString{String: "m2", Valid: true},
	}
	for _, r := range []*rawModel{a, b} {
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(r).Error; err != nil {
			t.Fatalf("create raw: %v", err)
		}
	}
	if err := recomputeEffective(db, "circ", "Circ"); err == nil || !strings.Contains(err.Error(), "expand extends") {
		t.Fatalf("expected expand error, got %v", err)
	}
}

func TestRecomputeEffective_LoadPriorServicesError(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	eff := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "prior-eff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Prior",
		Application: "prior",
		Path:        "/prior.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
		t.Fatalf("create effective: %v", err)
	}
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "prior-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Prior",
		Application: "prior",
		Path:        "/prior.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := db.Migrator().DropTable(&pkgmeta.Service{}); err != nil {
		t.Fatalf("drop service: %v", err)
	}
	if err := recomputeEffective(db, "prior", "Prior"); err == nil || !strings.Contains(err.Error(), "load prior effective services") {
		t.Fatalf("expected load prior services error, got %v", err)
	}
}

func TestRecomputeEffective_DeleteExistingWithRawsError(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	eff := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "del-with-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "DelRaw",
		Application: "delraw",
		Path:        "/del.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
		t.Fatalf("create effective: %v", err)
	}
	_ = db.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.Field{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "del-f", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "N",
		ModelId:   eff.Id,
	})
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "del-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "DelRaw",
		Application: "delraw",
		Path:        "/del.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := db.Migrator().DropTable(&pkgmeta.Field{}); err != nil {
		t.Fatalf("drop field: %v", err)
	}
	if err := recomputeEffective(db, "delraw", "DelRaw"); err == nil {
		t.Fatal("expected delete existing error during recompute with raws")
	}
}

func TestRecomputeEffective_PersistError(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "persist-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "PersistErr",
		Application: "perr",
		Path:        "/p.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawField{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "pf", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Name",
		ModelId:   raw.Id,
	}).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	if err := db.Migrator().DropTable(&pkgmeta.Field{}); err != nil {
		t.Fatalf("drop field: %v", err)
	}
	if err := recomputeEffective(db, "perr", "PersistErr"); err == nil || !strings.Contains(err.Error(), "persist effective") {
		t.Fatalf("expected persist error, got %v", err)
	}
}

func TestRecomputeEffective_PersistFailureRollsBackExisting(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "rb-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Rollback",
		Application: "rb",
		Path:        "/rb.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawField{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "rb-f", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Name",
		ModelId:   raw.Id,
	}).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	if err := recomputeEffective(db, "rb", "Rollback"); err != nil {
		t.Fatalf("first recompute: %v", err)
	}
	var before pkgmeta.Model
	if err := db.Where("application = ? AND name = ?", "rb", "Rollback").Take(&before).Error; err != nil {
		t.Fatalf("load before: %v", err)
	}

	const cbTag = "force-effective-field-create-fail"
	if err := db.Callback().Create().Before("gorm:create").Register(cbTag, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_field" {
			_ = tx.AddError(errors.New("forced field create fail"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove(cbTag) })

	if err := recomputeEffective(db, "rb", "Rollback"); err == nil || !strings.Contains(err.Error(), "persist effective") {
		t.Fatalf("expected persist error, got %v", err)
	}
	var after pkgmeta.Model
	if err := db.Preload("Fields").Where("application = ? AND name = ?", "rb", "Rollback").Take(&after).Error; err != nil {
		t.Fatalf("effective row missing after failed recompute: %v", err)
	}
	if after.Id.String != before.Id.String || len(after.Fields) != 1 || after.Fields[0].Name != "Name" {
		t.Fatalf("expected rolled-back effective unchanged, got id=%q fields=%#v", after.Id.String, after.Fields)
	}
}

func TestRecomputeEffective_EffIDFromRawTipWhenNoExisting(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-tip-id", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "RawTip",
		Application: "rtip",
		Path:        "/rtip.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}
	if err := recomputeEffective(db, "rtip", "RawTip"); err != nil {
		t.Fatalf("recompute: %v", err)
	}
	var eff pkgmeta.Model
	if err := db.Where("application = ? AND name = ?", "rtip", "RawTip").Take(&eff).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if eff.Id.String != "raw-tip-id" {
		t.Fatalf("eff id=%q want raw-tip-id", eff.Id.String)
	}
}

func TestRecomputeEffective_HookErrorPaths(t *testing.T) {
	db := openRecomputeTestDB(t)
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	raw := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "hook-raw", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Hook",
		Application: "hook",
		Path:        "/hook.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		t.Fatalf("create raw: %v", err)
	}

	prevLock := lockLogicalKeyFn
	lockLogicalKeyFn = func(*gorm.DB, LogicalKey) error { return errors.New("lock boom") }
	t.Cleanup(func() { lockLogicalKeyFn = prevLock })
	if err := recomputeEffective(db, "hook", "Hook"); err == nil || !strings.Contains(err.Error(), "lock boom") {
		t.Fatalf("expected lock error, got %v", err)
	}
	lockLogicalKeyFn = prevLock

	prevExpand := expandModelsAlongExtendsFn
	expandModelsAlongExtendsFn = func(*gorm.DB, []*pkgmeta.Model) error { return errors.New("expand boom") }
	t.Cleanup(func() { expandModelsAlongExtendsFn = prevExpand })
	if err := recomputeEffective(db, "hook", "Hook"); err == nil || !strings.Contains(err.Error(), "expand extends") {
		t.Fatalf("expected expand error, got %v", err)
	}
	expandModelsAlongExtendsFn = prevExpand

	prevMerge := mergeSameNameModelsByExtensionChainFn
	t.Cleanup(func() { mergeSameNameModelsByExtensionChainFn = prevMerge })
	mergeSameNameModelsByExtensionChainFn = func([]*pkgmeta.Model) (*pkgmeta.Model, error) { return nil, errors.New("merge boom") }
	if err := recomputeEffective(db, "hook", "Hook"); err == nil || !strings.Contains(err.Error(), "E2 merge") {
		t.Fatalf("expected merge error, got %v", err)
	}
	mergeSameNameModelsByExtensionChainFn = func([]*pkgmeta.Model) (*pkgmeta.Model, error) { return nil, nil }
	if err := recomputeEffective(db, "hook", "Hook"); err == nil || !strings.Contains(err.Error(), "returned nil") {
		t.Fatalf("expected nil merge error, got %v", err)
	}
	mergeSameNameModelsByExtensionChainFn = prevMerge
}

func TestDeleteTrees_LateDeleteHookErrors(t *testing.T) {
	prev := deleteWhereFn
	t.Cleanup(func() { deleteWhereFn = prev })

	seedEffective := func(t *testing.T, db *gorm.DB) string {
		t.Helper()
		ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
		id := xid.New().String()
		eff := &pkgmeta.Model{
			BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: id, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "EffHook",
			Application: "eh",
			Path:        "/eh-" + id + ".ts",
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
			t.Fatalf("create model: %v", err)
		}
		svc := &pkgmeta.Service{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "Create",
			ModelId:   eff.Id,
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(svc).Error; err != nil {
			t.Fatalf("create service: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.Field{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "Name",
			ModelId:   eff.Id,
		}).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}
		dec := &pkgmeta.Decorator{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "D",
			ModelId:   eff.Id,
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(dec).Error; err != nil {
			t.Fatalf("create decorator: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.Argument{
			BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			DecoratorId: dec.Id,
		}).Error; err != nil {
			t.Fatalf("create arg: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.Parameter{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "x",
			ServiceId: svc.Id,
		}).Error; err != nil {
			t.Fatalf("create param: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&pkgmeta.TypeParameter{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "T",
			ServiceId: svc.Id,
		}).Error; err != nil {
			t.Fatalf("create tp: %v", err)
		}
		return id
	}

	type failCase struct {
		match   func(value interface{}) bool
		wantMsg string
	}
	cases := []failCase{
		{func(v interface{}) bool { _, ok := v.(*pkgmeta.Decorator); return ok }, "delete effective decorators"},
		{func(v interface{}) bool { _, ok := v.(*pkgmeta.TypeParameter); return ok }, "delete effective type parameters"},
		{func(v interface{}) bool { _, ok := v.(*pkgmeta.Parameter); return ok }, "delete effective parameters"},
		{func(v interface{}) bool { _, ok := v.(*pkgmeta.Service); return ok }, "delete effective services"},
		{func(v interface{}) bool { _, ok := v.(*pkgmeta.Field); return ok }, "delete effective fields"},
		{func(v interface{}) bool { _, ok := v.(*pkgmeta.Model); return ok }, "delete effective model"},
	}
	for _, tc := range cases {
		db := openRecomputeTestDB(t)
		id := seedEffective(t, db)
		deleteWhereFn = func(db *gorm.DB, value interface{}, query interface{}, args ...interface{}) error {
			if tc.match(value) {
				return errors.New("boom")
			}
			return prev(db, value, query, args...)
		}
		err := deleteEffectiveModelTree(db, id)
		if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
			t.Fatalf("want %q, got %v", tc.wantMsg, err)
		}
	}
	deleteWhereFn = prev

	seedRaw := func(t *testing.T, db *gorm.DB, moduleID string) {
		t.Helper()
		ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
		raw := &rawModel{
			BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "RawHook",
			Application: "rh",
			Path:        "/rh-" + moduleID + ".ts",
			ModuleId:    sql.NullString{String: moduleID, Valid: true},
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
			t.Fatalf("create raw: %v", err)
		}
		rsvc := &rawService{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "Create",
			ModelId:   raw.Id,
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rsvc).Error; err != nil {
			t.Fatalf("create raw service: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawField{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "Name",
			ModelId:   raw.Id,
		}).Error; err != nil {
			t.Fatalf("create raw field: %v", err)
		}
		rdec := &rawDecorator{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "D",
			ModelId:   raw.Id,
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rdec).Error; err != nil {
			t.Fatalf("create raw decorator: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawArgument{
			BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			DecoratorId: rdec.Id,
		}).Error; err != nil {
			t.Fatalf("create raw arg: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawParameter{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "x",
			ServiceId: rsvc.Id,
		}).Error; err != nil {
			t.Fatalf("create raw param: %v", err)
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rawTypeParameter{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      "T",
			ServiceId: rsvc.Id,
		}).Error; err != nil {
			t.Fatalf("create raw tp: %v", err)
		}
	}

	rawCases := []failCase{
		{func(v interface{}) bool { _, ok := v.(*rawDecorator); return ok }, "delete declaration decorators"},
		{func(v interface{}) bool { _, ok := v.(*rawTypeParameter); return ok }, "delete declaration type parameters"},
		{func(v interface{}) bool { _, ok := v.(*rawParameter); return ok }, "delete declaration parameters"},
		{func(v interface{}) bool { _, ok := v.(*rawService); return ok }, "delete declaration services"},
		{func(v interface{}) bool { _, ok := v.(*rawField); return ok }, "delete declaration fields"},
		{func(v interface{}) bool { _, ok := v.(*rawModel); return ok }, "delete declaration models"},
	}
	for i, tc := range rawCases {
		db := openRecomputeTestDB(t)
		mod := "mod-hook-" + string(rune('a'+i))
		seedRaw(t, db, mod)
		deleteWhereFn = func(db *gorm.DB, value interface{}, query interface{}, args ...interface{}) error {
			if tc.match(value) {
				return errors.New("boom")
			}
			return prev(db, value, query, args...)
		}
		err := deleteRawModelsForModule(db, mod)
		if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
			t.Fatalf("raw want %q, got %v", tc.wantMsg, err)
		}
	}
	deleteWhereFn = prev
}
