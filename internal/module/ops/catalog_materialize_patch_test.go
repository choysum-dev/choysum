// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCatalogMaterialize_PersistSyncResultBranches(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if err := o.persistSyncResult(context.Background(), nil, SyncResultSummary{}); err == nil || !strings.Contains(err.Error(), "nil operation") {
		t.Fatalf("nil row: %v", err)
	}

	row := &modmeta.ModuleOperation{
		Action: ActionSync, Status: StatusRunning,
		PlanJson: datatypes.JSON(`{`),
	}
	rowWhitespaceBase := &modmeta.ModuleOperation{
		Action: ActionSync, Status: StatusRunning,
		PlanJson: datatypes.JSON(`{"baseRevision":"   "}`),
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed op: %v", err)
	}
	if err := db.Create(rowWhitespaceBase).Error; err != nil {
		t.Fatalf("seed whitespace base op: %v", err)
	}
	if err := o.persistSyncResult(context.Background(), row, SyncResultSummary{Origin: "all"}); err != nil {
		t.Fatalf("persist with bad plan json: %v", err)
	}
	if !strings.Contains(string(row.PlanJson), `"sync"`) {
		t.Fatalf("expected base revision sync in plan json, got %s", row.PlanJson)
	}
	if err := o.persistSyncResult(context.Background(), rowWhitespaceBase, SyncResultSummary{Origin: "all"}); err != nil {
		t.Fatalf("persist whitespace base: %v", err)
	}
	if !strings.Contains(string(rowWhitespaceBase.PlanJson), `"sync"`) {
		t.Fatalf("whitespace base revision not normalized: %s", rowWhitespaceBase.PlanJson)
	}

	prev := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("marshal fail") }
	t.Cleanup(func() { jsonMarshal = prev })
	if err := o.persistSyncResult(context.Background(), row, SyncResultSummary{}); err == nil || !strings.Contains(err.Error(), "marshal fail") {
		t.Fatalf("marshal fail: %v", err)
	}
}

func TestCatalogMaterialize_CountCatalogRowsErrors(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	closeOpsDB(t, db)
	if _, err := o.countCatalogRows(context.Background()); err == nil {
		t.Fatal("expected count error after close")
	}
	oNil := &Ops{Scope: &opsTestScope{ctx: context.Background(), session: &scope.Session{}}}
	if _, err := oNil.countCatalogRows(context.Background()); err == nil {
		t.Fatal("expected session error without db")
	}
}

func TestCatalogMaterialize_DependTableCreateAndReplaceEdges(t *testing.T) {
	t.Parallel()
	if err := ensureModuleCatalogDependTable(nil); err == nil || !strings.Contains(err.Error(), "nil db") {
		t.Fatalf("nil db: %v", err)
	}

	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Migrator().DropTable(&modmeta.ModuleCatalogDepend{}); err != nil {
		t.Fatalf("drop depend table: %v", err)
	}

	baseDepends := datatypes.JSON(`["base"]`)
	for _, name := range []string{"base", "auth"} {
		row := modmeta.ModuleCatalog{
			ModuleName: name,
			Installed:  true,
			DependsJson: func() datatypes.JSON {
				if name == "auth" {
					return baseDepends
				}
				return datatypes.JSON("[]")
			}(),
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed catalog %s: %v", name, err)
		}
	}
	if err := o.replaceCatalogDependsEdges(context.Background()); err != nil {
		t.Fatalf("replace edges: %v", err)
	}
	var edgeCount int64
	if err := db.Model(&modmeta.ModuleCatalogDepend{}).Count(&edgeCount).Error; err != nil {
		t.Fatalf("count edges: %v", err)
	}
	if edgeCount != 1 {
		t.Fatalf("edge count = %d, want 1 auth→base", edgeCount)
	}

	if err := db.Where("1=1").Delete(&modmeta.ModuleCatalogDepend{}).Error; err != nil {
		t.Fatalf("clear edges: %v", err)
	}
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "auth").Update("depends_json", datatypes.JSON("[]")).Error; err != nil {
		t.Fatalf("clear depends: %v", err)
	}
	if err := o.replaceCatalogDependsEdges(context.Background()); err != nil {
		t.Fatalf("replace empty edges: %v", err)
	}
	if err := db.Model(&modmeta.ModuleCatalogDepend{}).Count(&edgeCount).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if edgeCount != 0 {
		t.Fatalf("expected 0 edges after full clear, got %d", edgeCount)
	}
}

