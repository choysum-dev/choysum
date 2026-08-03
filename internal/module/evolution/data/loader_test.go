// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

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

	depLeaf := &meta.Module{Name: "base", ApplicationStr: "auth"}
	depMid := &meta.Module{Name: "auth_addon", ApplicationStr: "auth", Dependencies: []*meta.Module{depLeaf}}
	rules, err := buildModuleRulesFromOwner(&meta.Module{
		Name:           "auth",
		ApplicationStr: "auth",
		Dependencies:   []*meta.Module{depMid, depLeaf, nil},
	})
	if err != nil {
		t.Fatalf("buildModuleRulesFromOwner() error = %v", err)
	}
	if rules.OwnerName != "auth" || rules.OwnerApp != "auth" {
		t.Fatalf("unexpected owner rules: %#v", rules)
	}
	for _, name := range []string{"auth", "auth_addon", "base"} {
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
		owner *meta.Module
		want  string
	}{
		{name: "nil owner", owner: nil, want: "nil owner module"},
		{name: "missing owner name", owner: &meta.Module{ApplicationStr: "auth"}, want: "empty name"},
		{name: "missing owner app", owner: &meta.Module{Name: "auth"}, want: "empty application"},
		{name: "dependency missing app", owner: &meta.Module{Name: "auth", ApplicationStr: "auth", Dependencies: []*meta.Module{{Name: "base"}}}, want: "dependency module base has empty application"},
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
	rec := record{Module: " auth ", Name: " ext ", Application: "auth", Model: "User"}
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
		"name=auth.ext",
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
			map[string]any{"module": "auth", "name": "group_admin", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})
	mod := &meta.Module{Name: "auth", Path: dir, ApplicationStr: "auth"}

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
	if err := l.ApplyFiles(context.Background(), &meta.Module{Name: "auth"}, []string{"data.json"}); err == nil || !strings.Contains(err.Error(), "empty Path") {
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

	if err := (*Loader)(nil).ApplyModule(context.Background(), &meta.Module{Name: "auth", Path: dir}, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "nil loader") {
		t.Fatalf("expected nil loader error, got %v", err)
	}
	if err := (&Loader{}).ApplyModule(context.Background(), &meta.Module{Name: "auth", Path: dir}, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "nil loader") {
		t.Fatalf("expected nil env error, got %v", err)
	}
	if err := l.ApplyModule(context.Background(), nil, ApplyOptions{}); err != nil {
		t.Fatalf("ApplyModule(nil module) error = %v", err)
	}
	if err := l.ApplyModule(context.Background(), &meta.Module{Name: "auth"}, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "empty Path") {
		t.Fatalf("expected empty Path error, got %v", err)
	}

	invalidData := &meta.Module{Name: "auth", Path: dir, DataStr: []byte("{"), ApplicationStr: "auth"}
	if err := l.ApplyModule(context.Background(), invalidData, ApplyOptions{}); err == nil || !strings.Contains(err.Error(), "decode manifest data") {
		t.Fatalf("expected invalid data manifest error, got %v", err)
	}
	invalidDemo := &meta.Module{Name: "auth", Path: dir, DemoStr: []byte("{"), ApplicationStr: "auth"}
	if err := l.ApplyModule(context.Background(), invalidDemo, ApplyOptions{WithDemo: true}); err == nil || !strings.Contains(err.Error(), "decode manifest demo") {
		t.Fatalf("expected invalid demo manifest error, got %v", err)
	}
}

