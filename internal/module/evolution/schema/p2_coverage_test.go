// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestHelpersDDL_FullCoverage(t *testing.T) {
	if err := (HelperOptions{}).validate(); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := (HelperOptions{DB: db}).validate(); err == nil || !strings.Contains(err.Error(), "IntentBag") {
		t.Fatalf("nil intents: %v", err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}

	// validate failures through each exported helper
	bad := HelperOptions{DB: db}
	if err := RenameColumn(bad, "t", "a", "b"); err == nil || !strings.Contains(err.Error(), "IntentBag") {
		t.Fatalf("rename IntentBag: %v", err)
	}
	if err := DropColumn(bad, "t", "c"); err == nil || !strings.Contains(err.Error(), "IntentBag") {
		t.Fatalf("drop col IntentBag: %v", err)
	}
	if err := DropIndex(bad, "t", "idx_x"); err == nil || !strings.Contains(err.Error(), "IntentBag") {
		t.Fatalf("drop idx IntentBag: %v", err)
	}
	if err := DropCheck(bad, "t", "chk_x"); err == nil || !strings.Contains(err.Error(), "IntentBag") {
		t.Fatalf("drop chk IntentBag: %v", err)
	}
	if err := DropForeignKey(bad, "t", "fk_x"); err == nil || !strings.Contains(err.Error(), "IntentBag") {
		t.Fatalf("drop fk IntentBag: %v", err)
	}
	if err := DropCheck(opts, "t", "not_allowed"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("sqlite dropCheck: %v", err)
	}
	if err := DropForeignKey(opts, "t", "not_allowed"); err == nil || !strings.Contains(err.Error(), "rejects") {
		t.Fatalf("fk reject: %v", err)
	}

	if opts.allowName("t", "", "idx_") || opts.allowName("t", "custom", "idx_") {
		t.Fatal("deny empty/custom")
	}
	if !opts.allowName("t", "idx_x", "idx_") || opts.allowName("t", "chk_x", "idx_") {
		t.Fatal("prefix scoped to helper")
	}
	if !opts.allowName("t", "chk_x", "chk_", "ck_") || !opts.allowName("t", "ck_x", "chk_", "ck_") {
		t.Fatal("check prefixes")
	}
	if !opts.allowName("t", "fk_x", "fk_") || opts.allowName("t", "idx_x", "fk_") {
		t.Fatal("fk prefix")
	}
	opts.AllowedNames = map[string]map[string]struct{}{"t": {"custom_idx": {}}}
	if !opts.allowName("t", "custom_idx", "idx_") {
		t.Fatal("allowed names")
	}

	if err := RenameColumn(opts, "", "a", "b"); err == nil {
		t.Fatal("rename empty")
	}
	if err := RenameColumn(opts, "t", "same", "same"); err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("rename same: %v", err)
	}
	if err := DropColumn(opts, "", "c"); err == nil {
		t.Fatal("drop empty")
	}
	if err := DropIndex(opts, "t", ""); err == nil {
		t.Fatal("drop index empty")
	}
	if err := DropCheck(opts, "t", ""); err == nil {
		t.Fatal("drop check empty")
	}
	if err := DropForeignKey(opts, "t", ""); err == nil {
		t.Fatal("drop fk empty")
	}

	if err := db.Exec(`CREATE TABLE help_rename (old_c text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := RenameColumn(opts, "help_rename", "old_c", "new_c"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE help_drop (id integer, gone text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := DropColumn(opts, "help_drop", "gone"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE help_idx (code text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_help_idx_code ON help_idx (code)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := DropIndex(opts, "help_idx", "idx_missing"); err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("index ownership: %v", err)
	}
	if err := DropIndex(opts, "help_idx", "idx_help_idx_code"); err != nil {
		t.Fatal(err)
	}
	if err := DropCheck(opts, "help_idx", "chk_help_idx_code"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("sqlite dropCheck: %v", err)
	}
	pgOpts := HelperOptions{DB: db, Dialect: "postgres", Intents: NewMemoryIntentBag()}
	if err := DropCheck(pgOpts, "help_idx", "not_allowed"); err == nil || !strings.Contains(err.Error(), "rejects") {
		t.Fatalf("pg check reject: %v", err)
	}
	if err := DropForeignKey(opts, "help_idx", "fk_help"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("sqlite fk: %v", err)
	}

	// Dialect SQL builders (no DB exec for foreign dialects).
	if got := dropIndexSQL("mysql", "t", "idx_t"); !strings.Contains(got, "ON") {
		t.Fatalf("mysql drop index: %s", got)
	}
	if got := dropIndexSQL("sqlserver", "t", "idx_t"); !strings.Contains(got, "ON") {
		t.Fatalf("sqlserver drop index: %s", got)
	}
	if got := dropIndexSQL("postgres", "t", "idx_t"); !strings.Contains(got, "IF EXISTS") {
		t.Fatalf("postgres drop index: %s", got)
	}
	if _, err := dropForeignKeySQL("mysql", "t", "fk_t"); err != nil {
		t.Fatal(err)
	}
	if _, err := dropForeignKeySQL("postgres", "t", "fk_t"); err != nil {
		t.Fatal(err)
	}
	if quoteIdent("mysql", "a`b") != "`a``b`" {
		t.Fatal("mysql quote")
	}
	if quoteIdent("sqlserver", "a]b") != "[a]]b]" {
		t.Fatal("sqlserver quote")
	}
	if quoteIdent("postgres", `a"b`) != `"a""b"` {
		t.Fatal("postgres quote")
	}

	// Exec error paths with closed DB.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	closed := HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}
	if err := RenameColumn(closed, "help_rename", "new_c", "x"); err == nil {
		t.Fatal("closed rename")
	}
	if err := DropColumn(closed, "help_drop", "id"); err == nil {
		t.Fatal("closed drop col")
	}
	if err := DropIndex(closed, "help_idx", "idx_help_idx_code"); err == nil {
		t.Fatal("closed drop idx")
	}
	if err := DropCheck(HelperOptions{DB: db, Dialect: "postgres", Intents: NewMemoryIntentBag()}, "t", "chk_t"); err == nil {
		t.Fatal("closed drop chk")
	}
	if err := DropForeignKey(HelperOptions{DB: db, Dialect: "postgres", Intents: NewMemoryIntentBag()}, "t", "fk_t"); err == nil {
		t.Fatal("closed drop fk")
	}

	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(*gorm.DB, string) error { return nil }
	fkBag := NewMemoryIntentBag()
	if err := DropForeignKey(HelperOptions{DB: db, Dialect: "postgres", Intents: fkBag}, "t", "fk_t"); err != nil {
		t.Fatal(err)
	}
	if len(fkBag.List()) != 1 {
		t.Fatalf("fk intent %#v", fkBag.List())
	}
}

