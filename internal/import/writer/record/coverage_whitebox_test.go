// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/orm"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

func TestHelpers_FirstRecordIDAndMapORMError(t *testing.T) {
	if firstRecordID(nil) != "" {
		t.Fatal("nil")
	}
	if firstRecordID([]map[string]any{}) != "" {
		t.Fatal("empty maps")
	}
	if firstRecordID([]map[string]any{{"id": "x"}}) != "x" {
		t.Fatal("lowercase id in maps")
	}
	if firstRecordID([]any{}) != "" {
		t.Fatal("empty any")
	}
	id, ok := recordIDFromResult(map[string]any{"id": "abc"})
	if !ok || id != "abc" {
		t.Fatalf("lowercase id: %q %v", id, ok)
	}
	if _, ok := recordIDFromResult("nope"); ok {
		t.Fatal("non-map")
	}
	if mapORMError(recordplan.Unit{}, "", nil) != nil {
		t.Fatal("nil err")
	}
	err := mapORMError(recordplan.Unit{Index: 1}, "Code", errors.New("value is required"))
	imp, _ := importpkg.AsError(err)
	if imp.Code != importpkg.CodeEmptyRequired {
		t.Fatalf("code=%s", imp.Code)
	}
}

func TestResolveM2O_EmptyParts(t *testing.T) {
	db := openWBDB(t)
	unit := recordplan.Unit{Index: 1, Model: "base.Country"}
	caller := &wbCaller{result: []any{map[string]any{"Id": "1"}}}
	if _, err := ResolveM2O(context.Background(), db, caller, unit, "DefaultCurrencyId/", "x"); err == nil {
		t.Fatal("expected empty lookup field")
	}
	if _, err := ResolveM2O(context.Background(), db, caller, unit, "/Code", "x"); err == nil {
		t.Fatal("expected empty field name")
	}
}

func TestSplitModelFullName(t *testing.T) {
	app, name, err := SplitModelFullName("base.Country")
	if err != nil || app != "base" || name != "Country" {
		t.Fatalf("%s %s %v", app, name, err)
	}
	if _, _, err := SplitModelFullName("Country"); err == nil {
		t.Fatal("expected error")
	}
}

func TestIsInstalledModuleNamespace_Empty(t *testing.T) {
	ok, err := isInstalledModuleNamespace(nil, "  ")
	if err != nil || ok {
		t.Fatalf("empty module: %v %v", ok, err)
	}
}

func TestWithImportFileContext_NilScopeContext(t *testing.T) {
	s := &nilCtxScope{session: &scope.Session{}}
	marked := WithImportFileContext(s)
	if !ImportFileFromContext(marked.Context()) {
		t.Fatal("expected marker with nil original context")
	}
}

func TestWriter_WriteNilScope(t *testing.T) {
	if err := (Writer{}).Write(context.Background(), nil, nil); err == nil {
		t.Fatal("expected scope error")
	}
}

func TestUpsertByExternalID_Branches(t *testing.T) {
	db := openWBDB(t)
	seedMinimalCountryMeta(t, db)
	unit := recordplan.Unit{Index: 1, Model: "base.Country"}
	model, err := LookupModel(db, "base.Country")
	if err != nil {
		t.Fatal(err)
	}
	key := MetaModelDataKey{Module: "import", Name: "wb1"}

	caller := &wbCaller{result: map[string]any{}}
	if err := upsertByExternalID(context.Background(), caller, db, model, "base.Country", unit, key, map[string]any{"Code": "WB1"}); err == nil {
		t.Fatal("expected Create without Id")
	}

	caller = &wbCaller{err: errors.New("duplicate key")}
	err = upsertByExternalID(context.Background(), caller, db, model, "base.Country", unit, key, map[string]any{"Code": "WB1"})
	imp, _ := importpkg.AsError(err)
	if imp.Code != importpkg.CodeDuplicateKey {
		t.Fatalf("create err code=%s", imp.Code)
	}

	if err := db.Create(&modmeta.ModelData{Module: "import", Name: "wb1", Application: "base", ModelName: "Country", ModelId: model.Id.String, ResID: "gone"}).Error; err != nil {
		t.Fatal(err)
	}
	caller = &wbCaller{errOn: map[string]error{"base.Country.Search": errors.New("search failed")}}
	if err := upsertByExternalID(context.Background(), caller, db, model, "base.Country", unit, key, map[string]any{"Code": "WB1"}); err == nil {
		t.Fatal("expected search failure")
	}

	caller = &wbCaller{
		byKey: map[string]any{"base.Country.Search": []any{map[string]any{"Id": "gone"}}},
		errOn: map[string]error{"base.Country.UpdateById": errors.New("update failed")},
	}
	if err := upsertByExternalID(context.Background(), caller, db, model, "base.Country", unit, key, map[string]any{"Code": "WB1"}); err == nil {
		t.Fatal("expected update failure")
	}

	if ok, _ := recordExistsByID(context.Background(), &wbCaller{}, unit, "base.Country", "  "); ok {
		t.Fatal("empty id")
	}
}

