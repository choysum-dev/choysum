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

func TestAppendJoinTables_EdgeCases(t *testing.T) {
	if err := appendJoinTablesFromModels(nil, nil); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil desired: %v", err)
	}

	falseAM := false
	user := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "auth_user",
		Fields: []*meta.Field{
			nil,
			newFieldWithOptions(t, "Roles", `{
				"type":"ManyToMany",
				"relation":{"joinModel":"auth.UserRole","joinField":"UserId","inverseJoinField":"RoleId","targetModel":"auth.Role"}
			}`),
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

	joinOK := &meta.Model{
		Application: "auth", Name: "UserRole", ModelTable: "auth_user_role",
		Fields: []*meta.Field{newFieldWithOptions(t, "UserId", `{"type":"char","size":20}`)},
	}
	desired = DesiredSchema{
		Tables:     map[string][]ColumnSpec{"auth_user_role": {{Name: "user_id", PhysicalType: "char"}}},
		JoinTables: []JoinTableSpec{{Table: "auth_user_role"}}, // pre-seeded → dedupe
	}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{
		nil,
		{Name: "RO", ModelTable: "ro", Readonly: true},
		{Name: "Off", ModelTable: "off", AutoMigrate: &falseAM},
		user, joinOK,
	}); err != nil {
		t.Fatalf("dedupe: %v", err)
	}
	if len(desired.JoinTables) != 1 {
		t.Fatalf("JoinTables = %#v", desired.JoinTables)
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

	// Production-shaped joinModel arrow literal must resolve.
	arrowUser := &meta.Model{
		Application: "auth", Name: "User", ModelTable: "auth_user",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Roles", `{
				"type":"ManyToMany",
				"relation":{"joinModel":"() => UserRole","joinField":"UserId","inverseJoinField":"RoleId","targetModel":"() => Role"}
			}`),
		},
	}
	arrowJoin := &meta.Model{
		Application: "auth", Name: "UserRole", ModelTable: "auth_user_role",
		Fields: []*meta.Field{newFieldWithOptions(t, "UserId", `{"type":"char","size":20}`)},
	}
	arrowRole := &meta.Model{Application: "auth", Name: "Role", ModelTable: "auth_role"}
	desired = DesiredSchema{Tables: map[string][]ColumnSpec{
		"auth_user_role": {{Name: "user_id", PhysicalType: "char"}},
	}}
	if err := appendJoinTablesFromModels(&desired, []*meta.Model{arrowUser, arrowJoin, arrowRole}); err != nil {
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
			"auth": {{Name: "id"}},
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
	if err := runtimeScope.Session().DB.Create(&modmeta.SchemaSnapshot{
		ModelTable:  "own_empty",
		DesiredJSON: datatypes.JSON([]byte(`[]`)),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := markLeftoverOwnership(&SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverColumn, Table: "own_empty", Name: "c"},
	}}, runtimeScope); err != nil {
		t.Fatal(err)
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

	// Missing indexed column path: start from a bare table, then ensure adds columns+indexes.
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

	bare := newSchemaTestScope(t)
	if err := bare.Session().Exec(`CREATE TABLE task_job_execution (job_id varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	ensureTaskJobIndexesFn = func(*gorm.DB, string, ColumnSpec, string) error {
		return errString("add idx boom")
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
	// Extra live column → leftover; force snapshot load failure during ownership mark.
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
