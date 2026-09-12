// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestDefaultChangedBranches(t *testing.T) {
	liveDef := "hello"
	if defaultChanged(ColumnSpec{}, LiveColumn{}) {
		t.Fatal("both nil")
	}
	if !defaultChanged(ColumnSpec{}, LiveColumn{Default: &liveDef}) {
		t.Fatal("desired nil live set")
	}
	empty := "  "
	if !defaultChanged(ColumnSpec{Default: &empty}, LiveColumn{Default: &liveDef}) {
		t.Fatal("empty desired with live set")
	}
	if defaultChanged(ColumnSpec{Default: &empty}, LiveColumn{}) {
		t.Fatal("empty desired live nil")
	}
	want := "hello"
	if defaultChanged(ColumnSpec{Default: &want}, LiveColumn{}) {
		t.Fatal("unknown live default")
	}
	if defaultChanged(ColumnSpec{Default: &want}, LiveColumn{Default: &liveDef}) {
		t.Fatal("same default")
	}
	other := "other"
	if !defaultChanged(ColumnSpec{Default: &want}, LiveColumn{Default: &other}) {
		t.Fatal("different default")
	}
}

func TestLiveHasIndexBranches(t *testing.T) {
	if liveHasIndex(LiveSchema{}, "t", "x", false) {
		t.Fatal("nil map")
	}
	if liveHasIndex(LiveSchema{Indexes: map[string][]LiveIndex{"t": {}}}, "t", "  ", false) {
		t.Fatal("empty name")
	}
	live := LiveSchema{Indexes: map[string][]LiveIndex{
		"t": {
			{Name: "idx_a", Columns: []string{"a"}, Unique: false},
			{Name: "idx_b", Columns: []string{"b"}, Unique: true},
		},
	}}
	if !liveHasIndex(live, "t", "idx_a", false) {
		t.Fatal("name match non-unique")
	}
	if liveHasIndex(live, "t", "idx_a", true) {
		t.Fatal("non-unique name must not satisfy unique")
	}
	if !liveHasIndex(live, "t", "idx_b", true) {
		t.Fatal("unique name")
	}
	if !liveHasIndex(live, "t", "b", true) {
		t.Fatal("unique by column")
	}
	if liveHasIndex(live, "t", "a", true) {
		t.Fatal("non-unique column must not satisfy unique")
	}
	if !indexCoveredByDesiredKey(LiveIndex{Columns: []string{"Code"}}, map[string]struct{}{"code": {}}) {
		t.Fatal("covered")
	}
	if indexCoveredByDesiredKey(LiveIndex{Columns: []string{"other"}}, map[string]struct{}{"code": {}}) {
		t.Fatal("not covered")
	}
}