func TestUpsertByUniqueKeys_Branches(t *testing.T) {
	db := openWBDB(t)
	seedMinimalCountryMeta(t, db)
	fields, err := ListFields(db, mustLookupModel(t, db, "base.Country"))
	if err != nil {
		t.Fatal(err)
	}
	fieldByName := map[string]*meta.Field{}
	for i := range fields {
		fieldByName[fields[i].Name] = &fields[i]
	}
	unit := recordplan.Unit{Index: 1, Model: "base.Country"}

	if err := upsertByUniqueKeys(context.Background(), &wbCaller{err: errors.New("search boom")}, unit, "base.Country", fieldByName, map[string]any{"Code": "C1"}); err == nil {
		t.Fatal("search error")
	}
	caller := &wbCaller{
		byKey: map[string]any{"base.Country.Search": []any{map[string]any{"Id": "1"}}},
		errOn: map[string]error{"base.Country.UpdateById": errors.New("upd")},
	}
	if err := upsertByUniqueKeys(context.Background(), caller, unit, "base.Country", fieldByName, map[string]any{"Code": "C1"}); err == nil {
		t.Fatal("update error")
	}
	caller = &wbCaller{
		byKey: map[string]any{"base.Country.Search": []any{}},
		errOn: map[string]error{"base.Country.Create": errors.New("required field")},
	}
	err = upsertByUniqueKeys(context.Background(), caller, unit, "base.Country", fieldByName, map[string]any{"Code": "C1"})
	imp, _ := importpkg.AsError(err)
	if imp.Code != importpkg.CodeEmptyRequired {
		t.Fatalf("code=%s", imp.Code)
	}
	if err := upsertByUniqueKeys(context.Background(), caller, unit, "base.Country", fieldByName, map[string]any{"Name": "only"}); err == nil {
		t.Fatal("expected missing unique key")
	}
}

func TestBuildRecordVals_EmptyRaw(t *testing.T) {
	db := openWBDB(t)
	seedMinimalCountryMeta(t, db)
	fields, _ := ListFields(db, mustLookupModel(t, db, "base.Country"))
	fieldByName := map[string]*meta.Field{}
	for i := range fields {
		fieldByName[fields[i].Name] = &fields[i]
	}
	unit := recordplan.Unit{Index: 1, Model: "base.Country", Values: map[string]string{"Code": "  "}}
	if _, err := buildRecordVals(context.Background(), db, &wbCaller{}, unit, fieldByName); err == nil {
		t.Fatal("expected empty required")
	}
}

func TestUpsertExternalIDMapping_Update(t *testing.T) {
	db := openWBDB(t)
	model := &meta.Model{Name: "Country", Application: "base"}
	model.Id.String = "mid"
	if err := db.Create(model).Error; err != nil {
		t.Fatal(err)
	}
	key := MetaModelDataKey{Module: "import", Name: "u1"}
	if err := upsertExternalIDMapping(db, key, model, "r1"); err != nil {
		t.Fatal(err)
	}
	if err := upsertExternalIDMapping(db, key, model, "r2"); err != nil {
		t.Fatal(err)
	}
	var m modmeta.ModelData
	if err := db.Where("module = ? AND name = ?", "import", "u1").First(&m).Error; err != nil || m.ResID != "r2" {
		t.Fatalf("%#v %v", m, err)
	}
}

func TestAssertExternalIDWritable_DBErrors(t *testing.T) {
	db := openWBDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if err := AssertExternalIDWritable(db, MetaModelDataKey{Module: "import", Name: "x"}, 1); err == nil {
		t.Fatal("expected closed db error")
	}
	if _, err := lookupExternalID(db, MetaModelDataKey{Module: "import", Name: "x"}); err == nil {
		t.Fatal("expected lookup error")
	}
	if _, err := isInstalledModuleNamespace(db, "base"); err == nil {
		t.Fatal("expected count error")
	}
}

func TestUpsertRecord_ModelNotFound(t *testing.T) {
	db := openWBDB(t)
	scope := &wbScope{db: db}
	ctx := orm.ContextWithCaller(context.Background(), &wbCaller{})
	if err := UpsertRecord(ctx, scope, recordplan.Unit{Index: 1, Model: "base.Country", Values: map[string]string{"Code": "X"}}); err == nil {
		t.Fatal("expected model not found")
	}
}

