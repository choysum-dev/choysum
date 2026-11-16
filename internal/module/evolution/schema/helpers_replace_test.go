// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
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

func TestReplaceColumnPre_ObsoleteConflict(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE replace_conflict (code integer, code__obsolete integer)`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "replace_conflict", FromColumn: "code"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("got %v", err)
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
