// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type bootstrapAuthGroup struct {
	ID        string         `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (bootstrapAuthGroup) TableName() string { return "auth_group" }

type bootstrapAuthUser struct {
	ID        string         `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
	GroupID   string         `gorm:"column:group_id"`
}

func (bootstrapAuthUser) TableName() string { return "auth_user" }

type bootstrapModuleDependency struct {
	ModuleID       string `gorm:"column:module_id;primaryKey"`
	DependModuleID string `gorm:"column:depend_module_id;primaryKey"`
}

func (bootstrapModuleDependency) TableName() string { return "meta_module_dependencies" }

// BootstrapTestScope returns a scope seeded for loader and initdata writer tests.
func BootstrapTestScope(t *testing.T) scope.Scope {
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
	seedBootstrapTestSchema(t, runtimeScope.Session().DB)
	if sqlDB, err := runtimeScope.Session().DB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return runtimeScope
}

// WriteDataFileForTest writes a bootstrap JSON file for loader/import tests.
func WriteDataFileForTest(t *testing.T, dir string, df any) {
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

func seedBootstrapTestSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Module{}, &meta.Model{}, &meta.Field{}, &modmeta.ModelData{}, &bootstrapAuthGroup{}, &bootstrapAuthUser{}, &bootstrapModuleDependency{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	authGroupModel := &meta.Model{Name: "group", Application: "auth", Path: "/tmp", ModelTable: "auth_group"}
	if err := db.Create(authGroupModel).Error; err != nil {
		t.Fatalf("seed meta_model auth.group: %v", err)
	}
	authUserModel := &meta.Model{Name: "User", Application: "auth", Path: "/tmp", ModelTable: "auth_user"}
	if err := db.Create(authUserModel).Error; err != nil {
		t.Fatalf("seed meta_model auth.User: %v", err)
	}
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
