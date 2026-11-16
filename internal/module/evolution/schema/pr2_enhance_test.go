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
			{Kind: OpAlterColumn, Safety: SafetyGuarded, Table: "demo_order", Detail: "type change varchar → int", Column: &ColumnSpec{Name: "qty"}},
			{Kind: OpAddColumn, Safety: SafetyGuarded, Table: "demo_order", Detail: "add NOT NULL column on non-empty table", Column: &ColumnSpec{Name: "x"}},
			{Kind: OpAlterColumn, Safety: SafetyAuto, Table: "demo_order", Detail: "type change int → varchar", Column: &ColumnSpec{Name: "skip"}},
			{Kind: OpAlterColumn, Safety: SafetyGuarded, Table: "demo_order", Detail: "type change int → varchar", Column: nil},
			{Kind: OpAlterColumn, Safety: SafetyGuarded, Table: "", Detail: "type change int → varchar", Column: &ColumnSpec{Name: "orphan"}},
		},
	}
	lines := FormatReplaceColumnSuggestions(plan)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "replaceColumnPre(\"demo_order\", \"status_code\")") {
		t.Fatalf("missing Pre: %s", joined)
	}
	if !strings.Contains(joined, "status_code__obsolete") || !strings.Contains(joined, "int_to_varchar") {
		t.Fatalf("missing CopyCast int_to_varchar: %s", joined)
	}
	if !strings.Contains(joined, "choose replaceColumnCopyCast kind") {
		t.Fatalf("missing author-choice hint for varchar→int: %s", joined)
	}
	if strings.Count(joined, "replaceColumnPre(\"demo_order\", \"status_code\")") != 1 {
		t.Fatalf("expected dedupe, got %s", joined)
	}
	if FormatReplaceColumnSuggestions(SchemaPlan{}) != nil {
		t.Fatal("empty plan")
	}
}

func TestSuggestCopyCastKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		detail string
		kind   string
		ok     bool
	}{
		{"", "", false},
		{"narrow size 255 → 50", "as_text", true},
		{"type change int → varchar", "int_to_varchar", true},
		{"type change integer -> text", "int_to_varchar", true},
		{"type change varchar → text", "as_text", true},
		{"type change varchar → int", "", false},
		{"type change text → integer", "", false},
	}
	for _, tc := range cases {
		kind, ok := suggestCopyCastKind(tc.detail)
		if ok != tc.ok || kind != tc.kind {
			t.Fatalf("%q: got (%q,%v) want (%q,%v)", tc.detail, kind, ok, tc.kind, tc.ok)
		}
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
	if err := db.Exec(`CREATE TABLE null_missing_col (id integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := SetNotNull(opts, "null_missing_col", "nope"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing col: %v", err)
	}
}

func TestApplySetNotNull_DialectsViaStub(t *testing.T) {
	// Not parallel: stubs helperExec / getColumnTypes.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	bag := NewMemoryIntentBag()
	origExec := helperExec
	origCT := getColumnTypes
	t.Cleanup(func() {
		helperExec = origExec
		getColumnTypes = origCT
	})
	helperExec = func(*gorm.DB, string) error { return nil }
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return []gorm.ColumnType{
			fakeColumnTypeWithDecimal{
				fakeColumnTypeWithDefault: fakeColumnTypeWithDefault{
					fakeColumnType: fakeColumnType{name: "amt", dbType: "decimal", nullable: true, nullableOK: true},
					def:            "0",
					defOK:          true,
				},
				precision: 10, scale: 2, ok: true,
			},
			fakeColumnType{name: "code", dbType: "varchar", length: 20, lengthOK: true, nullable: true, nullableOK: true},
			nil,
		}, nil
	}

	for _, dialect := range []string{"postgres", "postgresql", "mysql", "mariadb", "sqlserver", "oracle"} {
		opts := HelperOptions{DB: db, Dialect: dialect, Intents: bag}
		if err := applySetNotNull(opts, "t", "amt"); err != nil {
			t.Fatalf("%s: %v", dialect, err)
		}
	}
	helperExec = func(*gorm.DB, string) error { return fmt.Errorf("exec boom") }
	if err := applySetNotNull(HelperOptions{DB: db, Dialect: "postgres", Intents: bag}, "t", "amt"); err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("postgres exec err: %v", err)
	}
	if err := applySetNotNull(HelperOptions{DB: db, Dialect: "mysql", Intents: bag}, "t", "amt"); err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("mysql exec err: %v", err)
	}
	if err := applySetNotNull(HelperOptions{DB: db, Dialect: "sqlserver", Intents: bag}, "t", "amt"); err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("sqlserver exec err: %v", err)
	}
	if err := applySetNotNull(HelperOptions{DB: db, Dialect: "oracle", Intents: bag}, "t", "amt"); err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("default exec err: %v", err)
	}
}

func TestLiveColumnTypeName(t *testing.T) {
	// Not parallel: stubs getColumnTypes.
	origCT := getColumnTypes
	t.Cleanup(func() { getColumnTypes = origCT })
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return nil, fmt.Errorf("inspect boom")
	}
	if _, err := liveColumnTypeName(nil, "t", "c"); err == nil || !strings.Contains(err.Error(), "inspect boom") {
		t.Fatalf("inspect: %v", err)
	}
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return []gorm.ColumnType{
			nil,
			fakeColumnType{name: "other", dbType: "int"},
			fakeColumnType{name: "empty", dbType: ""},
			fakeColumnTypeWithDecimal{
				fakeColumnTypeWithDefault: fakeColumnTypeWithDefault{
					fakeColumnType: fakeColumnType{name: "amt", dbType: "DECIMAL"},
					def:            "1.5",
					defOK:          true,
				},
				precision: 12, scale: 4, ok: true,
			},
			fakeColumnType{name: "code", dbType: "varchar", length: 32, lengthOK: true},
			fakeColumnType{name: "raw", dbType: "int"},
		}, nil
	}
	if _, err := liveColumnTypeName(nil, "t", "empty"); err == nil || !strings.Contains(err.Error(), "empty type") {
		t.Fatalf("empty type: %v", err)
	}
	got, err := liveColumnTypeName(nil, "t", "amt")
	if err != nil || got != "DECIMAL(12,4) DEFAULT 1.5" {
		t.Fatalf("decimal+default: %q %v", got, err)
	}
	got, err = liveColumnTypeName(nil, "t", "code")
	if err != nil || got != "varchar(32)" {
		t.Fatalf("varchar: %q %v", got, err)
	}
	got, err = liveColumnTypeName(nil, "t", "raw")
	if err != nil || got != "int" {
		t.Fatalf("raw: %q %v", got, err)
	}
	if _, err := liveColumnTypeName(nil, "t", "missing"); err == nil {
		t.Fatal("missing")
	}
}

type fakeColumnTypeWithDecimal struct {
	fakeColumnTypeWithDefault
	precision, scale int64
	ok               bool
}

func (f fakeColumnTypeWithDecimal) DecimalSize() (int64, int64, bool) {
	return f.precision, f.scale, f.ok
}

func TestSqliteRewriteTable_RenameAndDrop(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_tbl (a integer, b text, c integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_rw_b ON rw_tbl (b)`).Error; err != nil {
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
	var idxCount int
	if err := db.Raw(`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='rw_tbl' AND sql LIKE '%b%'`).Scan(&idxCount).Error; err != nil || idxCount < 1 {
		t.Fatalf("index restore failed count=%d err=%v", idxCount, err)
	}
}

func TestSqliteRewriteTable_PreservesPKDefaultAndSkipsDroppedIndex(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_pk (id integer PRIMARY KEY, code text DEFAULT 'x', gone integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_rw_gone ON rw_pk (gone)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER trg_rw_gone AFTER UPDATE OF gone ON rw_pk BEGIN SELECT 1; END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO rw_pk (id, code, gone) VALUES (1, 'a', 2)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqliteRewriteTable(db, "rw_pk", sqliteTableRewrite{
		Drop: map[string]struct{}{"gone": {}},
	}); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("rw_pk", "gone") {
		t.Fatal("gone should be dropped")
	}
	var idxCount int
	if err := db.Raw(`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='rw_pk' AND name='idx_rw_gone'`).Scan(&idxCount).Error; err != nil {
		t.Fatal(err)
	}
	if idxCount != 0 {
		t.Fatalf("index on dropped column should not be restored: %d", idxCount)
	}
	var trigCount int
	if err := db.Raw(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name='rw_pk' AND name='trg_rw_gone'`).Scan(&trigCount).Error; err != nil {
		t.Fatal(err)
	}
	if trigCount != 0 {
		t.Fatalf("trigger on dropped column should not be restored: %d", trigCount)
	}
	var ddl string
	if err := db.Raw(`SELECT sql FROM sqlite_master WHERE type='table' AND name='rw_pk'`).Scan(&ddl).Error; err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(ddl)
	if !strings.Contains(upper, "PRIMARY KEY") {
		t.Fatalf("expected PRIMARY KEY in %s", ddl)
	}
	if !strings.Contains(upper, "DEFAULT") {
		t.Fatalf("expected DEFAULT in %s", ddl)
	}
}

func TestSqliteRewriteTable_ErrorPaths(t *testing.T) {
	t.Parallel()
	if err := sqliteRewriteTable(nil, "t", sqliteTableRewrite{}); err == nil {
		t.Fatal("nil db")
	}
	if err := sqliteRewriteTable(&gorm.DB{}, "", sqliteTableRewrite{}); err == nil {
		t.Fatal("empty table")
	}
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := sqliteRewriteTable(db, "missing_rw", sqliteTableRewrite{}); err == nil {
		t.Fatal("missing table")
	}
	if err := db.Exec(`CREATE TABLE rw_emptyish (only_drop integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqliteRewriteTable(db, "rw_emptyish", sqliteTableRewrite{
		Drop: map[string]struct{}{"only_drop": {}},
	}); err == nil || !strings.Contains(err.Error(), "no columns") {
		t.Fatalf("drop-all: %v", err)
	}
}

func TestSqliteRewriteHelpers(t *testing.T) {
	t.Parallel()
	if got := sqliteRewriteTempName("a b/c"); !strings.HasPrefix(got, "choysum_rw_") || len(got) > 40 {
		t.Fatalf("temp name: %q", got)
	}
	if got := sqliteRewriteTempName(strings.Repeat("t", 80)); len(got) > 40 {
		t.Fatalf("temp too long: %d %q", len(got), got)
	}
	if intentPhysicalName("") != "" || intentPhysicalName("StatusCode") != "status_code" {
		t.Fatal("intentPhysicalName")
	}
	if _, ok := sqliteRewriteObjectSQL("", sqliteTableRewrite{}); ok {
		t.Fatal("empty sql")
	}
	sql, ok := sqliteRewriteObjectSQL(`CREATE INDEX idx ON t (a, b)`, sqliteTableRewrite{
		Rename: map[string]string{"a": "a2"},
	})
	if !ok || !strings.Contains(sql, "a2") || strings.Contains(sql, "(a,") {
		t.Fatalf("rename ident: %q", sql)
	}
	if _, ok := sqliteRewriteObjectSQL(`CREATE INDEX idx ON t (gone)`, sqliteTableRewrite{
		Drop: map[string]struct{}{"gone": {}},
	}); ok {
		t.Fatal("drop should skip")
	}
	if !sqliteSQLMentionsColumn(`create index idx on t ("gone")`, "gone") {
		t.Fatal("quoted mention")
	}
	if !sqliteSQLMentionsColumn(`create index idx on t ([gone])`, "gone") {
		t.Fatal("bracket mention")
	}
	if sqliteSQLMentionsColumn(`create index idx on t (gone_away)`, "gone") {
		t.Fatal("substring should not match")
	}
	if sqliteSQLMentionsColumn("xgone", "gone") {
		t.Fatal("leading ident byte should not match")
	}
	if sqliteSQLMentionsColumn("abc", "") {
		t.Fatal("empty col")
	}
	if !sqliteSQLMentionsColumn("`gone`", "gone") {
		t.Fatal("backtick mention")
	}
	if sqliteReplaceIdent("x", "", "y") != "x" || sqliteReplaceIdent("x", "a", "a") != "x" {
		t.Fatal("replace ident no-op")
	}
	if got := sqliteReplaceIdent(`CREATE INDEX idx ON t (prefix_a)`, "a", "b"); !strings.Contains(got, "prefix_a") {
		t.Fatalf("should not replace substring: %q", got)
	}
	if got := sqliteBareTypeName("INTEGER PRIMARY KEY AUTOINCREMENT"); got != "INTEGER" {
		t.Fatalf("bare type: %q", got)
	}
	if got := sqliteBareTypeName("TEXT NOT NULL DEFAULT 'x'"); got != "TEXT" {
		t.Fatalf("bare text: %q", got)
	}
	if got := sqliteBareTypeName(""); got != "TEXT" {
		t.Fatalf("empty bare: %q", got)
	}
}

func TestSqliteRewriteTable_ExecFailurePaths(t *testing.T) {
	// Not parallel: stubs helperExec.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_fail (a integer, b text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO rw_fail (a, b) VALUES (1, 'x')`).Error; err != nil {
		t.Fatal(err)
	}
	_ = db.Exec(`PRAGMA foreign_keys=ON`)
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })

	helperExec = func(db *gorm.DB, sql string) error {
		if strings.Contains(strings.ToUpper(sql), "INSERT INTO") {
			return fmt.Errorf("forced copy fail")
		}
		return origExec(db, sql)
	}
	if err := sqliteRewriteTable(db, "rw_fail", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"b": {}}}); err == nil || !strings.Contains(err.Error(), "copy") {
		t.Fatalf("copy fail: %v", err)
	}

	helperExec = func(db *gorm.DB, sql string) error {
		u := strings.ToUpper(sql)
		if strings.HasPrefix(strings.TrimSpace(u), "DROP TABLE") && !strings.Contains(u, "IF EXISTS") && strings.Contains(u, "RW_FAIL") {
			return fmt.Errorf("forced drop original fail")
		}
		return origExec(db, sql)
	}
	if err := sqliteRewriteTable(db, "rw_fail", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"b": {}}}); err == nil || !strings.Contains(err.Error(), "drop original") {
		t.Fatalf("drop original: %v", err)
	}

	helperExec = func(db *gorm.DB, sql string) error {
		u := strings.ToUpper(sql)
		if strings.Contains(u, "RENAME TO") {
			return fmt.Errorf("forced rename fail")
		}
		return origExec(db, sql)
	}
	if err := sqliteRewriteTable(db, "rw_fail", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"b": {}}}); err == nil || !strings.Contains(err.Error(), "rename rewrite") {
		t.Fatalf("rename fail: %v", err)
	}
	// Original table should still exist after rolled-back transaction (or cleaned temp).
	if !db.Migrator().HasTable("rw_fail") {
		t.Fatal("rw_fail should remain after failed rename")
	}

	helperExec = func(db *gorm.DB, sql string) error {
		if strings.EqualFold(strings.TrimSpace(sql), "PRAGMA foreign_keys=OFF") {
			return fmt.Errorf("forced fk off fail")
		}
		return origExec(db, sql)
	}
	_ = db.Exec(`PRAGMA foreign_keys=ON`)
	if err := sqliteRewriteTable(db, "rw_fail", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"b": {}}}); err == nil || !strings.Contains(err.Error(), "foreign_keys") {
		t.Fatalf("fk off: %v", err)
	}
	helperExec = origExec
}