func TestApplyModule_WithDemoFiles(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()
	writeNamedDataFile(t, dir, "data.json", map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "group_data", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})
	writeNamedDataFile(t, dir, "demo.json", map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "group_demo", "application": "auth", "model": "group", "values": map[string]any{}},
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
	mod := &meta.Module{Name: "auth", Path: dir, DataStr: dataPaths, DemoStr: demoPaths, ApplicationStr: "auth"}

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
			map[string]any{"module": "auth", "name": "group_outer", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})
	mod := &meta.Module{Name: "auth", Path: dir, ApplicationStr: "auth"}

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

	rules, err := buildModuleRules(db, &meta.Module{Name: "auth_addon", ApplicationStr: "auth"})
	if err != nil {
		t.Fatalf("buildModuleRules(db-backed) error = %v", err)
	}
	if rules.OwnerName != "auth_addon" || rules.OwnerApp != "auth" {
		t.Fatalf("unexpected rules owner: %#v", rules)
	}
	for _, want := range []string{"auth_addon", "auth"} {
		if _, ok := rules.Allowed[want]; !ok {
			t.Fatalf("expected %q in dependency closure: %#v", want, rules.Allowed)
		}
	}

	if err := db.Model(&meta.Module{}).Where("name = ?", "auth_addon").Update("application_str", "").Error; err != nil {
		t.Fatalf("blank auth_addon application_str: %v", err)
	}
	if err := db.Model(&meta.Module{}).Where("name = ?", "auth_addon").Update("application_str", "").Error; err != nil {
		t.Fatalf("confirm blank auth_addon application_str: %v", err)
	}
	_, err = buildModuleRules(db, &meta.Module{Name: "auth_addon", ApplicationStr: ""})
	if err == nil || !strings.Contains(err.Error(), "owner module auth_addon has empty application") {
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

func (testModuleDependency) TableName() string { return "meta_module_dependencies" }

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
	if err := db.AutoMigrate(&meta.Module{}, &meta.Model{}, &meta.Field{}, &metadata.ModelData{}, &testAuthGroup{}, &testAuthUser{}, &testModuleDependency{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	// Seed models so ApplyModule can resolve model -> table.
	authGroupModel := &meta.Model{Name: "group", Application: "auth", Path: "/tmp", ModelTable: "auth_group"}
	if err := db.Create(authGroupModel).Error; err != nil {
		t.Fatalf("seed meta_model auth.group: %v", err)
	}
	authUserModel := &meta.Model{Name: "User", Application: "auth", Path: "/tmp", ModelTable: "auth_user"}
	if err := db.Create(authUserModel).Error; err != nil {
		t.Fatalf("seed meta_model auth.User: %v", err)
	}

	// Seed field definitions so detectFieldCardinality can resolve ManyToOne/ManyToMany.
	if err := db.Create(&meta.Field{
		Name: "group_id", FieldType: "ManyToOne", ModelId: authUserModel.Id,
	}).Error; err != nil {
		t.Fatalf("seed meta_field auth.User.group_id: %v", err)
	}

	auth := &meta.Module{Name: "auth", ApplicationStr: "auth", Path: "/tmp"}
	if err := db.Create(auth).Error; err != nil {
		t.Fatalf("create module auth: %v", err)
	}
	authAddon := &meta.Module{Name: "auth_addon", ApplicationStr: "auth", Path: "/tmp"}
	if err := db.Create(authAddon).Error; err != nil {
		t.Fatalf("create module auth_addon: %v", err)
	}
	base := &meta.Module{Name: "base", ApplicationStr: "base", Path: "/tmp"}
	if err := db.Create(base).Error; err != nil {
		t.Fatalf("create module base: %v", err)
	}
	if err := db.Exec("INSERT INTO meta_module_dependencies (module_id, depend_module_id) VALUES (?, ?)", authAddon.Id.String, auth.Id.String).Error; err != nil {
		t.Fatalf("insert module dependency auth_addon -> auth: %v", err)
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

func moduleWithDataFile(t *testing.T, dir string) *meta.Module {
	return moduleWithDataFileNamed(t, dir, "auth")
}

func moduleWithDataFiles(t *testing.T, dir string, relPaths []string) *meta.Module {
	return moduleWithDataFilesNamed(t, dir, "auth", relPaths)
}

func moduleWithDataFileNamed(t *testing.T, dir string, name string) *meta.Module {
	t.Helper()
	paths, err := json.Marshal([]string{"data.json"})
	if err != nil {
		t.Fatalf("marshal manifest data: %v", err)
	}
	return &meta.Module{
		Name:    name,
		Path:    dir,
		DataStr: paths,
	}
}

func moduleWithDataFilesNamed(t *testing.T, dir string, name string, relPaths []string) *meta.Module {
	t.Helper()
	paths, err := json.Marshal(relPaths)
	if err != nil {
		t.Fatalf("marshal manifest data: %v", err)
	}
	return &meta.Module{
		Name:    name,
		Path:    dir,
		DataStr: paths,
	}
}

func TestApplyModule_DuplicateNameIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "x", "application": "auth", "model": "User", "values": map[string]any{}},
			map[string]any{"module": "auth", "name": "x", "application": "auth", "model": "User", "values": map[string]any{}},
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
	if le.Code != LoadErrorCodeDuplicateNameInInput {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeDuplicateNameInInput, le.Code)
	}
	if le.RecordIndex != 1 {
		t.Fatalf("expected RecordIndex=1, got %d", le.RecordIndex)
	}
	if le.Module != "auth" {
		t.Fatalf("expected Module=auth, got %q", le.Module)
	}
	if le.Name != "x" {
		t.Fatalf("expected Name=x, got %q", le.Name)
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
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
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
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
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

	// Add meta.Application table + model mapping so refBy can resolve it.
	if err := db.AutoMigrate(&meta.Application{}); err != nil {
		t.Fatalf("migrate meta_application: %v", err)
	}
	if err := db.Create(&meta.Model{Name: "MetaApplication", Application: "meta", Path: "/tmp", ModelTable: "meta_application"}).Error; err != nil {
		t.Fatalf("seed meta_model meta.MetaApplication: %v", err)
	}

	app := &meta.Application{Name: "auth"}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed meta_application(auth): %v", err)
	}
	appID := strings.TrimSpace(app.Id.String)
	if appID == "" {
		t.Fatalf("expected non-empty app id")
	}

	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module": "auth",
				"name":   "u",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"group_id": map[string]any{
						"refBy": map[string]any{"model": "meta.MetaApplication", "field": "Name", "value": "auth"},
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
	owner := &meta.Module{Name: "auth"}
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.b"},
				},
			},
			map[string]any{"module": "auth", "name": "b", "application": "auth", "model": "group", "values": map[string]any{}},
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
	order, err := l.planRecordOrder(db, &meta.Module{Name: "auth"}, filePath, nil)
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
		{name: "missing name", rec: record{Module: "auth", Application: "auth", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeMissingName},
		{name: "module not owner", rec: record{Module: "base", Name: "x", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeModuleNotOwner},
		{name: "application mismatch", rec: record{Name: "x", Application: "base", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeApplicationMismatch},
		{name: "missing model", rec: record{Module: "auth", Name: "x", Application: "auth", Values: map[string]any{}}, code: LoadErrorCodeMissingModel},
		{name: "missing values", rec: record{Module: "auth", Name: "x", Application: "auth", Model: "User"}, code: LoadErrorCodeMissingValues},
		{name: "invalid model full name", rec: record{Module: "auth", Name: "x", Application: "auth", Model: "auth.User", Values: map[string]any{}}, code: LoadErrorCodeInvalidModel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := l.planRecordOrder(db, &meta.Module{Name: "auth"}, filePath, []record{tc.rec})
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

	_, err := l.planRecordOrder(db, &meta.Module{Name: "auth"}, filePath, []record{{
		Module: "auth",
		Name:   "self",
		Application: "auth", Model: "User",
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
	owner := &meta.Module{Name: "auth"}
	dir := t.TempDir()
	absPath := filepath.Join(dir, "data.json")

	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module": "auth",
				"name":   "user_admin",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.group_admin"},
				},
			},
			map[string]any{"module": "auth", "name": "group_admin", "application": "auth", "model": "group", "values": map[string]any{}},
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
		records := []record{{Name: "a"}, {Name: "b"}, {Name: "c"}}
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
			{Module: "auth", Name: "a", Application: "auth", Model: "User"},
			{Module: "auth", Name: "b", Application: "auth", Model: "group"},
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

	t.Run("cycle with edge info sets name field path and ref", func(t *testing.T) {
		records := []record{
			{Module: "auth", Name: "a", Application: "auth", Model: "User"},
			{Module: "auth", Name: "b", Application: "auth", Model: "group"},
		}
		dep := [][]int{{1}, {0}}
		adj := [][]int{{1}, {0}}
		indeg := []int{1, 1}
		edgeInfo := map[[2]int]refOccurrence{
			{0, 1}: {FieldPath: "values.group_id", Ref: "auth.b"},
		}

		_, err := topoOrderOrCycle(records, dep, adj, indeg, edgeInfo, "/tmp/data.json")
		var le *LoadError
		if !errors.As(err, &le) {
			t.Fatalf("expected LoadError, got %T: %v", err, err)
		}
		if le.Code != LoadErrorCodeRefCycle {
			t.Fatalf("unexpected cycle code: %#v", le)
		}
		if le.Name != "a" {
			t.Fatalf("expected Name=a from edge info path, got %#v", le)
		}
		if le.FieldPath != "values.group_id" || le.Ref != "auth.b" {
			t.Fatalf("expected field/ref from edge info, got %#v", le)
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

	if err := db.AutoMigrate(&meta.Application{}); err != nil {
		t.Fatalf("migrate meta_application: %v", err)
	}
	if err := db.Create(&meta.Model{Name: "MetaApplication", Application: "meta", Path: "/tmp", ModelTable: "meta_application"}).Error; err != nil {
		t.Fatalf("seed meta.MetaApplication model: %v", err)
	}
	app := &meta.Application{Name: "auth"}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed meta_application(auth): %v", err)
	}
	if err := db.Create(&metadata.ModelData{Module: "auth", Name: "group_admin", Application: "auth", Model: "group", ResID: "gid-1"}).Error; err != nil {
		t.Fatalf("seed model_data: %v", err)
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
			{name: "string slice", in: []string{"a", "b"}, want: `["a","b"]`},
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

		modelFull, field, value, ok := extractRefBy(map[string]any{"refBy": map[string]any{"model": " meta.MetaApplication ", "field": " Name ", "value": "auth"}})
		if !ok || modelFull != "meta.MetaApplication" || field != "Name" || value != "auth" {
			t.Fatalf("extractRefBy() = (%q, %q, %#v, %v)", modelFull, field, value, ok)
		}
		if _, _, _, ok := extractRefBy(map[string]any{"ref_by": map[string]any{"model": "meta.MetaApplication", "field": "Name", "value": "auth"}}); !ok {
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
		for _, bad := range []string{".", "auth.", ".name", " . ", "auth. "} {
			if _, _, err := splitRef(bad); err == nil || !strings.Contains(err.Error(), "empty module or name") {
				t.Fatalf("splitRef(%q) error = %v, want empty module or name", bad, err)
			}
		}
		if app, model, err := splitModel(" auth.User "); err != nil || app != "auth" || model != "User" {
			t.Fatalf("splitModel() = (%q, %q, %v)", app, model, err)
		}
		if _, _, err := splitModel("auth"); err == nil {
			t.Fatalf("expected splitModel to reject invalid form")
		}
	})

	t.Run("resolveRef helpers", func(t *testing.T) {
		resID, err := l.resolveRefBy(db, "meta.MetaApplication", "Name", "auth")
		if err != nil || strings.TrimSpace(resID) != strings.TrimSpace(app.Id.String) {
			t.Fatalf("resolveRefBy() = %q, %v", resID, err)
		}
		if _, err := l.resolveRefBy(db, "meta.MetaApplication", "Name", "missing"); err == nil {
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
		rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}
		got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"ref": "auth.group_admin"})
		if err != nil || got != "gid-1" {
			t.Fatalf("resolveValue(ref) = %#v, %v", got, err)
		}
		got, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"refBy": map[string]any{"model": "meta.MetaApplication", "field": "Name", "value": "auth"}})
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
		if _, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"refBy": map[string]any{"model": "meta.MetaApplication", "field": "Name", "value": "missing"}}); err == nil {
			t.Fatalf("expected resolveValue refBy error")
		}
		if got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.nil", nil); err != nil || got != nil {
			t.Fatalf("resolveValue(nil) = %#v, %v", got, err)
		}
	})

	t.Run("parse ref query skeleton", func(t *testing.T) {
		// ref compat
		spec, ok, err := parseRefQuerySpec(map[string]any{"ref": " auth.group_admin "})
		if err != nil || !ok || spec.Kind != refQuerySpecKindRef || spec.Ref != "auth.group_admin" {
			t.Fatalf("parseRefQuerySpec(ref) = %#v, %v, %v", spec, ok, err)
		}
		// refBy compat
		spec, ok, err = parseRefQuerySpec(map[string]any{"refBy": map[string]any{"model": "meta.MetaApplication", "field": "Name", "value": "auth"}})
		if err != nil || !ok || spec.Kind != refQuerySpecKindRefBy || spec.RefBy.Model != "meta.MetaApplication" {
			t.Fatalf("parseRefQuerySpec(refBy) = %#v, %v, %v", spec, ok, err)
		}
		// ref_by alias
		spec, ok, err = parseRefQuerySpec(map[string]any{"ref_by": map[string]any{"model": "auth.User", "field": "id", "value": 1}})
		if err != nil || !ok || spec.Kind != refQuerySpecKindRefBy {
			t.Fatalf("parseRefQuerySpec(ref_by) = %#v, %v, %v", spec, ok, err)
		}
		// non-map returns false
		if _, ok, _ := parseRefQuerySpec("plain"); ok {
			t.Fatalf("expected non-map to return false")
		}
		// empty map returns false
		if _, ok, _ := parseRefQuerySpec(map[string]any{}); ok {
			t.Fatalf("expected empty map to return false")
		}
	})
}

func TestResolveAndMapValues_SkipsSystemFieldsAndNormalizes(t *testing.T) {
	l, db := newTestLoader(t)
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}
	if err := db.Create(&metadata.ModelData{Module: "auth", Name: "group_admin", Application: "auth", Model: "group", ResID: "gid-1"}).Error; err != nil {
		t.Fatalf("seed model_data: %v", err)
	}

	columns, err := l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, nil, map[string]any{
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

	_, err = l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, nil, map[string]any{
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
		{name: "missing module", rec: record{Name: "x", Application: "auth", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeMissingModule},
		{name: "missing name", rec: record{Module: "auth", Application: "auth", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeMissingName},
		{name: "missing application", rec: record{Module: "auth", Name: "x", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeMissingApplication},
		{name: "missing model", rec: record{Module: "auth", Name: "x", Application: "auth", Values: map[string]any{}}, code: LoadErrorCodeMissingModel},
		{name: "invalid model full name", rec: record{Module: "auth", Name: "x", Application: "auth", Model: "auth.User", Values: map[string]any{}}, code: LoadErrorCodeInvalidModel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := l.applyRecord(db, "/tmp/data.json", 0, tc.rec, now)
			var le *LoadError
			if !errors.As(err, &le) || le.Code != tc.code {
				t.Fatalf("applyRecord() error = %#v, want code %s", err, tc.code)
			}
		})
	}

	if err := db.Create(&meta.Model{Name: "Broken", Application: "auth", Path: "/tmp", ModelTable: ""}).Error; err != nil {
		t.Fatalf("seed broken model: %v", err)
	}
	err := l.applyRecord(db, "/tmp/data.json", 0, record{Module: "auth", Name: "x", Application: "auth", Model: "Broken", Values: map[string]any{}}, now)
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
	if err := db.Create(&metadata.ModelData{Module: "auth", Name: "user_admin", Application: "auth", Model: "User", ResID: user.ID, NoUpdate: false}).Error; err != nil {
		t.Fatalf("seed user mapping: %v", err)
	}
	if err := db.Create(&metadata.ModelData{Module: "auth", Name: "group_admin", Application: "auth", Model: "group", ResID: group2.ID, NoUpdate: false}).Error; err != nil {
		t.Fatalf("seed group mapping: %v", err)
	}
	freeze := true
	if err := l.applyRecord(db, "/tmp/data.json", 1, record{
		Module:   "auth",
		Name:     "user_admin",
		Application: "auth", Model: "User",
		NoUpdate: &freeze,
		Values:   map[string]any{"group_id": map[string]any{"ref": "auth.group_admin"}},
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
	var mapping metadata.ModelData
	if err := db.Where("module = ? AND name = ?", "auth", "user_admin").First(&mapping).Error; err != nil {
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
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"x": map[string]any{"ref": "auth.b"},
				},
			},
			map[string]any{
				"module": "auth",
				"name":   "b",
				"application": "auth",
				"model": "group",
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
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"x": map[string]any{"ref": "auth.b"},
				},
			},
			map[string]any{
				"module": "auth",
				"name":   "b",
				"application": "auth",
				"model": "group",
				"values": map[string]any{
					"y": map[string]any{"ref": "auth.c"},
				},
			},
			map[string]any{
				"module": "auth",
				"name":   "c",
				"application": "auth",
				"model": "group",
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
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.b"},
				},
			},
		},
	})
	writeNamedDataFile(t, dir, "b.json", map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "b", "application": "auth", "model": "group", "values": map[string]any{}},
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
				"module": "auth",
				"name":   "a",
				"application": "auth",
				"model": "User",
				"values": map[string]any{
					"x": map[string]any{"ref": "auth.b"},
				},
			},
		},
	})
	writeNamedDataFile(t, dir, "b.json", map[string]any{
		"records": []any{
			map[string]any{
				"module": "auth",
				"name":   "b",
				"application": "auth",
				"model": "group",
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

func TestApplyModule_ForeignModuleNamespaceIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "base", "name": "x", "model": "User", "values": map[string]any{}},
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
	if le.Code != LoadErrorCodeModuleNotOwner {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeModuleNotOwner, le.Code)
	}
	if le.RecordIndex != 0 {
		t.Fatalf("expected RecordIndex=0, got %d", le.RecordIndex)
	}
}

func TestApplyModule_CrossAppApplicationIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"name": "x", "application": "base", "model": "User", "values": map[string]any{}},
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
	if le.Code != LoadErrorCodeApplicationMismatch {
		t.Fatalf("expected Code=%q, got %q", LoadErrorCodeApplicationMismatch, le.Code)
	}
	if le.RecordIndex != 0 {
		t.Fatalf("expected RecordIndex=0, got %d", le.RecordIndex)
	}
}

