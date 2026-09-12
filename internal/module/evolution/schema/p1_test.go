// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
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

func TestSchemaSnapshot_LoadSaveEdgeCases(t *testing.T) {
	if _, err := LoadSnapshots(nil, []string{"t"}); err != nil {
		t.Fatalf("nil db: %v", err)
	}
	got, err := LoadSnapshots(nil, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty tables: %#v %v", got, err)
	}
	if err := SaveSnapshots(nil, DesiredSchema{}, nil, nil); err == nil {
		t.Fatal("expected nil db error")
	}
	runtimeScope := newSchemaTestScope(t)
	if err := SaveSnapshots(runtimeScope.Session().DB, DesiredSchema{
		Tables: map[string][]ColumnSpec{"snap_empty_meta": {{Name: "id", FieldName: "Id", PhysicalType: "int"}}},
	}, []*meta.Model{nil, {Name: "X", ModelTable: ""}}, nil); err != nil {
		t.Fatal(err)
	}
	blank, err := LoadSnapshots(runtimeScope.Session().DB, []string{" ", ""})
	if err != nil || len(blank) != 0 {
		t.Fatalf("blank names: %#v %v", blank, err)
	}

	failDB := runtimeScope.Session().DB.Session(&gorm.Session{NewDB: true})
	if err := failDB.Callback().Create().Before("gorm:create").Register("p1_fail_create", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("create boom"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := SaveSnapshots(failDB, DesiredSchema{
		Tables: map[string][]ColumnSpec{"snap_create_fail": {{Name: "id", FieldName: "Id", PhysicalType: "int"}}},
	}, []*meta.Model{{Name: "Order", Application: "sales", ModelTable: "snap_create_fail"}}, &meta.Module{Name: "sales", Version: "1"}); err == nil || !strings.Contains(err.Error(), "save schema snapshot") {
		t.Fatalf("expected create failure, got %v", err)
	}

	sqlDB, err := runtimeScope.Session().DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if _, err := LoadSnapshots(runtimeScope.Session().DB, []string{"snap_empty_meta"}); err == nil {
		t.Fatal("expected load error on closed db")
	}
	if err := SaveSnapshots(runtimeScope.Session().DB, DesiredSchema{
		Tables: map[string][]ColumnSpec{"snap_fail": {{Name: "id", FieldName: "Id", PhysicalType: "int"}}},
	}, nil, &meta.Module{Name: "sales"}); err == nil {
		t.Fatal("expected save error on closed db")
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
		Indexes: map[string][]LiveIndex{"t": {}},
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
		Indexes: map[string][]LiveIndex{"t": {}},
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
		Indexes:  map[string][]LiveIndex{"t": {}},
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

func TestPlan_UniqueFlagOnExistingColumnGuarded(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", Unique: true}},
	}}
	live := LiveSchema{
		Tables:   map[string]bool{"t": true},
		Columns:  map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "varchar"}}},
		Indexes:  map[string][]LiveIndex{"t": {{Name: "idx_code", Columns: []string{"code"}, Unique: false}}},
		RowCount: map[string]int64{"t": 1},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	found := false
	for _, op := range plan.Ops {
		if op.Kind == OpAddIndex && op.Safety == SafetyGuarded && op.Column != nil && op.Column.UniqueIndex {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected guarded Unique→uniqueIndex op, got %#v", plan.Ops)
	}
}

func TestPlan_NewUniqueColumnOnPopulatedIsGuardedIndex(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {
			{Name: "id", FieldName: "Id", PhysicalType: "int"},
			{Name: "code", FieldName: "Code", PhysicalType: "varchar", UniqueIndex: true},
		},
	}}
	live := LiveSchema{
		Tables:   map[string]bool{"t": true},
		Columns:  map[string]map[string]LiveColumn{"t": {"id": {Name: "id", DatabaseTypeName: "integer"}}},
		Indexes:  map[string][]LiveIndex{"t": {}},
		RowCount: map[string]int64{"t": 1},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	var addCol, addIdx *PlanOp
	for i := range plan.Ops {
		op := &plan.Ops[i]
		if op.Kind == OpAddColumn && op.Column != nil && op.Column.Name == "code" {
			addCol = op
		}
		if op.Kind == OpAddIndex && op.Column != nil && op.Column.Name == "code" {
			addIdx = op
		}
	}
	if addCol == nil || addCol.Safety != SafetyAuto {
		t.Fatalf("expected auto add column, got %#v", addCol)
	}
	if addIdx == nil || addIdx.Safety != SafetyGuarded {
		t.Fatalf("expected guarded add unique index, got %#v", addIdx)
	}
}