func TestIntentSatisfies_AllBranches(t *testing.T) {
	var nilBag *memoryIntentBag
	nilBag.Add(Intent{Kind: IntentDropColumn, Table: "t", Name: "c"})
	if nilBag.List() != nil {
		t.Fatal("nil list")
	}
	nilBag.Clear()

	if IntentSatisfies(PlanOp{}, nil) {
		t.Fatal("nil bag")
	}
	if ContextWithIntentBag(nil, nil) == nil {
		t.Fatal("nil bag keeps ctx")
	}
	if IntentBagFromContext(context.Background()) != nil {
		t.Fatal("empty ctx")
	}

	bag := NewMemoryIntentBag()
	bag.Add(Intent{Kind: IntentRenameColumn, Table: "t", FromName: "old", Name: "new"})
	bag.Add(Intent{Kind: IntentDropColumn, Table: "t", Name: "c"})
	bag.Add(Intent{Kind: IntentDropIndex, Table: "t", Name: "idx_t"})
	bag.Add(Intent{Kind: IntentDropCheck, Table: "t", Name: "chk_t"})
	bag.Add(Intent{Kind: IntentDropForeignKey, Table: "t", Name: "fk_t"})

	if !IntentSatisfies(PlanOp{Kind: OpRenameColumn, Table: "t", FromName: "old", Column: &ColumnSpec{Name: "new"}}, bag) {
		t.Fatal("rename")
	}
	if IntentSatisfies(PlanOp{Kind: OpRenameColumn, Table: "t", FromName: "old", Column: &ColumnSpec{Name: "other"}}, bag) {
		t.Fatal("rename dest mismatch")
	}
	if IntentSatisfies(PlanOp{Kind: OpRenameColumn, Table: "t", FromName: "nope", Column: &ColumnSpec{Name: "new"}}, bag) {
		t.Fatal("rename miss")
	}
	emptyNameBag := NewMemoryIntentBag()
	emptyNameBag.Add(Intent{Kind: IntentRenameColumn, Table: "t", FromName: "old", Name: ""})
	if IntentSatisfies(PlanOp{Kind: OpRenameColumn, Table: "t", FromName: "old", Column: &ColumnSpec{Name: "new"}}, emptyNameBag) {
		t.Fatal("empty rename Name must not wildcard")
	}
	if !IntentSatisfies(PlanOp{Kind: OpKind(IntentDropColumn), Table: "t", Column: &ColumnSpec{Name: "c"}}, bag) {
		t.Fatal("drop col")
	}
	if !IntentSatisfies(PlanOp{Kind: OpKind(IntentDropIndex), Table: "t", IndexName: "idx_t"}, bag) {
		t.Fatal("drop idx")
	}
	if !IntentSatisfies(PlanOp{Kind: OpKind(IntentDropCheck), Table: "t", CheckName: "chk_t"}, bag) {
		t.Fatal("drop chk")
	}
	if !IntentSatisfies(PlanOp{Kind: OpKind(IntentDropForeignKey), Table: "t", Detail: "fk_t"}, bag) {
		t.Fatal("drop fk detail name")
	}
	if !IntentSatisfies(PlanOp{Kind: OpKind(IntentDropForeignKey), Table: "t", Detail: "drop foreign key fk_t"}, bag) {
		t.Fatal("drop fk detail prefix")
	}
	if intentOpName(PlanOp{Detail: "drop column old_code"}) != "old_code" {
		t.Fatal("strip drop column prefix")
	}
	if intentKindForOp(PlanOp{Detail: "please drop column x", Safety: SafetyManual}) != IntentDropColumn {
		t.Fatal("detail drop column")
	}
	if intentKindForOp(PlanOp{Detail: "drop index x"}) != IntentDropIndex {
		t.Fatal("detail drop index")
	}
	if intentKindForOp(PlanOp{Detail: "drop check x"}) != IntentDropCheck {
		t.Fatal("detail drop check")
	}
	if intentKindForOp(PlanOp{Detail: "drop foreign key x"}) != IntentDropForeignKey {
		t.Fatal("detail drop fk")
	}
	if intentKindForOp(PlanOp{Detail: "noop"}) != "" {
		t.Fatal("unknown")
	}
	if IntentSatisfies(PlanOp{Kind: OpAlterColumn, Table: "t", Detail: "widen"}, bag) {
		t.Fatal("unguarded kind")
	}
}

