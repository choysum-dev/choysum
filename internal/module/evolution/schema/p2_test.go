// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestIntentBag_AddListClear(t *testing.T) {
	bag := NewMemoryIntentBag()
	bag.Add(
		Intent{Kind: IntentDropColumn, Table: "t", Name: "old_code"},
		Intent{Kind: "", Table: "t", Name: "skip"},
		Intent{Kind: IntentDropColumn, Table: "", Name: "no_table"},
		Intent{Kind: IntentDropColumn, Table: "t", Name: ""},
		Intent{Kind: IntentRenameColumn, Table: "t", FromName: "old", Name: "new"},
		Intent{Kind: IntentRenameColumn, Table: "t", FromName: "", Name: "new"},
		Intent{Kind: IntentDropIndex, Table: " t ", Name: " idx_t_code "},
	)
	got := bag.List()
	if len(got) != 3 {
		t.Fatalf("list = %#v", got)
	}
	if got[0].Name != "old_code" || got[1].Name != "new" || got[2].Name != "idx_t_code" {
		t.Fatalf("trimmed names: %#v", got)
	}
	bag.Clear()
	if len(bag.List()) != 0 {
		t.Fatal("expected clear")
	}
}

func TestValidate_ManualDropSatisfiedByIntent(t *testing.T) {
	bag := NewMemoryIntentBag()
	bag.Add(Intent{Kind: IntentDropColumn, Table: "sales_order", Name: "old_code"})
	plan := SchemaPlan{Ops: []PlanOp{{
		Kind:   OpKind(IntentDropColumn),
		Safety: SafetyManual,
		Table:  "sales_order",
		Detail: "drop column old_code",
		Column: &ColumnSpec{Name: "old_code"},
	}}}
	if err := ValidatePlan(plan, bag); err != nil {
		t.Fatalf("intent should satisfy manual drop: %v", err)
	}
	if err := ValidatePlan(plan, nil); err == nil {
		t.Fatal("expected failure without intent")
	}
}

func TestPlan_RenameFromAuto(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code", Size: intPtrValue(32)}},
	}}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"old_code": {Name: "old_code", DatabaseTypeName: "varchar"}}},
	}
	plan, err := buildPlan("sales", desired, live, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Ops) < 1 || plan.Ops[0].Kind != OpRenameColumn || plan.Ops[0].Safety != SafetyAuto {
		t.Fatalf("expected auto rename, got %#v", plan.Ops)
	}
	if plan.Ops[0].FromName != "old_code" {
		t.Fatalf("FromName=%q", plan.Ops[0].FromName)
	}
	for _, left := range plan.Leftover {
		if left.Kind == LeftoverColumn && strings.EqualFold(left.Name, "old_code") {
			t.Fatalf("renamed-from must not be leftover: %#v", plan.Leftover)
		}
	}
}

func TestPlan_RenameFollowedByAttributeDiff(t *testing.T) {
	nullable := true
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code", NotNull: true, Size: intPtrValue(64)}},
	}}
	live := LiveSchema{
		Tables: map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {
			"old_code": {Name: "old_code", DatabaseTypeName: "varchar", Nullable: &nullable, Length: int64Ptr(32)},
		}},
	}
	plan, err := buildPlan("sales", desired, live, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	hasRename, hasAlter := false, false
	for _, op := range plan.Ops {
		if op.Kind == OpRenameColumn && op.Safety == SafetyAuto {
			hasRename = true
		}
		if op.Kind == OpAlterColumn {
			hasAlter = true
		}
	}
	if !hasRename || !hasAlter {
		t.Fatalf("expected rename + attribute alter, got %#v", plan.Ops)
	}
}

func TestPlan_RenameTypeMismatchGuarded(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "integer", RenameFrom: "old_code"}},
	}}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"old_code": {Name: "old_code", DatabaseTypeName: "varchar"}}},
	}
	plan, err := buildPlan("sales", desired, live, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Ops) < 1 || plan.Ops[0].Kind != OpRenameColumn || plan.Ops[0].Safety != SafetyGuarded {
		t.Fatalf("expected guarded rename, got %#v", plan.Ops)
	}
}

func TestPlan_RenameMissingOldErrors(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code"}},
	}}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {}},
	}
	_, err := buildPlan("sales", desired, live, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "neither old column") {
		t.Fatalf("expected missing renameFrom error, got %v", err)
	}
}

func TestPlan_RenameAlreadyDoneNoOp(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code"}},
	}}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "TEXT"}}},
	}
	plan, err := buildPlan("sales", desired, live, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range plan.Ops {
		if op.Kind == OpRenameColumn {
			t.Fatalf("already-renamed must not emit rename: %#v", plan.Ops)
		}
	}
}

func TestApply_RenamePreservesData(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rename_data (old_code text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO rename_data (old_code) VALUES ('abc')`).Error; err != nil {
		t.Fatal(err)
	}
	col := ColumnSpec{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code"}
	plan := SchemaPlan{Ops: []PlanOp{{
		Kind: OpRenameColumn, Safety: SafetyAuto, Table: "rename_data",
		Detail: "rename", Column: &col, FromName: "old_code",
	}}}
	if err := applyPlan(runtimeScope, "sqlite", plan); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := db.Raw(`SELECT code FROM rename_data`).Scan(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestHelper_DropColumnSqlite(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE drop_me (id integer, old_code text)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	if err := DropColumn(HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}, "drop_me", "old_code"); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("drop_me", "old_code") {
		t.Fatal("column still present")
	}
	list := bag.List()
	if len(list) != 1 || list[0].Kind != IntentDropColumn || list[0].Name != "old_code" {
		t.Fatalf("intent = %#v", list)
	}
}

func TestHelper_RejectsNonChoysumName(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE idx_tbl (code text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX custom_name ON idx_tbl (code)`).Error; err != nil {
		t.Fatal(err)
	}
	err := DropIndex(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, "idx_tbl", "custom_name")
	if err == nil || !strings.Contains(err.Error(), "rejects non-Choysum name") {
		t.Fatalf("got %v", err)
	}
}

func TestDesired_RenameFromDropAfter(t *testing.T) {
	field := newFieldWithOptions(t, "Code", `{"type":"varchar","size":32,"renameFrom":"OldCode","dropAfter":"2.0.0"}`)
	model := &meta.Model{Name: "Order", ModelTable: "sales_order", Fields: []*meta.Field{field}}
	col, err := columnSpecFromField(field, model)
	if err != nil || col == nil {
		t.Fatalf("%v %#v", err, col)
	}
	if col.RenameFrom != "old_code" || col.DropAfter != "2.0.0" {
		t.Fatalf("%#v", col)
	}
}

func TestIntentBagFromContext(t *testing.T) {
	if IntentBagFromContext(nil) != nil {
		t.Fatal("nil ctx")
	}
	bag := NewMemoryIntentBag()
	ctx := ContextWithIntentBag(nil, bag)
	if IntentBagFromContext(ctx) != bag {
		t.Fatal("roundtrip")
	}
}