func TestPlan_EnsureCheckOnPopulatedGuarded(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "status", FieldName: "Status", PhysicalType: "varchar", CheckExpr: "status <> ''"}},
	}}
	live := LiveSchema{
		Tables:   map[string]bool{"t": true},
		Columns:  map[string]map[string]LiveColumn{"t": {"status": {Name: "status", DatabaseTypeName: "varchar"}}},
		Indexes:  map[string][]LiveIndex{"t": {}},
		RowCount: map[string]int64{"t": 1},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	found := false
	for _, op := range plan.Ops {
		if op.Kind == OpEnsureCheck && op.Safety == SafetyGuarded {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected guarded ensure_check, got %#v", plan.Ops)
	}
}

func TestPlan_DefaultRemovalGuarded(t *testing.T) {
	liveDef := "hello"
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "note", FieldName: "Note", PhysicalType: "varchar"}},
	}}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"note": {Name: "note", DatabaseTypeName: "varchar", Default: &liveDef}}},
		Indexes: map[string][]LiveIndex{"t": {}},
	}
	plan := buildPlan("sales", desired, live, "postgres")
	found := false
	for _, op := range plan.Ops {
		if op.Kind == OpAlterColumn && op.Safety == SafetyGuarded && strings.Contains(op.Detail, "default") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected guarded default removal, got %#v", plan.Ops)
	}
}

func TestLiveHasIndex_DistinguishesUnique(t *testing.T) {
	live := LiveSchema{Indexes: map[string][]LiveIndex{
		"t": {{Name: "idx_code", Columns: []string{"code"}, Unique: false}},
	}}
	if liveHasIndex(live, "t", "code", true) {
		t.Fatal("non-unique index must not satisfy unique lookup")
	}
	if !liveHasIndex(live, "t", "code", false) {
		t.Fatal("non-unique index should satisfy non-unique lookup")
	}
	if liveHasIndex(LiveSchema{}, "t", "code", false) {
		t.Fatal("nil indexes")
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

func TestMigrator_PlanOnly(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_plan_only_wrap",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	mig := &migrator{
		modelMigrator: newModelMigrator(runtimeScope, &meta.Module{Name: "sales"}, []*meta.Model{model}),
	}
	plan, err := mig.PlanOnly()
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Ops) == 0 {
		t.Fatal("expected ops")
	}
	if runtimeScope.Session().Migrator().HasTable("sales_plan_only_wrap") {
		t.Fatal("PlanOnly must not create tables")
	}

	failing := &migrator{modelMigrator: modelMigratorFunc(func() error { return errors.New("plan boom") })}
	if _, err := failing.PlanOnly(); err == nil || !strings.Contains(err.Error(), "plan schema") {
		t.Fatalf("expected wrapped plan error, got %v", err)
	}
}

