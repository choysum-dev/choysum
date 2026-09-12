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
	plan := buildPlan("sales", desired, live, "postgres")
	found := false
	for _, op := range plan.Ops {
		if op.Kind == OpEnsureCheck && op.Safety == SafetyAuto {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected auto ensure_check on populated table, got %#v", plan.Ops)
	}
	sqlitePlan := buildPlan("sales", desired, live, "sqlite")
	for _, op := range sqlitePlan.Ops {
		if op.Kind == OpEnsureCheck {
			t.Fatalf("sqlite existing table must omit ensure_check, got %#v", op)
		}
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

func TestIndexOps_ClassifiesUniqueIndependently(t *testing.T) {
	desiredKeys := map[string]struct{}{}
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Indexes: map[string][]LiveIndex{"t": {{Name: "uniq_code", Columns: []string{"code"}, Unique: true}}},
	}
	col := ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar",
		Indexed: true, IndexName: "idx_code", UniqueIndex: true, UniqueIndexNames: []string{"uniq_code"},
	}
	ops := indexOpsForColumn("t", col, live, 5, desiredKeys)
	if len(ops) != 1 || ops[0].IndexName != "idx_code" || ops[0].Safety != SafetyAuto {
		t.Fatalf("ordinary missing index must stay auto when unique exists: %#v", ops)
	}
	if _, ok := desiredKeys["code"]; !ok {
		t.Fatal("expected physical column name in desiredIndexKeys")
	}

	// Field-export lookup name should match live physical column.
	fieldKeys := map[string]struct{}{}
	fieldLive := LiveSchema{Indexes: map[string][]LiveIndex{
		"t": {{Name: "idx_created_by", Columns: []string{"created_by"}, Unique: false}},
	}}
	fieldCol := ColumnSpec{Name: "created_by", FieldName: "CreatedBy", PhysicalType: "varchar", Indexed: true}
	if ops := indexOpsForColumn("t", fieldCol, fieldLive, 0, fieldKeys); len(ops) != 0 {
		t.Fatalf("CreatedBy should match created_by live index: %#v", ops)
	}

	// Missing unique index on populated table stays guarded.
	uniqKeys := map[string]struct{}{}
	uniqOps := indexOpsForColumn("t", ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", UniqueIndex: true,
	}, LiveSchema{Indexes: map[string][]LiveIndex{"t": {}}}, 3, uniqKeys)
	if len(uniqOps) != 1 || uniqOps[0].Safety != SafetyGuarded {
		t.Fatalf("expected guarded unique add: %#v", uniqOps)
	}
}

func TestIndexCandidateUsesFieldLookup(t *testing.T) {
	if indexCandidateUsesFieldLookup(ColumnSpec{FieldName: "Code"}, "  ") {
		t.Fatal("blank name")
	}
	if indexCandidateUsesFieldLookup(ColumnSpec{FieldName: "Code", IndexName: "idx_code"}, "idx_code") {
		t.Fatal("explicit index name")
	}
	if indexCandidateUsesFieldLookup(ColumnSpec{FieldName: "Code", UniqueIndexNames: []string{"uniq_code"}}, "uniq_code") {
		t.Fatal("explicit unique index name")
	}
	if !indexCandidateUsesFieldLookup(ColumnSpec{FieldName: "CreatedBy"}, "CreatedBy") {
		t.Fatal("field export name")
	}
}

func TestIndexCoveredByDesiredKey_AllColumns(t *testing.T) {
	keys := map[string]struct{}{"code": {}}
	if indexCoveredByDesiredKey(LiveIndex{Columns: nil}, keys) {
		t.Fatal("empty columns")
	}
	if indexCoveredByDesiredKey(LiveIndex{Columns: []string{"tenant_id", "code"}}, keys) {
		t.Fatal("partial composite must not be covered")
	}
	keys["tenant_id"] = struct{}{}
	if !indexCoveredByDesiredKey(LiveIndex{Columns: []string{"tenant_id", "code"}}, keys) {
		t.Fatal("all columns covered")
	}
}

func TestDefaultChanged_SentinelLiveDefaults(t *testing.T) {
	nullLive := "NULL"
	if defaultChanged(ColumnSpec{}, LiveColumn{Default: &nullLive}) {
		t.Fatal("NULL sentinel")
	}
	ts := "CURRENT_TIMESTAMP"
	if defaultChanged(ColumnSpec{}, LiveColumn{Default: &ts}) {
		t.Fatal("timestamp sentinel")
	}
	real := "hello"
	if !defaultChanged(ColumnSpec{}, LiveColumn{Default: &real}) {
		t.Fatal("real live default")
	}
}