func TestSqliteRewriteTable_RestoreTriggerAndIndexFail(t *testing.T) {
	// Not parallel: stubs helperExec for index restore failure.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_trig (id integer PRIMARY KEY, code text NOT NULL DEFAULT 'x')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_rw_trig_code ON rw_trig (code)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER trg_rw_trig AFTER INSERT ON rw_trig BEGIN SELECT 1; END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO rw_trig (id, code) VALUES (1, 'a')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqliteRewriteTable(db, "rw_trig", sqliteTableRewrite{
		ForceNotNull: map[string]struct{}{"code": {}},
	}); err != nil {
		t.Fatalf("rewrite with trigger: %v", err)
	}
	var trig int
	if err := db.Raw(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name='rw_trig'`).Scan(&trig).Error; err != nil || trig < 1 {
		t.Fatalf("trigger restore trig=%d err=%v", trig, err)
	}

	if err := db.Exec(`CREATE TABLE rw_idx_fail (a integer, b text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_rw_idx_fail_b ON rw_idx_fail (b)`).Error; err != nil {
		t.Fatal(err)
	}
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(db *gorm.DB, sql string) error {
		if strings.Contains(strings.ToUpper(sql), "CREATE INDEX") {
			return fmt.Errorf("forced index restore fail")
		}
		return origExec(db, sql)
	}
	if err := sqliteRewriteTable(db, "rw_idx_fail", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"b": {}}}); err == nil || !strings.Contains(err.Error(), "restore index") {
		t.Fatalf("index restore: %v", err)
	}
}

func TestSqliteForeignKeyCheckAndMasterErrors(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := sqliteForeignKeyCheck(db); err != nil {
		t.Fatalf("empty check: %v", err)
	}
	on, err := sqliteForeignKeysEnabled(db)
	if err != nil {
		t.Fatal(err)
	}
	_ = on
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	closed, err := gorm.Open(db.Dialector, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := closed.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	if _, err := sqliteMasterSQLs(closed, "index", "t"); err == nil {
		t.Fatal("master on closed")
	}
	if _, err := sqliteForeignKeysEnabled(closed); err == nil {
		t.Fatal("fk pragma on closed")
	}
	if err := sqliteForeignKeyCheck(closed); err == nil {
		t.Fatal("fk check on closed")
	}
	_ = sqlDB
}

func TestSetNotNull_CountAndApplyErrors(t *testing.T) {
	// Not parallel: stubs getColumnTypes / helperExec.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE null_err (code text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO null_err (code) VALUES ('a')`).Error; err != nil {
		t.Fatal(err)
	}
	origExec := helperExec
	origCT := getColumnTypes
	t.Cleanup(func() {
		helperExec = origExec
		getColumnTypes = origCT
	})
	helperExec = func(*gorm.DB, string) error { return fmt.Errorf("apply boom") }
	err := SetNotNull(HelperOptions{DB: db, Dialect: "postgres", Intents: NewMemoryIntentBag()}, "null_err", "code")
	if err == nil || !strings.Contains(err.Error(), "apply boom") {
		t.Fatalf("apply: %v", err)
	}
	helperExec = origExec

	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return []gorm.ColumnType{fakeColumnType{name: "code", dbType: "text"}}, nil
	}
	err = SetNotNull(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, "no_such_null_table", "code")
	if err == nil || !strings.Contains(err.Error(), "count nulls") {
		t.Fatalf("count: %v", err)
	}
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return nil, fmt.Errorf("has boom")
	}
	if err := SetNotNull(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, "null_err", "code"); err == nil || !strings.Contains(err.Error(), "has boom") {
		t.Fatalf("has: %v", err)
	}
}