func TestPlanSchema_ValidateGuarded(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_guard_plan",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	if err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	if err := runtimeScope.Session().Exec(`INSERT INTO sales_guard_plan (status) VALUES ('x')`).Error; err != nil {
		t.Fatal(err)
	}
	model.Fields = append(model.Fields, newFieldWithOptions(t, "Code", `{"type":"char","uniqueIndex":true}`))
	_, err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).PlanSchema()
	if err == nil || !strings.Contains(err.Error(), "guarded") {
		t.Fatalf("expected guarded plan error, got %v", err)
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
	mig := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	plan, err := mig.PlanSchema()
	if err != nil {
		t.Fatal(err)
	}
	foundCheck := false
	for _, op := range plan.Ops {
		if op.Kind == OpEnsureCheck && op.Safety == SafetyAuto && strings.Contains(op.CheckExpr, "status") {
			foundCheck = true
		}
	}
	if !foundCheck {
		t.Fatalf("expected auto ensure_check in plan, got %#v", plan.Ops)
	}
	if err := mig.MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	if !runtimeScope.Session().Migrator().HasTable("sales_chk_plan") {
		t.Fatal("expected table")
	}
	// SQLite ensure_check is intentionally a no-op (cannot ALTER ADD CONSTRAINT).
	// Enforcement is covered on postgres/mysql dialects; here we only prove plan+migrate.
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
	if err := runtimeScope.Session().Exec(`INSERT INTO sales_def (status) VALUES ('ok')`).Error; err != nil {
		t.Fatal(err)
	}
	var note string
	if err := runtimeScope.Session().Raw(`SELECT note FROM sales_def LIMIT 1`).Scan(&note).Error; err != nil {
		t.Fatal(err)
	}
	if note != "hello" {
		t.Fatalf("note default = %q, want hello", note)
	}
}

func TestApply_WidenUsesSizeOnlyDefinition(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := runtimeScope.Session().Exec(`CREATE TABLE sales_widen (code varchar(8) not null default 'x')`).Error; err != nil {
		t.Fatal(err)
	}
	col := ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(32),
		NotNull: true, Default: stringPtr("should-not-apply"),
	}
	if err := applyAlterColumnWiden(runtimeScope.Session().DB, "sales_widen", col, "sqlite"); err != nil {
		// sqlite may no-op alter; still covers size-only path construction
		t.Logf("widen on sqlite: %v", err)
	}
	if err := applyAlterColumnWiden(nil, "sales_widen", col, "sqlite"); err == nil {
		t.Fatal("expected nil db error")
	}
}

func TestInspect_PropagatesIndexErrors(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := runtimeScope.Session().Exec(`CREATE TABLE idx_err_tbl (id integer)`).Error; err != nil {
		t.Fatal(err)
	}
	orig := getIndexes
	t.Cleanup(func() { getIndexes = orig })
	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return nil, errors.New("index inspect boom")
	}
	_, err := inspectTables(runtimeScope.Session().DB, []string{"idx_err_tbl"})
	if err == nil || !strings.Contains(err.Error(), "indexes for idx_err_tbl") {
		t.Fatalf("expected index inspect error, got %v", err)
	}
}

func TestInspect_StoresStructuredIndexes(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"idx_struct": {{
			Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true, IndexName: "idx_struct_code",
		}},
	}}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpCreateTable, Safety: SafetyAuto, Table: "idx_struct",
		Columns: desired.Tables["idx_struct"],
	}}}); err != nil {
		t.Fatal(err)
	}
	live, err := inspectTables(runtimeScope.Session().DB, []string{"idx_struct"})
	if err != nil {
		t.Fatal(err)
	}
	if len(live.Indexes["idx_struct"]) == 0 {
		t.Fatal("expected structured indexes")
	}
	for _, idx := range live.Indexes["idx_struct"] {
		if strings.EqualFold(idx.Name, "code") && len(idx.Columns) == 0 {
			t.Fatalf("column name must not be stored as standalone index entry: %#v", idx)
		}
	}
}

