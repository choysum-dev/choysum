// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"strings"
	"testing"
	"time"

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

func TestModuleUninstallerCleanModelsRebindsMetaModelDataTip(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&metadata.ModelData{}, &meta.Model{}, &meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
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

	baseModel := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        "@/partner/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    baseMod.Id,
	}
	if err := db.Create(baseModel).Error; err != nil {
		t.Fatalf("create base model: %v", err)
	}
	extModel := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        "@/partner_commercial/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    extMod.Id,
		Extends:     baseModel.Path,
	}
	// Ensure ext is newer tip for CreatedAt ordering.
	if err := db.Create(extModel).Error; err != nil {
		t.Fatalf("create ext model: %v", err)
	}
	if err := db.Model(extModel).Update("created_at", baseModel.CreatedAt.Add(time.Hour)).Error; err != nil {
		t.Fatalf("bump ext created_at: %v", err)
	}

	ownerMod := &meta.Module{Name: "seed_owner", Status: meta.Installed, Version: "1.0.0"}
	ownerMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(ownerMod).Error; err != nil {
		t.Fatalf("create owner module: %v", err)
	}
	mapping := &metadata.ModelData{
		Module:      ownerMod.Name,
		Name:        "partner_main",
		Application: "partner",
		ModelName:   "Partner",
		ModelId:     extModel.Id.String,
		ResID:       xid.New().String(),
	}
	if err := db.Create(mapping).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
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

	var updated metadata.ModelData
	if err := db.Where("module = ? AND name = ?", ownerMod.Name, "partner_main").First(&updated).Error; err != nil {
		t.Fatalf("mapping should remain: %v", err)
	}
	if updated.ModelId != baseModel.Id.String {
		t.Fatalf("ModelId = %q, want rebound tip %q", updated.ModelId, baseModel.Id.String)
	}

	var softDeleted meta.Model
	if err := db.Unscoped().Where("id = ?", extModel.Id.String).First(&softDeleted).Error; err != nil {
		t.Fatalf("ext model should still exist soft-deleted: %v", err)
	}
	if !softDeleted.DeletedAt.Valid {
		t.Fatalf("expected ext model soft-deleted")
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

func TestModuleUninstallerCleanModelsListingModelsError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)
	dropMetaTable(t, db, &meta.Model{})

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error listing models for uninstall") {
		t.Fatalf("cleanModels() error = %v, want listing models failure", err)
	}
}