func TestAssertExternalIDWritable_NoUpdateAndUninstalledNamespace(t *testing.T) {
	db := openWBDB(t)
	if err := db.Create(&modmeta.ModelData{
		Module: "import", Name: "locked", Application: "base", ModelName: "Country", ModelId: "m", ResID: "r", NoUpdate: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	err := AssertExternalIDWritable(db, MetaModelDataKey{Module: "import", Name: "locked"}, 1)
	imp, ok := importpkg.AsError(err)
	if !ok || imp.Code != importpkg.CodeExternalIDProtected {
		t.Fatalf("%v", err)
	}
	if err := AssertExternalIDWritable(db, MetaModelDataKey{Module: "custom", Name: "x"}, 1); err != nil {
		t.Fatalf("uninstalled namespace: %v", err)
	}
}

func TestUpsertExternalIDMapping_LookupError(t *testing.T) {
	db := openWBDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	model := &meta.Model{Name: "Country", Application: "base"}
	model.Id.String = "m"
	if err := upsertExternalIDMapping(db, MetaModelDataKey{Module: "import", Name: "x"}, model, "r"); err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestUpsertByExternalID_LookupError(t *testing.T) {
	db := openWBDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	model := &meta.Model{Name: "Country", Application: "base"}
	model.Id.String = "m"
	err := upsertByExternalID(context.Background(), &wbCaller{}, db, model, "base.Country", recordplan.Unit{Index: 1}, MetaModelDataKey{Module: "import", Name: "x"}, map[string]any{"Code": "A"})
	if err == nil {
		t.Fatal("expected lookup wrap error")
	}
}

func TestAssertModuleNamespaceWritable_DBError(t *testing.T) {
	db := openWBDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	if err := assertModuleNamespaceWritable(db, MetaModelDataKey{Module: "base", Name: "x"}, 1); err == nil {
		t.Fatal("expected module lookup error")
	}
}

type wbCaller struct {
	result any
	err    error
	byKey  map[string]any
	errOn  map[string]error
}

func (c *wbCaller) Call(_ context.Context, req orm.CallRequest) (any, error) {
	key := req.Model + "." + req.Method
	if c.errOn != nil {
		if e, ok := c.errOn[key]; ok {
			return nil, e
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.byKey != nil {
		if v, ok := c.byKey[key]; ok {
			return v, nil
		}
	}
	if c.result != nil {
		return c.result, nil
	}
	return nil, fmt.Errorf("no result for %s", key)
}

type nilCtxScope struct {
	session *scope.Session
}

func (s *nilCtxScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *nilCtxScope) Session() *scope.Session              { return s.session }
func (s *nilCtxScope) Transactor() scope.Transactor         { return nil }
func (s *nilCtxScope) Context() context.Context             { return nil }
func (s *nilCtxScope) WithContext(ctx context.Context) scope.Scope {
	return &markedScope{ctx: ctx, session: s.session}
}
func (s *nilCtxScope) Logger() *slog.Logger { return slog.Default() }

type markedScope struct {
	ctx     context.Context
	session *scope.Session
}

func (s *markedScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *markedScope) Session() *scope.Session              { return s.session }
func (s *markedScope) Transactor() scope.Transactor         { return nil }
func (s *markedScope) Context() context.Context             { return s.ctx }
func (s *markedScope) WithContext(ctx context.Context) scope.Scope {
	out := *s
	out.ctx = ctx
	return &out
}
func (s *markedScope) Logger() *slog.Logger { return slog.Default() }

type wbScope struct{ db *gorm.DB }

func (s *wbScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *wbScope) Session() *scope.Session              { return &scope.Session{DB: s.db} }
func (s *wbScope) Transactor() scope.Transactor         { return nil }
func (s *wbScope) Context() context.Context             { return context.Background() }
func (s *wbScope) WithContext(ctx context.Context) scope.Scope {
	return s
}
func (s *wbScope) Logger() *slog.Logger { return slog.Default() }

func openWBDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfg := &config.Config{Db: &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "wb.db")}}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	if err := runtimeScope.Session().AutoMigrate(&meta.Module{}, &meta.Model{}, &meta.Field{}, &modmeta.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope.Session().DB
}

func seedMinimalCountryMeta(t *testing.T, db *gorm.DB) {
	t.Helper()
	country := &meta.Model{Name: "Country", Application: "base", Path: "/tmp", ModelTable: "base_country"}
	if err := db.Create(country).Error; err != nil {
		t.Fatal(err)
	}
	unique := true
	code := &meta.Field{Name: "Code", FieldType: "varchar", NotNull: true, ModelId: country.Id}
	_ = code.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Name:         "Code",
			FieldType:    "varchar",
			StorageHints: &meta.FieldStructuralStorageHints{Unique: &unique},
		},
	})
	if err := db.Create(code).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Field{Name: "Name", FieldType: "varchar", ModelId: country.Id}).Error; err != nil {
		t.Fatal(err)
	}
}

func mustLookupModel(t *testing.T, db *gorm.DB, full string) *meta.Model {
	t.Helper()
	m, err := LookupModel(db, full)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
