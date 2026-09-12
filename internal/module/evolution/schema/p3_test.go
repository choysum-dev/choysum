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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestMigrateSchema_JoinTableEndToEnd(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	user := &meta.Model{Application: "auth", Name: "User", ModelTable: "auth_user", Fields: []*meta.Field{
		newFieldWithOptions(t, "Id", `{"type":"char","size":20}`),
		m2mField(t, "Roles", "auth.UserRole", "UserId", "RoleId", "auth.Role"),
	}}
	role := &meta.Model{Application: "auth", Name: "Role", ModelTable: "auth_role", Fields: []*meta.Field{
		newFieldWithOptions(t, "Id", `{"type":"char","size":20}`),
	}}
	userRole := &meta.Model{Application: "auth", Name: "UserRole", ModelTable: "auth_user_role", Fields: []*meta.Field{
		newFieldWithOptions(t, "UserId", `{"type":"char","size":20,"indexed":true}`),
		newFieldWithOptions(t, "RoleId", `{"type":"char","size":20,"indexed":true}`),
	}}
	mig := newModelMigrator(runtimeScope, nil, []*meta.Model{user, role, userRole})
	if err := mig.MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema: %v", err)
	}
	if !runtimeScope.Session().Migrator().HasTable("auth_user_role") {
		t.Fatal("expected join table created via Validate+apply")
	}
	var joinIdxCount int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='auth_user_role' AND sql LIKE '%user_id%'`,
	).Scan(&joinIdxCount).Error; err != nil || joinIdxCount < 1 {
		t.Fatalf("expected join-table index on user_id, count=%d err=%v", joinIdxCount, err)
	}
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='auth_user_role' AND sql LIKE '%role_id%'`,
	).Scan(&joinIdxCount).Error; err != nil || joinIdxCount < 1 {
		t.Fatalf("expected join-table index on role_id, count=%d err=%v", joinIdxCount, err)
	}
	if !runtimeScope.Session().Migrator().HasTable("auth_user") || !runtimeScope.Session().Migrator().HasTable("auth_role") {
		t.Fatal("expected parent tables")
	}
}

func TestDesired_JoinTableSpec(t *testing.T) {
	user := &meta.Model{
		Application: "auth",
		Name:        "User",
		ModelTable:  "auth_user",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Id", `{"type":"char","size":20}`),
			newFieldWithOptions(t, "Roles", `{
				"type":"ManyToMany",
				"relation":{
					"targetModel":"auth.Role",
					"joinModel":"auth.UserRole",
					"joinField":"UserId",
					"inverseJoinField":"RoleId"
				}
			}`),
		},
	}
	role := &meta.Model{
		Application: "auth",
		Name:        "Role",
		ModelTable:  "auth_role",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Id", `{"type":"char","size":20}`),
		},
	}
	userRole := &meta.Model{
		Application: "auth",
		Name:        "UserRole",
		ModelTable:  "auth_user_role",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "UserId", `{"type":"char","size":20,"indexed":true}`),
			newFieldWithOptions(t, "RoleId", `{"type":"char","size":20,"indexed":true}`),
		},
	}

	desired, err := buildDesired([]*meta.Model{user, role, userRole})
	if err != nil {
		t.Fatalf("buildDesired: %v", err)
	}
	if len(desired.JoinTables) != 1 {
		t.Fatalf("JoinTables = %#v, want 1", desired.JoinTables)
	}
	jt := desired.JoinTables[0]
	if jt.Table != "auth_user_role" {
		t.Fatalf("join table = %q", jt.Table)
	}
	if jt.Left.Column != "user_id" || jt.Left.ReferTable != "auth_user" {
		t.Fatalf("left = %#v", jt.Left)
	}
	if jt.Right.Column != "role_id" || jt.Right.ReferTable != "auth_role" {
		t.Fatalf("right = %#v", jt.Right)
	}
	if len(desired.Tables["auth_user_role"]) != 2 {
		t.Fatalf("join model columns = %#v", desired.Tables["auth_user_role"])
	}

	// Missing join model fails closed.
	_, err = buildDesired([]*meta.Model{user, role})
	if err == nil || !strings.Contains(err.Error(), "joinModel") {
		t.Fatalf("expected missing joinModel error, got %v", err)
	}

	// Bidirectional Role.Users must not conflict with User.Roles.
	role.Fields = append(role.Fields, newFieldWithOptions(t, "Users", `{
		"type":"ManyToMany",
		"relation":{
			"targetModel":"auth.User",
			"joinModel":"auth.UserRole",
			"joinField":"RoleId",
			"inverseJoinField":"UserId"
		}
	}`))
	desired, err = buildDesired([]*meta.Model{user, role, userRole})
	if err != nil {
		t.Fatalf("bidirectional buildDesired: %v", err)
	}
	if len(desired.JoinTables) != 1 {
		t.Fatalf("bidirectional JoinTables = %#v", desired.JoinTables)
	}
}

