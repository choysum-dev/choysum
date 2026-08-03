// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"strings"
	"testing"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
)

func TestModuleUninstallerCleanModelsClearsMetaModelDataOnly(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&metadata.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate ModelData: %v", err)
	}

	seed := seedCleanModelsFixture(t, db)
	otherModule := &meta.Module{Name: "other", Status: meta.Installed, Version: "1.0.0"}
	if err := db.Create(otherModule).Error; err != nil {
		t.Fatalf("create other module: %v", err)
	}

	businessTable := "demo_seed_row"
	if err := db.Exec(`CREATE TABLE ` + businessTable + ` (id TEXT PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatalf("create business table: %v", err)
	}
	resID := xid.New().String()
	otherResID := xid.New().String()
	if err := db.Exec(`INSERT INTO `+businessTable+`(id, name) VALUES (?, ?)`, resID, "kept").Error; err != nil {
		t.Fatalf("insert business row: %v", err)
	}
	if err := db.Exec(`INSERT INTO `+businessTable+`(id, name) VALUES (?, ?)`, otherResID, "other").Error; err != nil {
		t.Fatalf("insert other business row: %v", err)
	}

	demoMapping := &metadata.ModelData{
		Module: seed.module.Name, Name: "seed_row", Model: "demo.Thing", ResID: resID,
	}
	otherMapping := &metadata.ModelData{
		Module: otherModule.Name, Name: "other_row", Model: "other.Thing", ResID: otherResID,
	}
	if err := db.Create(demoMapping).Error; err != nil {
		t.Fatalf("create demo mapping: %v", err)
	}
	if err := db.Create(otherMapping).Error; err != nil {
		t.Fatalf("create other mapping: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels() error = %v", err)
	}

	var remaining int64
	if err := db.Model(&metadata.ModelData{}).Where("module = ?", seed.module.Name).Count(&remaining).Error; err != nil {
		t.Fatalf("count demo mappings: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("demo mappings remaining = %d, want 0", remaining)
	}

	var other metadata.ModelData
	if err := db.Where("module = ? AND name = ?", otherModule.Name, "other_row").First(&other).Error; err != nil {
		t.Fatalf("other module mapping should remain: %v", err)
	}
	if other.ResID != otherResID {
		t.Fatalf("other mapping ResID = %q, want %q", other.ResID, otherResID)
	}

	var softDeleted metadata.ModelData
	if err := db.Unscoped().Where("module = ? AND name = ?", seed.module.Name, "seed_row").First(&softDeleted).Error; err != nil {
		t.Fatalf("soft-deleted mapping lookup: %v", err)
	}
	if !softDeleted.DeletedAt.Valid {
		t.Fatalf("expected demo mapping soft-deleted, got DeletedAt=%v", softDeleted.DeletedAt)
	}

	var businessName string
	if err := db.Raw(`SELECT name FROM `+businessTable+` WHERE id = ?`, resID).Scan(&businessName).Error; err != nil {
		t.Fatalf("query business row: %v", err)
	}
	if businessName != "kept" {
		t.Fatalf("business row name = %q, want kept (must not cascade-delete seed rows)", businessName)
	}
}

func TestModuleUninstallerCleanModelsMetaModelDataError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&metadata.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate ModelData: %v", err)
	}
	seed := seedCleanModelsFixture(t, db)
	if err := db.Create(&metadata.ModelData{
		Module: seed.module.Name, Name: "seed_row", Model: "demo.Thing", ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}
	blockMetaSoftDeletes(t, db, "meta_model_data")

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error deleting meta model data mappings") {
		t.Fatalf("cleanModels() error = %v, want meta model data mappings failure", err)
	}
}
