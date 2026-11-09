// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestEnrichPlanDocument_InstallTargetAndSatisfiedDeps(t *testing.T) {
	o := &Ops{Resolver: &mapResolver{
		peek: map[string]*pkgmeta.Module{
			"partner": {Name: "partner", Version: "v1.0.0", ApplicationStr: "partner"},
			"core":    {Name: "core", Version: "v1.0.0", ApplicationStr: "core"},
			"auth":    {Name: "auth", Version: "v1.0.0", ApplicationStr: "auth"},
		},
		load: map[string]*pkgmeta.Module{
			"core": {Name: "core", Version: "v1.0.0", ApplicationStr: "core", Status: pkgmeta.Installed},
			"auth": {Name: "auth", Version: "v1.0.0", ApplicationStr: "auth", Status: pkgmeta.Installed},
		},
	}}
	built := plan.Plan{
		ModuleOrder:       []string{"core", "auth", "partner"},
		AffectedApps:      []string{"auth", "core", "partner"},
		NeedsGlobalWebBuild: true,
	}
	doc := o.enrichPlanDocument(context.Background(), ActionInstall, "partner", originLocal, built, nil)
	if !doc.NeedsWebBuild {
		t.Fatal("NeedsWebBuild should pass through")
	}
	byName := map[string]ModuleOrderEntry{}
	for _, e := range doc.ModuleOrder {
		byName[e.ModuleName] = e
	}
	partner := byName["partner"]
	if partner.Role != RoleTarget || partner.Effect != EffectInstall || partner.Reason != "root" {
		t.Fatalf("partner entry = %#v", partner)
	}
	if partner.TargetVersion != "1.0.0" {
		t.Fatalf("partner TargetVersion display = %q", partner.TargetVersion)
	}
	for _, name := range []string{"core", "auth"} {
		e := byName[name]
		if e.Role != RoleDependency || e.Effect != EffectAlreadySatisfied || e.Reason != "dependency" {
			t.Fatalf("%s entry = %#v", name, e)
		}
		if e.CurrentVersion != "1.0.0" || e.TargetVersion != "1.0.0" {
			t.Fatalf("%s versions = %#v", name, e)
		}
	}
}

func TestEnrichPlanDocument_UninstallDependentsShareApplication(t *testing.T) {
	o := &Ops{Resolver: &mapResolver{
		peek: map[string]*pkgmeta.Module{
			"partner":            {Name: "partner", Version: "1.0.0", ApplicationStr: "partner"},
			"partner_commercial": {Name: "partner_commercial", Version: "1.0.0", ApplicationStr: "partner"},
		},
		load: map[string]*pkgmeta.Module{
			"partner": {
				Name: "partner", Version: "1.0.0", ApplicationStr: "partner", Status: pkgmeta.Installed,
			},
			"partner_commercial": {
				Name: "partner_commercial", Version: "1.0.0", ApplicationStr: "partner", Status: pkgmeta.Installed,
			},
		},
	}}
	built := plan.Plan{
		ModuleOrder:  []string{"partner_commercial", "partner"},
		AffectedApps: []string{"partner"},
	}
	doc := o.enrichPlanDocument(context.Background(), ActionUninstall, "partner", originLocal, built, nil)
	if len(doc.ModuleOrder) != 2 {
		t.Fatalf("ModuleOrder = %#v", doc.ModuleOrder)
	}
	commercial := doc.ModuleOrder[0]
	if commercial.ModuleName != "partner_commercial" || commercial.Application != "partner" {
		t.Fatalf("commercial = %#v", commercial)
	}
	if commercial.Role != RoleDependent || commercial.Effect != EffectUninstall {
		t.Fatalf("commercial role/effect = %#v", commercial)
	}
	target := doc.ModuleOrder[1]
	if target.Role != RoleTarget || target.Effect != EffectUninstall || target.Reason != "root" {
		t.Fatalf("partner = %#v", target)
	}
}