func TestRebindMetaModelDataTipsBranches(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&metadata.ModelData{}, &meta.Model{}, &meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	t.Run("empty victims", func(t *testing.T) {
		if err := rebindMetaModelDataTips(db, nil); err != nil {
			t.Fatalf("empty victims: %v", err)
		}
	})

	t.Run("missing table", func(t *testing.T) {
		dropMetaTable(t, db, &metadata.ModelData{})
		err := rebindMetaModelDataTips(db, []modelVictim{{
			Id: xid.New().String(), Application: "partner", Name: "Partner",
		}})
		if err != nil {
			t.Fatalf("missing table should no-op: %v", err)
		}
		if err := db.AutoMigrate(&metadata.ModelData{}); err != nil {
			t.Fatalf("recreate ModelData: %v", err)
		}
	})

	t.Run("skips blank victims and continues when tip missing", func(t *testing.T) {
		goneID := xid.New().String()
		if err := rebindMetaModelDataTips(db, []modelVictim{
			{Id: "", Application: "partner", Name: "Partner"},
			{Id: goneID, Application: "  ", Name: "Partner"},
			{Id: goneID, Application: "partner", Name: " "},
			{Id: goneID, Application: "partner", Name: "Gone"},
		}); err != nil {
			t.Fatalf("blank/missing tip: %v", err)
		}
	})

	t.Run("tip resolve error", func(t *testing.T) {
		dropMetaTable(t, db, &meta.Model{})
		err := rebindMetaModelDataTips(db, []modelVictim{{
			Id: xid.New().String(), Application: "partner", Name: "Partner",
		}})
		if err == nil || !strings.Contains(err.Error(), "error resolving meta model tip") {
			t.Fatalf("tip resolve error = %v, want resolving tip failure", err)
		}
		if err := db.AutoMigrate(&meta.Model{}); err != nil {
			t.Fatalf("recreate meta_model: %v", err)
		}
	})

	t.Run("empty tip id skipped", func(t *testing.T) {
		// Insert a live tip row whose primary key is blank so tip.Id.String trims to "".
		if err := db.Exec(`
INSERT INTO meta_model (id, name, path, application, model_table, created_at, updated_at)
VALUES ('', 'EmptyTip', '@/partner/service/models/empty_tip.ts', 'partner', 'partner_empty_tip', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`).Error; err != nil {
			t.Fatalf("insert blank-id tip: %v", err)
		}
		if err := rebindMetaModelDataTips(db, []modelVictim{{
			Id: xid.New().String(), Application: "partner", Name: "EmptyTip",
		}}); err != nil {
			t.Fatalf("empty tip id: %v", err)
		}
		_ = db.Unscoped().Where("name = ?", "EmptyTip").Delete(&meta.Model{})
	})

	t.Run("update error propagates", func(t *testing.T) {
		tip := &meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name:        "UpdateFail",
			Path:        "@/partner/service/models/update_fail.ts",
			Application: "partner",
			ModelTable:  "partner_update_fail",
		}
		if err := db.Create(tip).Error; err != nil {
			t.Fatalf("create tip: %v", err)
		}
		victimID := xid.New().String()
		if err := db.Create(&metadata.ModelData{
			Module: "owner", Name: "row", Application: "partner", ModelName: "UpdateFail", ModelId: victimID, ResID: xid.New().String(),
		}).Error; err != nil {
			t.Fatalf("create mapping: %v", err)
		}
		if err := db.Exec(`
CREATE TRIGGER block_meta_model_data_rebind
BEFORE UPDATE OF model_id ON meta_model_data
BEGIN
  SELECT RAISE(ABORT, 'rebind blocked');
END`).Error; err != nil {
			t.Fatalf("create rebind trigger: %v", err)
		}
		err := rebindMetaModelDataTips(db, []modelVictim{{
			Id: victimID, Application: "partner", Name: "UpdateFail",
		}})
		if err == nil || !strings.Contains(err.Error(), "error rebinding meta model data") {
			t.Fatalf("update error = %v, want rebinding failure", err)
		}
	})
}

func TestModuleUninstallerCleanModelsRebindError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&metadata.ModelData{}, &meta.Model{}, &meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	baseMod := &meta.Module{Name: "partner_base", Status: meta.Installed, Version: "1.0.0"}
	baseMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(baseMod).Error; err != nil {
		t.Fatalf("create base module: %v", err)
	}
	extMod := &meta.Module{Name: "partner_ext", Status: meta.Installed, Version: "1.0.0"}
	extMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(extMod).Error; err != nil {
		t.Fatalf("create ext module: %v", err)
	}

	baseModel := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        "@/partner_base/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    baseMod.Id,
	}
	if err := db.Create(baseModel).Error; err != nil {
		t.Fatalf("create base model: %v", err)
	}
	extModel := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        "@/partner_ext/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    extMod.Id,
		Extends:     baseModel.Path,
	}
	if err := db.Create(extModel).Error; err != nil {
		t.Fatalf("create ext model: %v", err)
	}
	if err := db.Create(&metadata.ModelData{
		Module: "seed_owner", Name: "partner_main", Application: "partner", ModelName: "Partner",
		ModelId: extModel.Id.String, ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}
	if err := db.Exec(`
CREATE TRIGGER block_meta_model_data_rebind_clean
BEFORE UPDATE OF model_id ON meta_model_data
BEGIN
  SELECT RAISE(ABORT, 'rebind blocked');
END`).Error; err != nil {
		t.Fatalf("create rebind trigger: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        extMod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error rebinding meta model data") {
		t.Fatalf("cleanModels() error = %v, want rebind failure", err)
	}
}