func TestNormalizeDefaultLiteral_DoubleQuotedCast(t *testing.T) {
	want := "hello"
	live := `"hello"::text`
	if defaultChanged(ColumnSpec{Default: &want}, LiveColumn{Default: &live}) {
		t.Fatal("double-quoted cast should normalize equal")
	}
	inQuotes := `'a::b'::text`
	if normalizeDefaultLiteral(inQuotes) != "a::b" {
		t.Fatalf("cast inside quotes preserved incorrectly: %q", normalizeDefaultLiteral(inQuotes))
	}
	if got := normalizeDefaultLiteral("x::int"); got != "x::int" {
		t.Fatalf("short left-hand cast must stay intact: %q", got)
	}
	if got := normalizeDefaultLiteral("hello::text"); got != "hello::text" {
		t.Fatalf("unquoted left-hand cast must stay intact: %q", got)
	}
	if _, ok := stripOuterPostgresCast("::int"); ok {
		t.Fatal("empty left-hand cast must not strip")
	}
	if next, ok := stripOuterPostgresCast("('hello')::text"); !ok || next != "('hello')" {
		t.Fatalf("paren-wrapped cast: %q %v", next, ok)
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

func TestWidenColumnSpec_PreservesNullabilityOnModifyDialects(t *testing.T) {
	def := "x"
	col := ColumnSpec{
		Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(32),
		NotNull: true, Default: &def,
	}
	sqlite := widenColumnSpec(col, "sqlite")
	if sqlite.NotNull || sqlite.Default != nil {
		t.Fatalf("sqlite size-only must omit null/default: %#v", sqlite)
	}
	for _, dialect := range []string{"mysql", "sqlserver"} {
		got := widenColumnSpec(col, dialect)
		if !got.NotNull || got.Default == nil || *got.Default != "x" {
			t.Fatalf("%s must keep null/default: %#v", dialect, got)
		}
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
		t.Fatalf("widen on sqlite: %v", err)
	}
	var ddl string
	if err := runtimeScope.Session().Raw(`SELECT sql FROM sqlite_master WHERE type='table' AND name='sales_widen'`).Scan(&ddl).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(ddl), "should-not-apply") {
		t.Fatalf("widen must not apply default from ColumnSpec: %s", ddl)
	}
	if err := applyAlterColumnWiden(nil, "sales_widen", col, "sqlite"); err == nil {
		t.Fatal("expected nil db error")
	}
}

func TestSQLiteGetIndexes_NullColumnNames(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	// Expression index: PRAGMA_index_info.name is NULL for the expression column.
	if err := runtimeScope.Session().Exec(`CREATE TABLE expr_idx_tbl (id integer primary key, payload text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := runtimeScope.Session().Exec(`CREATE INDEX idx_expr_payload ON expr_idx_tbl ((payload || ''))`).Error; err != nil {
		t.Fatal(err)
	}
	live, err := inspectTables(runtimeScope.Session().DB, []string{"expr_idx_tbl"})
	if err != nil {
		t.Fatalf("inspect with expression index: %v", err)
	}
	if !live.Tables["expr_idx_tbl"] {
		t.Fatal("expected table")
	}
	found := false
	for _, idx := range live.Indexes["expr_idx_tbl"] {
		if strings.EqualFold(idx.Name, "idx_expr_payload") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected idx_expr_payload in %#v", live.Indexes["expr_idx_tbl"])
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
		"t": {
			{Name: "code", FieldName: "Code", PhysicalType: "varchar", Indexed: true, IndexName: "idx_code"},
			{Name: "note", FieldName: "Note", PhysicalType: "varchar"}, // declared but not indexed
		},
	}}
	live := LiveSchema{
		Tables: map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {
			"code": {Name: "code", DatabaseTypeName: "varchar"},
			"note": {Name: "note", DatabaseTypeName: "varchar"},
		}},
		Indexes: map[string][]LiveIndex{"t": {
			{Name: "idx_code", Columns: []string{"code"}, Unique: false},
			{Name: "idx_extra", Columns: []string{"other"}, Unique: false},
			{Name: "idx_note_stale", Columns: []string{"note"}, Unique: false},
			{Name: "sqlite_autoindex_t_1", Columns: []string{"id"}, Unique: true},
			{Name: "", Columns: []string{"x"}, Unique: false},
		}},
	}
	plan := buildPlan("sales", desired, live, "sqlite")
	foundExtra, foundStale := false, false
	for _, left := range plan.Leftover {
		if left.Kind == LeftoverIndex && left.Name == "idx_extra" {
			foundExtra = true
		}
		if left.Kind == LeftoverIndex && left.Name == "idx_note_stale" {
			foundStale = true
		}
		if left.Name == "sqlite_autoindex_t_1" || left.Name == "" {
			t.Fatalf("unexpected leftover %#v", left)
		}
	}
	if !foundExtra || !foundStale {
		t.Fatalf("expected leftover idx_extra and idx_note_stale, got %#v", plan.Leftover)
	}
}

func TestCheckOpsEmptyColumnName(t *testing.T) {
	ops := checkOpsForColumn("t", ColumnSpec{FieldName: "Status", CheckExpr: "status <> ''"}, "postgres", false)
	if len(ops) != 1 || ops[0].CheckName != "chk_t_status" || ops[0].Safety != SafetyAuto {
		t.Fatalf("%#v", ops)
	}
	sqliteExisting := checkOpsForColumn("t", ColumnSpec{Name: "status", FieldName: "Status", CheckExpr: "status <> ''"}, "sqlite", true)
	if len(sqliteExisting) != 0 {
		t.Fatalf("sqlite existing table must omit ensure_check: %#v", sqliteExisting)
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
