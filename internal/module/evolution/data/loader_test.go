// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"context"
	"encoding/json"
	"errors"
	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildModuleRulesFromOwner(t *testing.T) {
	t.Parallel()

	depLeaf := &meta.IrModule{Name: "base", ApplicationStr: "auth"}
	depMid := &meta.IrModule{Name: "auth_ext", ApplicationStr: "auth", Dependencies: []*meta.IrModule{depLeaf}}
	rules, err := buildModuleRulesFromOwner(&meta.IrModule{
		Name:           "auth",
		ApplicationStr: "auth",
		Dependencies:   []*meta.IrModule{depMid, depLeaf, nil},
	})
	if err != nil {
		t.Fatalf("buildModuleRulesFromOwner() error = %v", err)
	}
	if rules.OwnerName != "auth" || rules.OwnerApp != "auth" {
		t.Fatalf("unexpected owner rules: %#v", rules)
	}
	for _, name := range []string{"auth", "auth_ext", "base"} {
		if _, ok := rules.Allowed[name]; !ok {
			t.Fatalf("expected module %q in allowed set: %#v", name, rules.Allowed)
		}
		if strings.TrimSpace(rules.ModuleInfo[name].Application) != "auth" {
			t.Fatalf("expected module %q to inherit auth application, got %#v", name, rules.ModuleInfo[name])
		}
	}
}

func TestBuildModuleRulesFromOwner_Errors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		owner *meta.IrModule
		want  string
	}{
		{name: "nil owner", owner: nil, want: "nil owner module"},
		{name: "missing owner name", owner: &meta.IrModule{ApplicationStr: "auth"}, want: "empty name"},
		{name: "missing owner app", owner: &meta.IrModule{Name: "auth"}, want: "empty application"},
		{name: "dependency missing app", owner: &meta.IrModule{Name: "auth", ApplicationStr: "auth", Dependencies: []*meta.IrModule{{Name: "base"}}}, want: "dependency module base has empty application"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildModuleRulesFromOwner(tc.owner)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("buildModuleRulesFromOwner() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLoadErrorFormattingAndWrappingHelpers(t *testing.T) {
	t.Parallel()

	leaf := errors.New("db failed")
	rec := record{Module: " auth ", ExternalID: " ext ", Model: " auth.User "}
	wrapped := wrapLoadError(leaf, "/tmp/data.json", 2, rec, "insert failed")
	var le *LoadError
	if !errors.As(wrapped, &le) {
		t.Fatalf("expected wrapped LoadError, got %T", wrapped)
	}
	if !errors.Is(wrapped, leaf) {
		t.Fatal("expected wrapped error to preserve cause")
	}
	got := le.Error()
	for _, want := range []string{
		"insert failed",
		"kind=db",
		"code=db_error",
		"file=/tmp/data.json",
		"recordIndex=2",
		"external_id=auth.ext",
		"model=auth.User",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("LoadError.Error() = %q, want substring %q", got, want)
		}
	}

	wrappedWithCode := wrapLoadErrorWithCode(leaf, "/tmp/other.json", 1, rec, LoadErrorKindValidation, LoadErrorCodeInvalidModel, "bad model")
	if !errors.As(wrappedWithCode, &le) {
		t.Fatalf("expected wrapped LoadError with code, got %T", wrappedWithCode)
	}
	if le.Kind != LoadErrorKindValidation || le.Code != LoadErrorCodeInvalidModel {
		t.Fatalf("unexpected wrapped load error: %#v", le)
	}

	original := &LoadError{Message: "already wrapped", Cause: leaf}
	if wrapLoadError(original, "/tmp/ignored.json", 0, rec, "ignored") != original {
		t.Fatal("expected wrapLoadError to preserve existing LoadError")
	}
	if wrapLoadErrorWithCode(original, "/tmp/ignored.json", 0, rec, LoadErrorKindDB, LoadErrorCodeDBError, "ignored") != original {
		t.Fatal("expected wrapLoadErrorWithCode to preserve existing LoadError")
	}
	if wrapLoadError(nil, "/tmp/data.json", 0, rec, "ignored") != nil {
		t.Fatal("expected wrapLoadError(nil) to return nil")
	}
	if wrapLoadErrorWithCode(nil, "/tmp/data.json", 0, rec, LoadErrorKindDB, LoadErrorCodeDBError, "ignored") != nil {
		t.Fatal("expected wrapLoadErrorWithCode(nil) to return nil")
	}
	if (*LoadError)(nil).Error() != "<nil>" {
		t.Fatalf("nil LoadError Error() mismatch")
	}
	if (*LoadError)(nil).Unwrap() != nil {
		t.Fatalf("nil LoadError Unwrap() should be nil")
	}
}

func TestApplyFiles_PublicWrapperAndGuards(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "group_admin", "model": "auth.group", "values": map[string]any{}},
		},
	})
	mod := &meta.IrModule{Name: "auth", Path: dir, ApplicationStr: "auth"}

	if err := l.ApplyFiles(context.Background(), mod, []string{"data.json"}); err != nil {
		t.Fatalf("ApplyFiles() error = %v", err)
	}
	var count int64
	if err := db.Table("auth_group").Count(&count).Error; err != nil {
		t.Fatalf("count auth_group: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one inserted row, got %d", count)
	}

	if err := l.ApplyFiles(context.Background(), nil, []string{"data.json"}); err != nil {
		t.Fatalf("ApplyFiles(nil module) error = %v", err)
	}
	if err := l.ApplyFiles(context.Background(), &meta.IrModule{Name: "auth"}, []string{"data.json"}); err == nil || !strings.Contains(err.Error(), "empty Path") {
		t.Fatalf("expected empty Path error, got %v", err)
	}
	if err := (*Loader)(nil).ApplyFiles(context.Background(), mod, []string{"data.json"}); err == nil || !strings.Contains(err.Error(), "nil loader") {
		t.Fatalf("expected nil loader error, got %v", err)
	}
	if err := (&Loader{}).ApplyFiles(context.Background(), mod, []string{"data.json"}); err == nil || !strings.Contains(err.Error(), "nil loader") {
		t.Fatalf("expected nil env error, got %v", err)
	}
}