func TestApplySetNotNull_TypeInspectErrors(t *testing.T) {
	// Not parallel: stubs getColumnTypes.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	origCT := getColumnTypes
	t.Cleanup(func() { getColumnTypes = origCT })
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return nil, fmt.Errorf("type boom")
	}
	if err := applySetNotNull(HelperOptions{DB: db, Dialect: "mysql", Intents: NewMemoryIntentBag()}, "t", "c"); err == nil || !strings.Contains(err.Error(), "type boom") {
		t.Fatalf("mysql inspect: %v", err)
	}
	if err := applySetNotNull(HelperOptions{DB: db, Dialect: "sqlserver", Intents: NewMemoryIntentBag()}, "t", "c"); err == nil || !strings.Contains(err.Error(), "type boom") {
		t.Fatalf("sqlserver inspect: %v", err)
	}
}

func TestSqliteForeignKeyCheck_Violations(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE parent_fk (id integer PRIMARY KEY)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE child_fk (id integer, parent_id integer REFERENCES parent_fk(id))`).Error; err != nil {
		t.Fatal(err)
	}
	_ = db.Exec(`PRAGMA foreign_keys=OFF`)
	if err := db.Exec(`INSERT INTO child_fk (id, parent_id) VALUES (1, 999)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqliteForeignKeyCheck(db); err == nil {
		t.Fatal("expected fk violations")
	}
}

