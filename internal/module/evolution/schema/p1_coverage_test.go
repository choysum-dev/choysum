// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestDefaultChangedBranches(t *testing.T) {
	t.Parallel()
	liveDef := "hello"
	if defaultChanged(ColumnSpec{}, LiveColumn{}) {
		t.Fatal("both nil")
	}
	if !defaultChanged(ColumnSpec{}, LiveColumn{Default: &liveDef}) {
		t.Fatal("desired nil live set")
	}
	nullSentinel := "null"
	if defaultChanged(ColumnSpec{}, LiveColumn{Default: &nullSentinel}) {
		t.Fatal("null sentinel should be absent")
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
	pgLive := "'hello'::character varying"
	if defaultChanged(ColumnSpec{Default: &want}, LiveColumn{Default: &pgLive}) {
		t.Fatal("postgres cast should normalize equal")
	}
	parenLive := "('hello')"
	if defaultChanged(ColumnSpec{Default: &want}, LiveColumn{Default: &parenLive}) {
		t.Fatal("paren-wrapped default should normalize equal")
	}
	literalWithParens := "'foo()'"
	plainFoo := "foo"
	if !defaultChanged(ColumnSpec{Default: &plainFoo}, LiveColumn{Default: &literalWithParens}) {
		t.Fatal("'foo()' must not normalize equal to foo")
	}
}

func TestLiveHasIndexBranches(t *testing.T) {
	t.Parallel()
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
	composite := LiveSchema{Indexes: map[string][]LiveIndex{
		"t": {{Name: "idx_composite", Columns: []string{"tenant_id", "code"}, Unique: true}},
	}}
	if liveHasIndex(composite, "t", "code", true) {
		t.Fatal("composite unique index must not satisfy single-column unique lookup")
	}
	if !indexCoveredByDesiredKey(LiveIndex{Columns: []string{"Code"}}, map[string]struct{}{"code": {}}) {
		t.Fatal("covered")
	}
	if indexCoveredByDesiredKey(LiveIndex{Columns: []string{"other"}}, map[string]struct{}{"code": {}}) {
		t.Fatal("not covered")
	}
}

