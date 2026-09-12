// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"encoding/json"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestSchemaSnapshot_RoundTrip(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())

	desired := DesiredSchema{
		Tables: map[string][]ColumnSpec{
			"sales_order": {{
				Name: "status", FieldName: "Status", PhysicalType: "varchar", Size: intPtrValue(255),
			}},
		},
	}
	models := []*meta.Model{{Name: "Order", Application: "sales", ModelTable: "sales_order"}}
	mod := &meta.Module{Name: "sales", Version: "1.0.0"}
	if err := SaveSnapshots(runtimeScope.Session().DB, desired, models, mod); err != nil {
		t.Fatalf("SaveSnapshots: %v", err)
	}
	loaded, err := LoadSnapshots(runtimeScope.Session().DB, []string{"sales_order"})
	if err != nil {
		t.Fatalf("LoadSnapshots: %v", err)
	}
	row, ok := loaded["sales_order"]
	if !ok {
		t.Fatal("missing snapshot")
	}
	if row.UpdatedByModule != "sales" || row.ModuleVersion != "1.0.0" {
		t.Fatalf("unexpected snapshot meta: %#v", row)
	}
	var cols []ColumnSpec
	if err := json.Unmarshal(row.DesiredJSON, &cols); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(cols) != 1 || cols[0].Name != "status" {
		t.Fatalf("unexpected desired json: %#v", cols)
	}
}

func TestMigrate_WritesSnapshot(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", Application: "sales", ModelTable: "sales_snap",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	mod := &meta.Module{Name: "sales", Version: "2.0.0"}
	if err := newModelMigrator(runtimeScope, mod, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema: %v", err)
	}
	var row modmeta.SchemaSnapshot
	if err := runtimeScope.Session().Where("model_table = ?", "sales_snap").Take(&row).Error; err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if row.UpdatedByModule != "sales" {
		t.Fatalf("updated_by_module = %q", row.UpdatedByModule)
	}
}

func TestPlan_VarcharWidenAuto(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(64)}},
	}}
	length := int64(32)
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "varchar", Length: &length}}},
		Indexes: map[string]map[string]bool{"t": {}},
	}
	plan := buildPlan("sales", desired, live, "postgres")
	if len(plan.Ops) != 1 || plan.Ops[0].Kind != OpAlterColumn || plan.Ops[0].Safety != SafetyAuto {
		t.Fatalf("expected auto widen, got %#v", plan.Ops)
	}
	if !strings.Contains(plan.Ops[0].Detail, "widen") {
		t.Fatalf("detail = %q", plan.Ops[0].Detail)
	}
}

func TestPlan_VarcharNarrowGuarded(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(16)}},
	}}
	length := int64(32)
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "varchar", Length: &length}}},
		Indexes: map[string]map[string]bool{"t": {}},
	}
	plan := buildPlan("sales", desired, live, "postgres")
	if len(plan.Ops) != 1 || plan.Ops[0].Safety != SafetyGuarded || !strings.Contains(plan.Ops[0].Detail, "narrow") {
		t.Fatalf("expected guarded narrow, got %#v", plan.Ops)
	}
}

func TestPlan_UniqueOnPopulatedGuarded(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", UniqueIndex: true}},
	}}
	live := LiveSchema{
		Tables:   map[string]bool{"t": true},
		Columns:  map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "varchar"}}},
		Indexes:  map[string]map[string]bool{"t": {}},
		RowCount: map[string]int64{"t": 1},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	found := false
	for _, op := range plan.Ops {
		if op.Kind == OpAddIndex && op.Safety == SafetyGuarded {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected guarded add unique index, got %#v", plan.Ops)
	}
}

func TestPlanOnly_NoDDL(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_plan_only",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	mig := newModelMigrator(runtimeScope, &meta.Module{Name: "sales"}, []*meta.Model{model})
	plan, err := mig.PlanSchema()
	if err != nil {
		t.Fatalf("PlanSchema: %v", err)
	}
	if len(plan.Ops) == 0 {
		t.Fatal("expected create_table op")
	}
	if runtimeScope.Session().Migrator().HasTable("sales_plan_only") {
		t.Fatal("PlanSchema must not create tables")
	}
}

func TestMigrate_IdempotentSecondRun(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_idem",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	mig := newModelMigrator(runtimeScope, &meta.Module{Name: "sales"}, []*meta.Model{model})
	if err := mig.MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	plan, err := mig.PlanSchema()
	if err != nil {
		t.Fatal(err)
	}
	// ensure_check may still appear as Auto ops on every run (idempotent ensure).
	for _, op := range plan.Ops {
		if op.Kind == OpCreateTable || op.Kind == OpAddColumn {
			t.Fatalf("unexpected structural op on second plan: %#v", op)
		}
	}
}

func TestApply_EnsureCheckAfterCreate(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_chk_plan",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection","column":{"checkConstraint":"status <> ''"}}`)},
	}
	if err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	if !runtimeScope.Session().Migrator().HasTable("sales_chk_plan") {
		t.Fatal("expected table")
	}
}

func TestApply_DefaultOnAddColumn(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_def",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	if err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	field := &meta.Field{Name: "Note"}
	def := "hello"
	spec := &meta.FieldResolvedSpec{
		FieldName: "Note",
		Structural: meta.FieldStructuralSpec{
			Name: "Note", FieldType: "varchar",
			StorageHints: &meta.FieldStructuralStorageHints{Default: &def, Size: intPtrValue(32)},
		},
		Migration: meta.FieldMigrationDecision{ShouldCreateColumn: true, ResolvedColumnType: "varchar", StorageKind: "physical"},
	}
	_ = field.SetResolvedSpec(spec)
	model.Fields = append(model.Fields, field)
	if err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatalf("add with default: %v", err)
	}
	if !runtimeScope.Session().Migrator().HasColumn("sales_def", "note") {
		t.Fatal("expected note column")
	}
}