func TestSqliteRewriteTable_NilColumnsAndAI(t *testing.T) {
	// Not parallel: stubs getColumnTypes.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	origCT := getColumnTypes
	t.Cleanup(func() { getColumnTypes = origCT })
	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return []gorm.ColumnType{
			nil,
			fakeColumnType{name: "", dbType: "int"},
			fakeColumnTypeWithDefault{
				fakeColumnType: fakeColumnType{
					name: "label", dbType: "", nullable: true, nullableOK: true,
				},
				def: "x", defOK: true,
			},
			fakeColumnTypeAI{
				fakeColumnType: fakeColumnType{
					name: "n", dbType: "INTEGER", primaryKey: true, primaryKeySet: true, primaryKeyOK: true,
				},
				ai: true, aiOK: true,
			},
		}, nil
	}
	if err := db.Exec(`CREATE TABLE rw_ai (id integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqliteRewriteTable(db, "rw_ai", sqliteTableRewrite{}); err != nil {
		t.Fatalf("nil/empty/ai rewrite: %v", err)
	}
}

type fakeColumnTypeAI struct {
	fakeColumnType
	ai, aiOK bool
}

func (f fakeColumnTypeAI) AutoIncrement() (bool, bool) { return f.ai, f.aiOK }

func TestSqliteRewriteTable_SeamErrorsAndFKCheckFail(t *testing.T) {
	// Not parallel: stubs seams / getColumnTypes.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_seam (a integer)`).Error; err != nil {
		t.Fatal(err)
	}
	origList := sqliteListMasterSQLs
	origFK := sqliteReadForeignKeys
	origCheck := sqliteCheckForeignKeys
	origCT := getColumnTypes
	t.Cleanup(func() {
		sqliteListMasterSQLs = origList
		sqliteReadForeignKeys = origFK
		sqliteCheckForeignKeys = origCheck
		getColumnTypes = origCT
	})

	getColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return nil, nil
	}
	if err := sqliteRewriteTable(db, "rw_seam", sqliteTableRewrite{}); err == nil || !strings.Contains(err.Error(), "no columns") {
		t.Fatalf("empty types: %v", err)
	}
	getColumnTypes = origCT

	sqliteListMasterSQLs = func(*gorm.DB, string, string) ([]string, error) {
		return nil, fmt.Errorf("master boom")
	}
	if err := sqliteRewriteTable(db, "rw_seam", sqliteTableRewrite{}); err == nil || !strings.Contains(err.Error(), "master boom") {
		t.Fatalf("index master: %v", err)
	}
	calls := 0
	sqliteListMasterSQLs = func(*gorm.DB, string, string) ([]string, error) {
		calls++
		if calls == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("trigger master boom")
	}
	if err := sqliteRewriteTable(db, "rw_seam", sqliteTableRewrite{}); err == nil || !strings.Contains(err.Error(), "trigger master boom") {
		t.Fatalf("trigger master: %v", err)
	}
	sqliteListMasterSQLs = origList

	sqliteReadForeignKeys = func(*gorm.DB) (bool, error) {
		return false, fmt.Errorf("fk read boom")
	}
	if err := sqliteRewriteTable(db, "rw_seam", sqliteTableRewrite{}); err == nil || !strings.Contains(err.Error(), "fk read boom") {
		t.Fatalf("fk read: %v", err)
	}
	sqliteReadForeignKeys = origFK

	if err := db.Exec(`CREATE TABLE parent_fk2 (id integer PRIMARY KEY)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE child_fk2 (id integer, parent_id integer REFERENCES parent_fk2(id))`).Error; err != nil {
		t.Fatal(err)
	}
	_ = db.Exec(`PRAGMA foreign_keys=OFF`)
	if err := db.Exec(`INSERT INTO child_fk2 (id, parent_id) VALUES (1, 999)`).Error; err != nil {
		t.Fatal(err)
	}
	sqliteCheckForeignKeys = sqliteForeignKeyCheck
	if err := sqliteRewriteTable(db, "rw_seam", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"a": {}}}); err == nil || !strings.Contains(err.Error(), "foreign_key_check") {
		t.Fatalf("fk check during rewrite: %v", err)
	}
	_ = db.Exec(`DELETE FROM child_fk2`)
}

