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
	if isSafeSQLIdent("") || isSafeSQLIdent("a;drop") || isSafeSQLIdent("a b") || isSafeSQLIdent("123table") {
		t.Fatal("expected unsafe idents to be rejected")
	}
	if !isSafeSQLIdent("demo_seed_row") || !isSafeSQLIdent("SandboxCategory") || !isSafeSQLIdent("_private") {
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

func TestSnapshotModuleSeedRowsEdgeCases(t *testing.T) {
	t.Parallel()
	if snaps, err := snapshotModuleSeedRows(nil, "demo"); err != nil || snaps != nil {
		t.Fatalf("nil db: snaps=%v err=%v", snaps, err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if snaps, err := snapshotModuleSeedRows(db, "  "); err != nil || snaps != nil {
		t.Fatalf("empty module: snaps=%v err=%v", snaps, err)
	}
	if snaps, err := snapshotModuleSeedRows(db, "no_such_module"); err != nil || len(snaps) != 0 {
		t.Fatalf("no mappings: snaps=%v err=%v", snaps, err)
	}

	seed := seedCleanModelsFixture(t, db)
	businessTable := "demo_seed_snap"
	if err := db.Exec(`CREATE TABLE ` + businessTable + ` (id TEXT PRIMARY KEY)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	modelID := ensureEffectiveSeedModel(t, db, "demo", "SnapItem", businessTable)
	resID := xid.New().String()
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "with_empty_resid", Application: "demo", ModelName: "SnapItem", ModelId: modelID, ResID: "  ",
	}).Error; err != nil {
		t.Fatalf("create empty resid mapping: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "ok_row", Application: "demo", ModelName: "SnapItem", ModelId: modelID, ResID: resID,
	}).Error; err != nil {
		t.Fatalf("create ok mapping: %v", err)
	}
	snaps, err := snapshotModuleSeedRows(db, seed.module.Name)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ResID != resID || snaps[0].TableName != businessTable {
		t.Fatalf("snaps = %+v, want one row for %s", snaps, resID)
	}
}

func TestSnapshotModuleSeedRowsUnsafeTable(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	seed := seedCleanModelsFixture(t, db)
	modelID := ensureEffectiveSeedModel(t, db, "demo", "BadTable", "123bad")
	if err := db.Create(&modmeta.ModelData{
		Module: seed.module.Name, Name: "unsafe", Application: "demo", ModelName: "BadTable", ModelId: modelID, ResID: xid.New().String(),
	}).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}
	_, err := snapshotModuleSeedRows(db, seed.module.Name)
	if err == nil || !strings.Contains(err.Error(), "unsafe model_table") {
		t.Fatalf("err = %v, want unsafe model_table", err)
	}
}

func TestResolveSeedTableNameFallbacks(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if _, err := resolveSeedTableName(db, nil); err == nil || !strings.Contains(err.Error(), "nil model data") {
		t.Fatalf("nil mapping err = %v", err)
	}

	table := "demo_resolve_by_app"
	if err := db.Exec(`CREATE TABLE ` + table + ` (id TEXT PRIMARY KEY)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	_ = ensureEffectiveSeedModel(t, db, "demo", "ResolveByApp", table)

	got, err := resolveSeedTableName(db, &modmeta.ModelData{
		Module: "demo", Name: "x", Application: "demo", ModelName: "ResolveByApp", ModelId: "",
	})
	if err != nil || got != table {
		t.Fatalf("app+name resolve = %q err=%v, want %s", got, err, table)
	}

	emptyID := ensureEffectiveSeedModel(t, db, "demo", "EmptyTableModel", "")
	if _, err := resolveSeedTableName(db, &modmeta.ModelData{
		Module: "demo", Name: "empty_tbl", ModelId: emptyID,
	}); err == nil || !strings.Contains(err.Error(), "empty model_table") {
		t.Fatalf("empty model_table err = %v", err)
	}
}

func TestDialectNeedsDeleteSavepoint(t *testing.T) {
	t.Parallel()
	if dialectNeedsDeleteSavepoint(nil) {
		t.Fatal("nil db must be false")
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if dialectNeedsDeleteSavepoint(db) {
		t.Fatal("sqlite must not require savepoints")
	}
}

func TestHardDeleteSeedRowWithSavepoint(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	table := "demo_seed_sp"
	if err := db.Exec(`CREATE TABLE ` + table + ` (id TEXT PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	id := xid.New().String()
	if err := db.Exec(`INSERT INTO `+table+`(id, name) VALUES (?, ?)`, id, "a").Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
	}
	defer func() { _ = tx.Rollback() }()

	snap := seedRowSnapshot{Module: "demo", Name: "row", ResID: id, TableName: table}
	if err := hardDeleteSeedRow(tx, snap, true, 0); err != nil {
		t.Fatalf("hardDelete with savepoint: %v", err)
	}
	if err := hardDeleteSeedRow(tx, seedRowSnapshot{TableName: "1bad", ResID: id}, true, 1); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unsafe table err = %v", err)
	}
	// Missing row still succeeds (0 rows); savepoint release path still runs.
	if err := hardDeleteSeedRow(tx, seedRowSnapshot{Module: "demo", Name: "gone", ResID: xid.New().String(), TableName: table}, true, 2); err != nil {
		t.Fatalf("delete missing id: %v", err)
	}
	if err := releaseSeedDeleteSavepoint(nil, "sp_x"); err == nil {
		t.Fatal("nil db release must fail")
	}
	if err := releaseSeedDeleteSavepoint(tx, "1bad"); err == nil {
		t.Fatal("unsafe savepoint name must fail")
	}
}

func TestDeleteSeedRowsForSnapshotsNilAndEmpty(t *testing.T) {
	t.Parallel()
	if err := deleteSeedRowsForSnapshots(nil, nil); err != nil {
		t.Fatalf("nil: %v", err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	if err := deleteSeedRowsForSnapshots(runtimeScope.Session().DB, nil); err != nil {
		t.Fatalf("empty snaps: %v", err)
	}
}

func TestHardDeleteSeedRowSavepointRollbackOnError(t *testing.T) {
	t.Parallel()
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
	}
	defer func() { _ = tx.Rollback() }()

	err := hardDeleteSeedRow(tx, seedRowSnapshot{
		Module: "demo", Name: "x", ResID: "r1", TableName: "table_that_does_not_exist",
	}, true, 3)
	if err == nil {
		t.Fatal("expected missing-table delete error")
	}
	// Transaction must still be usable after savepoint rollback.
	if err := tx.Exec(`SELECT 1`).Error; err != nil {
		t.Fatalf("tx after savepoint rollback: %v", err)
	}
}
