// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestApply_StampActiveOpFailureFailsTerminal(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core", Status: pkgmeta.Installed}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}, load: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	if err := db.Create(&modmeta.ModuleCatalog{ModuleName: "core", Installed: true}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionUpgrade})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("fail_catalog_stamp", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("stamp boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	err = o.Apply(context.Background(), ApplyRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "stamp boom") {
		t.Fatalf("Apply err = %v", err)
	}
	row, loadErr := o.loadOperation(context.Background(), result.Id)
	if loadErr != nil {
		t.Fatalf("load: %v", loadErr)
	}
	if row.Status != StatusFailed {
		t.Fatalf("status = %q, want failed after stamp failure", row.Status)
	}
}

func TestRun_StampActiveOpFailureFailsTerminal(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	if err := db.Create(&modmeta.ModuleCatalog{ModuleName: "core", Installed: false}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// Apply without stamp failure first.
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Clear the stamp so Run's setCatalogActiveOp runs again, then fail catalog updates.
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "core").
		Update("active_operation_id", nil).Error; err != nil {
		t.Fatalf("clear stamp: %v", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("fail_run_stamp", func(tx *gorm.DB) {
		if tx.Statement == nil {
			return
		}
		if tx.Statement.Table != "meta_module_catalog" {
			return
		}
		// Fail only active_operation_id stamp (Valid true), not the clear after fail.
		_ = tx.AddError(errors.New("run stamp boom"))
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "run stamp boom") {
		t.Fatalf("Run err = %v", err)
	}
}

func TestCatalogActiveOp_EmptyAndSessionGuards(t *testing.T) {
	t.Parallel()
	o := &Ops{}
	if err := o.setCatalogActiveOp(context.Background(), []string{"a"}, " "); err != nil {
		t.Fatalf("empty opID: %v", err)
	}
	if err := o.setCatalogActiveOp(context.Background(), []string{" ", ""}, "op1"); err != nil {
		t.Fatalf("empty names: %v", err)
	}
	if err := o.clearCatalogActiveOp(context.Background(), " "); err != nil {
		t.Fatalf("clear empty: %v", err)
	}
	if err := o.setCatalogActiveOp(context.Background(), []string{"core"}, "op1"); err == nil {
		t.Fatal("expected session error")
	}
	if err := o.clearCatalogActiveOp(context.Background(), "op1"); err == nil {
		t.Fatal("expected clear session error")
	}

	if names := operationModuleNames(nil, PlanDocument{}); len(names) != 0 {
		t.Fatalf("empty names = %v", names)
	}
	if names := operationModuleNames(&modmeta.ModuleOperation{ModuleName: " root "}, PlanDocument{
		ModuleOrder: []ModuleOrderEntry{{ModuleName: "root"}, {ModuleName: " "}},
	}); len(names) != 1 || names[0] != "root" {
		t.Fatalf("dedupe names = %v", names)
	}
}

func TestMaterializeCatalog_ClearsDemoLoadedWhenUninstalled(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, InstalledVersion: nullString("1.0.0"),
		HasDemo: true, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Local snapshot keeps HasDemo via manifest; installed row is gone.
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "base", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("1.0.0"), ManifestJson: datatypes.JSON(`{"demo":["demo/a.json"]}`),
	}).Error; err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"base"}); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&row).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Installed {
		t.Fatal("expected not installed")
	}
	if row.DemoLoaded {
		t.Fatal("DemoLoaded must clear when uninstalled")
	}
}

func TestMaterializeCatalog_PreservesConcurrentDemoLoaded(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "base",
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, InstalledVersion: nullString("1.0.0"),
		HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	// Simulate race: markDemoLoaded after MaterializeCatalog read prior=false.
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "base").
		Update("demo_loaded", true).Error; err != nil {
		t.Fatalf("mark: %v", err)
	}
	// Re-run materialize with a fresh Ops that still sees prior from DB (true now).
	// Force stale false by writing via CreateInBatches path with prior false:
	// call materializeOne logic through Materialize after resetting prior in DB to true
	// and ensuring upsert CASE preserves it when excluded would be false.
	// Directly exercise upsert: set DemoLoaded false in a crafted row via prior empty.
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "base").
		Updates(map[string]any{"demo_loaded": true, "installed_version": "1.0.0"}).Error; err != nil {
		t.Fatalf("prep: %v", err)
	}
	// Delete prior so Materialize reads DemoLoaded=false from missing prior, then
	// re-insert concurrent true before upsert finishes — hard to race in unit test.
	// Instead: seed prior=false, mark true, Materialize with installed same version —
	// materializeOne sets DemoLoaded=prior.DemoLoaded=false, but SQL CASE preserves true.
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "base").
		Update("demo_loaded", false).Error; err != nil {
		t.Fatalf("reset: %v", err)
	}
	// Concurrent win: set true after "read" — we can't inject mid-function, so
	// update to true then run Materialize while forcing prior false via deleting
	// and recreating with false, then updating to true before CreateInBatches.
	// Simpler: call Materialize when prior is false, but update DB to true inside
	// a Before Create callback.
	if err := db.Callback().Create().Before("gorm:create").Register("race_demo_loaded", func(tx *gorm.DB) {
		_ = tx.Session(&gorm.Session{NewDB: true}).Model(&modmeta.ModuleCatalog{}).
			Where("module_name = ?", "base").Update("demo_loaded", true).Error
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"base"}); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&row).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if !row.DemoLoaded {
		t.Fatal("expected DemoLoaded preserved against stale false upsert")
	}
}

