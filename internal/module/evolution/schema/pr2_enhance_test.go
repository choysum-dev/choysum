// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestFormatReplaceColumnSuggestions(t *testing.T) {
	t.Parallel()
	plan := SchemaPlan{
		Ops: []PlanOp{
			{Kind: OpAlterColumn, Safety: SafetyGuarded, Table: "demo_order", Detail: "type change int → varchar", Column: &ColumnSpec{Name: "status_code"}},
			{Kind: OpAlterColumn, Safety: SafetyGuarded, Table: "demo_order", Detail: "narrow size 255 → 50", Column: &ColumnSpec{Name: "status_code"}},
			{Kind: OpAddColumn, Safety: SafetyGuarded, Table: "demo_order", Detail: "add NOT NULL column on non-empty table", Column: &ColumnSpec{Name: "x"}},
			{Kind: OpAlterColumn, Safety: SafetyAuto, Table: "demo_order", Detail: "type change int → varchar", Column: &ColumnSpec{Name: "skip"}},
		},
	}
	lines := FormatReplaceColumnSuggestions(plan)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "replaceColumnPre(\"demo_order\", \"status_code\")") {
		t.Fatalf("missing Pre: %s", joined)
	}
	if !strings.Contains(joined, "status_code__obsolete") || !strings.Contains(joined, "int_to_varchar") {
		t.Fatalf("missing CopyCast: %s", joined)
	}
	if strings.Count(joined, "replaceColumnPre") != 1 {
		t.Fatalf("expected dedupe, got %s", joined)
	}
	if FormatReplaceColumnSuggestions(SchemaPlan{}) != nil {
		t.Fatal("empty plan")
	}
}

func TestSetNotNull_SQLiteAndIntent(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE null_tighten (id integer, code text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO null_tighten (id, code) VALUES (1, 'a'), (2, 'b')`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	if err := SetNotNull(opts, "null_tighten", "code"); err != nil {
		t.Fatalf("SetNotNull: %v", err)
	}
	list := bag.List()
	if len(list) != 1 || list[0].Kind != IntentTightenNull || list[0].Name != "code" {
		t.Fatalf("intent = %#v", list)
	}
	plan := SchemaPlan{Ops: []PlanOp{{
		Kind: OpAlterColumn, Safety: SafetyGuarded, Table: "null_tighten",
		Detail: "tighten nullability to NOT NULL", Column: &ColumnSpec{Name: "code"},
	}}}
	if err := ValidatePlan(plan, bag); err != nil {
		t.Fatalf("ValidatePlan should accept IntentTightenNull: %v", err)
	}
	if IntentSatisfies(plan.Ops[0], bag) != true {
		t.Fatal("IntentSatisfies")
	}
}

func TestSetNotNull_RejectsNullRows(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE null_reject (code text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO null_reject (code) VALUES (NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	err := SetNotNull(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, "null_reject", "code")
	if err == nil || !strings.Contains(err.Error(), "NULL row") {
		t.Fatalf("got %v", err)
	}
}

func TestSetNotNull_ErrorPaths(t *testing.T) {
	t.Parallel()
	if err := SetNotNull(HelperOptions{}, "t", "c"); err == nil {
		t.Fatal("validate")
	}
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}
	if err := SetNotNull(opts, "", "c"); err == nil {
		t.Fatal("empty")
	}
	if err := SetNotNull(opts, "missing", "c"); err == nil {
		t.Fatal("missing table/col")
	}
}

func TestSqliteRewriteTable_RenameAndDrop(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_tbl (a integer, b text, c integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO rw_tbl (a, b, c) VALUES (1, 'x', 9)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqliteRewriteTable(db, "rw_tbl", sqliteTableRewrite{
		Rename: map[string]string{"a": "a__obsolete"},
		Drop:   map[string]struct{}{"c": {}},
	}); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("rw_tbl", "a") || !db.Migrator().HasColumn("rw_tbl", "a__obsolete") {
		t.Fatal("rename failed")
	}
	if db.Migrator().HasColumn("rw_tbl", "c") {
		t.Fatal("drop failed")
	}
	var got int
	if err := db.Raw(`SELECT a__obsolete FROM rw_tbl`).Scan(&got).Error; err != nil || got != 1 {
		t.Fatalf("data=%d err=%v", got, err)
	}
}

func TestReplaceColumnDropObsolete_SqliteRewriteFallback(t *testing.T) {
	// Not parallel: stubs helperExec.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE drop_fb (code integer, code__obsolete integer, keep text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO drop_fb (code, code__obsolete, keep) VALUES (1, 2, 'k')`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(db *gorm.DB, sql string) error {
		if strings.Contains(strings.ToUpper(sql), "DROP COLUMN") {
			return fmt.Errorf("forced drop column fail")
		}
		return origExec(db, sql)
	}
	if err := ReplaceColumnDropObsolete(opts, "drop_fb", "code"); err != nil {
		t.Fatalf("DropObsolete rewrite fallback: %v", err)
	}
	if db.Migrator().HasColumn("drop_fb", "code__obsolete") {
		t.Fatal("obsolete should be gone after rewrite")
	}
	if !db.Migrator().HasColumn("drop_fb", "keep") {
		t.Fatal("keep column must remain")
	}
}

func TestIntentKindForOp_TightenNull(t *testing.T) {
	t.Parallel()
	got := intentKindForOp(PlanOp{Kind: OpAlterColumn, Detail: "tighten nullability to NOT NULL"})
	if got != IntentTightenNull {
		t.Fatalf("got %q", got)
	}
	if intentKindForOp(PlanOp{Kind: OpAlterColumn, Detail: "drop column x"}) != "" {
		t.Fatal("alter_column must not match drop via Detail")
	}
	if intentKindForOp(PlanOp{Kind: OpAlterColumn, Detail: "type change int → varchar"}) != "" {
		t.Fatal("type change must stay unsatisfiable via Intent")
	}
}