func TestApplyPlan_AlterWidenAndIndexCheckErrors(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpCreateTable, Safety: SafetyAuto, Table: "widen_tbl",
		Columns: []ColumnSpec{{Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(8)}},
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpAlterColumn, Safety: SafetyAuto, Table: "widen_tbl",
		Column: &ColumnSpec{Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(32), NotNull: true},
	}}}); err != nil {
		t.Logf("sqlite alter widen: %v", err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpAlterColumn, Safety: SafetyAuto, Table: "widen_tbl",
	}}}); err == nil || !strings.Contains(err.Error(), "alter_column missing") {
		t.Fatalf("missing column: %v", err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpAddIndex, Safety: SafetyAuto, Table: "widen_tbl",
	}}}); err == nil || !strings.Contains(err.Error(), "add_index missing") {
		t.Fatalf("missing index column: %v", err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpEnsureCheck, Safety: SafetyAuto, Table: "widen_tbl",
	}}}); err == nil || !strings.Contains(err.Error(), "ensure_check missing") {
		t.Fatalf("missing check: %v", err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpAlterColumn, Safety: SafetyAuto, Table: "widen_tbl",
		Column: &ColumnSpec{Name: "code", FieldName: "Code", PhysicalType: "nope"},
	}}}); err == nil || !strings.Contains(err.Error(), "alter column") {
		t.Fatalf("alter wrap: %v", err)
	}
	if err := ensureIndexesForDesired(runtimeScope.Session().DB, DesiredSchema{}, "sqlite"); err != nil {
		t.Fatal(err)
	}
	if err := ensureIndexesForColumn(runtimeScope.Session().DB, "missing_idx_table", ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true,
	}, "sqlite"); err == nil {
		t.Fatal("expected create index failure")
	}
	origDialector := runtimeScope.Session().Config.Dialector
	t.Cleanup(func() { runtimeScope.Session().Config.Dialector = origDialector })
	runtimeScope.Session().Config.Dialector = fakeDialector{name: "postgres"}
	if err := applyPlan(runtimeScope, "postgres", SchemaPlan{Ops: []PlanOp{{
		Kind: OpEnsureCheck, Safety: SafetyAuto, Table: "widen_tbl",
		CheckName: "chk_widen_tbl_code", CheckExpr: "code <> ''",
	}}}); err == nil || !strings.Contains(err.Error(), "ensure check") {
		t.Fatalf("ensure check wrap: %v", err)
	}
	runtimeScope.Session().Config.Dialector = origDialector
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpCreateTable, Safety: SafetyAuto, Table: "widen_idx",
		Columns: []ColumnSpec{{Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true}},
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := ensureIndexesForColumn(runtimeScope.Session().DB, "widen_idx", ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true,
	}, "sqlite"); err != nil {
		t.Fatal(err)
	}
	if err := applyAlterColumnWiden(nil, "widen_tbl", ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(16),
	}, "sqlite"); err == nil {
		t.Fatal("expected nil db")
	}
	if err := applyAlterColumnWiden(runtimeScope.Session().DB, "no_such_widen_table", ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(16),
	}, "sqlite"); err == nil {
		t.Fatal("expected alter failure on missing table")
	}
	// Existing index should hit HasIndex continue path.
	if err := ensureIndexesForColumn(runtimeScope.Session().DB, "widen_tbl", ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true,
	}, "sqlite"); err != nil {
		t.Fatal(err)
	}
}

func TestInspect_SkipsNilAndEmptyIndexNames(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := runtimeScope.Session().Exec(`CREATE TABLE skip_idx (id integer)`).Error; err != nil {
		t.Fatal(err)
	}
	orig := getIndexes
	t.Cleanup(func() { getIndexes = orig })
	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return []gorm.Index{nil, fakeIndex{name: "  "}, fakeIndex{name: "idx_id", cols: []string{"", "id"}, unique: false}}, nil
	}
	live, err := inspectTables(runtimeScope.Session().DB, []string{"skip_idx", " ", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(live.Indexes["skip_idx"]) != 1 || live.Indexes["skip_idx"][0].Name != "idx_id" {
		t.Fatalf("indexes = %#v", live.Indexes["skip_idx"])
	}
}

func TestLiveColumnFromColumnType_Default(t *testing.T) {
	lc, ok := liveColumnFromColumnType(fakeColumnTypeWithDefault{fakeColumnType: fakeColumnType{name: "c", dbType: "TEXT"}, def: "x", defOK: true})
	if !ok || lc.Default == nil || *lc.Default != "x" {
		t.Fatalf("%#v", lc)
	}
	lc, ok = liveColumnFromColumnType(fakeColumnTypeWithDefault{fakeColumnType: fakeColumnType{name: "c", dbType: "TEXT"}, def: "  ", defOK: true})
	if !ok || lc.Default != nil {
		t.Fatalf("blank default %#v", lc)
	}
}

type fakeIndex struct {
	name   string
	cols   []string
	unique bool
}

func (f fakeIndex) Table() string            { return "" }
func (f fakeIndex) Name() string             { return f.name }
func (f fakeIndex) Columns() []string        { return f.cols }
func (f fakeIndex) PrimaryKey() (bool, bool) { return false, true }
func (f fakeIndex) Unique() (bool, bool)     { return f.unique, true }
func (f fakeIndex) Option() string           { return "" }

type fakeColumnTypeWithDefault struct {
	fakeColumnType
	def   string
	defOK bool
}

func (f fakeColumnTypeWithDefault) DefaultValue() (string, bool) { return f.def, f.defOK }