func TestRefreshCatalogDependents_EmptyAndSession(t *testing.T) {
	t.Parallel()
	o := &Ops{}
	if err := o.refreshCatalogDependents(context.Background()); err == nil {
		t.Fatal("expected session error")
	}
	db := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db)}
	if err := o2.refreshCatalogDependents(context.Background()); err != nil {
		t.Fatalf("empty catalog: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{ModuleName: "solo"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	closeOpsDB(t, db)
	if err := o2.refreshCatalogDependents(context.Background()); err == nil {
		t.Fatal("expected find error after close")
	}
}

func TestApply_CASSaveFailure(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("fail_apply_cas", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_operation" {
			_ = tx.AddError(errors.New("cas boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err == nil || !strings.Contains(err.Error(), "cas boom") {
		t.Fatalf("Apply err = %v", err)
	}
}

func TestSucceedAndAbandon_ClearActiveOpFailure(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{
		Scope:  newOpsTestScope(t, db),
		Events: &recordingBus{},
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &leaseTestLocker{}
		},
		SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
		},
		SyncRegistrySnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
		},
	}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionSync, Origin: originLocal})
	if err != nil {
		t.Fatalf("Plan sync: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "core", ActiveOperationId: nullString(result.Id),
	}).Error; err != nil {
		t.Fatalf("catalog: %v", err)
	}
	armed := false
	if err := db.Callback().Update().Before("gorm:update").Register("fail_clear_after_success", func(tx *gorm.DB) {
		if tx.Statement == nil {
			return
		}
		if tx.Statement.Table == "meta_module_operation" {
			if st, ok := tx.Statement.Dest.(map[string]any); ok {
				if status, _ := st["status"].(string); status == StatusSucceeded {
					armed = true
				}
			}
			return
		}
		if armed && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("clear boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	// Clear failure after terminal CAS must not fail the caller.
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err != nil {
		t.Fatalf("Run after clear boom: %v", err)
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", row.Status)
	}

	db2 := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db2), Events: &recordingBus{}, Resolver: &mapResolver{}}
	draft, err := o2.Plan(context.Background(), PlanRequest{ModuleName: "missing", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan draft: %v", err)
	}
	if err := db2.Create(&modmeta.ModuleCatalog{ModuleName: "missing", ActiveOperationId: nullString(draft.Id)}).Error; err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if err := db2.Callback().Update().Before("gorm:update").Register("fail_abandon_clear", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("abandon clear boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := o2.AbandonDraft(context.Background(), draft.Id, "bye"); err != nil {
		t.Fatalf("AbandonDraft after clear boom: %v", err)
	}
}

func TestWithModuleManagementLease_NonPositiveTTLInterval(t *testing.T) {
	// Not parallel: mutates package-level moduleManagementLeaseTTL.
	db := newOpsTestDB(t)
	prevTTL := moduleManagementLeaseTTL
	moduleManagementLeaseTTL = 0
	t.Cleanup(func() { moduleManagementLeaseTTL = prevTTL })
	locker := &leaseTestLocker{}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	if err := o.withModuleManagementLease(context.Background(), func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
}

func TestRefreshCatalogDependents_UpdateFailure(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&pkgmeta.Module{
		Name: "auth", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "auth",
		DependsStr: datatypes.JSON(`["base"]`),
	}).Error; err != nil {
		t.Fatalf("auth: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "base",
	}).Error; err != nil {
		t.Fatalf("base: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), nil); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	// Clear dependents so refresh must Update, then fail Updates.
	if err := db.Model(&modmeta.ModuleCatalog{}).Where("module_name = ?", "base").
		Update("dependents_json", datatypes.JSON("[]")).Error; err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("fail_dep_refresh", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("dep update boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := o.refreshCatalogDependents(context.Background()); err == nil || !strings.Contains(err.Error(), "dep update boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_PlanDecodeFailureFailsTerminal(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionSync, Origin: originLocal})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Corrupt plan_json then flip action so Run attempts decode for stamp.
	if err := db.Model(&modmeta.ModuleOperation{}).Where("id = ?", result.Id).Updates(map[string]any{
		"action":    ActionLoadDemo,
		"plan_json": datatypes.JSON(`{`),
	}).Error; err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err == nil {
		t.Fatal("expected plan decode failure")
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusFailed || row.ErrorCode != "PLAN_DECODE_FAILED" {
		t.Fatalf("row = %#v", row)
	}
}

func TestMaterializeCatalog_PriorFindFailure(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&pkgmeta.Module{
		Name: "core", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "core",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Break Find on prior by dropping the catalog table after modules/snapshots load.
	if err := db.Migrator().DropTable(&modmeta.ModuleCatalog{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"core"}); err == nil {
		t.Fatal("expected prior find error")
	}
}

func TestWithModuleManagementLease_NilContextWithLocker(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	locker := &leaseTestLocker{}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	if err := o.withModuleManagementLease(nil, func(ctx context.Context) error {
		if ctx == nil {
			t.Fatal("expected background ctx")
		}
		return nil
	}); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestDecodeDependsNames_Edges(t *testing.T) {
	t.Parallel()
	if got := decodeDependsNames(nil); got != nil {
		t.Fatalf("nil = %v", got)
	}
	if got := decodeDependsNames(datatypes.JSON(`{`)); got != nil {
		t.Fatalf("bad json = %v", got)
	}
	if got := decodeDependsNames(datatypes.JSON(`["a"," a ","b"]`)); len(got) != 2 {
		t.Fatalf("dedupe = %v", got)
	}
}

func TestExecuteLoadDemo_MaterializeFailure(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, Path: t.TempDir(),
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("catalog: %v", err)
	}
	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	applyDemoFiles = func(context.Context, scope.Scope, *pkgmeta.Module) error { return nil }

	// Break materialize by closing DB after mark — register callback on Create.
	if err := db.Callback().Create().Before("gorm:create").Register("fail_materialize_batch", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("materialize boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "materialize") {
		t.Fatalf("Run err = %v", err)
	}
}