func TestCatalogMaterialize_LifecycleTimesAndRefreshErrors(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	now := time.Now().UTC()

	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "timed", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	installAt := now.Add(-2 * time.Hour)
	upgradeAt := now.Add(-time.Hour)
	zeroTime := time.Time{}
	for _, op := range []modmeta.ModuleOperation{
		{Action: ActionInstall, ModuleName: "timed", Status: StatusSucceeded, FinishedAt: &installAt},
		{Action: ActionUpgrade, ModuleName: "timed", Status: StatusSucceeded, FinishedAt: &upgradeAt},
		{Action: ActionInstall, ModuleName: " ", Status: StatusSucceeded, FinishedAt: &now},
		{Action: ActionInstall, ModuleName: "timed", Status: StatusSucceeded, FinishedAt: &zeroTime},
		{Action: ActionInstall, ModuleName: "timed", Status: StatusRunning, FinishedAt: &now},
	} {
		if err := db.Create(&op).Error; err != nil {
			t.Fatalf("seed op: %v", err)
		}
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"timed"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}

	if err := db.Migrator().DropTable(&modmeta.ModuleOperation{}); err != nil {
		t.Fatalf("drop operations: %v", err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "broken", SourceType: originLocal, SourceRef: "local", Available: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"broken"}); err == nil {
		t.Fatal("expected lifecycle times query error")
	}

	db2 := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db2)}
	if err := db2.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "a", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db2.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "b", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	const blockRefresh = "ops_block_catalog_dependents_query"
	if err := db2.Callback().Query().Before("gorm:query").Register(blockRefresh, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "meta_module_catalog" {
			return
		}
		for _, col := range tx.Statement.Selects {
			if col == "dependents_json" {
				_ = tx.AddError(errors.New("refresh find blocked"))
				return
			}
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	t.Cleanup(func() { _ = db2.Callback().Query().Remove(blockRefresh) })
	if err := o2.MaterializeCatalog(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "refresh find blocked") {
		t.Fatalf("refresh dependents err: %v", err)
	}
}

func TestReplaceCatalogDependsEdges_SessionAndQueryErrors(t *testing.T) {
	t.Parallel()
	o := &Ops{Scope: &opsTestScope{ctx: context.Background(), session: &scope.Session{}}}
	if err := o.replaceCatalogDependsEdges(context.Background()); err == nil {
		t.Fatal("expected session error")
	}
	db := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db)}
	closeOpsDB(t, db)
	if err := o2.replaceCatalogDependsEdges(context.Background()); err == nil {
		t.Fatal("expected find error after close")
	}
}