func TestPlan_CreateJoinTableNoOp(t *testing.T) {
	cols := []ColumnSpec{
		{Name: "user_id", FieldName: "UserId", PhysicalType: "varchar"},
		{Name: "role_id", FieldName: "RoleId", PhysicalType: "varchar"},
	}
	desired := DesiredSchema{
		Tables: map[string][]ColumnSpec{
			"auth_user":      {{Name: "id", PhysicalType: "varchar"}},
			"auth_user_role": cols,
		},
		JoinTables: []JoinTableSpec{{
			Table: "auth_user_role",
			Left:  JoinEnd{Column: "user_id", ReferTable: "auth_user", ReferColumn: "id"},
			Right: JoinEnd{Column: "role_id", ReferTable: "auth_role", ReferColumn: "id"},
		}},
	}

	// Missing live join table → OpCreateJoinTable (not OpCreateTable).
	plan, err := buildPlan("auth", desired, LiveSchema{Tables: map[string]bool{}}, "sqlite")
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}
	var createTable, createJoin bool
	for _, op := range plan.Ops {
		if op.Kind == OpCreateTable && op.Table == "auth_user_role" {
			createTable = true
		}
		if op.Kind == OpCreateJoinTable && op.Table == "auth_user_role" {
			createJoin = true
		}
	}
	if createTable || !createJoin {
		t.Fatalf("ops = %#v (want create_join_table only for join)", plan.Ops)
	}

	// Live join table exists with columns but no indexes → plan OpAddIndex (not create).
	indexedCols := []ColumnSpec{
		{Name: "user_id", FieldName: "UserId", PhysicalType: "varchar", Indexed: true},
		{Name: "role_id", FieldName: "RoleId", PhysicalType: "varchar", Indexed: true},
	}
	desiredIndexed := DesiredSchema{
		Tables: map[string][]ColumnSpec{
			"auth_user":      {{Name: "id", PhysicalType: "varchar"}},
			"auth_user_role": indexedCols,
		},
		JoinTables: []JoinTableSpec{{
			Table: "auth_user_role",
			Left:  JoinEnd{Column: "user_id", ReferTable: "auth_user", ReferColumn: "id"},
			Right: JoinEnd{Column: "role_id", ReferTable: "auth_role", ReferColumn: "id"},
		}},
	}
	plan, err = buildPlan("auth", desiredIndexed, LiveSchema{
		Tables: map[string]bool{"auth_user": true, "auth_user_role": true},
		Columns: map[string]map[string]LiveColumn{
			"auth_user": {"id": {Name: "id", DatabaseTypeName: "TEXT"}},
			"auth_user_role": {
				"user_id": {Name: "user_id", DatabaseTypeName: "TEXT"},
				"role_id": {Name: "role_id", DatabaseTypeName: "TEXT"},
			},
		},
		Indexes: map[string][]LiveIndex{"auth_user": {}, "auth_user_role": {}},
	}, "sqlite")
	if err != nil {
		t.Fatalf("buildPlan missing join index: %v", err)
	}
	var joinAddIndex bool
	for _, op := range plan.Ops {
		if op.Kind == OpCreateTable || op.Kind == OpCreateJoinTable {
			t.Fatalf("unexpected create op %#v", op)
		}
		if op.Kind == OpAddIndex && op.Table == "auth_user_role" {
			joinAddIndex = true
		}
	}
	if !joinAddIndex {
		t.Fatalf("expected add_index for existing join table, ops=%#v", plan.Ops)
	}

	// Live table exists → neither create for join.
	plan, err = buildPlan("auth", desired, LiveSchema{
		Tables: map[string]bool{"auth_user": true, "auth_user_role": true},
		Columns: map[string]map[string]LiveColumn{
			"auth_user": {"id": {Name: "id", DatabaseTypeName: "TEXT"}},
			"auth_user_role": {
				"user_id": {Name: "user_id", DatabaseTypeName: "TEXT"},
				"role_id": {Name: "role_id", DatabaseTypeName: "TEXT"},
			},
		},
		Indexes: map[string][]LiveIndex{"auth_user": {}, "auth_user_role": {}},
	}, "sqlite")
	if err != nil {
		t.Fatalf("buildPlan live: %v", err)
	}
	for _, op := range plan.Ops {
		if op.Kind == OpCreateTable || op.Kind == OpCreateJoinTable {
			t.Fatalf("unexpected create op %#v", op)
		}
	}

	// Empty join table name is ignored.
	plan, err = buildPlan("auth", DesiredSchema{
		Tables:     map[string][]ColumnSpec{"t": {{Name: "c", PhysicalType: "varchar"}}},
		JoinTables: []JoinTableSpec{{Table: ""}},
	}, LiveSchema{Tables: map[string]bool{}}, "sqlite")
	if err != nil {
		t.Fatalf("empty join name: %v", err)
	}

	// JoinTables without columns fails.
	_, err = buildPlan("auth", DesiredSchema{
		Tables:     map[string][]ColumnSpec{},
		JoinTables: []JoinTableSpec{{Table: "orphan_join"}},
	}, LiveSchema{Tables: map[string]bool{}}, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "no desired columns") {
		t.Fatalf("expected no-columns error, got %v", err)
	}

	// Duplicate JoinTableSpec entries: first emits create_join_table, second hits plannedCreate skip.
	plan, err = buildPlan("auth", DesiredSchema{
		Tables: map[string][]ColumnSpec{"dup_join": {
			{Name: "a_id", PhysicalType: "varchar"},
			{Name: "b_id", PhysicalType: "varchar"},
		}},
		JoinTables: []JoinTableSpec{
			{Table: "dup_join"},
			{Table: "dup_join"},
		},
	}, LiveSchema{Tables: map[string]bool{}}, "sqlite")
	if err != nil {
		t.Fatalf("dup join: %v", err)
	}
	nJoin := 0
	for _, op := range plan.Ops {
		if op.Kind == OpCreateJoinTable && op.Table == "dup_join" {
			nJoin++
		}
	}
	if nJoin != 1 {
		t.Fatalf("expected one create_join_table, got %d in %#v", nJoin, plan.Ops)
	}
}