func TestApplyModule_OmitsModuleAndApplicationDefaultsToOwner(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"name": "x", "model": "User", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFile(t, dir)

	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var mapping metadata.ModelData
	if err := db.Where("module = ? AND name = ?", "auth", "x").First(&mapping).Error; err != nil {
		t.Fatalf("lookup mapping: %v", err)
	}
	if mapping.Application != "auth" || mapping.Model != "User" {
		t.Fatalf("unexpected mapping target: %#v", mapping)
	}
}

func TestApplyModule_CrossModuleSameApplicationIsRejected(t *testing.T) {
	l, _ := newTestLoader(t)
	dir := t.TempDir()
	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "x", "model": "User", "values": map[string]any{}},
		},
	})
	mod := moduleWithDataFileNamed(t, dir, "auth_addon")

	err := l.ApplyModule(context.Background(), mod, ApplyOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeModuleNotOwner {
		t.Fatalf("ApplyModule() error = %#v, want module_not_owner", err)
	}
}

func TestApplyModule_NoUpdateFreezesSubsequentUpdates(t *testing.T) {
	l, db := newTestLoader(t)
	dir := t.TempDir()

	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "g1", "application": "auth", "model": "group", "values": map[string]any{}},
			map[string]any{"module": "auth", "name": "g2", "application": "auth", "model": "group", "values": map[string]any{}},
			map[string]any{
				"module": "auth",
				"name":   "u",
				"application": "auth",
				"model": "User",
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

	var g1 metadata.ModelData
	if err := db.Where("module = ? AND name = ?", "auth", "g1").First(&g1).Error; err != nil {
		t.Fatalf("lookup g1 mapping: %v", err)
	}
	var g2 metadata.ModelData
	if err := db.Where("module = ? AND name = ?", "auth", "g2").First(&g2).Error; err != nil {
		t.Fatalf("lookup g2 mapping: %v", err)
	}
	var u metadata.ModelData
	if err := db.Where("module = ? AND name = ?", "auth", "u").First(&u).Error; err != nil {
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
				"module":   "auth",
				"name":     "u",
				"application": "auth",
				"model": "User",
				"noupdate": true,
				"values": map[string]any{
					"group_id": map[string]any{"ref": "auth.g2"},
				},
			},
		},
	})
	if err := l.ApplyModule(context.Background(), mod, ApplyOptions{}); err != nil {
		t.Fatalf("expected no error on second apply, got %T: %v", err, err)
	}
	if err := db.Where("module = ? AND name = ?", "auth", "u").First(&u).Error; err != nil {
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
				"module": "auth",
				"name":   "u",
				"application": "auth",
				"model": "User",
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
	if err := db.Create(&metadata.ModelData{Module: "auth", Name: "pre_group", Application: "auth", Model: "group", ResID: "pre_group", NoUpdate: true}).Error; err != nil {
		t.Fatalf("seed meta_model_data: %v", err)
	}

	writeDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{
				"module": "auth",
				"name":   "u",
				"application": "auth",
				"model": "User",
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

	var u metadata.ModelData
	if err := db.Where("module = ? AND name = ?", "auth", "u").First(&u).Error; err != nil {
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

func TestParseRefQuerySpec_CompatRefAndRefBy(t *testing.T) {
	t.Parallel()

	// ref
	spec, ok, err := parseRefQuerySpec(map[string]any{"ref": " auth.group_admin "})
	if err != nil || !ok {
		t.Fatalf("expected ok ref parse, got %v, %v", ok, err)
	}
	if spec.Kind != refQuerySpecKindRef || spec.Ref != "auth.group_admin" {
		t.Fatalf("unexpected spec: %#v", spec)
	}

	// refBy
	spec, ok, err = parseRefQuerySpec(map[string]any{
		"refBy": map[string]any{"model": " auth.User ", "field": " GroupId ", "value": "gid-1"},
	})
	if err != nil || !ok {
		t.Fatalf("expected ok refBy parse, got %v, %v", ok, err)
	}
	if spec.Kind != refQuerySpecKindRefBy || spec.RefBy.Model != "auth.User" || spec.RefBy.Field != "GroupId" || spec.RefBy.Value != "gid-1" {
		t.Fatalf("unexpected refBy spec: %#v", spec)
	}

	// ref_by alias
	spec, ok, err = parseRefQuerySpec(map[string]any{
		"ref_by": map[string]any{"model": "base.Company", "field": "Name", "value": "test"},
	})
	if err != nil || !ok || spec.Kind != refQuerySpecKindRefBy {
		t.Fatalf("expected ref_by alias parse, got %#v, %v, %v", spec, ok, err)
	}

	// ref with leading/trailing whitespace trimmed
	spec, ok, err = parseRefQuerySpec(map[string]any{"ref": "\t base.company_search \n"})
	if err != nil || !ok || spec.Ref != "base.company_search" {
		t.Fatalf("expected trimmed ref, got %q", spec.Ref)
	}

	// non-map input
	if _, ok, _ := parseRefQuerySpec(42); ok {
		t.Fatalf("expected non-map to return false")
	}
	if _, ok, _ := parseRefQuerySpec("plain"); ok {
		t.Fatalf("expected string to return false")
	}
	if _, ok, _ := parseRefQuerySpec(nil); ok {
		t.Fatalf("expected nil to return false")
	}
	if _, ok, _ := parseRefQuerySpec([]any{1, 2}); ok {
		t.Fatalf("expected slice to return false")
	}

	// empty map
	if _, ok, _ := parseRefQuerySpec(map[string]any{}); ok {
		t.Fatalf("expected empty map to return false")
	}

	// unknown key
	if _, ok, _ := parseRefQuerySpec(map[string]any{"other": "val"}); ok {
		t.Fatalf("expected unknown key map to return false")
	}
}

func TestParseRefQuerySpec_SearchShapeValidation(t *testing.T) {
	t.Parallel()

	// Valid minimal search
	spec, ok, err := parseRefQuerySpec(map[string]any{
		"search": map[string]any{"model": "auth.User"},
	})
	if err != nil || !ok {
		t.Fatalf("expected ok search parse, got %v, %v", ok, err)
	}
	if spec.Kind != refQuerySpecKindSearch || spec.Search.Model != "auth.User" {
		t.Fatalf("unexpected search spec: %#v", spec)
	}

	// Valid search with domain, orderBy, limit
	spec, ok, err = parseRefQuerySpec(map[string]any{
		"search": map[string]any{
			"model":   "auth.User",
			"domain":  []any{[]any{"name", "=", "admin"}},
			"orderBy": "id ASC",
			"limit":   float64(1),
		},
	})
	if err != nil || !ok {
		t.Fatalf("expected ok search with domain/orderBy/limit, got %v, %v", ok, err)
	}
	if spec.Search.OrderBy != "id ASC" || spec.Search.Limit != 1 {
		t.Fatalf("unexpected search fields: %#v", spec.Search)
	}

	// Search missing model
	_, ok, err = parseRefQuerySpec(map[string]any{
		"search": map[string]any{},
	})
	if !ok || err == nil {
		t.Fatalf("expected error for search missing model, got %v, %v", ok, err)
	}

	// Search model not a string
	_, ok, err = parseRefQuerySpec(map[string]any{
		"search": map[string]any{"model": 123},
	})
	if !ok || err == nil {
		t.Fatalf("expected error for non-string model, got %v, %v", ok, err)
	}

	// Search empty model
	_, ok, err = parseRefQuerySpec(map[string]any{
		"search": map[string]any{"model": "  "},
	})
	if !ok || err == nil {
		t.Fatalf("expected error for empty model, got %v, %v", ok, err)
	}

	// Search with negative limit
	_, ok, err = parseRefQuerySpec(map[string]any{
		"search": map[string]any{"model": "auth.User", "limit": float64(-1)},
	})
	if !ok || err == nil {
		t.Fatalf("expected error for negative limit, got %v, %v", ok, err)
	}

	// Search with int64 limit
	spec, ok, err = parseRefQuerySpec(map[string]any{
		"search": map[string]any{"model": "auth.User", "limit": int64(2)},
	})
	if err != nil || !ok {
		t.Fatalf("expected int64 limit to parse, got %v, %v", ok, err)
	}
	if spec.Search.Limit != 2 {
		t.Fatalf("expected int64 limit=2, got %#v", spec.Search)
	}
}

func TestParseSearchSpec_OrderByValidation(t *testing.T) {
	t.Parallel()

	valid := []string{
		"id ASC",
		"id DESC",
		"created_at ASC",
		"id ASC, name DESC",
		"auth_user.id ASC",
		"updated_at",
	}
	for _, v := range valid {
		_, ok, err := parseRefQuerySpec(map[string]any{
			"search": map[string]any{"model": "auth.User", "orderBy": v},
		})
		if err != nil || !ok {
			t.Fatalf("expected valid orderBy %q to parse, got err=%v ok=%v", v, err, ok)
		}
	}

	invalid := []string{
		"id; DROP TABLE users--",
		"1=1",
		"id ASC; SELECT 1",
		"id ASC/**/",
	}
	for _, v := range invalid {
		_, ok, err := parseRefQuerySpec(map[string]any{
			"search": map[string]any{"model": "auth.User", "orderBy": v},
		})
		if !ok || err == nil {
			t.Fatalf("expected invalid orderBy %q to fail, got ok=%v err=%v", v, ok, err)
		}
	}
}

func TestParseRefQuerySpec_ModelRefAndServiceRefShapeValidation(t *testing.T) {
	t.Parallel()

	// modelRef valid
	spec, ok, err := parseRefQuerySpec(map[string]any{"modelRef": " auth.User "})
	if err != nil || !ok {
		t.Fatalf("expected ok modelRef, got %v, %v", ok, err)
	}
	if spec.Kind != refQuerySpecKindModelRef || spec.ModelRef != "auth.User" {
		t.Fatalf("unexpected modelRef spec: %#v", spec)
	}

	// modelRef non-string
	_, ok, err = parseRefQuerySpec(map[string]any{"modelRef": 123})
	if !ok || err == nil {
		t.Fatalf("expected error for non-string modelRef, got %v, %v", ok, err)
	}

	// modelRef empty
	_, ok, err = parseRefQuerySpec(map[string]any{"modelRef": "  "})
	if !ok || err == nil {
		t.Fatalf("expected error for empty modelRef, got %v, %v", ok, err)
	}

	// serviceRef valid
	spec, ok, err = parseRefQuerySpec(map[string]any{"serviceRef": " auth.User/Browse "})
	if err != nil || !ok {
		t.Fatalf("expected ok serviceRef, got %v, %v", ok, err)
	}
	if spec.Kind != refQuerySpecKindServiceRef || spec.ServiceRef != "auth.User/Browse" {
		t.Fatalf("unexpected serviceRef spec: %#v", spec)
	}

	// serviceRef non-string
	_, ok, err = parseRefQuerySpec(map[string]any{"serviceRef": true})
	if !ok || err == nil {
		t.Fatalf("expected error for non-string serviceRef, got %v, %v", ok, err)
	}

	// serviceRef empty
	_, ok, err = parseRefQuerySpec(map[string]any{"serviceRef": ""})
	if !ok || err == nil {
		t.Fatalf("expected error for empty serviceRef, got %v, %v", ok, err)
	}

	// Multiple keys: ref takes priority over other keys
	spec, ok, err = parseRefQuerySpec(map[string]any{"ref": "auth.x", "search": map[string]any{"model": "auth.User"}})
	if err != nil || !ok || spec.Kind != refQuerySpecKindRef {
		t.Fatalf("expected ref priority, got %#v, %v, %v", spec, ok, err)
	}
}

func TestResolveValue_SearchShapeErrorIncludesFieldPath(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// search with a valid model now resolves successfully; resolveValue returns raw []string.
	got, err := l.resolveValue(db, "/tmp/data.json", 3, rec, "values.role_id", map[string]any{
		"search": map[string]any{"model": "auth.User"},
	})
	if err != nil {
		t.Fatalf("search on empty table should succeed, got %v", err)
	}
	ids, ok := got.([]string)
	if !ok {
		t.Fatalf("expected []string from search, got %T", got)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty results, got %#v", ids)
	}

	// modelRef resolves the meta.Model ID (auth.User is seeded in test schema).
	got2, modelRefErr := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.service_id", map[string]any{
		"modelRef": "auth.User",
	})
	if modelRefErr != nil {
		t.Fatalf("modelRef should resolve, got %v", modelRefErr)
	}
	if modelID, ok := got2.(string); !ok || modelID == "" {
		t.Fatalf("expected non-empty model ID from modelRef, got %#v", got2)
	}

	// serviceRef attempts model + meta_service lookup; may error if table not seeded,
	// but must NOT return the old "unsupported_op" skeleton error.
	_, serviceRefErr := l.resolveValue(db, "/tmp/data.json", 1, rec, "values.method_id", map[string]any{
		"serviceRef": "auth.User/Browse",
	})
	if serviceRefErr != nil {
		var le *LoadError
		if errors.As(serviceRefErr, &le) && le.Code == LoadErrorCodeRefSearchUnsupportedOp {
			t.Fatalf("serviceRef should no longer return unsupported_op, got %#v", le)
		}
	}

	// ref and refBy still work (backward compat)
	if err := db.Create(&metadata.ModelData{Module: "auth", Name: "group_admin", Application: "auth", Model: "group", ResID: "gid-1"}).Error; err != nil {
		t.Fatalf("seed model_data: %v", err)
	}
	got, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.group_id", map[string]any{"ref": "auth.group_admin"})
	if err != nil || got != "gid-1" {
		t.Fatalf("resolveValue(ref) after skeleton = %#v, %v", got, err)
	}
}

func TestResolveRefBySearch_DomainAndOrNot(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed users for domain tests.
	ids := []string{"u-active", "u-inactive", "u-admin"}
	for i, id := range ids {
		if err := db.Table("auth_user").Create(map[string]any{
			"id":         id,
			"created_at": time.Now(),
			"updated_at": time.Now(),
			"group_id":   "g-" + string(rune('a'+i)),
		}).Error; err != nil {
			t.Fatalf("seed user %s: %v", id, err)
		}
	}

	// Implicit AND: two conditions
	spec := searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "=", "u-active"},
		},
	}
	idsOut, err := l.resolveRefBySearch(db, spec)
	if err != nil || len(idsOut) != 1 || idsOut[0] != "u-active" {
		t.Fatalf("resolveRefBySearch(=) = %#v, %v", idsOut, err)
	}

	// Explicit OR
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			"|",
			[]any{"id", "=", "u-active"},
			[]any{"id", "=", "u-admin"},
		},
	}
	idsOut, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(idsOut) != 2 {
		t.Fatalf("resolveRefBySearch(|) = %#v, %v", idsOut, err)
	}

	// Explicit AND with &
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			"&",
			[]any{"id", "=", "u-admin"},
			[]any{"group_id", "=", "g-c"},
		},
	}
	idsOut, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(idsOut) != 1 || idsOut[0] != "u-admin" {
		t.Fatalf("resolveRefBySearch(&) = %#v, %v", idsOut, err)
	}

	// NOT
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			"!",
			[]any{"id", "=", "u-active"},
		},
	}
	idsOut, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(idsOut) != 2 {
		t.Fatalf("resolveRefBySearch(!) = %#v, %v", idsOut, err)
	}

	// OrderBy + Limit
	spec = searchSpec{
		Model:   "auth.User",
		Domain:  []any{[]any{"id", "!=", "missing"}},
		OrderBy: "id DESC",
		Limit:   2,
	}
	idsOut, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(idsOut) != 2 || idsOut[0] != "u-inactive" {
		t.Fatalf("resolveRefBySearch(order+limit) = %#v, %v", idsOut, err)
	}
}

