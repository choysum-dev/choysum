// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/plan"
	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type writerTestAuthGroup struct {
	ID        string         `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (writerTestAuthGroup) TableName() string { return "auth_group" }

type writerTestAuthUser struct {
	ID        string         `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
	GroupID   string         `gorm:"column:group_id"`
}

func (writerTestAuthUser) TableName() string { return "auth_user" }

type writerTestModuleDependency struct {
	ModuleID       string `gorm:"column:module_id;primaryKey"`
	DependModuleID string `gorm:"column:depend_module_id;primaryKey"`
}

func (writerTestModuleDependency) TableName() string { return "meta_module_dependencies" }

func newWriterTestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect:         "sqlite",
			DSN:             filepath.Join(t.TempDir(), "initdata-writer.db"),
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
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Module{}, &meta.Model{}, &meta.Field{}, &modmeta.ModelData{}, &writerTestAuthGroup{}, &writerTestAuthUser{}, &writerTestModuleDependency{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	authGroupModel := &meta.Model{Name: "group", Application: "auth", Path: "/tmp", ModelTable: "auth_group"}
	if err := db.Create(authGroupModel).Error; err != nil {
		t.Fatalf("seed auth.group: %v", err)
	}
	authUserModel := &meta.Model{Name: "User", Application: "auth", Path: "/tmp", ModelTable: "auth_user"}
	if err := db.Create(authUserModel).Error; err != nil {
		t.Fatalf("seed auth.User: %v", err)
	}
	if err := db.Create(&meta.Field{Name: "group_id", FieldType: "ManyToOne", ModelId: authUserModel.Id}).Error; err != nil {
		t.Fatalf("seed field: %v", err)
	}
	auth := &meta.Module{Name: "auth", ApplicationStr: "auth", Path: "/tmp"}
	if err := db.Create(auth).Error; err != nil {
		t.Fatalf("create auth: %v", err)
	}
	authAddon := &meta.Module{Name: "auth_addon", ApplicationStr: "auth", Path: "/tmp"}
	if err := db.Create(authAddon).Error; err != nil {
		t.Fatalf("create auth_addon: %v", err)
	}
	base := &meta.Module{Name: "base", ApplicationStr: "base", Path: "/tmp"}
	if err := db.Create(base).Error; err != nil {
		t.Fatalf("create base: %v", err)
	}
	if err := db.Exec("INSERT INTO meta_module_dependencies (module_id, depend_module_id) VALUES (?, ?)", authAddon.Id.String, auth.Id.String).Error; err != nil {
		t.Fatalf("insert dependency: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return runtimeScope
}

func writeWriterTestDataFile(t *testing.T, dir string, df any) {
	t.Helper()
	b, err := json.Marshal(df)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.json"), b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestWriter_Write(t *testing.T) {
	runtimeScope := newWriterTestScope(t)
	dir := t.TempDir()
	writeWriterTestDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "group_writer", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})

	unit := initdataplan.Unit{
		Index:       1,
		ModuleName:  "auth",
		ModulePath:  dir,
		Application: "auth",
		Files:       []string{"data.json"},
	}
	writer := Writer{}
	if err := writer.Write(context.Background(), runtimeScope, []plan.Unit{unit}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var count int64
	if err := runtimeScope.Session().Table("auth_group").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

func TestWriter_WriteUnexpectedUnitType(t *testing.T) {
	writer := Writer{}
	err := writer.Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 1}})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want CodeInvalidFormat", err)
	}
}

func TestWriter_WriteLoadErrorMapped(t *testing.T) {
	runtimeScope := newWriterTestScope(t)
	dir := t.TempDir()
	writeWriterTestDataFile(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "bad", "application": "auth", "model": "MissingModel", "values": map[string]any{}},
		},
	})

	unit := initdataplan.Unit{
		ModuleName:  "auth",
		ModulePath:  dir,
		Application: "auth",
		Files:       []string{"data.json"},
	}
	writer := Writer{}
	err := writer.Write(context.Background(), runtimeScope, []plan.Unit{unit})
	if err == nil {
		t.Fatal("expected load error")
	}
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) {
		t.Fatalf("error = %T, want *importpkg.Error", err)
	}
	if impErr.Code == "" || impErr.Text == "" {
		t.Fatalf("mapped error = %+v", impErr)
	}
}

func TestWriter_WritePlainError(t *testing.T) {
	runtimeScope := newWriterTestScope(t)
	unit := initdataplan.Unit{
		ModuleName: "auth",
		ModulePath: "/missing/path",
		Files:      []string{"data.json"},
	}
	writer := Writer{}
	err := writer.Write(context.Background(), runtimeScope, []plan.Unit{unit})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	var impErr *importpkg.Error
	if errors.As(err, &impErr) {
		t.Fatalf("plain filesystem error should not be wrapped as import error: %v", err)
	}
}

func TestWriter_WriteNilUnits(t *testing.T) {
	writer := Writer{}
	if err := writer.Write(context.Background(), nil, nil); err != nil {
		t.Fatalf("Write(nil units): %v", err)
	}
}

func TestMapLoaderError(t *testing.T) {
	loadErr := &dataloader.LoadError{Code: dataloader.LoadErrorCodeMissingName, Message: "missing name"}
	err := mapLoaderError(loadErr)
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != dataloader.LoadErrorCodeMissingName {
		t.Fatalf("mapLoaderError(loadErr) = %v", err)
	}
	plain := errors.New("plain")
	if mapLoaderError(plain) != plain {
		t.Fatal("non-LoadError should pass through unchanged")
	}
}