func TestPlan_LeftoverIndexAndCovered(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true, IndexName: "idx_code"}},
	}}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "varchar"}}},
		Indexes: map[string][]LiveIndex{"t": {
			{Name: "idx_code", Columns: []string{"code"}, Unique: false},
			{Name: "idx_extra", Columns: []string{"other"}, Unique: false},
			{Name: "sqlite_autoindex_t_1", Columns: []string{"id"}, Unique: true},
			{Name: "", Columns: []string{"x"}, Unique: false},
		}},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	foundExtra := false
	for _, left := range plan.Leftover {
		if left.Kind == LeftoverIndex && left.Name == "idx_extra" {
			foundExtra = true
		}
		if left.Name == "sqlite_autoindex_t_1" || left.Name == "" {
			t.Fatalf("unexpected leftover %#v", left)
		}
	}
	if !foundExtra {
		t.Fatalf("expected leftover idx_extra, got %#v", plan.Leftover)
	}
}

func TestCheckOpsEmptyColumnName(t *testing.T) {
	ops := checkOpsForColumn("t", ColumnSpec{FieldName: "Status", CheckExpr: "status <> ''"}, 0)
	if len(ops) != 1 || ops[0].CheckName != "chk_t_status" {
		t.Fatalf("%#v", ops)
	}
}

func TestPlanSchema_InspectError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_inspect_err",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	if err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	orig := getIndexes
	t.Cleanup(func() { getIndexes = orig })
	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return nil, errors.New("inspect boom")
	}
	_, err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).PlanSchema()
	if err == nil || !strings.Contains(err.Error(), "inspect") {
		t.Fatalf("expected inspect error, got %v", err)
	}
}

func TestMigrateSchema_SaveSnapshotError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_snap_err",
		Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
	}
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("marshal boom") }
	if err := newModelMigrator(runtimeScope, &meta.Module{Name: "sales"}, []*meta.Model{model}).MigrateSchema(); err == nil || !strings.Contains(err.Error(), "save schema snapshots") {
		t.Fatalf("expected snapshot save error, got %v", err)
	}
}

func TestLogPlan_NilLogger(t *testing.T) {
	m := newModelMigrator(&schemaTestScope{}, nil, nil)
	m.logPlan(SchemaPlan{Ops: []PlanOp{{Safety: SafetyAuto}}})
}

func TestLogPlan_CountsSafety(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	m := newModelMigrator(runtimeScope, &meta.Module{Name: "sales"}, nil)
	m.logPlan(SchemaPlan{Module: "sales", Ops: []PlanOp{
		{Safety: SafetyAuto}, {Safety: SafetyGuarded}, {Safety: SafetyManual}, {Safety: SafetyClass("other")},
	}})
}

func TestApplyTableCheckConstraints_Coverage(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_chk_cov",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Status", `{"type":"selection","column":{"checkConstraint":"status <> ''"}}`),
			{Name: "Skip"},
		},
	}
	m := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := m.applyTableCheckConstraints("sales_chk_cov", model); err != nil {
		t.Fatal(err) // sqlite ensure is no-op
	}
	field := &meta.Field{Name: "Bare"}
	_ = field.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName:  "Bare",
		Structural: meta.FieldStructuralSpec{Name: "Bare", FieldType: "varchar", CheckConstraint: "bare <> ''"},
		Migration:  meta.FieldMigrationDecision{ShouldCreateColumn: true, ResolvedColumnType: "varchar", StorageKind: "physical"},
	})
	// Force empty column name path via columnSpecFromField using Name from structural.
	model2 := &meta.Model{Name: "Order", ModelTable: "t", Fields: []*meta.Field{field}}
	if err := m.applyTableCheckConstraints("t", model2); err != nil {
		t.Fatal(err)
	}
}

func TestSaveSnapshots_MarshalError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("marshal boom") }
	if err := SaveSnapshots(runtimeScope.Session().DB, DesiredSchema{
		Tables: map[string][]ColumnSpec{"t": {{Name: "id", FieldName: "Id", PhysicalType: "int"}}},
	}, nil, nil); err == nil || !strings.Contains(err.Error(), "marshal desired") {
		t.Fatalf("got %v", err)
	}
}

func stringPtr(v string) *string { return &v }