func TestPlan_RenameConflictKeepsLeftover(t *testing.T) {
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code"}},
	}}
	live := LiveSchema{
		Tables: map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {
			"code":     {Name: "code", DatabaseTypeName: "varchar"},
			"old_code": {Name: "old_code", DatabaseTypeName: "varchar"},
		}},
	}
	plan, err := buildPlan("sales", desired, live, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	foundConflict, foundLeftover := false, false
	for _, op := range plan.Ops {
		if op.Kind == OpRenameColumn && op.Safety == SafetyGuarded {
			foundConflict = true
		}
	}
	for _, left := range plan.Leftover {
		if left.Kind == LeftoverColumn && strings.EqualFold(left.Name, "old_code") {
			foundLeftover = true
		}
	}
	if !foundConflict || !foundLeftover {
		t.Fatalf("conflict=%v leftover=%v plan=%#v", foundConflict, foundLeftover, plan)
	}
}

func TestRenamePhysicalCompatibleEdges(t *testing.T) {
	if renamePhysicalCompatible(ColumnSpec{PhysicalType: ""}, LiveColumn{DatabaseTypeName: "text"}, "sqlite") {
		t.Fatal("empty want must be incompatible")
	}
	if renamePhysicalCompatible(ColumnSpec{PhysicalType: "varchar"}, LiveColumn{DatabaseTypeName: ""}, "postgres") {
		t.Fatal("empty have must be incompatible")
	}
	if !renamePhysicalCompatible(ColumnSpec{PhysicalType: "varchar"}, LiveColumn{DatabaseTypeName: "TEXT"}, "sqlite") {
		t.Fatal("sqlite compat")
	}
	if renamePhysicalCompatible(ColumnSpec{PhysicalType: "integer"}, LiveColumn{DatabaseTypeName: "varchar"}, "postgres") {
		t.Fatal("mismatch")
	}
}