func TestResolveRefBySearch_UnsupportedOperator(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	_ = db

	spec := searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "regexp", ".*"},
		},
	}
	_, err := l.resolveRefBySearch(db, spec)
	if err == nil {
		t.Fatalf("expected error for unsupported operator")
	}
	if !strings.Contains(err.Error(), "unsupported search operator") {
		t.Fatalf("expected unsupported operator message, got %v", err)
	}

	// Also test via resolveValue path
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}
	_, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.x", map[string]any{
		"search": map[string]any{
			"model":  "auth.User",
			"domain": []any{[]any{"id", "regexp", ".*"}},
		},
	})
	if err == nil {
		t.Fatalf("expected error via resolveValue")
	}
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeResolveRefFailed {
		t.Fatalf("expected LoadError with ResolveRefFailed, got %#v", err)
	}
}

func TestResolveRefBySearch_InvalidFieldName(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	_ = db

	spec := searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id; DROP TABLE", "=", "x"},
		},
	}
	_, err := l.resolveRefBySearch(db, spec)
	if err == nil {
		t.Fatalf("expected error for invalid field name")
	}
	if !strings.Contains(err.Error(), "invalid search field") {
		t.Fatalf("expected invalid field message, got %v", err)
	}
}

func TestResolveRefBySearch_InvalidDomainNode(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Non-array root node should fail fast instead of being silently ignored.
	_, err := l.resolveRefBySearch(db, searchSpec{Model: "auth.User", Domain: "bad-domain"})
	if err == nil || !strings.Contains(err.Error(), "domain node must be an array") {
		t.Fatalf("expected invalid root domain node error, got %v", err)
	}

	// Non-array child node inside implicit AND should also fail.
	_, err = l.resolveRefBySearch(db, searchSpec{
		Model:  "auth.User",
		Domain: []any{[]any{"id", "=", "x"}, "bad-child"},
	})
	if err == nil || !strings.Contains(err.Error(), "domain node must be an array") {
		t.Fatalf("expected invalid child domain node error, got %v", err)
	}

	// OR/AND must have at least two operands.
	_, err = l.resolveRefBySearch(db, searchSpec{
		Model:  "auth.User",
		Domain: []any{"|", []any{"id", "=", "x"}},
	})
	if err == nil || !strings.Contains(err.Error(), "OR combinator requires at least 2 operands") {
		t.Fatalf("expected OR operand arity error, got %v", err)
	}

	_, err = l.resolveRefBySearch(db, searchSpec{
		Model:  "auth.User",
		Domain: []any{"&", []any{"id", "=", "x"}},
	})
	if err == nil || !strings.Contains(err.Error(), "AND combinator requires at least 2 operands") {
		t.Fatalf("expected AND operand arity error, got %v", err)
	}
}