func TestApply_CreateJoinTable(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	size := 20
	cols := []ColumnSpec{
		{Name: "user_id", FieldName: "UserId", PhysicalType: "varchar", Size: &size, Indexed: true},
		{Name: "role_id", FieldName: "RoleId", PhysicalType: "varchar", Size: &size, UniqueIndex: true},
		{Name: "scope", FieldName: "Scope", PhysicalType: "varchar", Size: &size, Unique: true},
		{Name: "tenant", FieldName: "Tenant", PhysicalType: "varchar", Size: &size, UniqueIndex: true, UniqueIndexNames: []string{"uniq_auth_user_role_tenant"}},
		{Name: "note", FieldName: "Note", PhysicalType: "varchar", Size: &size}, // non-indexed → skip branch
	}
	plan := SchemaPlan{Ops: []PlanOp{{
		Kind: OpCreateJoinTable, Safety: SafetyAuto, Table: "auth_user_role_apply",
		Columns: cols,
	}}}
	if err := applyPlan(runtimeScope, "sqlite", plan); err != nil {
		t.Fatalf("applyPlan: %v", err)
	}
	if !runtimeScope.Session().Migrator().HasTable("auth_user_role_apply") {
		t.Fatal("expected join table created")
	}
	if !runtimeScope.Session().Migrator().HasColumn("auth_user_role_apply", "user_id") {
		t.Fatal("expected user_id column")
	}
	var idxCount int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND sql LIKE '%user_id%'`,
		"auth_user_role_apply",
	).Scan(&idxCount).Error; err != nil || idxCount != 1 {
		t.Fatalf("expected exactly one index on user_id, count=%d err=%v", idxCount, err)
	}
	var uniqueRoleId int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql LIKE '%UNIQUE%' AND sql LIKE '%role_id%'`,
		"auth_user_role_apply",
	).Scan(&uniqueRoleId).Error; err != nil || uniqueRoleId < 1 {
		t.Fatalf("expected UNIQUE index on role_id, n=%d err=%v", uniqueRoleId, err)
	}
	var uniqueScope int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql LIKE '%UNIQUE%' AND sql LIKE '%scope%'`,
		"auth_user_role_apply",
	).Scan(&uniqueScope).Error; err != nil || uniqueScope < 1 {
		t.Fatalf("expected UNIQUE index on scope, n=%d err=%v", uniqueScope, err)
	}
	var namedTenant int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name=? AND name=?`,
		"auth_user_role_apply", "uniq_auth_user_role_tenant",
	).Scan(&namedTenant).Error; err != nil || namedTenant < 1 {
		t.Fatalf("expected named unique index on tenant, n=%d err=%v", namedTenant, err)
	}
	if columnNeedsIndex(ColumnSpec{}) || !columnNeedsIndex(ColumnSpec{Unique: true}) {
		t.Fatal("columnNeedsIndex")
	}

	// Index ensure failure on create_join_table.
	failScope := newSchemaTestScope(t)
	orig := ensureIndexesForColumnFn
	t.Cleanup(func() { ensureIndexesForColumnFn = orig })
	ensureIndexesForColumnFn = func(*gorm.DB, string, ColumnSpec, string) error {
		return errString("join idx boom")
	}
	if err := applyPlan(failScope, "sqlite", SchemaPlan{Ops: []PlanOp{{
		Kind: OpCreateJoinTable, Safety: SafetyAuto, Table: "auth_user_role_fail",
		Columns: []ColumnSpec{{Name: "user_id", FieldName: "UserId", PhysicalType: "varchar", Size: &size, Indexed: true}},
	}}}); err == nil || !strings.Contains(err.Error(), "join idx boom") {
		t.Fatalf("expected join index error, got %v", err)
	}
}