func TestApplyModule_GuardsAndDecodeErrors(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()

	if err := (*Loader)(nil).ApplyModule(context.Background(), &meta.IrModule{Name: "auth", Path: dir}, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "nil loader") {
		t.Fatalf("expected nil loader error, got %v", err)
	}
	if err := (&Loader{}).ApplyModule(context.Background(), &meta.IrModule{Name: "auth", Path: dir}, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "nil loader") {
		t.Fatalf("expected nil env error, got %v", err)
	}
	if err := l.ApplyModule(context.Background(), nil, ApplyOptions{}); err != nil {
		t.Fatalf("ApplyModule(nil module) error = %v", err)
	}
	if err := l.ApplyModule(context.Background(), &meta.IrModule{Name: "auth"}, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "empty Path") {
		t.Fatalf("expected empty Path error, got %v", err)
	}

	invalidData := &meta.IrModule{Name: "auth", Path: dir, DataStr: []byte("{"), ApplicationStr: "auth"}
	if err := l.ApplyModule(context.Background(), invalidData, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "decode manifest data") {
		t.Fatalf("expected invalid data manifest error, got %v", err)
	}
	invalidDemo := &meta.IrModule{Name: "auth", Path: dir, DemoStr: []byte("{"), ApplicationStr: "auth"}
	if err := l.ApplyModule(context.Background(), invalidDemo, ApplyOptions{WithDemo: true}); err == nil || !strings.Contains(err.Error(), "decode manifest demo") {
		t.Fatalf("expected invalid demo manifest error, got %v", err)
	}
}

func TestApplyModule_WithDemoFiles(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()
	writeNamedDataFile(t, dir, "data.json", map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "group_data", "model": "auth.group", "values": map[string]any{}},
		},
	})
	writeNamedDataFile(t, dir, "demo.json", map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "group_demo", "model": "auth.group", "values": map[string]any{}},
		},
	})
	dataPaths, err := json.Marshal([]string{"data.json"})
	if err != nil {
		t.Fatalf("marshal data paths: %v", err)
	}
	demoPaths, err := json.Marshal([]string{"demo.json"})
	if err != nil {
		t.Fatalf("marshal demo paths: %v", err)
	}
	mod := &meta.IrModule{Name: "auth", Path: dir, DataStr: dataPaths, DemoStr: demoPaths, ApplicationStr: "auth"}

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("ApplyModule(no demo) error = %v", err)
	}
	var count int64
	if err := db.Table("auth_group").Count(&count).Error; err != nil {
		t.Fatalf("count auth_group after data-only apply: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected only data records to load without demo, got %d", count)
	}

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{WithDemo: true}); err != nil {
		t.Fatalf("ApplyModule(with demo) error = %v", err)
	}
	if err := db.Table("auth_group").Count(&count).Error; err != nil {
		t.Fatalf("count auth_group after demo apply: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected data and demo records to load, got %d", count)
	}
}

func TestApplyFiles_UsesContextTransactionFromBaseScope(t *testing.T) {
	runtimeScope := newDefaultLoaderScope(t)
	loader := New(runtimeScope)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "group_outer", "model": "auth.group", "values": map[string]any{}},
		},
	})
	mod := &meta.IrModule{Name: "auth", Path: dir, ApplicationStr: "auth"}

	outerErr := errors.New("rollback outer transaction")
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		if err := loader.ApplyFiles(tx.Context(), mod, []string{"data.json"}); err != nil {
			return err
		}

		var count int64
		if err := txScope.Session().Table("auth_group").Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			t.Fatalf("expected row visible inside outer transaction, got %d", count)
		}
		return outerErr
	})
	if !errors.Is(err, outerErr) {
		t.Fatalf("outer Required error = %v, want %v", err, outerErr)
	}

	var count int64
	if err := runtimeScope.Session().Table("auth_group").Count(&count).Error; err != nil {
		t.Fatalf("count auth_group after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected outer rollback to remove loader writes, got %d persisted rows", count)
	}
}

func TestBuildModuleRules_DBBackedPaths(t *testing.T) {
	l, db := newTestLoader(t)
	_ = l

	rules, err := buildModuleRules(db, &meta.IrModule{Name: "auth_ext", ApplicationStr: "auth"})
	if err != nil {
		t.Fatalf("buildModuleRules(db-backed) error = %v", err)
	}
	if rules.OwnerName != "auth_ext" || rules.OwnerApp != "auth" {
		t.Fatalf("unexpected rules owner: %#v", rules)
	}
	for _, want := range []string{"auth_ext", "auth"} {
		if _, ok := rules.Allowed[want]; !ok {
			t.Fatalf("expected %q in dependency closure: %#v", want, rules.Allowed)
		}
	}

	if err := db.Model(&meta.IrModule{}).Where("name = ?", "auth_ext").Update("application_str", "").Error; err != nil {
		t.Fatalf("blank auth_ext application_str: %v", err)
	}
	if err := db.Model(&meta.IrModule{}).Where("name = ?", "auth_ext").Update("application_str", "").Error; err != nil {
		t.Fatalf("confirm blank auth_ext application_str: %v", err)
	}
	_, err = buildModuleRules(db, &meta.IrModule{Name: "auth_ext", ApplicationStr: ""})
	if err == nil || !strings.Contains(err.Error(), "owner module auth_ext has empty application") {
		t.Fatalf("expected empty application error, got %v", err)
	}
	if _, _, err := loadModuleIndex(nil); err == nil || !strings.Contains(err.Error(), "nil db session") {
		t.Fatalf("expected nil db session from loadModuleIndex, got %v", err)
	}
	if got, err := dependencyClosure(db, "", nil); err != nil || got != nil {
		t.Fatalf("expected empty ownerID dependency closure to return nil,nil, got %#v, %v", got, err)
	}
}

type testAuthGroup struct {
	ID        string         `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (testAuthGroup) TableName() string { return "auth_group" }

type testAuthUser struct {
	ID        string         `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
	GroupID   string         `gorm:"column:group_id"`
}