func TestApplyPlan_AlterWidenAndIndexCheckErrors(t *testing.T) {
	t.Parallel()
	runtimeScope := newSchemaTestScope(t)
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpCreateTable, Safety: SafetyAuto, Table: "widen_tbl",
		Columns: []ColumnSpec{{Name: "code", FieldName: "Code", PhysicalType: "varchar", Size: intPtrValue(8)}},
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(runtimeScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpAlterColumn, Safety: SafetyAuto, Table: "widen_tbl", Detail: "widen size 8 → 32",
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
		Kind: OpAlterColumn, Safety: SafetyAuto, Table: "widen_tbl", Detail: "widen size 8 → 32",
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

func TestDefaultGetIndexes_NonSQLiteAndSQLiteBranches(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE idx_cov (code text UNIQUE, name text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_cov_name ON idx_cov (name)`).Error; err != nil {
		t.Fatal(err)
	}

	// Non-sqlite / non-postgres dialector name uses Migrator().GetIndexes while
	// keeping a real sqlite migrator (postgres has its own catalog path).
	origDialector := db.Dialector
	db.Dialector = dialectorWithName{Dialector: origDialector, name: "mysql"}
	indexes, err := defaultGetIndexes(db, "idx_cov")
	db.Dialector = origDialector
	if err != nil {
		t.Fatalf("non-sqlite GetIndexes: %v", err)
	}
	if len(indexes) == 0 {
		t.Fatal("expected indexes from migrator path")
	}

	// Expression indexes yield NULL index_info names; NULL-safe sqlite path must skip them.
	if err := db.Exec(`CREATE INDEX idx_cov_expr ON idx_cov ((code || ''))`).Error; err != nil {
		t.Fatal(err)
	}

	// Real sqlite path: UNIQUE constraint origin "u" is skipped; expression NULL cols are skipped.
	sqliteIndexes, err := sqliteGetIndexes(db, "idx_cov")
	if err != nil {
		t.Fatalf("sqliteGetIndexes: %v", err)
	}
	for _, idx := range sqliteIndexes {
		if strings.TrimSpace(idx.Name()) == "" {
			t.Fatal("unexpected empty index name")
		}
	}

	origList := sqliteIndexListScan
	origInfo := sqliteIndexInfoScan
	t.Cleanup(func() {
		sqliteIndexListScan = origList
		sqliteIndexInfoScan = origInfo
	})

	sqliteIndexListScan = func(*gorm.DB, string, any) error {
		return gorm.ErrInvalidDB
	}
	if _, err := sqliteGetIndexes(db, "idx_cov"); err == nil {
		t.Fatal("expected index_list scan error")
	}

	sqliteIndexListScan = func(_ *gorm.DB, _ string, dest any) error {
		rows := dest.(*[]sqliteIndexListRow)
		*rows = []sqliteIndexListRow{
			{Name: sql.NullString{Valid: false}, Origin: "c"},
			{Name: sql.NullString{String: "  ", Valid: true}, Origin: "c"},
			{Name: sql.NullString{String: "uniq_from_constraint", Valid: true}, Origin: "u", Unique: true},
			{Name: sql.NullString{String: "idx_ok", Valid: true}, Origin: "c"},
			{Name: sql.NullString{String: "idx_pk", Valid: true}, Origin: "pk"},
		}
		return nil
	}
	sqliteIndexInfoScan = func(_ *gorm.DB, name string, dest any) error {
		if name == "idx_ok" {
			return gorm.ErrInvalidDB
		}
		cols := dest.(*[]sql.NullString)
		*cols = []sql.NullString{
			{Valid: false},
			{String: "  ", Valid: true},
			{String: "code", Valid: true},
		}
		return nil
	}
	if _, err := sqliteGetIndexes(db, "idx_cov"); err == nil {
		t.Fatal("expected index_info scan error for idx_ok")
	}

	sqliteIndexInfoScan = func(_ *gorm.DB, name string, dest any) error {
		cols := dest.(*[]sql.NullString)
		*cols = []sql.NullString{
			{Valid: false},
			{String: "  ", Valid: true},
			{String: "code", Valid: true},
		}
		return nil
	}
	out, err := sqliteGetIndexes(db, "idx_cov")
	if err != nil {
		t.Fatal(err)
	}
	// Invalid/empty names skipped; origin "u" retained for uniqueness visibility.
	if len(out) != 3 {
		t.Fatalf("expected uniq_from_constraint + idx_ok + idx_pk, got %#v", out)
	}

	// origin "u"/"pk" must mark Unique even when the pragma unique bool scans as false
	// (zero value), otherwise liveHasIndex misses sqlite_autoindex_* constraints.
	sqliteIndexListScan = func(_ *gorm.DB, _ string, dest any) error {
		rows := dest.(*[]sqliteIndexListRow)
		*rows = []sqliteIndexListRow{
			{Name: sql.NullString{String: "sqlite_autoindex_t_2", Valid: true}, Origin: "u", Unique: false},
			{Name: sql.NullString{String: "sqlite_autoindex_t_1", Valid: true}, Origin: "pk", Unique: false},
			{Name: sql.NullString{String: "idx_plain", Valid: true}, Origin: "c", Unique: false},
		}
		return nil
	}
	sqliteIndexInfoScan = func(_ *gorm.DB, name string, dest any) error {
		cols := dest.(*[]sql.NullString)
		switch name {
		case "sqlite_autoindex_t_2":
			*cols = []sql.NullString{{String: "code", Valid: true}}
		case "sqlite_autoindex_t_1":
			*cols = []sql.NullString{{String: "id", Valid: true}}
		default:
			*cols = []sql.NullString{{String: "note", Valid: true}}
		}
		return nil
	}
	out, err = sqliteGetIndexes(db, "idx_cov")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("origin-unique fallback indexes: %#v", out)
	}
	for _, idx := range out {
		u, ok := idx.Unique()
		switch idx.Name() {
		case "sqlite_autoindex_t_2", "sqlite_autoindex_t_1":
			if !ok || !u {
				t.Fatalf("%s must be unique via origin, got unique=%v ok=%v", idx.Name(), u, ok)
			}
		case "idx_plain":
			if !ok || u {
				t.Fatalf("idx_plain must stay non-unique, got unique=%v ok=%v", u, ok)
			}
		}
	}
}

func TestPostgresGetIndexes_IncludesUniqueConstraints(t *testing.T) {
	t.Parallel()
	origScan := postgresIndexScan
	t.Cleanup(func() { postgresIndexScan = origScan })

	// Execute the default scanner body (SQL string + Raw) for coverage. The
	// Postgres-only catalog SQL fails on the sqlite test DB after the query runs.
	if err := origScan(newSchemaTestScope(t).Session().DB, "auth_user", &[]postgresIndexRow{}); err == nil {
		t.Fatal("expected default postgresIndexScan to fail on sqlite test DB")
	}
	// Source contract: key-only ordinals + skip expression key columns (attnum 0).
	src, err := os.ReadFile("inspect.go")
	if err != nil {
		t.Fatal(err)
	}
	sqlBody := string(src)
	if !strings.Contains(sqlBody, "k.ord <= i.indnkeyatts") {
		t.Fatal("expected indnkeyatts key-column filter in postgresIndexScan SQL")
	}
	if !strings.Contains(sqlBody, "ke.attnum = 0") {
		t.Fatal("expected expression-index exclusion (attnum 0) in postgresIndexScan SQL")
	}
	if !strings.Contains(sqlBody, "i.indisvalid") {
		t.Fatal("expected indisvalid filter in postgresIndexScan SQL")
	}

	if _, err := postgresGetIndexes(nil, "auth_user"); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil db: %v", err)
	}
	out, err := postgresGetIndexes(&gorm.DB{}, "  ")
	if err != nil || len(out) != 0 {
		t.Fatalf("empty table: %#v %v", out, err)
	}

	postgresIndexScan = func(*gorm.DB, string, any) error {
		return gorm.ErrInvalidDB
	}
	if _, err := postgresGetIndexes(&gorm.DB{}, "auth_user"); err == nil {
		t.Fatal("expected scan error")
	}

	postgresIndexScan = func(_ *gorm.DB, _ string, dest any) error {
		rows := dest.(*[]postgresIndexRow)
		*rows = []postgresIndexRow{
			{IndexName: "", ColumnName: "skip"},
			{IndexName: "uni_auth_user_username", IsUnique: true, ColumnName: "username", ColumnOrd: 1},
			{IndexName: "uni_auth_user_email", IsUnique: true, ColumnName: "email", ColumnOrd: 1},
			{IndexName: "auth_user_pkey", IsUnique: true, IsPrimary: true, ColumnName: "id", ColumnOrd: 1},
			{IndexName: "idx_auth_user_company_id", IsUnique: false, ColumnName: "company_id", ColumnOrd: 1},
			// Composite unique keeps column order from ColumnOrd scan order.
			{IndexName: "uq_pair", IsUnique: true, ColumnName: "a", ColumnOrd: 1},
			{IndexName: "uq_pair", IsUnique: true, ColumnName: "b", ColumnOrd: 2},
			// Later rows OR unique/primary onto an existing aggregate; blank column skipped.
			{IndexName: "uq_pair", IsUnique: false, IsPrimary: true, ColumnName: "  ", ColumnOrd: 3},
		}
		return nil
	}
	out, err = postgresGetIndexes(&gorm.DB{}, "auth_user")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]gorm.Index{}
	for _, idx := range out {
		byName[idx.Name()] = idx
	}
	if len(byName) != 5 {
		t.Fatalf("indexes = %#v", out)
	}
	u, ok := byName["uni_auth_user_username"].Unique()
	if !ok || !u {
		t.Fatal("username unique constraint must be visible")
	}
	if got := byName["uni_auth_user_username"].Columns(); len(got) != 1 || got[0] != "username" {
		t.Fatalf("username cols = %#v", got)
	}
	pk, pkOK := byName["auth_user_pkey"].PrimaryKey()
	if !pkOK || !pk {
		t.Fatal("pkey must be primary")
	}
	if got := byName["uq_pair"].Columns(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("composite cols = %#v", got)
	}
	pkPair, pkPairOK := byName["uq_pair"].PrimaryKey()
	if !pkPairOK || !pkPair {
		t.Fatal("later primary flag must OR onto existing aggregate")
	}

	// defaultGetIndexes routes postgres/postgresql dialector names.
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	origDialector := db.Dialector
	t.Cleanup(func() { db.Dialector = origDialector })
	postgresIndexScan = func(_ *gorm.DB, table string, dest any) error {
		if table != "routed" {
			t.Fatalf("table = %q", table)
		}
		rows := dest.(*[]postgresIndexRow)
		*rows = []postgresIndexRow{{IndexName: "uni_x", IsUnique: true, ColumnName: "x", ColumnOrd: 1}}
		return nil
	}
	for _, name := range []string{"postgres", "postgresql", "Postgres"} {
		db.Dialector = dialectorWithName{Dialector: origDialector, name: name}
		got, err := defaultGetIndexes(db, "routed")
		if err != nil || len(got) != 1 || got[0].Name() != "uni_x" {
			t.Fatalf("route %s: %#v %v", name, got, err)
		}
	}
}

func TestPlan_PostgresStyleUniqueConstraintSatisfiesDesired(t *testing.T) {
	t.Parallel()
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{
		"auth_user": {
			{Name: "username", FieldName: "Username", PhysicalType: "varchar", Unique: true},
			{Name: "email", FieldName: "Email", PhysicalType: "varchar", Unique: true},
		},
	}}
	live := LiveSchema{
		Tables: map[string]bool{"auth_user": true},
		Columns: map[string]map[string]LiveColumn{
			"auth_user": {
				"username": {Name: "username", DatabaseTypeName: "varchar"},
				"email":    {Name: "email", DatabaseTypeName: "varchar"},
			},
		},
		Indexes: map[string][]LiveIndex{
			"auth_user": {
				{Name: "uni_auth_user_username", Columns: []string{"username"}, Unique: true},
				{Name: "uni_auth_user_email", Columns: []string{"email"}, Unique: true},
			},
		},
		RowCount: map[string]int64{"auth_user": 3},
	}
	plan, err := buildPlan("auth", desired, live, "postgres")
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}
	for _, op := range plan.Ops {
		if op.Kind == OpAddIndex {
			t.Fatalf("existing UNIQUE CONSTRAINT must not plan add_index, got %#v", plan.Ops)
		}
	}
}

func TestLiveColumnFromColumnType_Default(t *testing.T) {
	t.Parallel()
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
	name          string
	cols          []string
	unique        bool
	uniqueUnknown bool
}

func (f fakeIndex) Table() string            { return "" }
func (f fakeIndex) Name() string             { return f.name }
func (f fakeIndex) Columns() []string        { return f.cols }
func (f fakeIndex) PrimaryKey() (bool, bool) { return false, true }
func (f fakeIndex) Unique() (bool, bool) {
	if f.uniqueUnknown {
		return false, false
	}
	return f.unique, true
}
func (f fakeIndex) Option() string { return "" }

type fakeColumnTypeWithDefault struct {
	fakeColumnType
	def   string
	defOK bool
}

func (f fakeColumnTypeWithDefault) DefaultValue() (string, bool) { return f.def, f.defOK }