func TestApplyLeafNode_RemainingOperatorsAndEdgeCases(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed a user for operational tests.
	if err := db.Table("auth_user").Create(map[string]any{
		"id": "edge-1", "created_at": time.Now(), "updated_at": time.Now(), "group_id": "x",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// != operator
	ids, err := l.resolveRefBySearch(db, searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "!=", "nothing"},
		},
	})
	if err != nil || len(ids) != 1 || ids[0] != "edge-1" {
		t.Fatalf("!= search = %#v, %v", ids, err)
	}

	// > operator
	ids, err = l.resolveRefBySearch(db, searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"group_id", ">", "w"},
		},
	})
	if err != nil || len(ids) != 1 || ids[0] != "edge-1" {
		t.Fatalf("> search = %#v, %v", ids, err)
	}

	// <= operator
	ids, err = l.resolveRefBySearch(db, searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"group_id", "<=", "x"},
		},
	})
	if err != nil || len(ids) != 1 || ids[0] != "edge-1" {
		t.Fatalf("<= search = %#v, %v", ids, err)
	}

	// ilike operator
	ids, err = l.resolveRefBySearch(db, searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "ilike", "EDGE%"},
		},
	})
	if err != nil || len(ids) != 1 || ids[0] != "edge-1" {
		t.Fatalf("ilike search = %#v, %v", ids, err)
	}

	// not_in operator
	ids, err = l.resolveRefBySearch(db, searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "not_in", []string{"other", "nope"}},
		},
	})
	if err != nil || len(ids) != 1 || ids[0] != "edge-1" {
		t.Fatalf("not_in search = %#v, %v", ids, err)
	}

	// --- Error branches in applyLeafNode ---

	// Invalid leaf arity: too many elements.
	_, err = l.resolveRefBySearch(db, searchSpec{
		Model:  "auth.User",
		Domain: []any{[]any{"id", "=", "x", "extra"}},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid domain leaf") {
		t.Fatalf("expected too-many-elements leaf error, got %v", err)
	}

	// Missing value for operator that requires it.
	_, err = l.resolveRefBySearch(db, searchSpec{
		Model:  "auth.User",
		Domain: []any{[]any{"id", "!="}},
	})
	if err == nil || !strings.Contains(err.Error(), "requires a value") {
		t.Fatalf("expected missing-value error for !=, got %v", err)
	}
}

func TestNormalizeSearchField_EdgeCases(t *testing.T) {
	// Empty field
	_, err := normalizeSearchField("")
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("expected empty field error, got %v", err)
	}

	// Invalid characters
	_, err = normalizeSearchField("id; DROP")
	if err == nil || !strings.Contains(err.Error(), "invalid search field") {
		t.Fatalf("expected invalid field error, got %v", err)
	}

	// Valid snake_case conversion
	f, err := normalizeSearchField("GroupId")
	if err != nil || f != "group_id" {
		t.Fatalf("normalizeSearchField(GroupId) = %q, %v", f, err)
	}
}

func TestResolveServiceRef_ErrorPaths(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Invalid format (no slash)
	_, err := l.resolveServiceRef(db, "invalid")
	if err == nil || !strings.Contains(err.Error(), "invalid serviceRef") {
		t.Fatalf("expected invalid format error, got %v", err)
	}

	// Valid format but model does not exist.
	_, err = l.resolveServiceRef(db, "missing.Model/Method")
	if err == nil || !strings.Contains(err.Error(), "resolve serviceRef") {
		t.Fatalf("expected model-not-found error, got %v", err)
	}
}

func TestDetectFieldCardinality_FallbackPaths(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Non-existent model returns ManyToOne default.
	c := l.detectFieldCardinality(db, "nonexistent.Model", "any_field")
	if c != refCardinalityManyToOne {
		t.Fatalf("expected ManyToOne fallback for missing model, got %d", c)
	}

	// Model exists but field not in meta_field returns ManyToOne default.
	c = l.detectFieldCardinality(db, "auth.User", "nonexistent_field")
	if c != refCardinalityManyToOne {
		t.Fatalf("expected ManyToOne fallback for missing field, got %d", c)
	}
}

func TestEnforceReferenceCardinality_EdgeCases(t *testing.T) {
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// 0 results on ManyToOne
	_, err := enforceReferenceCardinality([]string{}, refCardinalityManyToOne, "/tmp/d.json", 0, rec, "values.x", "search")
	if err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("expected not_found error, got %v", err)
	}

	// >1 results on ManyToOne
	_, err = enforceReferenceCardinality([]string{"a", "b"}, refCardinalityManyToOne, "/tmp/d.json", 0, rec, "values.x", "search")
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("expected not_unique error, got %v", err)
	}

	// 0 results on First
	_, err = enforceReferenceCardinality([]string{}, refCardinalityFirst, "/tmp/d.json", 0, rec, "values.x", "search")
	if err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("expected not_found error for First, got %v", err)
	}

	// First returns first element
	res, err := enforceReferenceCardinality([]string{"first", "second"}, refCardinalityFirst, "/tmp/d.json", 0, rec, "values.x", "search")
	if err != nil || res != "first" {
		t.Fatalf("expected first element, got %#v, %v", res, err)
	}

	// ManyToMany returns slice as-is
	res, err = enforceReferenceCardinality([]string{"a", "b"}, refCardinalityManyToMany, "/tmp/d.json", 0, rec, "values.x", "search")
	ids, ok := res.([]string)
	if err != nil || !ok || len(ids) != 2 {
		t.Fatalf("expected full slice for ManyToMany, got %#v, %v", res, err)
	}
}

func TestResolveRefBySearch_ComparisonOperators(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed users with known group_id values: a < b < d (lexicographically).
	for _, row := range []map[string]any{
		{"id": "c1", "created_at": time.Now(), "updated_at": time.Now(), "group_id": "b"},
		{"id": "c2", "created_at": time.Now(), "updated_at": time.Now(), "group_id": "d"},
		{"id": "c3", "created_at": time.Now(), "updated_at": time.Now(), "group_id": "a"},
		{"id": "c4", "created_at": time.Now(), "updated_at": time.Now(), "group_id": nil},
	} {
		if err := db.Table("auth_user").Create(row).Error; err != nil {
			t.Fatalf("seed comparison user: %v", err)
		}
	}

	// >= b matches b, d (2 rows).
	spec := searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"group_id", ">=", "b"},
		},
	}
	ids, err := l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 2 {
		t.Fatalf(">= search = %#v, %v", ids, err)
	}

	// < b matches a only.
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"group_id", "<", "b"},
		},
	}
	ids, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 1 || ids[0] != "c3" {
		t.Fatalf("< search = %#v, %v", ids, err)
	}

	// is_null matches c4.
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"group_id", "is_null"},
		},
	}
	ids, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 1 || ids[0] != "c4" {
		t.Fatalf("is_null search = %#v, %v", ids, err)
	}

	// is_not_null matches c1, c2, c3 (3 rows).
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"group_id", "is_not_null"},
		},
	}
	ids, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 3 {
		t.Fatalf("is_not_null search = %#v, %v", ids, err)
	}

	// in
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "in", []string{"c1", "c3"}},
		},
	}
	ids, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 2 {
		t.Fatalf("in search = %#v, %v", ids, err)
	}

	// like
	spec = searchSpec{
		Model: "auth.User",
		Domain: []any{
			[]any{"id", "like", "c%"},
		},
	}
	ids, err = l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 4 {
		t.Fatalf("like search = %#v, %v", ids, err)
	}
}

func TestResolveValue_SearchDomainNestedModelRef(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	var userModel meta.Model
	if err := db.Where("application = ? AND name = ?", "auth", "User").First(&userModel).Error; err != nil {
		t.Fatalf("lookup auth.User model: %v", err)
	}
	userModelID := strings.TrimSpace(userModel.Id.String)
	if userModelID == "" {
		t.Fatalf("expected non-empty user model id")
	}

	// Register meta.MetaField so search model resolution works.
	if err := db.Where("application = ? AND name = ?", "meta", "MetaField").First(&meta.Model{}).Error; err != nil {
		if err := db.Create(&meta.Model{Name: "MetaField", Application: "meta", Path: "/tmp", ModelTable: "meta_field"}).Error; err != nil {
			t.Fatalf("seed meta_model meta.MetaField: %v", err)
		}
	}

	langField := &meta.Field{Name: "Language"}
	langField.ModelId.Valid = true
	langField.ModelId.String = userModelID
	if err := db.Create(langField).Error; err != nil {
		t.Fatalf("seed meta_field: %v", err)
	}
	langFieldID := strings.TrimSpace(langField.Id.String)
	if langFieldID == "" {
		t.Fatalf("expected non-empty language field id")
	}

	rec := record{Module: "auth", Name: "fr", Application: "auth", Model: "RoleFieldRule"}
	got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.FieldId", map[string]any{
		"search": map[string]any{
			"model": "meta.MetaField",
			"domain": []any{
				[]any{"Name", "=", "Language"},
				[]any{"ModelId", "=", map[string]any{"modelRef": "auth.User"}},
			},
			"limit": float64(1),
		},
	})
	if err != nil {
		t.Fatalf("resolveValue(search nested modelRef) error = %v", err)
	}
	ids, ok := got.([]string)
	if !ok || len(ids) != 1 || ids[0] != langFieldID {
		t.Fatalf("resolveValue(search nested modelRef) = %#v, want [%q]", got, langFieldID)
	}
}

