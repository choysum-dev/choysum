// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
)

func TestModuleUninstallerCleanModelsRecomputesAfterExtRawDelete(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
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
	baseRawID := xid.New().String()
	if _, err := modmeta.ReplaceModuleDeclarations(db, baseMod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: baseRawID, Valid: true}},
		Name:        "Partner",
		Path:        basePath,
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    baseMod.Id,
	}}); err != nil {
		t.Fatalf("create base raw model: %v", err)
	}
	extRawID := xid.New().String()
	if _, err := modmeta.ReplaceModuleDeclarations(db, extMod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: extRawID, Valid: true}},
		Name:        "Partner",
		Path:        "@/partner_commercial/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    extMod.Id,
		Extends:     basePath,
	}}); err != nil {
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

	var extCount int64
	if err := db.Unscoped().Table("meta_raw_model").Where("id = ?", extRawID).Count(&extCount).Error; err != nil {
		t.Fatalf("count ext raw: %v", err)
	}
	if extCount != 0 {
		t.Fatalf("ext raw model should be hard-deleted, count=%d", extCount)
	}

	var baseRawCount int64
	if err := db.Table("meta_raw_model").Where("id = ?", baseRawID).Count(&baseRawCount).Error; err != nil {
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
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)
	modelID := ensureEffectiveSeedModel(t, db, "demo", "Item", "demo_seed_trigger_table")
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "seed_row", Application: "demo", ModelName: "Item", ModelId: modelID, ResID: xid.New().String(),
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
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)
	dropMetaTable(t, db, "meta_raw_model")

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error removing module declarations") {
		t.Fatalf("cleanModels() error = %v, want removing module declarations failure", err)
	}
}
