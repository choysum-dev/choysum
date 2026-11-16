// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func TestObsoleteColumnName(t *testing.T) {
	t.Parallel()
	if got := ObsoleteColumnName("status_code"); got != "status_code__obsolete" {
		t.Fatalf("got %q", got)
	}
	if ObsoleteColumnName("  ") != "" || ObsoleteColumnName("") != "" {
		t.Fatal("empty input should yield empty obsolete name")
	}
}

func TestReplaceColumn_FullFlowIntToVarchar(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	table := "sales_order_replace"
	field := newFieldWithOptions(t, "CodeVal", `{"type":"int"}`)
	model := &meta.Model{
		Name:       "Order",
		Path:       "sales/order_replace.ts",
		ModelTable: table,
		Fields:     []*meta.Field{field},
	}
	mig := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("initial MigrateSchema: %v", err)
	}
	db := runtimeScope.Session().DB
	if err := db.Exec(`INSERT INTO ` + table + ` (code_val) VALUES (1), (2)`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	spec, err := field.GetResolvedSpec()
	if err != nil {
		t.Fatalf("GetResolvedSpec: %v", err)
	}
	spec.Migration.ResolvedColumnType = "varchar"
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}

	mig = newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	err = mig.MigrateSchema()
	if err == nil || !strings.Contains(err.Error(), "schema plan has guarded/manual operations") {
		t.Fatalf("expected guarded type-change, got %v", err)
	}
	if !strings.Contains(err.Error(), "ReplaceColumn") || !strings.Contains(err.Error(), "replaceColumnPre") {
		t.Fatalf("expected ReplaceColumn hint, got %v", err)
	}

	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: table, FromColumn: "code_val"}); err != nil {
		t.Fatalf("ReplaceColumnPre: %v", err)
	}
	if db.Migrator().HasColumn(table, "code_val") {
		t.Fatal("code_val should have been renamed away")
	}
	if !db.Migrator().HasColumn(table, "code_val__obsolete") {
		t.Fatal("expected obsolete column")
	}

	mig = newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	mig.intents = bag
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema after Pre: %v", err)
	}
	if !db.Migrator().HasColumn(table, "code_val") {
		t.Fatal("expected new code_val column")
	}
	if !db.Migrator().HasColumn(table, "code_val__obsolete") {
		t.Fatal("obsolete leftover must remain until Drop")
	}

	if err := ReplaceColumnCopyCast(opts, table, "code_val__obsolete", "code_val", "int_to_varchar"); err != nil {
		t.Fatalf("CopyCast: %v", err)
	}
	var vals []string
	if err := db.Raw(`SELECT code_val FROM ` + table + ` ORDER BY code_val`).Scan(&vals).Error; err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(vals) != 2 || vals[0] != "1" || vals[1] != "2" {
		t.Fatalf("copied values = %#v", vals)
	}

	if err := ReplaceColumnDropObsolete(opts, table, "code_val"); err != nil {
		t.Fatalf("DropObsolete: %v", err)
	}
	if db.Migrator().HasColumn(table, "code_val__obsolete") {
		t.Fatal("obsolete column should be gone")
	}
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema after Drop: %v", err)
	}

	// Drop is idempotent when obsolete is already gone.
	if err := ReplaceColumnDropObsolete(opts, table, "code_val"); err != nil {
		t.Fatalf("idempotent Drop: %v", err)
	}
}

func TestReplaceColumnPre_IdempotentAfterRename(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_pre_once (code_val integer, keep text)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_pre_once", FromColumn: "code_val"}); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_pre_once", FromColumn: "code_val"}); err != nil {
		t.Fatalf("second Pre should no-op when only obsolete remains: %v", err)
	}
}