func TestLeftover_ChoysumOwnedAndIntentFilter(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB

	// Seed snapshot marking old_code as previously desired.
	payload, _ := json.Marshal([]ColumnSpec{{Name: "code"}, {Name: "old_code"}})
	if err := db.Create(&modmeta.SchemaSnapshot{
		ModelTable:  "sales_owned",
		DesiredJSON: datatypes.JSON(payload),
	}).Error; err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	plan := SchemaPlan{Leftover: []Leftover{
		{Kind: LeftoverIndex, Table: "sales_owned", Name: "idx_sales_owned_code"},
		{Kind: LeftoverIndex, Table: "sales_owned", Name: "custom_ix"},
		{Kind: LeftoverColumn, Table: "sales_owned", Name: "old_code"},
		{Kind: LeftoverColumn, Table: "sales_owned", Name: "stranger"},
	}}
	if err := markLeftoverOwnership(&plan, runtimeScope); err != nil {
		t.Fatalf("markLeftoverOwnership: %v", err)
	}
	ownedIdx, unownedIdx, ownedCol, unownedCol := false, false, false, false
	for _, left := range plan.Leftover {
		switch {
		case left.Kind == LeftoverIndex && left.Name == "idx_sales_owned_code":
			ownedIdx = left.ChoysumOwned
		case left.Kind == LeftoverIndex && left.Name == "custom_ix":
			unownedIdx = !left.ChoysumOwned
		case left.Kind == LeftoverColumn && left.Name == "old_code":
			ownedCol = left.ChoysumOwned
		case left.Kind == LeftoverColumn && left.Name == "stranger":
			unownedCol = !left.ChoysumOwned
		}
	}
	if !ownedIdx || !unownedIdx || !ownedCol || !unownedCol {
		t.Fatalf("ownership flags wrong: %#v", plan.Leftover)
	}

	bag := NewMemoryIntentBag()
	bag.Add(Intent{Kind: IntentDropColumn, Table: "sales_owned", Name: "old_code"})
	filtered := filterIntentCoveredLeftovers(plan, bag)
	for _, left := range filtered.Leftover {
		if left.Kind == LeftoverColumn && left.Name == "old_code" {
			t.Fatal("expected Intent-covered leftover column filtered out")
		}
	}
	if len(filtered.Leftover) != 3 {
		t.Fatalf("filtered leftovers = %#v", filtered.Leftover)
	}
}

