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
	"gorm.io/gorm"
)

func ensureEffectiveSeedModel(t *testing.T, db *gorm.DB, application, name, modelTable string) string {
	t.Helper()
	modelID := xid.New().String()
	model := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: modelID, Valid: true}},
		Name:        name,
		Path:        application + "/" + name,
		Application: application,
		ModelTable:  modelTable,
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create effective meta_model: %v", err)
	}
	return modelID
}

func TestIsSafeSQLIdent(t *testing.T) {
	t.Parallel()
	if isSafeSQLIdent("") || isSafeSQLIdent("a;drop") || isSafeSQLIdent("a b") {
		t.Fatal("expected unsafe idents to be rejected")
	}
	if !isSafeSQLIdent("demo_seed_row") || !isSafeSQLIdent("SandboxCategory") {
		t.Fatal("expected safe idents to be accepted")
	}
}

func TestModuleUninstallerCascadesSeedRowsAndClearsMetaModelData(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB

	seed := seedCleanModelsFixture(t, db)
	otherModule := &meta.Module{Name: "other", Status: meta.Installed, Version: "1.0.0"}
	if err := db.Create(otherModule).Error; err != nil {
		t.Fatalf("create other module: %v", err)
	}

	businessTable := "demo_seed_row"
	if err := db.Exec(`CREATE TABLE ` + businessTable + ` (id TEXT PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatalf("create business table: %v", err)
	}
	modelID := ensureEffectiveSeedModel(t, db, "demo", "Item", businessTable)
	otherModelID := ensureEffectiveSeedModel(t, db, "other", "Item", businessTable)

	resID := xid.New().String()
	otherResID := xid.New().String()
	if err := db.Exec(`INSERT INTO `+businessTable+`(id, name) VALUES (?, ?)`, resID, "demo_seed").Error; err != nil {
		t.Fatalf("insert business row: %v", err)
	}
	if err := db.Exec(`INSERT INTO `+businessTable+`(id, name) VALUES (?, ?)`, otherResID, "other").Error; err != nil {
		t.Fatalf("insert other business row: %v", err)
	}

	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "seed_row", Application: "demo", ModelName: "Item", ModelId: modelID, ResID: resID,
	}).Error; err != nil {
		t.Fatalf("create demo mapping: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: otherModule.Name, Name: "other_row", Application: "other", ModelName: "Item", ModelId: otherModelID, ResID: otherResID,
	}).Error; err != nil {
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
	if err := db.Model(&modmeta.ModelData{}).Where("module = ?", seed.module.Name).Count(&remaining).Error; err != nil {
		t.Fatalf("count demo mappings: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("demo mappings remaining = %d, want 0", remaining)
	}

	var other modmeta.ModelData
	if err := db.Where("module = ? AND name = ?", otherModule.Name, "other_row").First(&other).Error; err != nil {
		t.Fatalf("other module mapping should remain: %v", err)
	}
	if other.ResID != otherResID {
		t.Fatalf("other mapping ResID = %q, want %q", other.ResID, otherResID)
	}

	var demoCount int64
	if err := db.Raw(`SELECT COUNT(1) FROM `+businessTable+` WHERE id = ?`, resID).Scan(&demoCount).Error; err != nil {
		t.Fatalf("query demo business row: %v", err)
	}
	if demoCount != 0 {
		t.Fatalf("demo seed row remaining = %d, want 0 (E15 cascade)", demoCount)
	}

	var otherName string
	if err := db.Raw(`SELECT name FROM `+businessTable+` WHERE id = ?`, otherResID).Scan(&otherName).Error; err != nil {
		t.Fatalf("query other business row: %v", err)
	}
	if otherName != "other" {
		t.Fatalf("other module seed name = %q, want other", otherName)
	}

	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "seed_row", Application: "demo", ModelName: "Item", ModelId: xid.New().String(), ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("recreate mapping after uninstall cleanup: %v", err)
	}
}

func TestModuleUninstallerCascadeSeedRowAlreadyMissing(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)

	businessTable := "demo_seed_missing"
	if err := db.Exec(`CREATE TABLE ` + businessTable + ` (id TEXT PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatalf("create business table: %v", err)
	}
	modelID := ensureEffectiveSeedModel(t, db, "demo", "MissingItem", businessTable)
	resID := xid.New().String()
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "gone_row", Application: "demo", ModelName: "MissingItem", ModelId: modelID, ResID: resID,
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
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
}

func TestModuleUninstallerCascadeMissingBusinessTableSkipped(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)

	modelID := ensureEffectiveSeedModel(t, db, "demo", "GoneTable", "demo_table_does_not_exist")
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "orphan_map", Application: "demo", ModelName: "GoneTable", ModelId: modelID, ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels() error = %v (missing table should be skipped)", err)
	}
}

func TestModuleUninstallerCascadeUnresolvedTableFails(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)

	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "bad_map", Application: "demo", ModelName: "NoSuchModel", ModelId: xid.New().String(), ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "cannot resolve model_table") {
		t.Fatalf("cleanModels() error = %v, want cannot resolve model_table", err)
	}
}

