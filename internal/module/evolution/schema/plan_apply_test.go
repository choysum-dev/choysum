// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestMigrateSchema_CreateTableAndAddNullableColumn(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name:       "Order",
		Path:       "sales/order.ts",
		ModelTable: "sales_order_plan",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Status", `{"type":"selection"}`),
		},
	}
	migrator := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("initial MigrateSchema() error = %v", err)
	}
	if !runtimeScope.Session().Migrator().HasTable("sales_order_plan") {
		t.Fatal("expected table to be created")
	}
	if !runtimeScope.Session().Migrator().HasColumn("sales_order_plan", "status") {
		t.Fatal("expected status column")
	}

	model.Fields = append(model.Fields, newFieldWithOptions(t, "Note", `{"type":"text"}`))
	migrator = newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("add-column MigrateSchema() error = %v", err)
	}
	if !runtimeScope.Session().Migrator().HasColumn("sales_order_plan", "note") {
		t.Fatal("expected note column to be added")
	}
}

func TestMigrateSchema_LeftoverColumnDoesNotFail(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name:       "Order",
		Path:       "sales/order.ts",
		ModelTable: "sales_order_leftover",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Status", `{"type":"selection"}`),
		},
	}
	migrator := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("initial MigrateSchema() error = %v", err)
	}
	if err := runtimeScope.Session().Exec(`ALTER TABLE sales_order_leftover ADD COLUMN legacy_col text`).Error; err != nil {
		t.Fatalf("add leftover column: %v", err)
	}

	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema with leftover column error = %v", err)
	}
	if !runtimeScope.Session().Migrator().HasColumn("sales_order_leftover", "legacy_col") {
		t.Fatal("leftover column must remain")
	}
}

func TestMigrateSchema_TypeChangeFailsValidate(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	field := newFieldWithOptions(t, "Status", `{"type":"selection"}`)
	model := &meta.Model{
		Name:       "Order",
		Path:       "sales/order.ts",
		ModelTable: "sales_order_type_change",
		Fields:     []*meta.Field{field},
	}
	migrator := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("initial MigrateSchema() error = %v", err)
	}

	spec, err := field.GetResolvedSpec()
	if err != nil {
		t.Fatalf("GetResolvedSpec() error = %v", err)
	}
	spec.Migration.ResolvedColumnType = "text"
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec() error = %v", err)
	}

	migrator = newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	err = migrator.MigrateSchema()
	if err == nil || !strings.Contains(err.Error(), "schema plan has guarded/manual operations") {
		t.Fatalf("expected guarded plan error, got %v", err)
	}
}

func TestValidatePlan_RejectsGuarded(t *testing.T) {
	plan := SchemaPlan{
		Module: "sales",
		Ops: []PlanOp{{
			Kind:   OpAlterColumn,
			Safety: SafetyGuarded,
			Table:  "sales_order",
			Detail: "type change varchar → text (desired physical text)",
			Column: &ColumnSpec{Name: "status", PhysicalType: "text"},
		}},
	}
	err := ValidatePlan(plan)
	if err == nil || !strings.Contains(err.Error(), "schema plan has guarded/manual operations") {
		t.Fatalf("ValidatePlan() error = %v", err)
	}
	if !strings.Contains(err.Error(), "sales_order") {
		t.Fatalf("expected table in error, got %v", err)
	}
}

func TestBuildPlan_AddNotNullOnNonEmptyTableIsGuarded(t *testing.T) {
	desired := DesiredSchema{
		Tables: map[string][]ColumnSpec{
			"sales_order": {{
				Name:         "code",
				FieldName:    "Code",
				PhysicalType: "varchar",
				NotNull:      true,
				Size:         intPtrValue(32),
			}},
		},
	}
	live := LiveSchema{
		Tables:   map[string]bool{"sales_order": true},
		Columns:  map[string]map[string]LiveColumn{"sales_order": {}},
		RowCount: map[string]int64{"sales_order": 1},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	if len(plan.Ops) != 1 || plan.Ops[0].Safety != SafetyGuarded {
		t.Fatalf("expected guarded add NOT NULL, got %#v", plan.Ops)
	}
	if err := ValidatePlan(plan); err == nil {
		t.Fatal("expected ValidatePlan to reject guarded add NOT NULL")
	}
}