func (testAuthUser) TableName() string { return "auth_user" }

type testModuleDependency struct {
	ModuleID       string `gorm:"column:module_id;primaryKey"`
	DependModuleID string `gorm:"column:depend_module_id;primaryKey"`
}

func (testModuleDependency) TableName() string { return "meta_ir_module_dependencies" }

type testScope struct {
	db *gorm.DB
}

func (e *testScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *testScope) Transactor() scope.Transactor                      { return scopetest.NewPassthroughTransactor(e) }
func (e *testScope) Session() *scope.Session                           { return &scope.Session{DB: e.db} }
func (e *testScope) WithContext(ctx context.Context) scope.Scope       { return e }
func (e *testScope) Context() context.Context                          { return context.Background() }
func (e *testScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *testScope) Config() *config.Config { return &config.Config{} }

func (e *testScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newTestLoader(t *testing.T) (*Loader, *gorm.DB) {
	t.Helper()
	db := newLoaderTestDB(t)
	return New(&testScope{db: db}), db
}

func newLoaderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique shared in-memory database per test to avoid cross-test interference.
	dsn := "file:dataloader_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	seedLoaderTestSchema(t, db)
	return db
}

func newDefaultLoaderScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect:         "sqlite",
			DSN:             filepath.Join(t.TempDir(), "dataloader.db"),
			MaxIdleConns:    1,
			MaxOpenConns:    1,
			ConnMaxLifetime: 60,
		},
		Log: config.NewDefaultLogConfig(),
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	seedLoaderTestSchema(t, runtimeScope.Session().DB)
	if sqlDB, err := runtimeScope.Session().DB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return runtimeScope
}

func seedLoaderTestSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.IrModule{}, &meta.IrModel{}, &metadata.IrModelData{}, &testAuthGroup{}, &testAuthUser{}, &testModuleDependency{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	// Seed models so ApplyModule can resolve model -> table.
	if err := db.Create(&meta.IrModel{Name: "group", Application: "auth", Path: "/tmp", ModelTable: "auth_group"}).Error; err != nil {
		t.Fatalf("seed meta_ir_model auth.group: %v", err)
	}
	if err := db.Create(&meta.IrModel{Name: "User", Application: "auth", Path: "/tmp", ModelTable: "auth_user"}).Error; err != nil {
		t.Fatalf("seed meta_ir_model auth.User: %v", err)
	}

	auth := &meta.IrModule{Name: "auth", ApplicationStr: "auth", Path: "/tmp"}
	if err := db.Create(auth).Error; err != nil {
		t.Fatalf("create module auth: %v", err)
	}
	authExt := &meta.IrModule{Name: "auth_ext", ApplicationStr: "auth", Path: "/tmp"}
	if err := db.Create(authExt).Error; err != nil {
		t.Fatalf("create module auth_ext: %v", err)
	}
	base := &meta.IrModule{Name: "base", ApplicationStr: "base", Path: "/tmp"}
	if err := db.Create(base).Error; err != nil {
		t.Fatalf("create module base: %v", err)
	}
	if err := db.Exec("INSERT INTO meta_ir_module_dependencies (module_id, depend_module_id) VALUES (?, ?)", authExt.Id.String, auth.Id.String).Error; err != nil {
		t.Fatalf("insert module dependency auth_ext -> auth: %v", err)
	}
}

func writeDataFile(t *testing.T, dir string, df any) {
	t.Helper()
	b, err := json.Marshal(df)
	if err != nil {
		t.Fatalf("marshal data file: %v", err)
	}
	absPath := filepath.Join(dir, "data.json")
	if err := os.WriteFile(absPath, b, 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}
}

func writeNamedDataFile(t *testing.T, dir string, name string, df any) {
	t.Helper()
	b, err := json.Marshal(df)
	if err != nil {
		t.Fatalf("marshal data file: %v", err)
	}
	absPath := filepath.Join(dir, name)
	if err := os.WriteFile(absPath, b, 0o644); err != nil {
		t.Fatalf("write data file %s: %v", name, err)
	}
}

func moduleWithDataFile(t *testing.T, dir string) *meta.IrModule {
	return moduleWithDataFileNamed(t, dir, "auth")
}

func moduleWithDataFiles(t *testing.T, dir string, relPaths []string) *meta.IrModule {
	return moduleWithDataFilesNamed(t, dir, "auth", relPaths)
}

func moduleWithDataFileNamed(t *testing.T, dir string, name string) *meta.IrModule {
	t.Helper()
	paths, err := json.Marshal([]string{"data.json"})
	if err != nil {
		t.Fatalf("marshal manifest data: %v", err)
	}
	return &meta.IrModule{
		Name:    name,
		Path:    dir,
		DataStr: paths,
	}
}

func moduleWithDataFilesNamed(t *testing.T, dir string, name string, relPaths []string) *meta.IrModule {
	t.Helper()
	paths, err := json.Marshal(relPaths)
	if err != nil {
		t.Fatalf("marshal manifest data: %v", err)
	}
	return &meta.IrModule{
		Name:    name,
		Path:    dir,
		DataStr: paths,
	}
}

func TestApplyModule_DuplicateExternalIDIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "x", "model": "auth.User", "values": map[string]any{}},
			map[string]any{"module": "auth", "external_id": "x", "model": "auth.User", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindValidation {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindValidation, le.Kind)
	}
	if le.Code != LoadErrorCodeDuplicateExternalIDInInput {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeDuplicateExternalIDInInput, le.Code)
	}
	if le.RecordIndex != 1 {
		t.Fatalf("expected RecordIndex=1, got %d", le.RecordIndex)
	}
	if le.Module != "auth" {
		t.Fatalf("expected Module=auth, got %q", le.Module)
	}
	if le.ExternalID != "x" {
		t.Fatalf("expected ExternalID=x, got %q", le.ExternalID)
	}
	if le.Message == "" {
		t.Fatalf("expected non-empty message")
	}
}