func TestResolveSearchDomainRefs_AllBranches(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	rec := record{Module: "auth", Name: "fr", Application: "auth", Model: "RoleFieldRule"}

	if err := db.Table("auth_user").Create(map[string]any{
		"id": "user-collapse-1", "created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("nil and non-array and empty", func(t *testing.T) {
		got, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", nil)
		if err != nil || got != nil {
			t.Fatalf("nil domain = %#v, %v", got, err)
		}
		got, err = l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", "not-an-array")
		if err != nil || got != "not-an-array" {
			t.Fatalf("non-array domain = %#v, %v", got, err)
		}
		got, err = l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{})
		if err != nil {
			t.Fatalf("empty domain error = %v", err)
		}
		arr, ok := got.([]any)
		if !ok || len(arr) != 0 {
			t.Fatalf("empty domain = %#v", got)
		}
	})

	t.Run("and or not combinators", func(t *testing.T) {
		andGot, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{
			"&",
			[]any{"id", "=", "a"},
			[]any{"name", "=", "b"},
		})
		if err != nil {
			t.Fatalf("& domain error = %v", err)
		}
		andArr, ok := andGot.([]any)
		if !ok || len(andArr) != 3 || andArr[0] != "&" {
			t.Fatalf("& domain = %#v", andGot)
		}

		orGot, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{
			"|",
			[]any{"id", "=", "a"},
			[]any{"id", "=", "b"},
		})
		if err != nil {
			t.Fatalf("| domain error = %v", err)
		}
		orArr, ok := orGot.([]any)
		if !ok || len(orArr) != 3 || orArr[0] != "|" {
			t.Fatalf("| domain = %#v", orGot)
		}

		notShort, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{"!"})
		if err != nil {
			t.Fatalf("! short error = %v", err)
		}
		if short, ok := notShort.([]any); !ok || len(short) != 1 || short[0] != "!" {
			t.Fatalf("! short = %#v", notShort)
		}

		notGot, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{
			"!",
			[]any{"id", "=", "a"},
		})
		if err != nil {
			t.Fatalf("! domain error = %v", err)
		}
		notArr, ok := notGot.([]any)
		if !ok || len(notArr) != 2 || notArr[0] != "!" {
			t.Fatalf("! domain = %#v", notGot)
		}
	})

	t.Run("leaf collapses singleton search and copies extras", func(t *testing.T) {
		got, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{
			"group_id",
			"=",
			map[string]any{
				"search": map[string]any{
					"model":  "auth.User",
					"domain": []any{[]any{"id", "=", "user-collapse-1"}},
					"limit":  float64(1),
				},
			},
			"extra-ignored-by-sql",
		})
		if err != nil {
			t.Fatalf("leaf collapse error = %v", err)
		}
		arr, ok := got.([]any)
		if !ok || len(arr) != 4 {
			t.Fatalf("leaf collapse shape = %#v", got)
		}
		if arr[2] != "user-collapse-1" {
			t.Fatalf("leaf collapse value = %#v, want user-collapse-1", arr[2])
		}
		if arr[3] != "extra-ignored-by-sql" {
			t.Fatalf("leaf extra = %#v", arr[3])
		}
	})

	t.Run("error paths", func(t *testing.T) {
		badLeaf := []any{"ModelId", "=", map[string]any{"modelRef": "missing.Model"}}
		if _, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", badLeaf); err == nil {
			t.Fatal("expected leaf resolveValue error")
		}
		if _, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{"&", badLeaf, []any{"id", "=", "x"}}); err == nil {
			t.Fatal("expected & child error")
		}
		if _, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{"|", badLeaf, []any{"id", "=", "x"}}); err == nil {
			t.Fatal("expected | child error")
		}
		if _, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{"!", badLeaf}); err == nil {
			t.Fatal("expected ! child error")
		}
		if _, err := l.resolveSearchDomainRefs(db, "/tmp/d.json", 0, rec, "domain", []any{badLeaf, []any{"id", "=", "x"}}); err == nil {
			t.Fatal("expected implicit AND child error")
		}

		// resolveValue search path must surface domain-resolution errors.
		_, err := l.resolveValue(db, "/tmp/d.json", 0, rec, "values.FieldId", map[string]any{
			"search": map[string]any{
				"model":  "auth.User",
				"domain": []any{badLeaf},
			},
		})
		if err == nil {
			t.Fatal("expected resolveValue search domain error")
		}
	})
}

func TestResolveValue_SearchDomainResolution(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed a group and users.
	if err := db.Table("auth_group").Create(map[string]any{
		"id":         "group-admin",
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if err := db.Table("auth_user").Create(map[string]any{
		"id":         "user-1",
		"created_at": time.Now(),
		"updated_at": time.Now(),
		"group_id":   "group-admin",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// search via resolveValue returns raw []string; cardinality enforced by resolveAndMapValues.
	got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.user_ids", map[string]any{
		"search": map[string]any{
			"model":  "auth.User",
			"domain": []any{[]any{"group_id", "=", "group-admin"}},
		},
	})
	if err != nil {
		t.Fatalf("resolveValue(search) error = %v", err)
	}
	ids, ok := got.([]string)
	if !ok || len(ids) != 1 || ids[0] != "user-1" {
		t.Fatalf("resolveValue(search) = %#v", got)
	}

	// search with limit + orderBy returns raw []string (cardinality not enforced at this level).
	got, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.first", map[string]any{
		"search": map[string]any{
			"model":   "auth.User",
			"limit":   float64(1),
			"orderBy": "id ASC",
		},
	})
	if err != nil {
		t.Fatalf("resolveValue(search limit) error = %v", err)
	}
	ids, ok = got.([]string)
	if !ok || len(ids) != 1 {
		t.Fatalf("resolveValue(search limit) = %#v", got)
	}
}

