// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

func TestPlan_MatchesBuildPlanOrderAndPersistsDraft(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	runtimeScope := newOpsTestScope(t, db)

	core := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	auth := &pkgmeta.Module{
		Name:           "auth",
		Version:        "1.0.0",
		ApplicationStr: "auth",
		DependsStr:     datatypes.JSON(`["core"]`),
	}
	resolver := &mapResolver{
		peek: map[string]*pkgmeta.Module{"core": core, "auth": auth},
		load: map[string]*pkgmeta.Module{},
	}
	o := &Ops{Scope: runtimeScope, Resolver: resolver}

	built, err := plan.BuildPlan(context.Background(), plan.OpInstall, auth, resolver)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	result, err := o.Plan(context.Background(), PlanRequest{
		ModuleName: "auth",
		Action:     ActionInstall,
		Origin:     originLocal,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if result.Id == "" {
		t.Fatal("expected operation id")
	}
	if len(result.ModuleOrder) != len(built.ModuleOrder) {
		t.Fatalf("moduleOrder len = %d, want %d (%v vs %v)", len(result.ModuleOrder), len(built.ModuleOrder), result.ModuleOrder, built.ModuleOrder)
	}
	for i, name := range built.ModuleOrder {
		if result.ModuleOrder[i].ModuleName != name {
			t.Fatalf("moduleOrder[%d] = %q, want %q", i, result.ModuleOrder[i].ModuleName, name)
		}
		if result.ModuleOrder[i].Origin != originLocal {
			t.Fatalf("origin = %q, want local", result.ModuleOrder[i].Origin)
		}
		if result.ModuleOrder[i].TargetVersion == "" {
			t.Fatalf("targetVersion empty for %q", name)
		}
	}

	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("loadOperation: %v", err)
	}
	if row.Status != StatusDraft {
		t.Fatalf("status = %q, want draft", row.Status)
	}
	doc, err := unmarshalPlanDocument(row.PlanJson)
	if err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}
	if doc.BaseRevision == "" || doc.BaseRevision != result.BaseRevision {
		t.Fatalf("baseRevision mismatch doc=%q result=%q", doc.BaseRevision, result.BaseRevision)
	}
}

func TestPlan_MissingModuleStillWritesDraftWithBlocker(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{
		Scope:    newOpsTestScope(t, db),
		Resolver: &mapResolver{},
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "missing", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(result.Blockers) == 0 {
		t.Fatal("expected blockers")
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusDraft {
		t.Fatalf("status = %q, want draft", row.Status)
	}
}

func TestApply_RevisionMismatchAndSuccess(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	runtimeScope := newOpsTestScope(t, db)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	resolver := &mapResolver{
		peek: map[string]*pkgmeta.Module{"core": mod},
		load: map[string]*pkgmeta.Module{},
	}
	o := &Ops{Scope: runtimeScope, Resolver: resolver}

	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Tamper baseRevision.
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	doc, err := unmarshalPlanDocument(row.PlanJson)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	doc.BaseRevision = "stale"
	raw, _ := json.Marshal(doc)
	row.PlanJson = raw
	if err := o.saveOperation(context.Background(), row); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err == nil || !strings.Contains(err.Error(), "baseRevision") {
		t.Fatalf("Apply error = %v, want baseRevision mismatch", err)
	}

	// Restore revision and apply.
	doc.BaseRevision = computeBaseRevision(context.Background(), o, doc.ModuleOrder)
	raw, _ = json.Marshal(doc)
	row.PlanJson = raw
	if err := o.saveOperation(context.Background(), row); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	row, err = o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusQueued || row.Phase != PhaseQueued {
		t.Fatalf("status/phase = %q/%q, want queued/queued", row.Status, row.Phase)
	}
}

func TestApply_RejectsBlockers(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Resolver: &mapResolver{}}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "missing", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err == nil || !strings.Contains(err.Error(), "blockers") {
		t.Fatalf("Apply error = %v, want blockers", err)
	}
}

func TestApply_RejectsEmptyModuleOrder(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	row := &modmeta.ModuleOperation{
		Action:   ActionInstall,
		Status:   StatusDraft,
		PlanJson: datatypes.JSON(`{"moduleOrder":[],"baseRevision":"x"}`),
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: row.Id.String}); err == nil || !strings.Contains(err.Error(), "empty module order") {
		t.Fatalf("Apply error = %v, want empty module order", err)
	}
}
