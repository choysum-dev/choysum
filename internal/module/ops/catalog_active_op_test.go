// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestApplyAndRun_SetsAndClearsActiveOperationId(t *testing.T) {
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core", Status: pkgmeta.Installed}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}, load: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	if err := db.Create(&modmeta.ModuleCatalog{ModuleName: "core", Installed: true}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionUpgrade})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var mid modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "core").Take(&mid).Error; err != nil {
		t.Fatalf("catalog mid: %v", err)
	}
	if !mid.ActiveOperationId.Valid || mid.ActiveOperationId.String != result.Id {
		t.Fatalf("ActiveOperationId after Apply = %#v, want %q", mid.ActiveOperationId, result.Id)
	}
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var done modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "core").Take(&done).Error; err != nil {
		t.Fatalf("catalog done: %v", err)
	}
	if done.ActiveOperationId.Valid && done.ActiveOperationId.String != "" {
		t.Fatalf("ActiveOperationId after Run = %#v, want empty", done.ActiveOperationId)
	}
}

func TestMaterializeCatalog_ClearsDemoLoadedOnUpgrade(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&modmeta.ModuleCatalog{
		ModuleName: "base", Installed: true, InstalledVersion: nullString("1.0.0"),
		HasDemo: true, DemoLoaded: true,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "2.0.0", Status: pkgmeta.Installed, ApplicationStr: "base",
		DemoStr: []byte(`["demo/a.json"]`),
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"base"}); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	var row modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&row).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.DemoLoaded {
		t.Fatal("DemoLoaded should clear when installed version changes")
	}
	if !row.HasDemo {
		t.Fatal("HasDemo should stay true")
	}
}

func TestMaterializeCatalog_FillsDependentsJson(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.Create(&pkgmeta.Module{
		Name: "auth", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "auth",
		DependsStr: []byte(`["base"]`),
	}).Error; err != nil {
		t.Fatalf("seed auth: %v", err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name: "base", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "base",
	}).Error; err != nil {
		t.Fatalf("seed base: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), nil); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	var base modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "base").Take(&base).Error; err != nil {
		t.Fatalf("base: %v", err)
	}
	if !strings.Contains(string(base.DependentsJson), `"auth"`) {
		t.Fatalf("DependentsJson = %s, want auth", base.DependentsJson)
	}
	var auth modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "auth").Take(&auth).Error; err != nil {
		t.Fatalf("auth: %v", err)
	}
	var edges []modmeta.ModuleCatalogDepend
	if err := db.Find(&edges).Error; err != nil {
		t.Fatalf("depend edges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("depend edges len = %d, want 1", len(edges))
	}
	if edges[0].ModuleID != auth.Id.String || edges[0].DependID != base.Id.String {
		t.Fatalf("edge = %+v, want auth(%s)->base(%s)", edges[0], auth.Id.String, base.Id.String)
	}
}