func TestPlan_RenameFromClaimConflicts(t *testing.T) {
	live := LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"old_code": {Name: "old_code", DatabaseTypeName: "varchar"}}},
	}
	_, err := buildPlan("sales", DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {
			{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code"},
			{Name: "note", FieldName: "Note", PhysicalType: "varchar", RenameFrom: "old_code"},
		},
	}}, live, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "duplicate renameFrom") {
		t.Fatalf("duplicate: %v", err)
	}

	_, err = buildPlan("sales", DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {
			{Name: "old_code", FieldName: "OldCode", PhysicalType: "varchar"},
			{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "old_code"},
		},
	}}, live, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "conflicts with desired column") {
		t.Fatalf("desired collision: %v", err)
	}

	plan, err := buildPlan("sales", DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "code", FieldName: "Code", PhysicalType: "varchar", RenameFrom: "code"}},
	}}, LiveSchema{
		Tables:  map[string]bool{"t": true},
		Columns: map[string]map[string]LiveColumn{"t": {"code": {Name: "code", DatabaseTypeName: "varchar"}}},
	}, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range plan.Ops {
		if op.Kind == OpRenameColumn {
			t.Fatalf("self renameFrom must be a no-op, got %#v", op)
		}
	}
}

func TestApply_RenameColumnErrorPaths(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpRenameColumn, Safety: SafetyAuto, Table: "t",
	}}}); err == nil || !strings.Contains(err.Error(), "missing column") {
		t.Fatalf("nil column: %v", err)
	}
	col := ColumnSpec{Name: "code", FieldName: "Code", PhysicalType: "varchar"}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpRenameColumn, Safety: SafetyAuto, Table: "t", Column: &col,
	}}}); err == nil || !strings.Contains(err.Error(), "missing from") {
		t.Fatalf("empty from: %v", err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpRenameColumn, Safety: SafetyAuto, Table: "missing_tbl",
		Column: &col, FromName: "old",
	}}}); err == nil || !strings.Contains(err.Error(), "rename column") {
		t.Fatalf("rename apply wrap: %v", err)
	}
	if err := renameColumn(nil, "t", "a", col, "sqlite"); err == nil {
		t.Fatal("nil db")
	}
	if err := renameColumn(runtimeScope.Session().DB, "t", "", col, "sqlite"); err == nil {
		t.Fatal("empty from")
	}
	bad := ColumnSpec{Name: "x", FieldName: "", PhysicalType: "nope"}
	_ = renameColumn(runtimeScope.Session().DB, "missing", "a", bad, "sqlite")
}

