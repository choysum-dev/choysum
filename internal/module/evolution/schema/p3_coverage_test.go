// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func m2mField(t *testing.T, name, joinModel, joinField, inverse, target string) *meta.Field {
	t.Helper()
	return newFieldWithOptions(t, name, `{
		"type":"ManyToMany",
		"relation":{
			"joinModel":"`+joinModel+`",
			"joinField":"`+joinField+`",
			"inverseJoinField":"`+inverse+`",
			"targetModel":"`+target+`"
		}
	}`)
}

func TestAppendJoinTables_EdgeCases(t *testing.T) {
	if err := appendJoinTablesFromModels(nil, nil); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil desired: %v", err)
	}

	falseAM := false
	user := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "auth_user",
		Fields: []*meta.Field{
			nil,
			m2mField(t, "Roles", "auth.UserRole", "UserId", "RoleId", "auth.Role"),
		},
	}

	joinEmptyTable := &meta.Model{Application: "auth", Name: "UserRole", ModelTable: ""}
	desired := DesiredSchema{Tables: map[string][]ColumnSpec{}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{user, joinEmptyTable}); err == nil || !strings.Contains(err.Error(), "empty ModelTable") {
		t.Fatalf("empty ModelTable: %v", err)
	}

	joinNoCols := &meta.Model{Application: "auth", Name: "UserRole", ModelTable: "auth_user_role"}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{user, joinNoCols}); err == nil || !strings.Contains(err.Error(), "no desired columns") {
		t.Fatalf("no columns: %v", err)
	}

	// AutoMigrate=false join model is skipped (not an error).
	joinDisabled := &meta.Model{
		Application: "auth", Name: "UserRole", ModelTable: "auth_user_role", AutoMigrate: &falseAM,
		Fields: []*meta.Field{newFieldWithOptions(t, "UserId", `{"type":"char","size":20}`)},
	}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{user, joinDisabled}); err != nil {
		t.Fatalf("disabled join: %v", err)
	}
	if len(desired.JoinTables) != 0 {
		t.Fatalf("disabled join should skip, got %#v", desired.JoinTables)
	}

	joinOK := &meta.Model{
		Application: "auth", Name: "UserRole", ModelTable: "auth_user_role",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "UserId", `{"type":"char","size":20}`),
			newFieldWithOptions(t, "RoleId", `{"type":"char","size":20}`),
		},
	}
	role := &meta.Model{Application: "auth", Name: "Role", ModelTable: "auth_role"}
	match := JoinTableSpec{
		Table: "auth_user_role",
		Left:  JoinEnd{Column: "user_id", ReferTable: "auth_user", ReferColumn: "id"},
		Right: JoinEnd{Column: "role_id", ReferTable: "auth_role", ReferColumn: "id"},
	}
	desired = DesiredSchema{
		Tables: map[string][]ColumnSpec{"auth_user_role": {
			{Name: "user_id", PhysicalType: "char"},
			{Name: "role_id", PhysicalType: "char"},
		}},
		JoinTables: []JoinTableSpec{match},
	}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{
		nil,
		{Name: "RO", ModelTable: "ro", Readonly: true},
		{Name: "Off", ModelTable: "off", AutoMigrate: &falseAM},
		user, joinOK, role,
	}); err != nil {
		t.Fatalf("identical dedupe: %v", err)
	}
	if len(desired.JoinTables) != 1 {
		t.Fatalf("JoinTables = %#v", desired.JoinTables)
	}

	// Conflicting duplicate join table definition fails.
	conflictUser := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "auth_user",
		Fields: []*meta.Field{m2mField(t, "Roles", "auth.UserRole", "UserId", "RoleId", "auth.Role")},
	}
	desired = DesiredSchema{
		Tables: map[string][]ColumnSpec{"auth_user_role": {
			{Name: "user_id", PhysicalType: "char"},
			{Name: "role_id", PhysicalType: "char"},
		}},
		JoinTables: []JoinTableSpec{{
			Table: "auth_user_role",
			Left:  JoinEnd{Column: "other_id", ReferTable: "auth_user", ReferColumn: "id"},
			Right: JoinEnd{Column: "role_id", ReferTable: "auth_role", ReferColumn: "id"},
		}},
	}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{conflictUser, joinOK, role}); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("conflict: %v", err)
	}

	// Missing join fields fail closed.
	noFields := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "auth_user",
		Fields: []*meta.Field{m2mField(t, "Roles", "auth.UserRole", "", "", "auth.Role")},
	}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{"auth_user_role": {
		{Name: "user_id", PhysicalType: "char"}, {Name: "role_id", PhysicalType: "char"},
	}}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{noFields, joinOK}); err == nil || !strings.Contains(err.Error(), "joinField") {
		t.Fatalf("empty join fields: %v", err)
	}

	// Join column missing from desired table fails (left and right).
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{"auth_user_role": {
		{Name: "role_id", PhysicalType: "char"},
	}}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{user, joinOK, role}); err == nil || !strings.Contains(err.Error(), "user_id") {
		t.Fatalf("missing left join col: %v", err)
	}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{"auth_user_role": {
		{Name: "user_id", PhysicalType: "char"},
	}}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{user, joinOK, role}); err == nil || !strings.Contains(err.Error(), "role_id") {
		t.Fatalf("missing right join col: %v", err)
	}

	// Empty parent ModelTable fails.
	emptyParent := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "",
		Fields: []*meta.Field{m2mField(t, "Roles", "auth.UserRole", "UserId", "RoleId", "auth.Role")},
	}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{"auth_user_role": {
		{Name: "user_id", PhysicalType: "char"}, {Name: "role_id", PhysicalType: "char"},
	}}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{emptyParent, joinOK}); err == nil || !strings.Contains(err.Error(), "parent model") {
		t.Fatalf("empty parent: %v", err)
	}

	if resolveModelRef(nil, "") != nil {
		t.Fatal("empty ref")
	}
	if got := normalizeModelRefLiteral("() => UserRole"); got != "UserRole" {
		t.Fatalf("arrow joinModel = %q", got)
	}
	if got := normalizeModelRefLiteral("()=>auth.UserRole"); got != "auth.UserRole" {
		t.Fatalf("dotted arrow = %q", got)
	}
	if got := normalizeModelRefLiteral(`() => "UserRole"`); got != "UserRole" {
		t.Fatalf("quoted arrow = %q", got)
	}
	if got := normalizeModelRefLiteral("() => UserRole()"); got != "UserRole" {
		t.Fatalf("call suffix = %q", got)
	}

	arrowUser := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "auth_user",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Roles", `{
				"type":"ManyToMany",
				"relation":{"joinModel":"() => UserRole","joinField":"UserId","inverseJoinField":"RoleId","targetModel":"() => Role"}
			}`),
		},
	}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{
		"auth_user_role": {{Name: "user_id", PhysicalType: "char"}, {Name: "role_id", PhysicalType: "char"}},
	}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{arrowUser, joinOK, role}); err != nil {
		t.Fatalf("arrow joinModel: %v", err)
	}
	if len(desired.JoinTables) != 1 || desired.JoinTables[0].Table != "auth_user_role" {
		t.Fatalf("arrow JoinTables = %#v", desired.JoinTables)
	}
	if desired.JoinTables[0].Right.ReferTable != "auth_role" {
		t.Fatalf("target arrow resolve = %#v", desired.JoinTables[0].Right)
	}
}

func TestDesiredTableNames_JoinOnly(t *testing.T) {
	names := desiredTableNames(DesiredSchema{
		Tables: map[string][]ColumnSpec{
			"":     {{Name: "x"}},
			"Auth": {{Name: "id"}},
			"auth": {{Name: "id2"}}, // case-variant hits seen dedupe
		},
		JoinTables: []JoinTableSpec{
			{Table: ""},
			{Table: "auth"},
			{Table: "auth_user_role"},
		},
	})
	if len(names) != 2 {
		t.Fatalf("names = %#v", names)
	}
}

func TestMarkLeftoverOwnership_EdgeCases(t *testing.T) {
	if err := markLeftoverOwnership(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := markLeftoverOwnership(&SchemaPlan{}, nil); err != nil {
		t.Fatal(err)
	}
	plan := SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverIndex, Name: "idx_x"},
		{Kind: LeftoverColumn, Table: "", Name: "c"},
		{Kind: LeftoverColumn, Table: "t", Name: "a"},
		{Kind: LeftoverColumn, Table: "t", Name: "b"},
	}}
	if err := markLeftoverOwnership(&plan, nil); err != nil {
		t.Fatal(err)
	}
	if err := markLeftoverOwnership(&plan, &schemaTestScope{}); err != nil {
		t.Fatal(err)
	}

	runtimeScope := newSchemaTestScope(t)
	// Truly empty DesiredJSON skips ownership map for that snap.
	if err := runtimeScope.Session().DB.Create(&modmeta.SchemaSnapshot{
		ModelTable:  "own_empty",
		DesiredJSON: datatypes.JSON(nil),
	}).Error; err != nil {
		t.Fatal(err)
	}
	emptyPlan := SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "own_empty", Name: "c"},
		{Kind: LeftoverColumn, Table: "no_snap", Name: "x"}, // names==nil branch
	}}
	if err := markLeftoverOwnership(&emptyPlan, runtimeScope); err != nil {
		t.Fatal(err)
	}
	if emptyPlan.Leftover[0].ChoysumOwned || emptyPlan.Leftover[1].ChoysumOwned {
		t.Fatalf("expected unowned: %#v", emptyPlan.Leftover)
	}

	if err := runtimeScope.Session().DB.Create(&modmeta.SchemaSnapshot{
		ModelTable:  "own_bad",
		DesiredJSON: datatypes.JSON([]byte(`{`)),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := markLeftoverOwnership(&SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "own_bad", Name: "c"},
	}}, runtimeScope); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("bad json: %v", err)
	}

	if err := runtimeScope.Session().DB.Create(&modmeta.SchemaSnapshot{
		ModelTable:  "own_rf",
		DesiredJSON: datatypes.JSON([]byte(`[{"name":"code","renameFrom":"old_code"}]`)),
	}).Error; err != nil {
		t.Fatal(err)
	}
	rfPlan := SchemaPlan{Leftover: []Leftover{{Kind: LeftoverColumn, Table: "own_rf", Name: "old_code"}}}
	if err := markLeftoverOwnership(&rfPlan, runtimeScope); err != nil {
		t.Fatal(err)
	}
	if !rfPlan.Leftover[0].ChoysumOwned {
		t.Fatal("renameFrom should mark owned")
	}

	orig := loadSnapshotsFn
	t.Cleanup(func() { loadSnapshotsFn = orig })
	loadSnapshotsFn = func(*gorm.DB, []string) (map[string]modmeta.SchemaSnapshot, error) {
		return nil, errString("snap boom")
	}
	if err := markLeftoverOwnership(&SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "t", Name: "c"},
	}}, runtimeScope); err == nil || !strings.Contains(err.Error(), "snap boom") {
		t.Fatalf("load err: %v", err)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestEnsureTaskJobExecution_DialectBranches(t *testing.T) {
	if got := schemaDialectName(nil); got != "sqlite" {
		t.Fatalf("nil dialector = %q", got)
	}
	for _, tc := range []struct {
		name string
		want string
	}{
		{"postgres", "postgres"},
		{"postgresql", "postgres"},
		{"mysql", "mysql"},
		{"mariadb", "mysql"},
		{"sqlserver", "sqlserver"},
		{"sqlite", "sqlite"},
		{"oracle", "sqlite"},
	} {
		if got := schemaDialectName(fakeDialector{name: tc.name}); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}

	runtimeScope := newSchemaTestScope(t)
	if err := runtimeScope.Session().Exec(`CREATE TABLE task_job_execution (job_id varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("ensure bare table: %v", err)
	}
	if !runtimeScope.Session().Migrator().HasColumn("task_job_execution", "status") {
		t.Fatal("status missing")
	}
	var n int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='task_job_execution' AND (sql LIKE '%job_id%' OR name LIKE '%job_id%')`,
	).Scan(&n).Error; err != nil || n < 1 {
		t.Fatalf("expected job_id index after reconcile, n=%d err=%v", n, err)
	}
}

func TestEnsureTaskJobExecution_ErrorHooks(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	origCreate := structForCreateTableFn
	origAdd := structForAddColumnFn
	origIdx := ensureTaskJobIndexesFn
	t.Cleanup(func() {
		structForCreateTableFn = origCreate
		structForAddColumnFn = origAdd
		ensureTaskJobIndexesFn = origIdx
	})

	structForCreateTableFn = func(string, []ColumnSpec, string) (any, error) {
		return nil, errString("create struct boom")
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err == nil || !strings.Contains(err.Error(), "create struct boom") {
		t.Fatalf("create struct: %v", err)
	}
	structForCreateTableFn = origCreate

	ensureTaskJobIndexesFn = func(*gorm.DB, string, ColumnSpec, string) error {
		return errString("idx boom")
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err == nil || !strings.Contains(err.Error(), "idx boom") {
		t.Fatalf("create idx: %v", err)
	}
	ensureTaskJobIndexesFn = origIdx
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatal(err)
	}

	if err := runtimeScope.Session().Exec(`ALTER TABLE task_job_execution DROP COLUMN result_hash`).Error; err != nil {
		t.Fatal(err)
	}
	structForAddColumnFn = func(string, ColumnSpec, string) (any, error) {
		return nil, errString("add struct boom")
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err == nil || !strings.Contains(err.Error(), "add struct boom") {
		t.Fatalf("add struct: %v", err)
	}
	structForAddColumnFn = origAdd

	// AddColumn failure via injectable migrator hook.
	hookScope := newSchemaTestScope(t)
	if err := hookScope.Session().Exec(`CREATE TABLE task_job_execution (job_id varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	origAddCol := taskJobAddColumnFn
	t.Cleanup(func() { taskJobAddColumnFn = origAddCol })
	taskJobAddColumnFn = func(gorm.Migrator, any, string) error {
		return errString("add column boom")
	}
	if err := ensureTaskJobExecutionTable(hookScope); err == nil || !strings.Contains(err.Error(), "add column boom") {
		t.Fatalf("add column: %v", err)
	}
	taskJobAddColumnFn = origAddCol

	// Index ensure on existing columns: fail only for newly added indexed column.
	bare := newSchemaTestScope(t)
	if err := bare.Session().Exec(`CREATE TABLE task_job_execution (job_id varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	ensureTaskJobIndexesFn = func(_ *gorm.DB, _ string, col ColumnSpec, _ string) error {
		if col.Name == "status" {
			return errString("add idx boom")
		}
		return nil
	}
	if err := ensureTaskJobExecutionTable(bare); err == nil || !strings.Contains(err.Error(), "add idx boom") {
		t.Fatalf("add idx: %v", err)
	}
}

func TestFilterIntentCoveredLeftovers_NilBag(t *testing.T) {
	plan := SchemaPlan{Leftover: []Leftover{{Kind: LeftoverColumn, Name: "c"}}}
	got := filterIntentCoveredLeftovers(plan, nil)
	if len(got.Leftover) != 1 {
		t.Fatalf("%#v", got.Leftover)
	}
}

func TestBuildSchemaPlan_OwnershipError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_own_err",
		Fields: []*meta.Field{newFieldWithOptions(t, "Code", `{"type":"varchar","size":20}`)},
	}
	if err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	if err := runtimeScope.Session().Exec(`ALTER TABLE sales_own_err ADD COLUMN leftover_x TEXT`).Error; err != nil {
		t.Fatal(err)
	}
	orig := loadSnapshotsFn
	t.Cleanup(func() { loadSnapshotsFn = orig })
	loadSnapshotsFn = func(*gorm.DB, []string) (map[string]modmeta.SchemaSnapshot, error) {
		return nil, errString("own boom")
	}
	_, err := newModelMigrator(runtimeScope, nil, []*meta.Model{model}).PlanSchema()
	if err == nil || !strings.Contains(err.Error(), "own boom") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

func TestMigrateSchema_LeftoverWithoutIntentSucceeds(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	model := &meta.Model{
		Name: "Order", ModelTable: "sales_left_ok",
		Fields: []*meta.Field{newFieldWithOptions(t, "Code", `{"type":"varchar","size":20}`)},
	}
	mig := newModelMigrator(runtimeScope, nil, []*meta.Model{model})
	if err := mig.MigrateSchema(); err != nil {
		t.Fatal(err)
	}
	if err := runtimeScope.Session().Exec(`ALTER TABLE sales_left_ok ADD COLUMN ghost TEXT`).Error; err != nil {
		t.Fatal(err)
	}
	plan, err := mig.PlanSchema()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, left := range plan.Leftover {
		if left.Kind == LeftoverColumn && left.Name == "ghost" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ghost leftover, got %#v", plan.Leftover)
	}
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("migrate with leftover: %v", err)
	}
}