func TestReplaceCatalogDependsEdges_FindAndDeleteErrors(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	const blockFind = "ops_block_catalog_depends_find"
	if err := db.Callback().Query().Before("gorm:query").Register(blockFind, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "meta_module_catalog" {
			return
		}
		for _, col := range tx.Statement.Selects {
			if col == "depends_json" {
				_ = tx.AddError(errors.New("depends find blocked"))
				return
			}
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(blockFind) })
	if err := o.replaceCatalogDependsEdges(context.Background()); err == nil || !strings.Contains(err.Error(), "depends find blocked") {
		t.Fatalf("find blocked: %v", err)
	}

	db2 := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db2)}
	for _, row := range []modmeta.ModuleCatalog{
		{ModuleName: "base", DependsJson: datatypes.JSON(`[]`)},
		{ModuleName: "leaf", DependsJson: datatypes.JSON(`["base"]`)},
	} {
		if err := db2.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	const blockDelete = "ops_block_depend_delete"
	if err := db2.Callback().Delete().Before("gorm:delete").Register(blockDelete, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog_depend" {
			_ = tx.AddError(errors.New("delete blocked"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db2.Callback().Delete().Remove(blockDelete) })
	if err := o2.replaceCatalogDependsEdges(context.Background()); err == nil || !strings.Contains(err.Error(), "delete blocked") {
		t.Fatalf("delete blocked: %v", err)
	}
}

func TestDependsNamesForEdges(t *testing.T) {
	t.Parallel()
	if dependsNamesForEdges(nil) != nil {
		t.Fatal("empty raw should be nil")
	}
	if dependsNamesForEdges(datatypes.JSON(`{`)) != nil {
		t.Fatal("bad json should be nil")
	}
	got := dependsNamesForEdges(datatypes.JSON(`[" base ","","auth"]`))
	if len(got) != 2 || got[0] != "base" || got[1] != "auth" {
		t.Fatalf("normalize = %v", got)
	}
}

func TestReplaceCatalogDependsEdges_DedupesDuplicateDepends(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	for _, name := range []string{"base", "auth"} {
		if err := db.Create(&modmeta.ModuleCatalog{ModuleName: name, DependsJson: datatypes.JSON("[]")}).Error; err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "auth").
		Update("depends_json", datatypes.JSON(`["base","base"]`)).Error; err != nil {
		t.Fatalf("set depends: %v", err)
	}
	if err := o.replaceCatalogDependsEdges(context.Background()); err != nil {
		t.Fatalf("replace: %v", err)
	}
	var n int64
	if err := db.Model(&modmeta.ModuleCatalogDepend{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("edge count = %d, want 1 after duplicate depends dedupe", n)
	}
}

func TestReplaceCatalogDependsEdges_SkipsInvalidRows(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Exec(`INSERT INTO meta_module_catalog (id, module_name, depends_json, created_at, updated_at)
		VALUES ('', 'noop', '[]', datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("seed empty id: %v", err)
	}
	selfRow := modmeta.ModuleCatalog{ModuleName: "solo", DependsJson: datatypes.JSON(`["solo"]`)}
	if err := db.Create(&selfRow).Error; err != nil {
		t.Fatalf("seed self dep: %v", err)
	}
	if err := o.replaceCatalogDependsEdges(context.Background()); err != nil {
		t.Fatalf("replace: %v", err)
	}
	var n int64
	if err := db.Model(&modmeta.ModuleCatalogDepend{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("edge count = %d, want 0 after skips", n)
	}
}

func TestRegistryOriginFromSnapshot_Branches(t *testing.T) {
	t.Parallel()
	if pkg, tarball := registryOriginFromSnapshot(nil); pkg != "" || tarball != "" {
		t.Fatalf("nil snap = %q %q", pkg, tarball)
	}
	snap := &modmeta.ModuleSourceSnapshot{
		Version: nullString("1.0.0"),
		ManifestJson: datatypes.JSON(`{
			"package":"@scope/mod",
			"versions":{
				"1.0.0":{"package":"@scope/mod","tarball":"https://ex/t.tgz"},
				"v1.0.0":{"tarball":"https://ex/raw.tgz"}
			}
		}`),
	}
	pkg, tarball := registryOriginFromSnapshot(snap)
	if pkg != "@scope/mod" || tarball != "https://ex/t.tgz" {
		t.Fatalf("semver key = %q %q", pkg, tarball)
	}
	fromVersions := &modmeta.ModuleSourceSnapshot{
		Version: nullString("1.0.0"),
		ManifestJson: datatypes.JSON(`{"versions":{"1.0.0":{"package":"@from/entry","tarball":"https://ex/entry.tgz"}}}`),
	}
	pkg3, tb3 := registryOriginFromSnapshot(fromVersions)
	if pkg3 != "@from/entry" || tb3 != "https://ex/entry.tgz" {
		t.Fatalf("version map entry package = %q %q", pkg3, tb3)
	}
	snap2 := &modmeta.ModuleSourceSnapshot{
		Version:      nullString("v9.9.9"),
		ManifestJson: datatypes.JSON(`{"versions":{"v9.9.9":{"package":"@scope/raw","tarball":"https://ex/raw-only.tgz"}}}`),
	}
	pkg2, tb2 := registryOriginFromSnapshot(snap2)
	if tb2 != "https://ex/raw-only.tgz" || pkg2 != "@scope/raw" {
		t.Fatalf("raw version key = %q %q", pkg2, tb2)
	}
	bad := &modmeta.ModuleSourceSnapshot{ManifestJson: datatypes.JSON(`{`)}
	if p, _ := registryOriginFromSnapshot(bad); p != "" {
		t.Fatalf("bad json package = %q", p)
	}
}

func TestLoadCatalogLifecycleTimes_NilDB(t *testing.T) {
	t.Parallel()
	got, err := loadCatalogLifecycleTimes(nil, nil, false)
	if err != nil || len(got) != 0 {
		t.Fatalf("nil db => %#v %v", got, err)
	}
}

func TestExecuteSync_MaterializeAndPersistErrors(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return testLocker{}
		},
		SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{Total: 1, Success: 1}, nil
		},
		SyncRegistrySnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
		},
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	row := &modmeta.ModuleOperation{Action: ActionSync, Status: StatusRunning, PlanJson: datatypes.JSON(`{"baseRevision":"sync"}`)}
	if err := db.Create(row).Error; err != nil {
		t.Fatal(err)
	}

	const blockMat = "ops_block_catalog_mat"
	if err := db.Callback().Create().Before("gorm:create").Register(blockMat, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("materialize blocked"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove(blockMat) })
	if err := o.executeSync(context.Background(), row); err == nil || !strings.Contains(err.Error(), "materialize blocked") {
		t.Fatalf("materialize error: %v", err)
	}

	db2 := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db2)}
	row2 := &modmeta.ModuleOperation{Action: ActionSync, Status: StatusRunning, PlanJson: datatypes.JSON(`{"baseRevision":"sync"}`)}
	if err := db2.Create(row2).Error; err != nil {
		t.Fatal(err)
	}
	prev := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("persist marshal") }
	if err := o2.executeSync(context.Background(), row2); err == nil || !strings.Contains(err.Error(), "persist marshal") {
		jsonMarshal = prev
		t.Fatalf("persist error: %v", err)
	}
	jsonMarshal = prev

	db3 := newOpsTestDB(t)
	o3 := &Ops{
		Scope: newOpsTestScope(t, db3),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return testLocker{}
		},
		SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{Total: 1, Success: 1}, nil
		},
		SyncRegistrySnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
		},
	}
	row3 := &modmeta.ModuleOperation{Action: ActionSync, Status: StatusRunning, PlanJson: datatypes.JSON(`{"baseRevision":"sync"}`)}
	if err := db3.Create(row3).Error; err != nil {
		t.Fatal(err)
	}
	const blockCount = "ops_block_catalog_count"
	if err := db3.Callback().Query().Before("gorm:query").Register(blockCount, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			if _, ok := tx.Statement.Dest.(*int64); ok {
				_ = tx.AddError(errors.New("count blocked"))
			}
		}
	}); err != nil {
		t.Fatalf("count callback: %v", err)
	}
	t.Cleanup(func() { _ = db3.Callback().Query().Remove(blockCount) })
	// countCatalogRows fails softly; materialize on empty snapshots still succeeds.
	if err := o3.executeSync(context.Background(), row3); err != nil {
		t.Fatalf("executeSync with count error: %v", err)
	}
}