func TestApplyModule_RefNotFoundIsRejectedWithFieldPath(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.missing"},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindRef {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindRef, le.Kind)
	}
	if le.Code != LoadErrorCodeRefNotFound {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeRefNotFound, le.Code)
	}
	if le.RecordIndex != 0 {
		t.Fatalf("expected RecordIndex=0, got %d", le.RecordIndex)
	}
	if le.FieldPath != "values.group_id" {
		t.Fatalf("expected FieldPath=values.group_id, got %q", le.FieldPath)
	}
	if le.Ref != "auth.missing" {
		t.Fatalf("expected Ref=auth.missing, got %q", le.Ref)
	}
}

func TestApplyModule_CrossModuleRefNotFoundIncludesHint(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "base.missing"},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindRef {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindRef, le.Kind)
	}
	if le.Code != LoadErrorCodeRefNotFound {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeRefNotFound, le.Code)
	}
	if le.Message == "" || !strings.Contains(le.Message, "dependency") {
		t.Fatalf("expected message to include dependency hint, got %q", le.Message)
	}
	if !strings.Contains(le.Message, "base") || !strings.Contains(le.Message, "auth") {
		t.Fatalf("expected message to mention base/auth, got %q", le.Message)
	}
}

func TestApplyModule_RefByResolvesExistingRow(t *testing.T) {
	l, db := newTestLoader(t)

	// Add meta.IrApplication table + model mapping so refBy can resolve it.
	if err := db.AutoMigrate(&meta.IrApplication{}); err != nil {
		t.Fatalf("migrate meta_ir_application: %v", err)
	}
	if err := db.Create(&meta.IrModel{Name: "IrApplication", Application: "meta", Path: "/tmp", ModelTable: "meta_ir_application"}).Error; err != nil {
		t.Fatalf("seed meta_ir_model meta.IrApplication: %v", err)
	}

	app := &meta.IrApplication{Name: "auth"}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed meta_ir_application(auth): %v", err)
	}
	appID := strings.TrimSpace(app.Id.String)
	if appID == "" {
		t.Fatalf("expected non-empty app id")
	}

	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "u",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{
						"refBy": map[string]any{"model": "meta.IrApplication", "field": "Name", "value": "auth"},
					},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("ApplyModule: %v", err)
	}

	var got string
	if err := db.Table("auth_user").Select("group_id").Limit(1).Scan(&got).Error; err != nil {
		t.Fatalf("query auth_user.group_id: %v", err)
	}
	if strings.TrimSpace(got) != appID {
		t.Fatalf("expected group_id=%q, got %q", appID, got)
	}
}

func TestPlanRecordOrder_ForwardRefIsReordered(t *testing.T) {
	l, db := newTestLoader(t)
	owner := &meta.IrModule{Name: "auth"}
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.b"},
				},
			},
			map[string]any{"module": "auth", "external_id": "b", "model": "auth.group", "values": map[string]any{}},
		},
	})
	// Parse the file into dataFile records to test only the planner.
	b, err := os.ReadFile(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("read data file: %v", err)
	}
	var df dataFile
	if err := json.Unmarshal(b, &df); err != nil {
		t.Fatalf("unmarshal data file: %v", err)
	}

	order, err := l.planRecordOrder(db, owner, filepath.Join(dir, "data.json"), df.Records)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("expected order length 2, got %d", len(order))
	}
	// b must be applied before a.
	if order[0] != 1 || order[1] != 0 {
		t.Fatalf("expected order [1 0], got %v", order)
	}
}

func TestPlanRecordOrder_GuardsAndValidationErrors(t *testing.T) {
	l, db := newTestLoader(t)
	filePath := filepath.Join(t.TempDir(), "data.json")

	if _, err := l.planRecordOrder(db, nil, filePath, nil); err == nil || !strings.Contains(err.Error(), "nil owner module") {
		t.Fatalf("expected nil owner error, got %v", err)
	}
	order, err := l.planRecordOrder(db, &meta.IrModule{Name: "auth"}, filePath, nil)
	if err != nil {
		t.Fatalf("empty planRecordOrder() error = %v", err)
	}
	if order != nil {
		t.Fatalf("expected nil order for empty records, got %v", order)
	}

	for _, tc := range []struct {
		name string
		rec  record
		code string
	}{
		{name: "missing external id", rec: record{Module: "auth", Model: "auth.User", Values: map[string]any{}}, code: LoadErrorCodeMissingExternalID},
		{name: "missing model", rec: record{Module: "auth", ExternalID: "x", Values: map[string]any{}}, code: LoadErrorCodeMissingModel},
		{name: "missing values", rec: record{Module: "auth", ExternalID: "x", Model: "auth.User"}, code: LoadErrorCodeMissingValues},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := l.planRecordOrder(db, &meta.IrModule{Name: "auth"}, filePath, []record{tc.rec})
			var le *LoadError
			if !errors.As(err, &le) || le.Code != tc.code {
				t.Fatalf("planRecordOrder() error = %v, want code %s", err, tc.code)
			}
		})
	}
}

func TestPlanRecordOrder_SelfReferenceIsRejected(t *testing.T) {
	l, db := newTestLoader(t)
	filePath := filepath.Join(t.TempDir(), "data.json")

	_, err := l.planRecordOrder(db, &meta.IrModule{Name: "auth"}, filePath, []record{{
		Module:     "auth",
		ExternalID: "self",
		Model:      "auth.User",
		Values: map[string]any{
			"group_id": map[string]any{"ref": "auth.self"},
		},
	}})
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeRefSelfCycle || le.FieldPath != "values.group_id" || le.Ref != "auth.self" {
		t.Fatalf("planRecordOrder() error = %#v, want self-cycle load error", err)
	}
}

