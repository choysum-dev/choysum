// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestPlanAndRun_LoadDemoSetsDemoLoadedAndLatch(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}

	if err := db.Create(&pkgmeta.Module{
		Name:           "base",
		Version:        "1.0.0",
		Status:         pkgmeta.Installed,
		ApplicationStr: "base",
		Path:           t.TempDir(),
		DemoStr:        datatypes.JSON(`["demo/smoke.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName:       "base",
		Installed:        true,
		InstalledVersion: sql.NullString{String: "1.0.0", Valid: true},
		HasDemo:          true,
		DemoLoaded:       false,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	// Already-loaded / no-demo rows must be excluded from the plan.
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "core",
		Installed:  true,
		HasDemo:    true,
		DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed core catalog: %v", err)
	}

	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	var applied []string
	applyDemoFiles = func(_ context.Context, _ scope.Scope, mod *pkgmeta.Module) error {
		applied = append(applied, mod.Name)
		return nil
	}

	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(result.Blockers) != 0 {
		t.Fatalf("blockers = %#v", result.Blockers)
	}
	if len(result.ModuleOrder) != 1 || result.ModuleOrder[0].ModuleName != "base" {
		t.Fatalf("moduleOrder = %#v", result.ModuleOrder)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(applied) != 1 || applied[0] != "base" {
		t.Fatalf("applied = %#v", applied)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&row).Error; err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if !row.DemoLoaded {
		t.Fatal("expected DemoLoaded=true")
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected modules.with_demo latch")
	}
	op, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load op: %v", err)
	}
	if op.Status != StatusSucceeded {
		t.Fatalf("status = %q", op.Status)
	}
}

func TestPlan_LoadDemoNoPendingBlocker(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(result.ModuleOrder) != 0 {
		t.Fatalf("moduleOrder = %#v", result.ModuleOrder)
	}
	if len(result.Blockers) != 1 || result.Blockers[0].Code != "NO_DEMO_PENDING" {
		t.Fatalf("blockers = %#v", result.Blockers)
	}
}

func TestRun_LoadDemoPropagatesApplyError(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, Path: t.TempDir(),
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	applyDemoFiles = func(context.Context, scope.Scope, *pkgmeta.Module) error {
		return errors.New("demo boom")
	}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "demo boom") {
		t.Fatalf("Run = %v", err)
	}
}

func TestRun_LoadDemoMarksEachModuleBeforeNext(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	for _, name := range []string{"alpha", "beta"} {
		if err := db.Create(&pkgmeta.Module{
			Name: name, Version: "1.0.0", Status: pkgmeta.Installed, Path: t.TempDir(),
			DemoStr: datatypes.JSON(`["demo/a.json"]`),
		}).Error; err != nil {
			t.Fatalf("seed module %s: %v", name, err)
		}
		if err := db.Create(&modmeta.ModuleCatalog{
			ModuleName: name, Installed: true, HasDemo: true, DemoLoaded: false,
		}).Error; err != nil {
			t.Fatalf("seed catalog %s: %v", name, err)
		}
	}
	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	applyDemoFiles = func(_ context.Context, _ scope.Scope, mod *pkgmeta.Module) error {
		if mod.Name == "beta" {
			return errors.New("beta boom")
		}
		return nil
	}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "beta boom") {
		t.Fatalf("Run = %v", err)
	}
	var alpha, beta modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "alpha").Take(&alpha).Error; err != nil {
		t.Fatalf("alpha: %v", err)
	}
	if err := db.Where("module_name = ?", "beta").Take(&beta).Error; err != nil {
		t.Fatalf("beta: %v", err)
	}
	if !alpha.DemoLoaded {
		t.Fatal("alpha should be marked DemoLoaded after successful import")
	}
	if beta.DemoLoaded {
		t.Fatal("beta must stay unmarked after failed import")
	}
	// Latch is set on the first successful import (before mark), so a later
	// module failure does not leave with_demo unset after partial progress.
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected modules.with_demo after alpha succeeded")
	}
}

func TestRun_LoadDemoSkipsStaleAndUninstalled(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := db.Create(&pkgmeta.Module{
		Name: "gone", Version: "1.0.0", Status: pkgmeta.Uninstalled, Path: t.TempDir(),
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed gone: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name: "kept", Version: "1.0.0", Status: pkgmeta.Installed, Path: t.TempDir(),
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed kept: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "gone", Installed: false, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("catalog gone: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "kept", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("catalog kept: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "already", Installed: true, HasDemo: true, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("catalog already: %v", err)
	}

	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	var applied []string
	applyDemoFiles = func(_ context.Context, _ scope.Scope, mod *pkgmeta.Module) error {
		applied = append(applied, mod.Name)
		return nil
	}

	// Craft a queued plan that still lists stale names captured at an earlier Plan.
	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder: []ModuleOrderEntry{
			{ModuleName: "gone"},
			{ModuleName: "already"},
			{ModuleName: "kept"},
			{ModuleName: "  "},
		},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	op := &modmeta.ModuleOperation{
		Action: ActionLoadDemo, PlanJson: raw, Status: StatusQueued, Phase: PhaseQueued,
	}
	if err := db.Create(op).Error; err != nil {
		t.Fatalf("create op: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: op.Id.String}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(applied) != 1 || applied[0] != "kept" {
		t.Fatalf("applied = %#v", applied)
	}
}

func TestExecuteLoadDemo_EmptyOrderAndNilScope(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{
		PlanJson: datatypes.JSON(`{"moduleOrder":[{"moduleName":""}],"baseRevision":"load-demo"}`),
	}); err != nil {
		t.Fatalf("empty names: %v", err)
	}
	bare := &Ops{}
	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder:  []ModuleOrderEntry{{ModuleName: "base"}},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := bare.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{PlanJson: raw}); err == nil ||
		!strings.Contains(err.Error(), "scope is required") {
		t.Fatalf("nil scope = %v", err)
	}
	if err := o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{
		PlanJson: datatypes.JSON(`{`),
	}); err == nil {
		t.Fatal("expected bad plan json")
	}
}

func TestExecuteLoadDemo_SetWithDemoFailure(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, Path: t.TempDir(),
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	applyDemoFiles = func(context.Context, scope.Scope, *pkgmeta.Module) error { return nil }
	if err := db.Callback().Create().Before("gorm:create").Register("ops_fail_load_demo_latch", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_app_setting" {
			_ = tx.AddError(errors.New("forced latch fail"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}

	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder:  []ModuleOrderEntry{{ModuleName: "base"}},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{PlanJson: raw})
	if err == nil || !strings.Contains(err.Error(), "forced latch fail") {
		t.Fatalf("SetWithDemo fail: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&row).Error; err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if row.DemoLoaded {
		t.Fatal("DemoLoaded must stay false when latch write fails (retry can remount)")
	}
}

func TestExecuteLoadDemo_MarkDemoLoadedFailure(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, Path: t.TempDir(),
		DemoStr: datatypes.JSON(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	applyDemoFiles = func(context.Context, scope.Scope, *pkgmeta.Module) error { return nil }
	if err := db.Callback().Update().Before("gorm:update").Register("ops_fail_mark_demo", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("forced mark fail"))
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder:  []ModuleOrderEntry{{ModuleName: "base"}},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{PlanJson: raw})
	if err == nil || !strings.Contains(err.Error(), "forced mark fail") {
		t.Fatalf("mark fail: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("latch is set before mark; mark failure must leave with_demo true")
	}
}

func TestExecuteLoadDemo_AllStaleNoLatch(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, HasDemo: true, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	oldApply := applyDemoFiles
	defer func() { applyDemoFiles = oldApply }()
	applyDemoFiles = func(context.Context, scope.Scope, *pkgmeta.Module) error {
		t.Fatal("should not apply stale module")
		return nil
	}
	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder:  []ModuleOrderEntry{{ModuleName: "base"}},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{PlanJson: raw}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if o.ResolveWithDemo(context.Background()) {
		t.Fatal("latch must stay false when nothing loaded")
	}
}

func TestExecuteLoadDemo_MissingInstalledModule(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "ghost", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder:  []ModuleOrderEntry{{ModuleName: "ghost"}},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{PlanJson: raw})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("missing module: %v", err)
	}
}

func TestPlanLoadDemo_MarshalAndSessionErrors(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	prev := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("marshal boom") }
	_, err := o.planLoadDemo(context.Background(), "")
	jsonMarshal = prev
	if err == nil || !strings.Contains(err.Error(), "marshal boom") {
		t.Fatalf("marshal: %v", err)
	}

	if err := db.Callback().Create().Before("gorm:create").Register("ops_fail_plan_demo_create", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_operation" {
			_ = tx.AddError(errors.New("forced create fail"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if _, err := o.planLoadDemo(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "forced create fail") {
		t.Fatalf("createOperation: %v", err)
	}

	bare := &Ops{}
	if _, err := bare.listPendingDemoCatalog(context.Background()); err == nil {
		t.Fatal("expected session error")
	}
	if _, err := bare.isPendingDemoModule(context.Background(), "x"); err == nil {
		t.Fatal("expected session error")
	}
	if _, err := bare.loadInstalledModule(context.Background(), "x"); err == nil {
		t.Fatal("expected session error")
	}
	if err := bare.markDemoLoaded(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected session error")
	}
	if err := bare.markDemoLoaded(context.Background(), nil); err != nil {
		t.Fatalf("empty mark: %v", err)
	}
}

func TestExecuteLoadDemo_PendingCheckQueryFailure(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Callback().Query().Before("gorm:query").Register("ops_fail_pending_count", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_module_catalog" {
			_ = tx.AddError(errors.New("forced count fail"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	raw, err := marshalPlanDocument(PlanDocument{
		ModuleOrder:  []ModuleOrderEntry{{ModuleName: "base"}},
		BaseRevision: "load-demo",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = o.executeLoadDemo(context.Background(), &modmeta.ModuleOperation{PlanJson: raw})
	if err == nil || !strings.Contains(err.Error(), "forced count fail") {
		t.Fatalf("pending check: %v", err)
	}
}

func TestMarkDemoLoaded_OnlyPendingRows(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "pending", Installed: true, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "done", Installed: true, HasDemo: true, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed done: %v", err)
	}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "gone", Installed: false, HasDemo: true, DemoLoaded: false,
	}).Error; err != nil {
		t.Fatalf("seed gone: %v", err)
	}
	if err := o.markDemoLoaded(context.Background(), []string{"pending", "done", "gone"}); err != nil {
		t.Fatalf("mark: %v", err)
	}
	var pending, done, gone modmeta.ModuleCatalog
	_ = db.Where("module_name = ?", "pending").Take(&pending).Error
	_ = db.Where("module_name = ?", "done").Take(&done).Error
	_ = db.Where("module_name = ?", "gone").Take(&gone).Error
	if !pending.DemoLoaded {
		t.Fatal("pending should be marked")
	}
	if !done.DemoLoaded {
		t.Fatal("done must stay loaded")
	}
	if gone.DemoLoaded {
		t.Fatal("uninstalled row must not be marked")
	}
}

func TestLoadInstalledModule_RequiresInstalledStatus(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Uninstalled, Path: t.TempDir(),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := o.loadInstalledModule(context.Background(), "base"); err == nil {
		t.Fatal("expected not found for uninstalled")
	}
}

func TestDefaultApplyDemoFiles(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })

	if err := defaultApplyDemoFiles(context.Background(), nil, nil); err != nil {
		t.Fatalf("nil mod: %v", err)
	}
	if err := defaultApplyDemoFiles(context.Background(), nil, &pkgmeta.Module{Name: "x"}); err != nil {
		t.Fatalf("empty demo: %v", err)
	}
	if err := defaultApplyDemoFiles(context.Background(), nil, &pkgmeta.Module{
		Name: "x", DemoStr: datatypes.JSON(`{`),
	}); err == nil {
		t.Fatal("expected manifest decode error")
	}
	if err := defaultApplyDemoFiles(context.Background(), nil, &pkgmeta.Module{
		Name: "x", DemoStr: datatypes.JSON(`["demo/a.json"]`), Path: "  ",
	}); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("empty path: %v", err)
	}

	var got importpkg.Spec
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
		got = spec
		return importpkg.Report{}, nil
	})
	demoJSON, err := json.Marshal([]string{"demo/smoke.json"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := defaultApplyDemoFiles(context.Background(), nil, &pkgmeta.Module{
		Name: "base", ApplicationStr: "base", Path: "/modules/base", DemoStr: demoJSON,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got.Module != "base" || !got.Options.WithDemo || len(got.Options.DemoFiles) != 1 || len(got.Options.InitdataFiles) != 0 {
		t.Fatalf("spec = %+v", got)
	}

	importpkg.SetRun(func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, errors.New("import boom")
	})
	if err := defaultApplyDemoFiles(context.Background(), nil, &pkgmeta.Module{
		Name: "base", Path: "/modules/base", DemoStr: demoJSON,
	}); err == nil || !strings.Contains(err.Error(), "import boom") {
		t.Fatalf("import err: %v", err)
	}
}