func TestEnsureTaskJobExecution_Path1(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !runtimeScope.Session().Migrator().HasTable("task_job_execution") {
		t.Fatal("expected table")
	}
	if !runtimeScope.Session().Migrator().HasColumn("task_job_execution", "job_id") {
		t.Fatal("expected job_id")
	}
	var uniqueJobId int
	if err := runtimeScope.Session().Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='task_job_execution' AND sql LIKE '%UNIQUE%' AND sql LIKE '%job_id%'`,
	).Scan(&uniqueJobId).Error; err != nil || uniqueJobId < 1 {
		t.Fatalf("expected UNIQUE index on job_id, n=%d err=%v", uniqueJobId, err)
	}
	// Idempotent when table exists (warm path: skip index re-reconcile).
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("re-ensure: %v", err)
	}
	ready, err := taskJobExecutionUniqueJobIDReady(runtimeScope.Session().DB, "task_job_execution")
	if err != nil || !ready {
		t.Fatalf("unique ready after warm path: ready=%v err=%v", ready, err)
	}
	if ready, err := taskJobExecutionUniqueJobIDReady(nil, "task_job_execution"); err != nil || ready {
		t.Fatalf("nil db ready=%v err=%v", ready, err)
	}
	// Add missing column path: drop one column then re-ensure.
	if err := runtimeScope.Session().Exec(`ALTER TABLE task_job_execution DROP COLUMN result_hash`).Error; err != nil {
		t.Fatalf("drop column: %v", err)
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("add missing: %v", err)
	}
	if !runtimeScope.Session().Migrator().HasColumn("task_job_execution", "result_hash") {
		t.Fatal("expected result_hash restored")
	}

	// All columns present but UNIQUE missing → must reconcile (not warm-skip).
	db := runtimeScope.Session().DB
	indexes, err := getIndexes(db, "task_job_execution")
	if err != nil {
		t.Fatalf("getIndexes: %v", err)
	}
	mig := db.Table("task_job_execution").Migrator()
	for _, idx := range indexes {
		if idx == nil {
			continue
		}
		unique, ok := idx.Unique()
		if !ok || !unique {
			continue
		}
		cols := idx.Columns()
		if len(cols) != 1 || !strings.EqualFold(cols[0], "job_id") {
			continue
		}
		if err := mig.DropIndex(&struct {
			JobId string `gorm:"column:job_id;size:64;uniqueIndex"`
		}{}, idx.Name()); err != nil {
			// Fall back to DROP INDEX by name for sqlite.
			if dropErr := db.Exec(`DROP INDEX IF EXISTS "` + idx.Name() + `"`).Error; dropErr != nil {
				t.Fatalf("drop unique %s: mig=%v drop=%v", idx.Name(), err, dropErr)
			}
		}
	}
	ready, err = taskJobExecutionUniqueJobIDReady(db, "task_job_execution")
	if err != nil {
		t.Fatalf("ready after drop: %v", err)
	}
	if ready {
		t.Fatal("expected UNIQUE missing after drop")
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("reconcile unique: %v", err)
	}
	ready, err = taskJobExecutionUniqueJobIDReady(db, "task_job_execution")
	if err != nil || !ready {
		t.Fatalf("unique restored: ready=%v err=%v", ready, err)
	}

	// UNIQUE job_id present but a secondary indexed column missing → must reconcile.
	indexes, err = getIndexes(db, "task_job_execution")
	if err != nil {
		t.Fatalf("getIndexes before status drop: %v", err)
	}
	statusDropped := false
	for _, idx := range indexes {
		if idx == nil {
			continue
		}
		unique, ok := idx.Unique()
		if ok && unique {
			continue
		}
		cols := idx.Columns()
		if len(cols) != 1 || !strings.EqualFold(cols[0], "status") {
			continue
		}
		if err := db.Exec(`DROP INDEX IF EXISTS "` + idx.Name() + `"`).Error; err != nil {
			t.Fatalf("drop status index %s: %v", idx.Name(), err)
		}
		statusDropped = true
		break
	}
	if !statusDropped {
		t.Fatal("expected a non-unique status index to drop")
	}
	indexesReady, err := taskJobExecutionIndexesReady(db, "task_job_execution", taskJobExecutionColumns())
	if err != nil {
		t.Fatalf("indexesReady after status drop: %v", err)
	}
	if indexesReady {
		t.Fatal("expected indexesReady=false after dropping status index")
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("reconcile secondary index: %v", err)
	}
	indexesReady, err = taskJobExecutionIndexesReady(db, "task_job_execution", taskJobExecutionColumns())
	if err != nil || !indexesReady {
		t.Fatalf("status index restored: ready=%v err=%v", indexesReady, err)
	}
	if ready, err := taskJobExecutionIndexesReady(nil, "task_job_execution", nil); err != nil || ready {
		t.Fatalf("nil db indexesReady=%v err=%v", ready, err)
	}
	origGet := getIndexes
	t.Cleanup(func() { getIndexes = origGet })
	getIndexes = func(*gorm.DB, string) ([]gorm.Index, error) {
		return nil, errors.New("indexes inspect boom")
	}
	if _, err := taskJobExecutionIndexesReady(db, "task_job_execution", taskJobExecutionColumns()); err == nil || !strings.Contains(err.Error(), "indexes inspect boom") {
		t.Fatalf("indexesReady inspect err: %v", err)
	}
	getIndexes = origGet
}

func TestSaveSnapshots_MissingTable(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Migrator().DropTable(&modmeta.SchemaSnapshot{}); err != nil {
		t.Fatalf("drop snapshot table: %v", err)
	}
	err := SaveSnapshots(db, DesiredSchema{Tables: map[string][]ColumnSpec{
		"t": {{Name: "c", PhysicalType: "varchar"}},
	}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "meta_schema_snapshot missing") {
		t.Fatalf("expected missing table error, got %v", err)
	}
}