func TestApplyFile_SuccessAndErrors(t *testing.T) {
	l, db := newTestLoader(t)
	owner := &meta.IrModule{Name: "auth"}
	dir := t.TempDir()
	absPath := filepath.Join(dir, "data.json")

	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "user_admin",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.group_admin"},
				},
			},
			map[string]any{"module": "auth", "external_id": "group_admin", "model": "auth.group", "values": map[string]any{}},
		},
	})

	if err := l.applyFile(context.Background(), owner, absPath); err != nil {
		t.Fatalf("applyFile() error = %v", err)
	}
	var insertedUser testAuthUser
	if err := db.Table("auth_user").First(&insertedUser).Error; err != nil {
		t.Fatalf("query auth_user: %v", err)
	}
	if strings.TrimSpace(insertedUser.GroupID) == "" {
		t.Fatalf("expected resolved group id, got %#v", insertedUser)
	}

	if err := l.applyFile(context.Background(), owner, filepath.Join(dir, "missing.json")); err == nil || !strings.Contains(err.Error(), "read data file") {
		t.Fatalf("expected read error, got %v", err)
	}
	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid file: %v", err)
	}
	if err := l.applyFile(context.Background(), owner, invalidPath); err == nil || !strings.Contains(err.Error(), "parse data file") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestTopoOrderOrCycle_AcyclicAndCycleFallback(t *testing.T) {
	t.Run("acyclic stable order", func(t *testing.T) {
		records := []record{{ExternalID: "a"}, {ExternalID: "b"}, {ExternalID: "c"}}
		dep := [][]int{{}, {0}, {0}}
		adj := [][]int{{1, 2}, {}, {}}
		indeg := []int{0, 1, 1}

		order, err := topoOrderOrCycle(records, dep, adj, indeg, nil, "/tmp/data.json")
		if err != nil {
			t.Fatalf("topoOrderOrCycle() error = %v", err)
		}
		if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
			t.Fatalf("unexpected topological order: %v", order)
		}
	})

	t.Run("cycle without edge info still returns structured error", func(t *testing.T) {
		records := []record{
			{Module: "auth", ExternalID: "a", Model: "auth.User"},
			{Module: "auth", ExternalID: "b", Model: "auth.group"},
		}
		dep := [][]int{{1}, {0}}
		adj := [][]int{{1}, {0}}
		indeg := []int{1, 1}

		_, err := topoOrderOrCycle(records, dep, adj, indeg, map[[2]int]refOccurrence{}, "/tmp/data.json")
		var le *LoadError
		if !errors.As(err, &le) {
			t.Fatalf("expected LoadError, got %T: %v", err, err)
		}
		if le.Code != LoadErrorCodeRefCycle || !strings.Contains(le.Message, "auth.a") || !strings.Contains(le.Message, "auth.b") {
			t.Fatalf("unexpected cycle error: %#v", le)
		}
		if le.FieldPath != "" || le.Ref != "" {
			t.Fatalf("expected empty field/ref without edge info, got %#v", le)
		}
	})
}

func TestFindCycleAndCollectRefOccurrencesHelpers(t *testing.T) {
	t.Parallel()

	if cycle := findCycle([][]int{{1}, {2}, {0}}); len(cycle) != 4 || cycle[0] != 0 || cycle[3] != 0 {
		t.Fatalf("unexpected cycle result: %v", cycle)
	}
	if cycle := findCycle([][]int{{1}, {}}); cycle != nil {
		t.Fatalf("expected nil cycle, got %v", cycle)
	}

	occ := collectRefOccurrences(map[string]any{
		"items": []any{
			map[string]any{"ref": "auth.a"},
			[]any{map[string]any{"ref": "auth.b"}},
		},
		"name": "plain",
	})
	if len(occ) != 2 {
		t.Fatalf("expected 2 ref occurrences, got %#v", occ)
	}
	if occ[0].FieldPath != "values.items[0]" || occ[0].Ref != "auth.a" {
		t.Fatalf("unexpected first occurrence: %#v", occ[0])
	}
	if occ[1].FieldPath != "values.items[1][0]" || occ[1].Ref != "auth.b" {
		t.Fatalf("unexpected second occurrence: %#v", occ[1])
	}
}