func TestReplaceColumnPre_InterruptedSameNameIsNoop(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	// After Pre+Migrate with From==To: both the new column and __obsolete remain (e.g. Post failed).
	if err := db.Exec(`CREATE TABLE replace_interrupted (code integer, code__obsolete integer, keep text)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_interrupted", FromColumn: "code"}); err != nil {
		t.Fatalf("interrupted same-name Pre must no-op: %v", err)
	}
}

func TestReplaceColumnPre_ObsoleteConflictDifferentTo(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_conflict (old_code integer, old_code__obsolete integer)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	err := ReplaceColumnPre(opts, ReplaceColumnOptions{
		Table: "replace_conflict", FromColumn: "old_code", ToColumn: "status_code",
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("got %v", err)
	}
}

func TestReplaceColumn_IndexedFieldSqlite(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	table := "sales_order_replace_idx"
	field := newFieldWithOptions(t, "CodeVal", `{"type":"int","indexed":true}`)
	model := &meta.Model{
		Name:       "Order",
		Path:       "sales/order_replace_idx.ts",
		ModelTable: table,
		Fields:     []*meta.Field{field},
	}
	mig := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("initial MigrateSchema: %v", err)
	}
	db := runtimeScope.Session().DB
	if err := db.Exec(`INSERT INTO ` + table + ` (code_val) VALUES (9)`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	spec, err := field.GetResolvedSpec()
	if err != nil {
		t.Fatal(err)
	}
	spec.Migration.ResolvedColumnType = "varchar"
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatal(err)
	}

	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: table, FromColumn: "code_val"}); err != nil {
		t.Fatalf("Pre with index: %v", err)
	}
	mig = newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	mig.intents = bag
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("Migrate after Pre: %v", err)
	}
	if err := ReplaceColumnCopyCast(opts, table, "code_val__obsolete", "code_val", "as_text"); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceColumnDropObsolete(opts, table, "code_val"); err != nil {
		t.Fatalf("DropObsolete with prior index: %v", err)
	}
	if db.Migrator().HasColumn(table, "code_val__obsolete") {
		t.Fatal("obsolete should be gone")
	}
	// Recreate index on the new column.
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("Migrate after Drop: %v", err)
	}
}

func TestReplaceColumnPre_ToColumnDifferent(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_rename (old_code integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO replace_rename (old_code) VALUES (7)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{
		Table: "replace_rename", FromColumn: "old_code", ToColumn: "status_code",
	}); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasColumn("replace_rename", "old_code__obsolete") {
		t.Fatal("expected obsolete")
	}
	if db.Migrator().HasColumn("replace_rename", "old_code") {
		t.Fatal("old_code should be renamed")
	}
	// Simulate Migrate adding the new column.
	if err := db.Exec(`ALTER TABLE replace_rename ADD COLUMN status_code text`).Error; err != nil {
		t.Fatal(err)
	}
	if err := ReplaceColumnCopyCast(opts, "replace_rename", "old_code__obsolete", "status_code", "int_to_varchar"); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := db.Raw(`SELECT status_code FROM replace_rename`).Scan(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != "7" {
		t.Fatalf("got %q", got)
	}
}

func TestReplaceColumnCopyCast_UnknownKind(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_kind (a integer, b text)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	err := ReplaceColumnCopyCast(HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}, "replace_kind", "a", "b", "nope")
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePlan_ReplaceColumnHintOnTypeChange(t *testing.T) {
	t.Parallel()
	plan := SchemaPlan{
		Ops: []PlanOp{{
			Kind:   OpAlterColumn,
			Safety: SafetyGuarded,
			Table:  "t",
			Detail: "type change int → varchar (desired physical varchar)",
		}},
	}
	err := ValidatePlan(plan, nil)
	if err == nil || !strings.Contains(err.Error(), replaceColumnHint) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "int_to_varchar") {
		t.Fatalf("hint should list copy kinds, got %v", err)
	}
}

func TestValidatePlan_ReplaceColumnHintOnNarrowSize(t *testing.T) {
	t.Parallel()
	plan := SchemaPlan{
		Ops: []PlanOp{{
			Kind:   OpAlterColumn,
			Safety: SafetyGuarded,
			Table:  "t",
			Detail: "narrow size 255 → 50",
		}},
	}
	err := ValidatePlan(plan, nil)
	if err == nil || !strings.Contains(err.Error(), replaceColumnHint) {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePlan_NoReplaceHintOnUnrelatedGuarded(t *testing.T) {
	t.Parallel()
	plan := SchemaPlan{
		Ops: []PlanOp{{
			Kind:   OpAddColumn,
			Safety: SafetyGuarded,
			Table:  "t",
			Detail: "add NOT NULL column on non-empty table",
		}},
	}
	err := ValidatePlan(plan, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "ReplaceColumn") {
		t.Fatalf("should not hint ReplaceColumn for add NOT NULL: %v", err)
	}
}

func TestCopyCastExpr_DialectsAndKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		dialect string
		kind    string
		want    string
	}{
		{"mysql", "int_to_varchar", `CAST("c" AS CHAR)`},
		{"mariadb", "as_text", `CAST("c" AS CHAR)`},
		{"sqlserver", "int_to_varchar", `CAST("c" AS NVARCHAR(MAX))`},
		{"sqlite", "as_text", `CAST("c" AS TEXT)`},
		{"postgres", "int_to_varchar", `CAST("c" AS TEXT)`},
	}
	for _, tc := range cases {
		got, err := copyCastExpr(tc.dialect, `"c"`, tc.kind)
		if err != nil {
			t.Fatalf("%s/%s: %v", tc.dialect, tc.kind, err)
		}
		if got != tc.want {
			t.Fatalf("%s/%s: got %q want %q", tc.dialect, tc.kind, got, tc.want)
		}
	}
	if _, err := copyCastExpr("sqlite", `"c"`, "nope"); err == nil {
		t.Fatal("expected unknown kind")
	}
}

func TestReplaceColumnHelpers_ErrorPaths(t *testing.T) {
	// Not parallel: stubs getColumnTypes / helperExec.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_errs (a integer, b text)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}

	if err := ReplaceColumnPre(HelperOptions{}, ReplaceColumnOptions{Table: "t", FromColumn: "a"}); err == nil {
		t.Fatal("Pre validate")
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{}); err == nil || !strings.Contains(err.Error(), "requires table") {
		t.Fatalf("Pre empty: %v", err)
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_errs", FromColumn: "missing"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Pre missing: %v", err)
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_errs", FromColumn: "missing", ToColumn: "b"}); err != nil {
		t.Fatalf("Pre !from && hasTo no-op: %v", err)
	}

	if err := ReplaceColumnCopyCast(HelperOptions{}, "replace_errs", "a", "b", "as_text"); err == nil {
		t.Fatal("CopyCast validate")
	}
	if err := ReplaceColumnCopyCast(opts, "", "a", "b", "as_text"); err == nil {
		t.Fatal("CopyCast empty")
	}
	if err := ReplaceColumnCopyCast(opts, "replace_errs", "missing", "b", "as_text"); err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("CopyCast missing col: %v", err)
	}

	if err := ReplaceColumnDropObsolete(HelperOptions{}, "replace_errs", "a"); err == nil {
		t.Fatal("Drop validate")
	}
	if err := ReplaceColumnDropObsolete(opts, "", "a"); err == nil {
		t.Fatal("Drop empty")
	}

	if _, err := helperHasColumn(nil, "t", "a"); err == nil {
		t.Fatal("helperHasColumn nil db")
	}

	origCT := getColumnTypes
	t.Cleanup(func() { getColumnTypes = origCT })
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return nil, fmt.Errorf("boom")
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_errs", FromColumn: "a"}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Pre hasColumn err: %v", err)
	}
	if err := ReplaceColumnCopyCast(opts, "replace_errs", "a", "b", "as_text"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("CopyCast hasColumn err: %v", err)
	}
	if err := ReplaceColumnDropObsolete(opts, "replace_errs", "a"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Drop hasColumn err: %v", err)
	}
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return []gorm.ColumnType{nil, fakeColumnType{name: "a", dbType: "INTEGER"}}, nil
	}
	has, err := helperHasColumn(db, "replace_errs", "a")
	if err != nil || !has {
		t.Fatalf("nil ct skip: has=%v err=%v", has, err)
	}

	getColumnTypes = origCT
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(*gorm.DB, string) error { return fmt.Errorf("exec boom") }
	if err := ReplaceColumnCopyCast(opts, "replace_errs", "a", "b", "as_text"); err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("CopyCast exec: %v", err)
	}
	helperExec = origExec
}

func TestReplaceColumnHelpers_StubsForRemainingBranches(t *testing.T) {
	// Not parallel: stubs getColumnTypes / getIndexes / helperExec.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_stub (a integer, b text, a__obsolete integer)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}

	origCT := getColumnTypes
	t.Cleanup(func() { getColumnTypes = origCT })
	calls := 0
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		calls++
		if calls == 1 {
			return []gorm.ColumnType{fakeColumnType{name: "a", dbType: "INTEGER"}}, nil
		}
		return nil, fmt.Errorf("ct boom %d", calls)
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_stub", FromColumn: "a"}); err == nil || !strings.Contains(err.Error(), "ct boom 2") {
		t.Fatalf("Pre obsolete hasColumn: %v", err)
	}
	calls = 0
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		calls++
		if calls <= 2 {
			return []gorm.ColumnType{fakeColumnType{name: "a", dbType: "INTEGER"}}, nil
		}
		return nil, fmt.Errorf("ct boom %d", calls)
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_stub", FromColumn: "a", ToColumn: "b"}); err == nil || !strings.Contains(err.Error(), "ct boom 3") {
		t.Fatalf("Pre to hasColumn: %v", err)
	}

	calls = 0
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		calls++
		if calls == 1 {
			return []gorm.ColumnType{fakeColumnType{name: "a", dbType: "INTEGER"}}, nil
		}
		return nil, fmt.Errorf("ct boom %d", calls)
	}
	if err := ReplaceColumnCopyCast(opts, "replace_stub", "a", "b", "as_text"); err == nil || !strings.Contains(err.Error(), "ct boom 2") {
		t.Fatalf("CopyCast to hasColumn: %v", err)
	}
	getColumnTypes = origCT

	origIdx := getIndexes
	t.Cleanup(func() { getIndexes = origIdx })
	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return nil, fmt.Errorf("idx boom")
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_stub", FromColumn: "b"}); err == nil || !strings.Contains(err.Error(), "idx boom") {
		t.Fatalf("Pre dropIndexes: %v", err)
	}
	if err := dropIndexesReferencingColumn(opts, "replace_stub", "a"); err == nil || !strings.Contains(err.Error(), "idx boom") {
		t.Fatalf("dropIndexes list: %v", err)
	}

	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return []gorm.Index{
			nil,
			fakeIndex{name: "", cols: []string{"a"}},
			fakeIndex{name: "  ", cols: []string{"a"}},
			fakeIndex{name: "idx_other", cols: []string{"b"}},
			fakeIndex{name: "idx_a", cols: []string{"a"}},
		}, nil
	}
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(_ *gorm.DB, sql string) error {
		if strings.Contains(sql, "idx_a") {
			return fmt.Errorf("drop idx boom")
		}
		return nil
	}
	if err := dropIndexesReferencingColumn(opts, "replace_stub", "a"); err == nil || !strings.Contains(err.Error(), "drop idx boom") {
		t.Fatalf("dropIndexes exec: %v", err)
	}

	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return nil, fmt.Errorf("idx drop boom")
	}
	helperExec = origExec
	// DropObsolete finds a__obsolete via real columns, then fails listing indexes.
	getColumnTypes = origCT
	if err := ReplaceColumnDropObsolete(opts, "replace_stub", "a"); err == nil || !strings.Contains(err.Error(), "idx drop boom") {
		t.Fatalf("DropObsolete dropIndexes: %v", err)
	}
	getIndexes = origIdx
}