func TestSqliteRewriteTable_TriggerRestoreFail(t *testing.T) {
	// Not parallel: stubs helperExec.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE rw_trig_fail (a integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER trg_rw_trig_fail AFTER INSERT ON rw_trig_fail BEGIN SELECT 1; END`).Error; err != nil {
		t.Fatal(err)
	}
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(db *gorm.DB, sql string) error {
		if strings.Contains(strings.ToUpper(sql), "CREATE TRIGGER") {
			return fmt.Errorf("forced trigger restore fail")
		}
		return origExec(db, sql)
	}
	if err := sqliteRewriteTable(db, "rw_trig_fail", sqliteTableRewrite{ForceNotNull: map[string]struct{}{"a": {}}}); err == nil || !strings.Contains(err.Error(), "restore trigger") {
		t.Fatalf("trigger restore: %v", err)
	}
}

func TestReplaceColumnPre_SqliteRewriteFallback(t *testing.T) {
	// Not parallel: stubs renameColumnHelper.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE pre_fb (code integer, keep text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO pre_fb (code, keep) VALUES (7, 'k')`).Error; err != nil {
		t.Fatal(err)
	}
	bag := NewMemoryIntentBag()
	opts := HelperOptions{DB: db, Dialect: "sqlite", Intents: bag}
	origRename := renameColumnHelper
	t.Cleanup(func() { renameColumnHelper = origRename })
	renameColumnHelper = func(HelperOptions, string, string, string) error {
		return fmt.Errorf("forced rename fail")
	}
	if err := ReplaceColumnPre(opts, ReplaceColumnOptions{Table: "pre_fb", FromColumn: "code"}); err != nil {
		t.Fatalf("Pre rewrite fallback: %v", err)
	}
	if db.Migrator().HasColumn("pre_fb", "code") || !db.Migrator().HasColumn("pre_fb", "code__obsolete") {
		t.Fatal("rename via rewrite failed")
	}
	list := bag.List()
	if len(list) != 1 || list[0].Kind != IntentRenameColumn || list[0].Name != "code__obsolete" {
		t.Fatalf("intent=%#v", list)
	}

	// Rewrite failure path.
	if err := db.Exec(`CREATE TABLE pre_fb2 (code integer)`).Error; err != nil {
		t.Fatal(err)
	}
	origExec := helperExec
	t.Cleanup(func() { helperExec = origExec })
	helperExec = func(db *gorm.DB, sql string) error {
		if strings.Contains(strings.ToUpper(sql), "CREATE TABLE") && strings.Contains(strings.ToLower(sql), "choysum_rw_") {
			return fmt.Errorf("forced rewrite create fail")
		}
		return origExec(db, sql)
	}
	err := ReplaceColumnPre(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, ReplaceColumnOptions{Table: "pre_fb2", FromColumn: "code"})
	if err == nil || !strings.Contains(err.Error(), "sqlite rewrite") {
		t.Fatalf("expected rewrite wrap err, got %v", err)
	}

	// Non-sqlite rename failure returns original error.
	renameColumnHelper = func(HelperOptions, string, string, string) error {
		return fmt.Errorf("pg rename boom")
	}
	err = ReplaceColumnPre(HelperOptions{DB: db, Dialect: "postgres", Intents: NewMemoryIntentBag()}, ReplaceColumnOptions{Table: "pre_fb2", FromColumn: "code"})
	if err == nil || !strings.Contains(err.Error(), "pg rename boom") {
		t.Fatalf("non-sqlite: %v", err)
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
	list := bag.List()
	if len(list) != 1 || list[0].Kind != IntentDropColumn || list[0].Name != "code__obsolete" {
		t.Fatalf("intent=%#v", list)
	}

	if err := db.Exec(`CREATE TABLE drop_fb2 (code integer, code__obsolete integer)`).Error; err != nil {
		t.Fatal(err)
	}
	helperExec = func(db *gorm.DB, sql string) error {
		u := strings.ToUpper(sql)
		if strings.Contains(u, "DROP COLUMN") {
			return fmt.Errorf("forced drop column fail")
		}
		if strings.Contains(u, "CREATE TABLE") && strings.Contains(strings.ToLower(sql), "choysum_rw_") {
			return fmt.Errorf("forced rewrite fail")
		}
		return origExec(db, sql)
	}
	err := ReplaceColumnDropObsolete(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, "drop_fb2", "code")
	if err == nil || !strings.Contains(err.Error(), "sqlite rewrite") {
		t.Fatalf("rewrite fail wrap: %v", err)
	}

	helperExec = func(db *gorm.DB, sql string) error {
		if strings.Contains(strings.ToUpper(sql), "DROP COLUMN") {
			return fmt.Errorf("pg drop boom")
		}
		return origExec(db, sql)
	}
	err = ReplaceColumnDropObsolete(HelperOptions{DB: db, Dialect: "postgres", Intents: NewMemoryIntentBag()}, "drop_fb2", "code")
	if err == nil || !strings.Contains(err.Error(), "pg drop boom") {
		t.Fatalf("non-sqlite drop: %v", err)
	}
}

func TestReplaceColumnDropObsolete_RejectsPrimaryKey(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE drop_pk (code__obsolete integer PRIMARY KEY, code integer)`).Error; err != nil {
		t.Fatal(err)
	}
	err := ReplaceColumnDropObsolete(HelperOptions{DB: db, Dialect: "sqlite", Intents: NewMemoryIntentBag()}, "drop_pk", "code")
	if err == nil || !strings.Contains(err.Error(), "primary key") {
		t.Fatalf("got %v", err)
	}
	if !db.Migrator().HasColumn("drop_pk", "code__obsolete") {
		t.Fatal("PK obsolete must remain")
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
