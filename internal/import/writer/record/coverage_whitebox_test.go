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
	unit := recordplan.Unit{Index: 1}
	caller := &wbCaller{result: []any{map[string]any{"Id": "1"}}}
	if _, err := ResolveM2O(context.Background(), caller, unit, "DefaultCurrencyId/", "x"); err == nil {
		t.Fatal("expected empty lookup field")
	}
	if _, err := ResolveM2O(context.Background(), caller, unit, "/Code", "x"); err == nil {
		t.Fatal("expected empty field name")
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

func TestUpsertCountryByExternalID_Branches(t *testing.T) {
	unit := recordplan.Unit{Index: 1, Model: countryModelFull}
	model := &meta.Model{Name: "Country", Application: "base"}
	model.Id.String = "mid"
	key := MetaModelDataKey{Module: "import", Name: "wb1"}
	db := openWBDB(t)

	caller := &wbCaller{result: map[string]any{}}
	if err := upsertCountryByExternalID(context.Background(), caller, db, model, unit, key, map[string]any{"Code": "WB1"}); err == nil {
		t.Fatal("expected Create without Id")
	}

	caller = &wbCaller{err: errors.New("duplicate key")}
	err := upsertCountryByExternalID(context.Background(), caller, db, model, unit, key, map[string]any{"Code": "WB1"})
	imp, _ := importpkg.AsError(err)
	if imp.Code != importpkg.CodeDuplicateKey {
		t.Fatalf("create err code=%s", imp.Code)
	}

	if err := db.Create(&modmeta.ModelData{Module: "import", Name: "wb1", Application: "base", ModelName: "Country", ModelId: "mid", ResID: "gone"}).Error; err != nil {
		t.Fatal(err)
	}
	caller = &wbCaller{errOn: map[string]error{"base.Country.Search": errors.New("search failed")}}
	if err := upsertCountryByExternalID(context.Background(), caller, db, model, unit, key, map[string]any{"Code": "WB1"}); err == nil {
		t.Fatal("expected search failure")
	}

	caller = &wbCaller{
		byKey: map[string]any{"base.Country.Search": []any{map[string]any{"Id": "gone"}}},
		errOn: map[string]error{"base.Country.UpdateById": errors.New("update failed")},
	}
	if err := upsertCountryByExternalID(context.Background(), caller, db, model, unit, key, map[string]any{"Code": "WB1"}); err == nil {
		t.Fatal("expected update failure")
	}

	if ok, _ := countryExistsByID(context.Background(), &wbCaller{}, unit, "  "); ok {
		t.Fatal("empty id")
	}
}

func TestUpsertCountryByCode_Branches(t *testing.T) {
	unit := recordplan.Unit{Index: 1}
	if err := upsertCountryByCode(context.Background(), &wbCaller{err: errors.New("search boom")}, unit, map[string]any{"Code": "C1"}); err == nil {
		t.Fatal("search error")
	}
	caller := &wbCaller{
		byKey: map[string]any{"base.Country.Search": []any{map[string]any{"Id": "1"}}},
		errOn: map[string]error{"base.Country.UpdateById": errors.New("upd")},
	}
	if err := upsertCountryByCode(context.Background(), caller, unit, map[string]any{"Code": "C1"}); err == nil {
		t.Fatal("update error")
	}
	caller = &wbCaller{
		byKey: map[string]any{"base.Country.Search": []any{}},
		errOn: map[string]error{"base.Country.Create": errors.New("required field")},
	}
	err := upsertCountryByCode(context.Background(), caller, unit, map[string]any{"Code": "C1"})
	imp, _ := importpkg.AsError(err)
	if imp.Code != importpkg.CodeEmptyRequired {
		t.Fatalf("code=%s", imp.Code)
	}
	if err := upsertCountryByCode(context.Background(), caller, unit, map[string]any{"Code": "  "}); err == nil {
		t.Fatal("empty code")
	}
}

func TestBuildCountryVals_EmptyRaw(t *testing.T) {
	unit := recordplan.Unit{Index: 1, Values: map[string]string{"Code": "  "}}
	if _, err := buildCountryVals(context.Background(), &wbCaller{}, unit); err == nil {
		t.Fatal("expected empty required")
	}
}

func TestUpsertExternalIDMapping_Update(t *testing.T) {
	db := openWBDB(t)
	model := &meta.Model{Name: "Country", Application: "base"}
	model.Id.String = "mid"
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

func TestUpsertCountry_ModelNotFound(t *testing.T) {
	db := openWBDB(t)
	scope := &wbScope{db: db}
	ctx := orm.ContextWithCaller(context.Background(), &wbCaller{})
	if err := UpsertCountry(ctx, scope, recordplan.Unit{Index: 1, Model: countryModelFull, Values: map[string]string{"Code": "X"}}); err == nil {
		t.Fatal("expected model not found")
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
	if err := runtimeScope.Session().AutoMigrate(&meta.Module{}, &meta.Model{}, &modmeta.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope.Session().DB
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

func TestUpsertCountryByExternalID_LookupError(t *testing.T) {
	db := openWBDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	model := &meta.Model{Name: "Country", Application: "base"}
	model.Id.String = "m"
	err := upsertCountryByExternalID(context.Background(), &wbCaller{}, db, model, recordplan.Unit{Index: 1}, MetaModelDataKey{Module: "import", Name: "x"}, map[string]any{"Code": "A"})
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
