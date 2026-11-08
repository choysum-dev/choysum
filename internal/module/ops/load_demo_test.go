// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/datatypes"
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