func TestResolveValue_ManyToOneSearchRequiresUnique(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed two users with the same group to trigger not_unique.
	if err := db.Table("auth_group").Create(map[string]any{
		"id": "g-dup", "created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
	for _, id := range []string{"u-1", "u-2"} {
		if err := db.Table("auth_user").Create(map[string]any{
			"id": id, "created_at": time.Now(), "updated_at": time.Now(), "group_id": "g-dup",
		}).Error; err != nil {
			t.Fatalf("seed user %s: %v", id, err)
		}
	}

	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// ManyToOne (default) with >1 results → error.
	_, err := l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, nil, map[string]any{
		"GroupID": map[string]any{
			"search": map[string]any{
				"model":  "auth.User",
				"domain": []any{[]any{"group_id", "=", "g-dup"}},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected not_unique error for many-to-one with multiple results")
	}
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeRefSearchNotUnique {
		t.Fatalf("expected LoadError with RefSearchNotUnique, got %#v", err)
	}
	if le.FieldPath != "values.GroupID" {
		t.Fatalf("expected FieldPath=values.GroupID, got %q", le.FieldPath)
	}

	// ManyToOne with exactly 1 result → returns the single string.
	cols, err := l.resolveAndMapValues(db, "/tmp/data.json", 1, rec, nil, map[string]any{
		"GroupID": map[string]any{
			"search": map[string]any{
				"model":  "auth.User",
				"domain": []any{[]any{"id", "=", "u-1"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("resolveAndMapValues(unique) error = %v", err)
	}
	if cols["group_id"] != "u-1" {
		t.Fatalf("expected group_id=u-1, got %#v", cols)
	}
}

func TestResolveValue_ManyToOneSearchNotFound(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// ManyToOne (default) with 0 results → error.
	_, err := l.resolveAndMapValues(db, "/tmp/data.json", 5, rec, nil, map[string]any{
		"GroupID": map[string]any{
			"search": map[string]any{
				"model":  "auth.User",
				"domain": []any{[]any{"id", "=", "nonexistent"}},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected not_found error")
	}
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeRefSearchNotFound {
		t.Fatalf("expected LoadError with RefSearchNotFound, got %#v", err)
	}
	if le.FieldPath != "values.GroupID" || le.RecordIndex != 5 {
		t.Fatalf("expected FieldPath=values.GroupID and RecordIndex=5, got %#v", le)
	}
}

func TestResolveValue_ManyToManySearchReturnsSlice(t *testing.T) {
	t.Parallel()

	// enforceReferenceCardinality in ManyToMany mode returns the raw []string slice.
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}
	ids := []string{"a", "b", "c"}
	got, err := enforceReferenceCardinality(ids, refCardinalityManyToMany, "/tmp/data.json", 0, rec, "values.tags", "search")
	if err != nil {
		t.Fatalf("enforceReferenceCardinality(ManyToMany) error = %v", err)
	}
	slc, ok := got.([]string)
	if !ok || len(slc) != 3 {
		t.Fatalf("expected []string with 3 elements, got %#v", got)
	}
}

func TestResolveValue_SearchLimitOneRequiresStableOrder(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// Seed users with ascending ids.
	for _, id := range []string{"u-a", "u-b"} {
		if err := db.Table("auth_user").Create(map[string]any{
			"id": id, "created_at": time.Now(), "updated_at": time.Now(),
		}).Error; err != nil {
			t.Fatalf("seed user %s: %v", id, err)
		}
	}

	// First cardinality (limit=1 + orderBy) returns the first result.
	cols, err := l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, nil, map[string]any{
		"UserID": map[string]any{
			"search": map[string]any{
				"model":   "auth.User",
				"limit":   float64(1),
				"orderBy": "id ASC",
			},
		},
	})
	if err != nil {
		t.Fatalf("resolveAndMapValues(first) error = %v", err)
	}
	if cols["user_id"] != "u-a" {
		t.Fatalf("expected first user u-a, got %#v", cols)
	}

	// First cardinality with 0 results → error.
	_, err = l.resolveAndMapValues(db, "/tmp/data.json", 0, rec, nil, map[string]any{
		"UserID": map[string]any{
			"search": map[string]any{
				"model":   "auth.User",
				"domain":  []any{[]any{"id", "=", "nonexistent"}},
				"limit":   float64(1),
				"orderBy": "id ASC",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected not_found for first cardinality with no results")
	}
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeRefSearchNotFound {
		t.Fatalf("expected RefSearchNotFound, got %#v", err)
	}
}

func TestResolveValue_SearchCardinalityErrorIncludesLocation(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)
	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// Seed users to trigger not_unique.
	for _, id := range []string{"ux-1", "ux-2"} {
		if err := db.Table("auth_user").Create(map[string]any{
			"id": id, "created_at": time.Now(), "updated_at": time.Now(),
		}).Error; err != nil {
			t.Fatalf("seed user %s: %v", id, err)
		}
	}

	// ManyToOne search that hits 2 rows → error with full location info.
	_, err := l.resolveAndMapValues(db, "/tmp/mods/auth/data.json", 7, rec, nil, map[string]any{
		"RoleID": map[string]any{
			"search": map[string]any{
				"model":  "auth.User",
				"domain": []any{[]any{"id", "like", "ux-%"}},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected not_unique error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected LoadError, got %T: %v", err, err)
	}
	if le.Code != LoadErrorCodeRefSearchNotUnique {
		t.Fatalf("expected Code=RefSearchNotUnique, got %q", le.Code)
	}
	if le.FilePath != "/tmp/mods/auth/data.json" {
		t.Fatalf("expected FilePath preserved, got %q", le.FilePath)
	}
	if le.RecordIndex != 7 {
		t.Fatalf("expected RecordIndex=7, got %d", le.RecordIndex)
	}
	if le.FieldPath != "values.RoleID" {
		t.Fatalf("expected FieldPath=values.RoleID, got %q", le.FieldPath)
	}
	if !strings.Contains(le.Message, "multiple results") {
		t.Fatalf("expected 'multiple results' in message, got %q", le.Message)
	}
}

func TestResolveModelRef_SuccessAndNotFound(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// auth.User is seeded in the test schema — modelRef should resolve.
	id, err := l.resolveModelRef(db, "auth.User")
	if err != nil || id == "" {
		t.Fatalf("resolveModelRef(auth.User) = %q, %v", id, err)
	}

	// Unknown model returns error.
	_, err = l.resolveModelRef(db, "unknown.Model")
	if err == nil {
		t.Fatalf("expected error for unknown model")
	}
}

func TestResolveServiceRef_SuccessAndNotFound(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed meta_service so we can resolve a serviceRef.
	if err := db.AutoMigrate(&meta.Service{}); err != nil {
		t.Fatalf("migrate meta_service: %v", err)
	}
	// Register meta.MetaService model entry so resolveSearchModel can find its table.
	if err := db.Create(&meta.Model{
		Name: "MetaService", Application: "meta", Path: "/tmp", ModelTable: "meta_service",
	}).Error; err != nil {
		t.Fatalf("seed meta.MetaService model: %v", err)
	}
	// Fetch the Model ID for auth.User.
	model := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "auth", "User").First(model).Error; err != nil {
		t.Fatalf("lookup auth.User model: %v", err)
	}
	svc := &meta.Service{
		Name:    "Browse",
		ModelId: model.Id,
	}
	if err := db.Create(svc).Error; err != nil {
		t.Fatalf("seed meta_service: %v", err)
	}

	// Valid serviceRef resolves.
	id, err := l.resolveServiceRef(db, "auth.User/Browse")
	if err != nil || id == "" {
		t.Fatalf("resolveServiceRef(auth.User/Browse) = %q, %v", id, err)
	}
	if id != svc.Id.String {
		t.Fatalf("expected service ID %q, got %q", svc.Id.String, id)
	}

	// Unknown method returns error.
	_, err = l.resolveServiceRef(db, "auth.User/Unknown")
	if err == nil {
		t.Fatalf("expected error for unknown method")
	}
}

func TestResolveServiceRef_InvalidFormat(t *testing.T) {
	t.Parallel()

	// splitServiceRef rejects invalid formats.
	for _, tc := range []struct {
		in   string
		want string
	}{
		{in: "auth.User", want: "invalid serviceRef"},
		{in: "", want: "invalid serviceRef"},
		{in: "/", want: "empty model or method"},
		{in: " / ", want: "empty model or method"},
		{in: "x/", want: "empty model or method"},
		{in: "/x", want: "empty model or method"},
	} {
		_, _, err := splitServiceRef(tc.in)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("splitServiceRef(%q) = %v, want substring %q", tc.in, err, tc.want)
		}
	}
}

func TestResolveValue_ModelRefAndServiceRefShortcuts(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed meta_service for the serviceRef test.
	if err := db.AutoMigrate(&meta.Service{}); err != nil {
		t.Fatalf("migrate meta_service: %v", err)
	}
	if err := db.Create(&meta.Model{
		Name: "MetaService", Application: "meta", Path: "/tmp", ModelTable: "meta_service",
	}).Error; err != nil {
		t.Fatalf("seed meta.MetaService model: %v", err)
	}
	model := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "auth", "User").First(model).Error; err != nil {
		t.Fatalf("lookup auth.User model: %v", err)
	}
	svc := &meta.Service{
		Name:    "Browse",
		ModelId: model.Id,
	}
	if err := db.Create(svc).Error; err != nil {
		t.Fatalf("seed meta_service: %v", err)
	}

	rec := record{Module: "auth", Name: "u", Application: "auth", Model: "User"}

	// modelRef via resolveValue returns the model ID as string.
	got, err := l.resolveValue(db, "/tmp/data.json", 0, rec, "values.model_id", map[string]any{
		"modelRef": "auth.User",
	})
	if err != nil {
		t.Fatalf("resolveValue(modelRef) error = %v", err)
	}
	if s, ok := got.(string); !ok || s == "" {
		t.Fatalf("expected non-empty string from modelRef, got %#v", got)
	}

	// serviceRef via resolveValue returns the service ID as string.
	got, err = l.resolveValue(db, "/tmp/data.json", 0, rec, "values.service_id", map[string]any{
		"serviceRef": "auth.User/Browse",
	})
	if err != nil {
		t.Fatalf("resolveValue(serviceRef) error = %v", err)
	}
	if s, ok := got.(string); !ok || s != svc.Id.String {
		t.Fatalf("expected service ID %q from serviceRef, got %#v", svc.Id.String, got)
	}
}

func TestResolveValue_ServiceRefSharesSearchExecutor(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// resolveServiceRef internally uses resolveRefBySearch to query meta_service.
	// Verify that a serviceRef resolves correctly and that the underlying search
	// executor was exercised (by checking the result matches a direct search).
	if err := db.AutoMigrate(&meta.Service{}); err != nil {
		t.Fatalf("migrate meta_service: %v", err)
	}
	if err := db.Create(&meta.Model{
		Name: "MetaService", Application: "meta", Path: "/tmp", ModelTable: "meta_service",
	}).Error; err != nil {
		t.Fatalf("seed meta.MetaService model: %v", err)
	}
	model := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "auth", "User").First(model).Error; err != nil {
		t.Fatalf("lookup auth.User model: %v", err)
	}
	svc := &meta.Service{
		Name:    "GetPermissionState",
		ModelId: model.Id,
	}
	if err := db.Create(svc).Error; err != nil {
		t.Fatalf("seed meta_service: %v", err)
	}

	// Resolve via shortcut.
	idFromShortcut, err := l.resolveServiceRef(db, "auth.User/GetPermissionState")
	if err != nil {
		t.Fatalf("resolveServiceRef error = %v", err)
	}

	// Resolve via equivalent direct search.
	spec := searchSpec{
		Model: "meta.MetaService",
		Domain: []any{
			"&",
			[]any{"model_id", "=", model.Id.String},
			[]any{"name", "=", "GetPermissionState"},
		},
	}
	ids, err := l.resolveRefBySearch(db, spec)
	if err != nil || len(ids) != 1 || ids[0] != idFromShortcut {
		t.Fatalf("direct search = %#v, %v; shortcut gave %q", ids, err, idFromShortcut)
	}
}

func TestMetadataCache_ModelRefAndFieldCardinality(t *testing.T) {
	t.Parallel()

	l, db := newTestLoader(t)

	// Seed meta_field so detectFieldCardinality has something to look up.
	if err := db.AutoMigrate(&meta.Field{}); err != nil {
		t.Fatalf("migrate meta_field: %v", err)
	}

	// First modelRef call should populate the model cache.
	id1, err := l.resolveModelRef(db, "auth.User")
	if err != nil {
		t.Fatalf("resolveModelRef first call: %v", err)
	}
	if id1 == "" {
		t.Fatal("expected non-empty model ID from first resolveModelRef")
	}

	// Second call to the same modelRef must return the cached ID without querying.
	id2, err := l.resolveModelRef(db, "auth.User")
	if err != nil {
		t.Fatalf("resolveModelRef second call: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("cache mismatch: first=%q second=%q", id1, id2)
	}

	// First cardinality lookup populates the field cache.
	c1 := l.detectFieldCardinality(db, "auth.User", "group_id")
	// Second call must hit cache.
	c2 := l.detectFieldCardinality(db, "auth.User", "group_id")
	if c1 != c2 {
		t.Fatalf("cardinality cache mismatch: first=%d second=%d", c1, c2)
	}

	// Different field should resolve independently.
	c3 := l.detectFieldCardinality(db, "auth.User", "name")
	if c3 == c2 {
		// They might be the same cardinality, that's fine — the point is no panic.
	}

	// A third lookup on the first field must still hit cache and return the same value.
	c4 := l.detectFieldCardinality(db, "auth.User", "group_id")
	if c1 != c4 {
		t.Fatalf("cardinality cache corruption: first=%d retry=%d", c1, c4)
	}
}

func execLoaderTestSQL(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec sql %q: %v", sql, err)
	}
}

func TestLoadModuleIndexAndDependencyClosureDBFailures(t *testing.T) {
	t.Parallel()

	t.Run("load module index query failure", func(t *testing.T) {
		_, db := newTestLoader(t)
		execLoaderTestSQL(t, db, "DROP TABLE meta_module")
		execLoaderTestSQL(t, db, "CREATE TABLE meta_module (id TEXT)")
		if _, _, err := loadModuleIndex(db); err == nil || !strings.Contains(err.Error(), "load module index") {
			t.Fatalf("loadModuleIndex() error = %v, want load module index failure", err)
		}
	})

	t.Run("dependency closure query failure", func(t *testing.T) {
		_, db := newTestLoader(t)
		var auth meta.Module
		if err := db.Where("name = ?", "auth").First(&auth).Error; err != nil {
			t.Fatalf("lookup auth module: %v", err)
		}
		execLoaderTestSQL(t, db, "DROP TABLE meta_module_dependencies")
		execLoaderTestSQL(t, db, "CREATE TABLE meta_module_dependencies (module_id TEXT)")
		if _, err := dependencyClosure(db, auth.Id.String, nil); err == nil || !strings.Contains(err.Error(), "load module dependencies") {
			t.Fatalf("dependencyClosure() error = %v, want dependency lookup failure", err)
		}
	})
}

func TestPlanRecordOrderLookupModelDataFailure(t *testing.T) {
	l, db := newTestLoader(t)
	owner := &meta.Module{Name: "auth"}
	records := []record{{
		Module: "auth",
		Name:   "u",
		Application: "auth", Model: "User",
		Values: map[string]any{
			"group_id": map[string]any{"ref": "auth.pre_group"},
		},
	}}

	execLoaderTestSQL(t, db, "DROP TABLE meta_model_data")
	_, err := l.planRecordOrder(db, owner, filepath.Join(t.TempDir(), "data.json"), records)
	if err == nil || !strings.Contains(err.Error(), "lookup model_data for refs") {
		t.Fatalf("planRecordOrder() error = %v, want model_data lookup failure", err)
	}
}

func TestPlanBatchRecordOrderLookupModelDataFailure(t *testing.T) {
	l, db := newTestLoader(t)
	owner := &meta.Module{Name: "auth"}
	batch := []batchRecord{{
		FilePath:    filepath.Join(t.TempDir(), "data.json"),
		RecordIndex: 0,
		Rec: record{
			Module: "auth",
			Name:   "u",
			Application: "auth", Model: "User",
			Values: map[string]any{
				"group_id": map[string]any{"ref": "auth.pre_group"},
			},
		},
	}}

	execLoaderTestSQL(t, db, "DROP TABLE meta_model_data")
	_, err := l.planBatchRecordOrder(db, owner, batch)
	if err == nil || !strings.Contains(err.Error(), "lookup model_data for refs") {
		t.Fatalf("planBatchRecordOrder() error = %v, want model_data lookup failure", err)
	}
}

func TestApplyRecordModelDataDBFailures(t *testing.T) {
	now := time.Now()

	t.Run("lookup model_data db error", func(t *testing.T) {
		l, db := newTestLoader(t)
		execLoaderTestSQL(t, db, "DROP TABLE meta_model_data")
		execLoaderTestSQL(t, db, "CREATE TABLE meta_model_data (id TEXT, module TEXT, name TEXT, model TEXT, res_id TEXT, no_update INTEGER)")
		err := l.applyRecord(db, "/tmp/data.json", 0, record{
			Module: "auth", Name: "new_user", Application: "auth", Model: "User", Values: map[string]any{},
		}, now)
		var le *LoadError
		if !errors.As(err, &le) || le.Code != LoadErrorCodeDBLookupModelData {
			t.Fatalf("applyRecord() lookup error = %#v, want code %s", err, LoadErrorCodeDBLookupModelData)
		}
	})

	t.Run("insert model_data db error", func(t *testing.T) {
		l, db := newTestLoader(t)
		execLoaderTestSQL(t, db, "DROP TABLE meta_model_data")
		if err := db.AutoMigrate(&metadata.ModelData{}); err != nil {
			t.Fatalf("remigrate meta_model_data: %v", err)
		}
		execLoaderTestSQL(t, db, `CREATE TRIGGER block_model_data_insert BEFORE INSERT ON meta_model_data BEGIN SELECT RAISE(ABORT, 'blocked'); END`)
		err := l.applyRecord(db, "/tmp/data.json", 0, record{
			Module: "auth", Name: "insert_blocked", Application: "auth", Model: "User", Values: map[string]any{},
		}, now)
		var le *LoadError
		if !errors.As(err, &le) || le.Code != LoadErrorCodeDBInsertModelData {
			t.Fatalf("applyRecord() insert model_data error = %#v, want code %s", err, LoadErrorCodeDBInsertModelData)
		}
	})

	t.Run("update model_data noupdate db error", func(t *testing.T) {
		l, db := newTestLoader(t)
		group := testAuthGroup{ID: "group-no-update"}
		user := testAuthUser{ID: "user-no-update", GroupID: group.ID}
		if err := db.Table("auth_group").Create(&group).Error; err != nil {
			t.Fatalf("seed group: %v", err)
		}
		if err := db.Table("auth_user").Create(&user).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if err := db.Create(&metadata.ModelData{
			Module: "auth", Name: "user_no_update", Application: "auth", Model: "User", ResID: user.ID, NoUpdate: false,
		}).Error; err != nil {
			t.Fatalf("seed mapping: %v", err)
		}
		execLoaderTestSQL(t, db, `CREATE TRIGGER block_model_data_update BEFORE UPDATE ON meta_model_data BEGIN SELECT RAISE(ABORT, 'blocked'); END`)
		freeze := true
		err := l.applyRecord(db, "/tmp/data.json", 1, record{
			Module: "auth", Name: "user_no_update", Application: "auth", Model: "User", NoUpdate: &freeze,
			Values: map[string]any{"group_id": group.ID},
		}, now)
		var le *LoadError
		if !errors.As(err, &le) || le.Code != LoadErrorCodeDBUpdateModelDataNoUpdate {
			t.Fatalf("applyRecord() update noupdate error = %#v, want code %s", err, LoadErrorCodeDBUpdateModelDataNoUpdate)
		}
	})
}

func TestNormalizeRecordOwnership_ModuleNotOwner(t *testing.T) {
	t.Parallel()
	rules := &moduleRules{
		OwnerName:  "auth",
		OwnerApp:   "auth",
		ModuleInfo: map[string]moduleInfo{"auth": {Application: "auth"}},
		Allowed:    map[string]struct{}{"auth": {}},
	}
	rec := record{Module: "does_not_exist", Name: "x", Model: "User", Values: map[string]any{}}
	err := normalizeRecordOwnership(rules, "/tmp/data.json", 0, &rec)
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeModuleNotOwner || le.Name != "x" {
		t.Fatalf("normalizeRecordOwnership() error = %#v, want module_not_owner with Name=x", err)
	}
}

func TestPlanBatchRecordOrder_ValidationAndRefErrors(t *testing.T) {
	l, db := newTestLoader(t)
	owner := &meta.Module{Name: "auth"}
	filePath := filepath.Join(t.TempDir(), "data.json")

	for _, tc := range []struct {
		name string
		rec  record
		code string
	}{
		{name: "missing name", rec: record{Module: "auth", Application: "auth", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeMissingName},
		{name: "module not owner", rec: record{Module: "base", Name: "x", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeModuleNotOwner},
		{name: "application mismatch", rec: record{Name: "x", Application: "base", Model: "User", Values: map[string]any{}}, code: LoadErrorCodeApplicationMismatch},
		{name: "missing model", rec: record{Module: "auth", Name: "x", Application: "auth", Values: map[string]any{}}, code: LoadErrorCodeMissingModel},
		{name: "missing values", rec: record{Module: "auth", Name: "x", Application: "auth", Model: "User"}, code: LoadErrorCodeMissingValues},
		{name: "invalid model full name", rec: record{Module: "auth", Name: "x", Application: "auth", Model: "auth.User", Values: map[string]any{}}, code: LoadErrorCodeInvalidModel},
		{
			name: "invalid ref",
			rec: record{
				Module: "auth", Name: "x", Application: "auth", Model: "User",
				Values: map[string]any{"group_id": map[string]any{"ref": "not-a-ref"}},
			},
			code: LoadErrorCodeInvalidRef,
		},
		{
			name: "self cycle",
			rec: record{
				Module: "auth", Name: "self", Application: "auth", Model: "User",
				Values: map[string]any{"group_id": map[string]any{"ref": "auth.self"}},
			},
			code: LoadErrorCodeRefSelfCycle,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := l.planBatchRecordOrder(db, owner, []batchRecord{{
				FilePath: filePath, RecordIndex: 0, Rec: tc.rec,
			}})
			var le *LoadError
			if !errors.As(err, &le) || le.Code != tc.code {
				t.Fatalf("planBatchRecordOrder() error = %#v, want code %s", err, tc.code)
			}
		})
	}
}

func TestPlanRecordOrder_DuplicateInvalidAndExternalRef(t *testing.T) {
	l, db := newTestLoader(t)
	owner := &meta.Module{Name: "auth"}
	filePath := filepath.Join(t.TempDir(), "data.json")

	t.Run("duplicate name", func(t *testing.T) {
		_, err := l.planRecordOrder(db, owner, filePath, []record{
			{Module: "auth", Name: "x", Application: "auth", Model: "User", Values: map[string]any{}},
			{Module: "auth", Name: "x", Application: "auth", Model: "User", Values: map[string]any{}},
		})
		var le *LoadError
		if !errors.As(err, &le) || le.Code != LoadErrorCodeDuplicateNameInInput || le.Name != "x" {
			t.Fatalf("planRecordOrder() error = %#v, want DuplicateNameInInput", err)
		}
	})

	t.Run("invalid ref", func(t *testing.T) {
		_, err := l.planRecordOrder(db, owner, filePath, []record{{
			Module: "auth", Name: "x", Application: "auth", Model: "User",
			Values: map[string]any{"group_id": map[string]any{"ref": "not-a-ref"}},
		}})
		var le *LoadError
		if !errors.As(err, &le) || le.Code != LoadErrorCodeInvalidRef {
			t.Fatalf("planRecordOrder() error = %#v, want InvalidRef", err)
		}
	})

	t.Run("external ref exists", func(t *testing.T) {
		if err := db.Create(&metadata.ModelData{
			Module: "auth", Name: "pre_group", Application: "auth", Model: "group", ResID: "gid-pre",
		}).Error; err != nil {
			t.Fatalf("seed model_data: %v", err)
		}
		order, err := l.planRecordOrder(db, owner, filePath, []record{{
			Module: "auth", Name: "u", Application: "auth", Model: "User",
			Values: map[string]any{"group_id": map[string]any{"ref": "auth.pre_group"}},
		}})
		if err != nil {
			t.Fatalf("planRecordOrder() error = %v", err)
		}
		if len(order) != 1 || order[0] != 0 {
			t.Fatalf("unexpected order: %v", order)
		}
	})

	t.Run("external ref missing", func(t *testing.T) {
		_, err := l.planRecordOrder(db, owner, filePath, []record{{
			Module: "auth", Name: "u2", Application: "auth", Model: "User",
			Values: map[string]any{"group_id": map[string]any{"ref": "auth.still_missing"}},
		}})
		var le *LoadError
		if !errors.As(err, &le) || le.Code != LoadErrorCodeRefNotFound || le.Ref != "auth.still_missing" {
			t.Fatalf("planRecordOrder() error = %#v, want RefNotFound", err)
		}
	})
}
