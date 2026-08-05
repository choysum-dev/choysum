// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
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
		Module: seed.module.Name, Name: "seed_row", Application: "demo", ModelName: "Item", ModelId: xid.New().String(), ResID: resID,
	}
	otherMapping := &metadata.ModelData{
		Module: otherModule.Name, Name: "other_row", Application: "other", ModelName: "Item", ModelId: xid.New().String(), ResID: otherResID,
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

	var hardDeleted int64
	if err := db.Unscoped().Model(&metadata.ModelData{}).Where("module = ? AND name = ?", seed.module.Name, "seed_row").Count(&hardDeleted).Error; err != nil {
		t.Fatalf("count hard-deleted mapping: %v", err)
	}
	if hardDeleted != 0 {
		t.Fatalf("demo mapping remaining (incl. soft-deleted) = %d, want 0 (hard delete)", hardDeleted)
	}

	var businessName string
	if err := db.Raw(`SELECT name FROM `+businessTable+` WHERE id = ?`, resID).Scan(&businessName).Error; err != nil {
		t.Fatalf("query business row: %v", err)
	}
	if businessName != "kept" {
		t.Fatalf("business row name = %q, want kept (must not cascade-delete seed rows)", businessName)
	}

	// Reinstall must be able to recreate the same (module, name) key after cleanup.
	if err := db.Create(&metadata.ModelData{
		Module: seed.module.Name, Name: "seed_row", Application: "demo", ModelName: "Item", ModelId: xid.New().String(), ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("recreate mapping after uninstall cleanup: %v", err)
	}
}

func TestModuleUninstallerCleanModelsRecomputesAfterExtRawDelete(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}

	baseMod := &meta.Module{Name: "partner", Status: meta.Installed, Version: "1.0.0"}
	baseMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(baseMod).Error; err != nil {
		t.Fatalf("create base module: %v", err)
	}
	extMod := &meta.Module{Name: "partner_commercial", Status: meta.Installed, Version: "1.0.0"}
	extMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(extMod).Error; err != nil {
		t.Fatalf("create ext module: %v", err)
	}

	basePath := "@/partner/service/models/partner.ts"
	baseRaw := &meta.RawModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        basePath,
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    baseMod.Id,
	}
	if err := db.Create(baseRaw).Error; err != nil {
		t.Fatalf("create base raw model: %v", err)
	}
	extRaw := &meta.RawModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        "@/partner_commercial/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    extMod.Id,
		Extends:     basePath,
	}
	if err := db.Create(extRaw).Error; err != nil {
		t.Fatalf("create ext raw model: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        extMod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels() error = %v", err)
	}

	var extRawDeleted meta.RawModel
	err := db.Unscoped().Where("id = ?", extRaw.Id.String).First(&extRawDeleted).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("ext raw model should be hard-deleted, got err=%v row=%+v", err, extRawDeleted)
	}

	var baseRawCount int64
	if err := db.Model(&meta.RawModel{}).Where("id = ?", baseRaw.Id.String).Count(&baseRawCount).Error; err != nil {
		t.Fatalf("count base raw: %v", err)
	}
	if baseRawCount != 1 {
		t.Fatalf("base raw model should survive, count = %d", baseRawCount)
	}

	var effectives []meta.Model
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").Find(&effectives).Error; err != nil {
		t.Fatalf("load effective models: %v", err)
	}
	if len(effectives) != 1 {
		t.Fatalf("survivor raw should yield one effective model, count = %d", len(effectives))
	}
	// Effective projections intentionally omit ModuleId (merged across declarations);
	// Path still identifies which tip declaration was reprojected.
	if effectives[0].Path != basePath {
		t.Fatalf("effective model path = %q, want survivor base path %q", effectives[0].Path, basePath)
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
		Module: seed.module.Name, Name: "seed_row", Application: "demo", ModelName: "Item", ModelId: xid.New().String(), ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}
	stmt := `
CREATE TRIGGER block_meta_model_data_delete
BEFORE DELETE ON meta_model_data
BEGIN
  SELECT RAISE(ABORT, 'hard delete blocked');
END`
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("create hard delete trigger: %v", err)
	}

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

func TestModuleUninstallerCleanModelsListingRawModelsError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)
	dropMetaTable(t, db, &meta.RawModel{})

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error listing raw models for uninstall") {
		t.Fatalf("cleanModels() error = %v, want listing raw models failure", err)
	}
}