func TestMigratorOptions_Coverage(t *testing.T) {
	WithIntentBag(NewMemoryIntentBag())(nil)
	WithToVersion("1")(nil)
	m := &migrator{modelMigrator: modelMigratorFunc(func() error { return nil })}
	WithIntentBag(NewMemoryIntentBag())(m)
	WithToVersion("2.0.0")(m)

	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())
	mod := &meta.Module{Name: "sales", Version: "1.0.0", ApplicationStr: "sales"}
	got, err := NewMigrator(runtimeScope, mod, nil, WithIntentBag(NewMemoryIntentBag()), WithToVersion("1.0.0"))
	if err != nil {
		// module may lack models in empty DB — option application still ran
		t.Log(err)
	}
	_ = got
}

func TestWarnDropAfterLeftovers_Coverage(t *testing.T) {
	m := &modelMigrator{}
	m.warnDropAfterLeftovers(SchemaPlan{}) // empty toVersion

	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())

	m = newModelMigrator(runtimeScope, &meta.Module{Name: "sales", Version: "2.0.0"}, nil)
	m.toVersion = "2.0.0"
	m.warnDropAfterLeftovers(SchemaPlan{Leftover: []Leftover{{Kind: LeftoverIndex, Table: "t", Name: "idx"}}})

	if err := runtimeScope.Session().Exec(`CREATE TABLE warn_tbl (old_code text)`).Error; err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal([]ColumnSpec{{Name: "old_code", DropAfter: "2.0.0"}})
	row := modmeta.SchemaSnapshot{ModelTable: "warn_tbl", DesiredJSON: datatypes.JSON(payload)}
	if err := runtimeScope.Session().Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	m.warnDropAfterLeftovers(SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "warn_tbl", Name: "old_code"},
		{Kind: LeftoverIndex, Table: "warn_tbl", Name: "idx_x"},
		{Kind: LeftoverColumn, Table: "missing", Name: "x"},
	}})

	bag := NewMemoryIntentBag()
	bag.Add(Intent{Kind: IntentDropColumn, Table: "warn_tbl", Name: "old_code"})
	m.intents = bag
	m.warnDropAfterLeftovers(SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "warn_tbl", Name: "old_code"},
	}})

	payloadBad := datatypes.JSON([]byte(`{`))
	_ = runtimeScope.Session().Model(&modmeta.SchemaSnapshot{}).Where("model_table = ?", "warn_tbl").Update("desired_json", payloadBad)
	m.intents = nil
	m.warnDropAfterLeftovers(SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "warn_tbl", Name: "old_code"},
	}})

	m.toVersion = "9.9.9"
	payload2, _ := json.Marshal([]ColumnSpec{{Name: "old_code", DropAfter: "2.0.0"}})
	_ = runtimeScope.Session().Model(&modmeta.SchemaSnapshot{}).Where("model_table = ?", "warn_tbl").Update("desired_json", datatypes.JSON(payload2))
	m.warnDropAfterLeftovers(SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "warn_tbl", Name: "other"},
		{Kind: LeftoverColumn, Table: "warn_tbl", Name: "old_code"},
	}})
}

func TestBuildSchemaPlan_RenameFromError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	field := newFieldWithOptions(t, "Code", `{"type":"varchar","size":32,"renameFrom":"OldCode"}`)
	model := &meta.Model{Name: "Order", Application: "sales", ModelTable: "sales_rename_err", Fields: []*meta.Field{field}}
	if err := runtimeScope.Session().Exec(`CREATE TABLE sales_rename_err (id integer)`).Error; err != nil {
		t.Fatal(err)
	}
	m := newModelMigrator(runtimeScope, &meta.Module{Name: "sales"}, []*meta.Model{model})
	_, _, err := m.buildSchemaPlan()
	if err == nil || !strings.Contains(err.Error(), "neither old column") {
		t.Fatalf("got %v", err)
	}
}

// Ensure closed-db LoadSnapshots path in warn is hit when possible.
func TestWarnDropAfter_LoadSnapshotsEmpty(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	m := newModelMigrator(runtimeScope, nil, nil)
	m.toVersion = "1"
	m.warnDropAfterLeftovers(SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "no_snap", Name: "c"},
	}})
}