func TestValueResolutionHelpers(t *testing.T) {
	l, db := newTestLoader(t)

	if err := db.AutoMigrate(&meta.IrApplication{}); err != nil {
		t.Fatalf("migrate meta_ir_application: %v", err)
	}
	if err := db.Create(&meta.IrModel{Name: "IrApplication", Application: "meta", Path: "/tmp", ModelTable: "meta_ir_application"}).Error; err != nil {
		t.Fatalf("seed meta.IrApplication model: %v", err)
	}
	app := &meta.IrApplication{Name: "auth"}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed meta_ir_application(auth): %v", err)
	}
	if err := db.Create(&metadata.IrModelData{Module: "auth", ExternalID: "group_admin", Model: "auth.group", ResID: "gid-1"}).Error; err != nil {
		t.Fatalf("seed ir_model_data: %v", err)
	}

	t.Run("normalizeSeedDBValue", func(t *testing.T) {
		cases := []struct {
			name string
			in   any
			want any
		}{
			{name: "nil", in: nil, want: nil},
			{name: "scalar", in: 12, want: 12},
			{name: "slice", in: []any{"a", true}, want: `["a",true]`},
			{name: "map", in: map[string]any{"enabled": true}, want: `{"enabled":true}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := normalizeSeedDBValue(tc.in)
				if err != nil || got != tc.want {
					t.Fatalf("normalizeSeedDBValue(%#v) = %#v, %v; want %#v", tc.in, got, err, tc.want)
				}
			})
		}
		if _, err := normalizeSeedDBValue(map[string]any{"bad": func() {}}); err == nil {
			t.Fatalf("expected marshal error for unsupported value")
		}
	})

	t.Run("extractors and split helpers", func(t *testing.T) {
		if ref, ok := extractRef(map[string]any{"ref": " auth.group_admin "}); !ok || ref != "auth.group_admin" {
			t.Fatalf("extractRef() = %q, %v", ref, ok)
		}
		if _, ok := extractRef(map[string]any{"ref": 1}); ok {
			t.Fatalf("expected extractRef to reject non-string")
		}

		modelFull, field, value, ok := extractRefBy(map[string]any{"refBy": map[string]any{"model": " meta.IrApplication ", "field": " Name ", "value": "auth"}})
		if !ok || modelFull != "meta.IrApplication" || field != "Name" || value != "auth" {
			t.Fatalf("extractRefBy() = (%q, %q, %#v, %v)", modelFull, field, value, ok)
		}
		if _, _, _, ok := extractRefBy(map[string]any{"ref_by": map[string]any{"model": "meta.IrApplication", "field": "Name", "value": "auth"}}); !ok {
			t.Fatalf("expected extractRefBy to support ref_by alias")
		}
		if _, _, _, ok := extractRefBy(map[string]any{"refBy": map[string]any{"model": "", "field": "Name", "value": "auth"}}); ok {
			t.Fatalf("expected extractRefBy to reject empty model")
		}

		if mod, ext, err := splitRef(" auth.group_admin "); err != nil || mod != "auth" || ext != "group_admin" {
			t.Fatalf("splitRef() = (%q, %q, %v)", mod, ext, err)
		}
		if _, _, err := splitRef("auth"); err == nil {
			t.Fatalf("expected splitRef to reject invalid form")
		}
		if app, model, err := splitModel(" auth.User "); err != nil || app != "auth" || model != "User" {
			t.Fatalf("splitModel() = (%q, %q, %v)", app, model, err)
		}
		if _, _, err := splitModel("auth"); err == nil {
			t.Fatalf("expected splitModel to reject invalid form")
		}
	})

	t.Run("resolveRef helpers", func(t *testing.T) {
		resID, err := l.resolveRefBy(db, "meta.IrApplication", "Name", "auth")
		if err != nil || strings.TrimSpace(resID) != strings.TrimSpace(app.Id.String) {
			t.Fatalf("resolveRefBy() = %q, %v", resID, err)
		}
		if _, err := l.resolveRefBy(db, "meta.IrApplication", "Name", "missing"); err == nil {
			t.Fatalf("expected resolveRefBy not found error")
		}
		if _, err := l.resolveRefBy(db, "", "Name", "auth"); err == nil {
			t.Fatalf("expected resolveRefBy empty model/field error")
		}

		resID, err = l.resolveRef(db, "auth.group_admin")
		if err != nil || resID != "gid-1" {
			t.Fatalf("resolveRef() = %q, %v", resID, err)
		}
		if _, err := l.resolveRef(db, "auth.missing"); err == nil {
			t.Fatalf("expected resolveRef not found error")
		}
	})

	t.Run("resolveValue", func(t *testing.T) {
		rec := record{Module: "auth", ExternalID: "u", Model: "auth.User"}
		got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"ref": "auth.group_admin"})
		if err != nil || got != "gid-1" {
			t.Fatalf("resolveValue(ref) = %#v, %v", got, err)
		}
		got, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"refBy": map[string]any{"model": "meta.IrApplication", "field": "Name", "value": "auth"}})
		if err != nil || strings.TrimSpace(got.(string)) != strings.TrimSpace(app.Id.String) {
			t.Fatalf("resolveValue(refBy) = %#v, %v", got, err)
		}
		got, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.items", []any{map[string]any{"ref": "auth.group_admin"}, "plain"})
		arr, ok := got.([]any)
		if err != nil || !ok || len(arr) != 2 || arr[0] != "gid-1" || arr[1] != "plain" {
			t.Fatalf("resolveValue(array) = %#v, %v", got, err)
		}
		if _, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"ref": "auth.missing"}); err == nil {
			t.Fatalf("expected resolveValue ref error")
		}
		if _, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"refBy": map[string]any{"model": "meta.IrApplication", "field": "Name", "value": "missing"}}); err == nil {
			t.Fatalf("expected resolveValue refBy error")
		}
		if got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.nil", nil); err != nil || got != nil {
			t.Fatalf("resolveValue(nil) = %#v, %v", got, err)
		}
	})
}

func TestResolveAndMapValues_SkipsSystemFieldsAndNormalizes(t *testing.T) {
	l, db := newTestLoader(t)
	rec := record{Module: "auth", ExternalID: "u", Model: "auth.User"}
	if err := db.Create(&metadata.IrModelData{Module: "auth", ExternalID: "group_admin", Model: "auth.group", ResID: "gid-1"}).Error; err != nil {
		t.Fatalf("seed ir_model_data: %v", err)
	}

	columns, err := l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, map[string]any{
		"Id":        "ignored",
		"CreatedAt": "ignored",
		"GroupID":   map[string]any{"ref": "auth.group_admin"},
		"Meta":      map[string]any{"enabled": true},
		"Tags":      []any{"a", true},
	})
	if err != nil {
		t.Fatalf("resolveAndMapValues() error = %v", err)
	}
	if _, ok := columns["id"]; ok {
		t.Fatalf("expected system field id to be skipped: %#v", columns)
	}
	if columns["group_id"] != "gid-1" {
		t.Fatalf("expected resolved group_id, got %#v", columns)
	}
	if columns["meta"] != `{"enabled":true}` {
		t.Fatalf("expected normalized meta JSON, got %#v", columns["meta"])
	}
	if columns["tags"] != `["a",true]` {
		t.Fatalf("expected normalized tags JSON, got %#v", columns["tags"])
	}

	_, err = l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, map[string]any{
		"Bad": map[string]any{"fn": func() {}},
	})
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeMissingValues || le.FieldPath != "values.Bad" {
		t.Fatalf("expected normalization load error, got %#v", err)
	}
}

func TestApplyRecord_DirectBranches(t *testing.T) {
	l, db := newTestLoader(t)
	now := time.Now()

	for _, tc := range []struct {
		name string
		rec  record
		code string
	}{
		{name: "missing module", rec: record{ExternalID: "x", Model: "auth.User", Values: map[string]any{}}, code: LoadErrorCodeMissingModule},
		{name: "missing external id", rec: record{Module: "auth", Model: "auth.User", Values: map[string]any{}}, code: LoadErrorCodeMissingExternalID},
		{name: "missing model", rec: record{Module: "auth", ExternalID: "x", Values: map[string]any{}}, code: LoadErrorCodeMissingModel},
		{name: "invalid model", rec: record{Module: "auth", ExternalID: "x", Model: "broken", Values: map[string]any{}}, code: LoadErrorCodeInvalidModel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := l.applyRecord(db, nil, "/tmp/data.json", 0, tc.rec, now)
			var le *LoadError
			if !errors.As(err, &le) || le.Code != tc.code {
				t.Fatalf("applyRecord() error = %#v, want code %s", err, tc.code)
			}
		})
	}

	if err := db.Create(&meta.IrModel{Name: "Broken", Application: "auth", Path: "/tmp", ModelTable: ""}).Error; err != nil {
		t.Fatalf("seed broken model: %v", err)
	}
	err := l.applyRecord(db, nil, "/tmp/data.json", 0, record{Module: "auth", ExternalID: "x", Model: "auth.Broken", Values: map[string]any{}}, now)
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeDBModelTableEmpty {
		t.Fatalf("expected model table empty error, got %#v", err)
	}

	group1 := testAuthGroup{ID: "group-1"}
	group2 := testAuthGroup{ID: "group-2"}
	user := testAuthUser{ID: "user-1", GroupID: group1.ID}
	if err := db.Table("auth_group").Create(&group1).Error; err != nil {
		t.Fatalf("seed group1: %v", err)
	}
	if err := db.Table("auth_group").Create(&group2).Error; err != nil {
		t.Fatalf("seed group2: %v", err)
	}
	if err := db.Table("auth_user").Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&metadata.IrModelData{Module: "auth", ExternalID: "user_admin", Model: "auth.User", ResID: user.ID, NoUpdate: false}).Error; err != nil {
		t.Fatalf("seed user mapping: %v", err)
	}
	if err := db.Create(&metadata.IrModelData{Module: "auth", ExternalID: "group_admin", Model: "auth.group", ResID: group2.ID, NoUpdate: false}).Error; err != nil {
		t.Fatalf("seed group mapping: %v", err)
	}
	freeze := true
	if err := l.applyRecord(db, nil, "/tmp/data.json", 1, record{
		Module:     "auth",
		ExternalID: "user_admin",
		Model:      "auth.User",
		NoUpdate:   &freeze,
		Values:     map[string]any{"group_id": map[string]any{"ref": "auth.group_admin"}},
	}, now); err != nil {
		t.Fatalf("applyRecord(update existing) error = %v", err)
	}
	var updatedUser testAuthUser
	if err := db.Table("auth_user").Where("id = ?", user.ID).First(&updatedUser).Error; err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if updatedUser.GroupID != group2.ID {
		t.Fatalf("expected updated group id %q, got %#v", group2.ID, updatedUser)
	}
	var mapping metadata.IrModelData
	if err := db.Where("module = ? AND external_id = ?", "auth", "user_admin").First(&mapping).Error; err != nil {
		t.Fatalf("query updated mapping: %v", err)
	}
	if !mapping.NoUpdate {
		t.Fatalf("expected mapping no_update to flip true, got %#v", mapping)
	}
}

func TestApplyModule_CycleRefIsRejectedWithChain(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"x": map[string]any{"ref": "auth.b"},
				},
			},
			map[string]any{
				"module":      "auth",
				"external_id": "b",
				"model":       "auth.group",
				"values": map[string]any{
					"y": map[string]any{"ref": "auth.a"},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindRef {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindRef, le.Kind)
	}
	if le.Code != LoadErrorCodeRefCycle {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeRefCycle, le.Code)
	}
	if le.Message == "" || !strings.Contains(le.Message, "circular ref detected") {
		t.Fatalf("expected cycle message, got %q", le.Message)
	}
	if le.FilePath == "" {
		t.Fatalf("expected FilePath")
	}
	if le.RecordIndex < 0 {
		t.Fatalf("expected RecordIndex >= 0, got %d", le.RecordIndex)
	}
	if le.FieldPath == "" {
		t.Fatalf("expected FieldPath to be set")
	}
	if le.Ref == "" {
		t.Fatalf("expected Ref to be set")
	}
}

func TestApplyModule_CycleRef3NodesIsRejectedWithChain(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"x": map[string]any{"ref": "auth.b"},
				},
			},
			map[string]any{
				"module":      "auth",
				"external_id": "b",
				"model":       "auth.group",
				"values": map[string]any{
					"y": map[string]any{"ref": "auth.c"},
				},
			},
			map[string]any{
				"module":      "auth",
				"external_id": "c",
				"model":       "auth.group",
				"values": map[string]any{
					"z": map[string]any{"ref": "auth.a"},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Message == "" || !strings.Contains(le.Message, "circular ref detected") {
		t.Fatalf("expected cycle message, got %q", le.Message)
	}
	// Don't assert exact chain order (implementation detail); ensure key nodes appear.
	if !strings.Contains(le.Message, "auth.a") || !strings.Contains(le.Message, "auth.b") || !strings.Contains(le.Message, "auth.c") {
		t.Fatalf("expected message to mention auth.a/auth.b/auth.c, got %q", le.Message)
	}
	if le.FieldPath == "" {
		t.Fatalf("expected FieldPath to be set")
	}
	if le.Ref == "" {
		t.Fatalf("expected Ref to be set")
	}
}

func TestApplyModule_CrossFileForwardRefIsReordered(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeNamedDataFile(t, dir, "a.json", map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.b"},
				},
			},
		},
	})
	writeNamedDataFile(t, dir, "b.json", map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "b", "model": "auth.group", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFiles(t, dir, []string{"a.json", "b.json"})

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %T: %v", err, err)
	}
}

func TestApplyModule_CrossFileCycleIsRejectedWithFileInfo(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeNamedDataFile(t, dir, "a.json", map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "a",
				"model":       "auth.User",
				"values": map[string]any{
					"x": map[string]any{"ref": "auth.b"},
				},
			},
		},
	})
	writeNamedDataFile(t, dir, "b.json", map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "b",
				"model":       "auth.group",
				"values": map[string]any{
					"y": map[string]any{"ref": "auth.a"},
				},
			},
		},
	})
	mod := moduleWithDataFiles(t, dir, []string{"a.json", "b.json"})

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindRef {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindRef, le.Kind)
	}
	if le.Code != LoadErrorCodeRefCycle {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeRefCycle, le.Code)
	}
	if le.Message == "" || !strings.Contains(le.Message, "circular ref detected") {
		t.Fatalf("expected cycle message, got %q", le.Message)
	}
	if !strings.Contains(le.Message, "file=a.json") || !strings.Contains(le.Message, "file=b.json") {
		t.Fatalf("expected message to include file basenames, got %q", le.Message)
	}
	if !strings.Contains(le.Message, "recordIndex=0") {
		t.Fatalf("expected message to include recordIndex, got %q", le.Message)
	}
	if le.FilePath == "" {
		t.Fatalf("expected FilePath to be set")
	}
	if le.RecordIndex < 0 {
		t.Fatalf("expected RecordIndex >= 0, got %d", le.RecordIndex)
	}
	if le.FieldPath == "" {
		t.Fatalf("expected FieldPath to be set")
	}
	if le.Ref == "" {
		t.Fatalf("expected Ref to be set")
	}
}

func TestApplyModule_ModuleCrossApplicationIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "base", "external_id": "x", "model": "auth.User", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindValidation {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindValidation, le.Kind)
	}
	if le.Code != LoadErrorCodeModuleCrossApplication {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeModuleCrossApplication, le.Code)
	}
	if le.RecordIndex != 0 {
		t.Fatalf("expected RecordIndex=0, got %d", le.RecordIndex)
	}
}

func TestApplyModule_ModuleNotInDependencyChainIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth_ext", "external_id": "x", "model": "auth.User", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFile(t, dir)

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Kind != LoadErrorKindValidation {
		t.Fatalf("expected Kind=%q, got %q", LoadErrorKindValidation, le.Kind)
	}
	if le.Code != LoadErrorCodeModuleNotInDependencyChain {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeModuleNotInDependencyChain, le.Code)
	}
	if le.RecordIndex != 0 {
		t.Fatalf("expected RecordIndex=0, got %d", le.RecordIndex)
	}
}

func TestApplyModule_CrossModuleSameApplicationAllowedWithDependency(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "x", "model": "auth.User", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFileNamed(t, dir, "auth_ext")

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestApplyModule_NoUpdateFreezesSubsequentUpdates(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()

	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "external_id": "g1", "model": "auth.group", "values": map[string]any{}},
			map[string]any{"module": "auth", "external_id": "g2", "model": "auth.group", "values": map[string]any{}},
			map[string]any{
				"module":      "auth",
				"external_id": "u",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.g1"},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error on first apply, got %T: %v", err, err)
	}

	var g1 metadata.IrModelData
	if err := db.Where("module = ? AND external_id = ?", "auth", "g1").First(&g1).Error; err != nil {
		t.Fatalf("lookup g1 mapping: %v", err)
	}
	var g2 metadata.IrModelData
	if err := db.Where("module = ? AND external_id = ?", "auth", "g2").First(&g2).Error; err != nil {
		t.Fatalf("lookup g2 mapping: %v", err)
	}
	var u metadata.IrModelData
	if err := db.Where("module = ? AND external_id = ?", "auth", "u").First(&u).Error; err != nil {
		t.Fatalf("lookup u mapping: %v", err)
	}
	if u.NoUpdate {
		t.Fatalf("expected u.NoUpdate=false initially")
	}

	var row testAuthUser
	if err := db.Where("id = ?", u.ResID).First(&row).Error; err != nil {
		t.Fatalf("lookup auth_user row: %v", err)
	}
	if row.GroupID != g1.ResID {
		t.Fatalf("expected initial group_id=%q, got %q", g1.ResID, row.GroupID)
	}

	// Second apply flips group_id to g2 and freezes the record (noupdate=true).
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "u",
				"model":       "auth.User",
				"noupdate":    true,
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.g2"},
				},
			},
		},
	})
	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error on second apply, got %T: %v", err, err)
	}
	if err := db.Where("module = ? AND external_id = ?", "auth", "u").First(&u).Error; err != nil {
		t.Fatalf("reload u mapping: %v", err)
	}
	if !u.NoUpdate {
		t.Fatalf("expected u.NoUpdate=true after second apply")
	}
	if err := db.Where("id = ?", u.ResID).First(&row).Error; err != nil {
		t.Fatalf("reload auth_user row: %v", err)
	}
	if row.GroupID != g2.ResID {
		t.Fatalf("expected updated group_id=%q, got %q", g2.ResID, row.GroupID)
	}

	// Third apply attempts to change group_id back to g1, but must be ignored because mapping.noupdate=true.
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "u",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.g1"},
				},
			},
		},
	})
	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error on third apply, got %T: %v", err, err)
	}
	if err := db.Where("id = ?", u.ResID).First(&row).Error; err != nil {
		t.Fatalf("reload auth_user row: %v", err)
	}
	if row.GroupID != g2.ResID {
		t.Fatalf("expected group_id to remain %q, got %q", g2.ResID, row.GroupID)
	}
}

func TestApplyModule_RefCanResolveExistingDBMappingOutsideBatch(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()

	// Seed a referenced record + model_data mapping directly in DB (not part of this batch).
	if err := db.Table("auth_group").Create(map[string]any{
		"id":         "pre_group",
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed auth_group: %v", err)
	}
	if err := db.Create(&metadata.IrModelData{Module: "auth", ExternalID: "pre_group", Model: "auth.group", ResID: "pre_group", NoUpdate: true}).Error; err != nil {
		t.Fatalf("seed meta_ir_model_data: %v", err)
	}

	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module":      "auth",
				"external_id": "u",
				"model":       "auth.User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.pre_group"},
				},
			},
		},
	})
	mod := moduleWithDataFile(t, dir)

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error, got %T: %v", err, err)
	}

	var u metadata.IrModelData
	if err := db.Where("module = ? AND external_id = ?", "auth", "u").First(&u).Error; err != nil {
		t.Fatalf("lookup u mapping: %v", err)
	}
	var row testAuthUser
	if err := db.Where("id = ?", u.ResID).First(&row).Error; err != nil {
		t.Fatalf("lookup auth_user row: %v", err)
	}
	if row.GroupID != "pre_group" {
		t.Fatalf("expected group_id=pre_group, got %q", row.GroupID)
	}
}