func TestEnrichPlanDocument_UpgradeEnsureModules(t *testing.T) {
	o := &Ops{Resolver: &mapResolver{
		peek: map[string]*pkgmeta.Module{
			"partner":  {Name: "partner", Version: "2.0.0", ApplicationStr: "partner"},
			"document": {Name: "document", Version: "1.0.0", ApplicationStr: "document"},
		},
		load: map[string]*pkgmeta.Module{
			"partner": {Name: "partner", Version: "1.0.0", ApplicationStr: "partner", Status: pkgmeta.Installed},
		},
	}}
	built := plan.Plan{
		ModuleOrder:  []string{"partner"},
		EnsureOrder:  []string{"document"},
		AffectedApps: []string{"document", "partner"},
	}
	doc := o.enrichPlanDocument(context.Background(), ActionUpgrade, "partner", originLocal, built, nil)
	if len(doc.ModuleOrder) != 1 || doc.ModuleOrder[0].Effect != EffectUpgrade || doc.ModuleOrder[0].Role != RoleTarget {
		t.Fatalf("ModuleOrder = %#v", doc.ModuleOrder)
	}
	if len(doc.EnsureModules) != 1 {
		t.Fatalf("EnsureModules = %#v", doc.EnsureModules)
	}
	ens := doc.EnsureModules[0]
	if ens.ModuleName != "document" || ens.Role != RoleDependency || ens.Effect != EffectInstall {
		t.Fatalf("ensure = %#v", ens)
	}
	if len(doc.EnsureOrder) != 1 || doc.EnsureOrder[0] != "document" {
		t.Fatalf("EnsureOrder = %#v", doc.EnsureOrder)
	}
}

func TestBuildOrderEntry_EnsureStepEffects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := newOpsTestDB(t)
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		Resolver: &mapResolver{
			load: map[string]*pkgmeta.Module{
				"core": {Name: "core", Version: "1.0.0", ApplicationStr: "core", Status: pkgmeta.Installed},
				"auth": {Name: "auth", Version: "2.0.0", ApplicationStr: "auth", Status: pkgmeta.Installed},
			},
			peek: map[string]*pkgmeta.Module{
				"core": {Name: "core", Version: "1.0.0", ApplicationStr: "core"},
				"auth": {Name: "auth", Version: "3.0.0", ApplicationStr: "auth"},
			},
		},
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "auth", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("3.0.0"),
	}).Error; err != nil {
		t.Fatalf("seed auth snapshot: %v", err)
	}
	satisfied := o.buildOrderEntry(ctx, ActionInstall, "partner", "core", originLocal, true)
	if satisfied.Effect != EffectAlreadySatisfied {
		t.Fatalf("ensure satisfied = %#v", satisfied)
	}
	ensureInstalled := o.buildOrderEntry(ctx, ActionInstall, "partner", "auth", originLocal, true)
	if ensureInstalled.Effect != EffectAlreadySatisfied {
		t.Fatalf("ensure installed (version skew) = %#v", ensureInstalled)
	}
	installDepInstalled := o.buildOrderEntry(ctx, ActionInstall, "partner", "auth", originLocal, false)
	if installDepInstalled.Effect != EffectAlreadySatisfied {
		t.Fatalf("install dep already installed = %#v", installDepInstalled)
	}
	depUpgrade := o.buildOrderEntry(ctx, ActionUpgrade, "partner", "auth", originLocal, false)
	if depUpgrade.Effect != EffectUpgrade {
		t.Fatalf("dependency upgrade = %#v", depUpgrade)
	}
	app, installed := (&Ops{}).applicationAndInstalled(ctx, "core")
	if app != "" || installed {
		t.Fatalf("nil resolver applicationAndInstalled = %q %v", app, installed)
	}
}

func TestEnrichPlanDocument_InstallAlreadyInstalledTargetSatisfied(t *testing.T) {
	o := &Ops{Resolver: &mapResolver{
		peek: map[string]*pkgmeta.Module{
			"core": {Name: "core", Version: "1.0.0", ApplicationStr: "core"},
		},
		load: map[string]*pkgmeta.Module{
			"core": {Name: "core", Version: "1.0.0", ApplicationStr: "core", Status: pkgmeta.Installed},
		},
	}}
	built := plan.Plan{ModuleOrder: []string{"core"}, AffectedApps: []string{"core"}}
	doc := o.enrichPlanDocument(context.Background(), ActionInstall, "core", originLocal, built, nil)
	if doc.ModuleOrder[0].Effect != EffectAlreadySatisfied || doc.ModuleOrder[0].Role != RoleTarget {
		t.Fatalf("entry = %#v", doc.ModuleOrder[0])
	}
}