func TestModuleUninstallerCascadeParentChildMultiPass(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`PRAGMA foreign_keys = ON`).Error; err != nil {
		t.Fatalf("enable foreign_keys: %v", err)
	}
	seed := seedCleanModelsFixture(t, db)

	parentTable := "demo_seed_parent"
	childTable := "demo_seed_child"
	if err := db.Exec(`CREATE TABLE ` + parentTable + ` (id TEXT PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatalf("create parent table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE ` + childTable + ` (
		id TEXT PRIMARY KEY,
		parent_id TEXT NOT NULL,
		FOREIGN KEY(parent_id) REFERENCES ` + parentTable + `(id)
	)`).Error; err != nil {
		t.Fatalf("create child table: %v", err)
	}

	parentModelID := ensureEffectiveSeedModel(t, db, "demo", "Parent", parentTable)
	childModelID := ensureEffectiveSeedModel(t, db, "demo", "Child", childTable)
	parentID := xid.New().String()
	childID := xid.New().String()
	if err := db.Exec(`INSERT INTO `+parentTable+`(id, name) VALUES (?, ?)`, parentID, "root").Error; err != nil {
		t.Fatalf("insert parent: %v", err)
	}
	if err := db.Exec(`INSERT INTO `+childTable+`(id, parent_id) VALUES (?, ?)`, childID, parentID).Error; err != nil {
		t.Fatalf("insert child: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "parent", Application: "demo", ModelName: "Parent", ModelId: parentModelID, ResID: parentID,
	}).Error; err != nil {
		t.Fatalf("create parent mapping: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "child", Application: "demo", ModelName: "Child", ModelId: childModelID, ResID: childID,
	}).Error; err != nil {
		t.Fatalf("create child mapping: %v", err)
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

	var parentCount, childCount int64
	_ = db.Raw(`SELECT COUNT(1) FROM ` + parentTable).Scan(&parentCount)
	_ = db.Raw(`SELECT COUNT(1) FROM ` + childTable).Scan(&childCount)
	if parentCount != 0 || childCount != 0 {
		t.Fatalf("after cascade parent=%d child=%d, want 0/0", parentCount, childCount)
	}
}

func TestModuleUninstallerCascadeUnresolvableFKFails(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`PRAGMA foreign_keys = ON`).Error; err != nil {
		t.Fatalf("enable foreign_keys: %v", err)
	}
	seed := seedCleanModelsFixture(t, db)

	parentTable := "demo_fk_parent"
	childTable := "demo_fk_child"
	if err := db.Exec(`CREATE TABLE ` + parentTable + ` (id TEXT PRIMARY KEY)`).Error; err != nil {
		t.Fatalf("create parent table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE ` + childTable + ` (
		id TEXT PRIMARY KEY,
		parent_id TEXT NOT NULL,
		FOREIGN KEY(parent_id) REFERENCES ` + parentTable + `(id)
	)`).Error; err != nil {
		t.Fatalf("create child table: %v", err)
	}

	parentModelID := ensureEffectiveSeedModel(t, db, "demo", "FkParent", parentTable)
	parentID := xid.New().String()
	childID := xid.New().String()
	if err := db.Exec(`INSERT INTO `+parentTable+`(id) VALUES (?)`, parentID).Error; err != nil {
		t.Fatalf("insert parent: %v", err)
	}
	if err := db.Exec(`INSERT INTO `+childTable+`(id, parent_id) VALUES (?, ?)`, childID, parentID).Error; err != nil {
		t.Fatalf("insert child: %v", err)
	}
	// Only map the parent: child row is not a module seed, so cascade cannot clear the FK.
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "fk_parent", Application: "demo", ModelName: "FkParent", ModelId: parentModelID, ResID: parentID,
	}).Error; err != nil {
		t.Fatalf("create parent mapping: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        seed.module,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error cascade-deleting seed rows") {
		t.Fatalf("cleanModels() error = %v, want cascade-deleting failure", err)
	}

	var mappingCount int64
	if err := db.Model(&modmeta.ModelData{}).Where("module = ? AND name = ?", seed.module.Name, "fk_parent").Count(&mappingCount).Error; err != nil {
		t.Fatalf("count mapping: %v", err)
	}
	if mappingCount != 1 {
		t.Fatalf("mapping remaining = %d, want 1 (uninstall must abort before clearing mappings)", mappingCount)
	}
}

func TestModuleUninstallerCascadeThenReinstallUniqueCode(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)

	businessTable := "demo_sandbox_category"
	if err := db.Exec(`CREATE TABLE ` + businessTable + ` (
		id TEXT PRIMARY KEY,
		code TEXT NOT NULL UNIQUE
	)`).Error; err != nil {
		t.Fatalf("create business table: %v", err)
	}
	modelID := ensureEffectiveSeedModel(t, db, "demo", "SandboxCategory", businessTable)
	resID := xid.New().String()
	if err := db.Exec(`INSERT INTO `+businessTable+`(id, code) VALUES (?, ?)`, resID, "ROOT").Error; err != nil {
		t.Fatalf("insert seed: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "category_root", Application: "demo", ModelName: "SandboxCategory", ModelId: modelID, ResID: resID,
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
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

	// Simulate loader Create after reinstall: same business unique key must succeed.
	newID := xid.New().String()
	if err := db.Exec(`INSERT INTO `+businessTable+`(id, code) VALUES (?, ?)`, newID, "ROOT").Error; err != nil {
		t.Fatalf("reinstall insert with same code: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "category_root", Application: "demo", ModelName: "SandboxCategory", ModelId: modelID, ResID: newID,
	}).Error; err != nil {
		t.Fatalf("recreate mapping: %v", err)
	}
}
