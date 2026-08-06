// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
)

type fakeModelMigrator struct{ err error }

func (f fakeModelMigrator) MigrateSchema() error { return f.err }

type fakeForeignKeyMigrator struct{ err error }

func (f fakeForeignKeyMigrator) MigrateForeignKeys() error { return f.err }

func TestMigratorMigrateOrdersSchemaBeforeForeignKeys(t *testing.T) {
	order := []string{}
	m := &migrator{
		modelMigrator:      fakeModelMigrator{err: nil},
		foreignKeyMigrator: fakeForeignKeyMigrator{err: nil},
	}
	// override with closures via adapter structs
	m.modelMigrator = modelMigratorFunc(func() error {
		order = append(order, "schema")
		return nil
	})
	m.foreignKeyMigrator = foreignKeyMigratorFunc(func() error {
		order = append(order, "fk")
		return nil
	})

	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate error: %v", err)
	}
	if len(order) != 2 || order[0] != "schema" || order[1] != "fk" {
		t.Fatalf("unexpected call order: %#v", order)
	}
}

func TestMigratorMigrateCallsTerminologyEditorAllows(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)

	authMigrator := &migrator{
		runtimeScope:       runtimeScope,
		module:             &meta.Module{Name: "auth", ApplicationStr: "auth"},
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	if err := authMigrator.Migrate(); err != nil {
		t.Fatalf("Migrate(auth): %v", err)
	}
	// Table creation is MetaModel migrate's job; schema migrator only seeds ACL.
	if runtimeScope.Session().Migrator().HasTable("auth_translation_term") {
		t.Fatal("Migrate must not create auth_translation_term directly")
	}

	coreMigrator := &migrator{
		runtimeScope:       runtimeScope,
		module:             &meta.Module{Name: "base", ApplicationStr: "core"},
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	if err := coreMigrator.Migrate(); err != nil {
		t.Fatalf("Migrate(core): %v", err)
	}
	if runtimeScope.Session().Migrator().HasTable("core_translation_term") {
		t.Fatal("expected no core_translation_term when application == core")
	}
}

func TestMigratorMigrateWrapsErrors(t *testing.T) {
	m := &migrator{
		modelMigrator:      modelMigratorFunc(func() error { return errors.New("schema broken") }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	if err := m.Migrate(); err == nil || !strings.Contains(err.Error(), "migrate schema") || !strings.Contains(err.Error(), "schema broken") {
		t.Fatalf("expected wrapped schema error, got %v", err)
	}

	m = &migrator{
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return errors.New("fk broken") }),
	}
	if err := m.Migrate(); err == nil || !strings.Contains(err.Error(), "migrate foreign keys") || !strings.Contains(err.Error(), "fk broken") {
		t.Fatalf("expected wrapped foreign key error, got %v", err)
	}
}

func TestMigratorMigrateWrapsTerminologyEditorAllowsError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE auth_role (
			id TEXT PRIMARY KEY,
			code TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE auth_role_method_access (
			id TEXT PRIMARY KEY,
			role_id TEXT,
			meta_application_id TEXT,
			meta_model_id TEXT,
			meta_service_id TEXT,
			mode TEXT,
			source TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}
	if err := db.Exec(`CREATE TRIGGER no_acl_insert BEFORE INSERT ON auth_role_method_access
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, "role-1", "terminology.editor").Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	ts := time.Now().UTC()
	model := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "model-tt", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "TranslationTerm",
		Path:        "auth/translation_term.ts",
		Application: "auth",
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	for _, methodName := range []string{"Search", "Browse", "Update", "Count"} {
		svc := &meta.Service{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: "svc-" + methodName, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      methodName,
			ModelId:   model.Id,
		}
		if err := db.Create(svc).Error; err != nil {
			t.Fatalf("create service %s: %v", methodName, err)
		}
	}

	m := &migrator{
		runtimeScope:       runtimeScope,
		module:             &meta.Module{Name: "auth", ApplicationStr: "auth"},
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	err := m.Migrate()
	if err == nil || !strings.Contains(err.Error(), "ensure terminology editor allows") {
		t.Fatalf("expected wrapped ACL error, got %v", err)
	}
}

func TestMigratorMigrateSeedsTerminologyEditorAllows(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE auth_role (
			id TEXT PRIMARY KEY,
			code TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE auth_role_method_access (
			id TEXT PRIMARY KEY,
			role_id TEXT,
			meta_application_id TEXT,
			meta_model_id TEXT,
			meta_service_id TEXT,
			mode TEXT,
			source TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, "role-1", "terminology.editor").Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	ts := time.Now().UTC()
	model := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "model-tt", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "TranslationTerm",
		Path:        "auth/translation_term.ts",
		Application: "auth",
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	for _, methodName := range []string{"Search", "Browse", "Update", "Count"} {
		svc := &meta.Service{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: "svc-" + methodName, Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      methodName,
			ModelId:   model.Id,
		}
		if err := db.Create(svc).Error; err != nil {
			t.Fatalf("create service %s: %v", methodName, err)
		}
	}

	m := &migrator{
		runtimeScope:       runtimeScope,
		module:             &meta.Module{Name: "auth", ApplicationStr: "auth"},
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	var count int64
	if err := db.Table("auth_role_method_access").Where("role_id = ?", "role-1").Count(&count).Error; err != nil {
		t.Fatalf("count acl: %v", err)
	}
	if count != 4 {
		t.Fatalf("acl rows = %d, want 4", count)
	}
}

func TestGetModuleModelsFiltersAndWrapsDBErrors(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())

	module := &meta.Module{Name: "sales"}
	if err := runtimeScope.Session().Create(module).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	disabledAutoMigrate := false
	decls := []*meta.Model{
		{
			Name: "Order", Path: "sales/order.ts", ModelTable: "sales_order", ModuleId: module.Id,
			Fields: []*meta.Field{newFieldWithOptions(t, "Status", `{"type":"selection"}`)},
		},
		{Name: "Readonly", Path: "sales/readonly.ts", ModelTable: "sales_readonly", ModuleId: module.Id, Readonly: true},
		{Name: "Disabled", Path: "sales/disabled.ts", ModelTable: "sales_disabled", ModuleId: module.Id, AutoMigrate: &disabledAutoMigrate},
		{Name: "Abstract", Path: "sales/abstract.ts", ModelTable: "sales_abstract", ModuleId: module.Id, Abstract: true},
	}
	for _, model := range decls {
		if err := meta.PersistModelTreeAsRaw(runtimeScope.Session().DB, model); err != nil {
			t.Fatalf("persist declaration %s: %v", model.Name, err)
		}
	}

	loaded, err := getModuleModels(runtimeScope, module)
	if err != nil {
		t.Fatalf("getModuleModels() error = %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 active model, got %#v", loaded)
	}
	if loaded[0].ModelTable != "sales_order" {
		t.Fatalf("loaded model table = %q, want sales_order", loaded[0].ModelTable)
	}
	if len(loaded[0].Fields) != 1 || len(loaded[0].Fields[0].Decorators) != 1 || len(loaded[0].Fields[0].Decorators[0].Arguments) != 1 {
		t.Fatalf("expected fields/decorators/arguments to be preloaded, got %#v", loaded[0].Fields)
	}

	sqlDB, err := runtimeScope.Session().DB.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}
	if _, err := getModuleModels(runtimeScope, module); err == nil || !strings.Contains(err.Error(), "error getting models by module id") {
		t.Fatalf("expected wrapped db error, got %v", err)
	}
}

func TestGetModuleModels_CircularExtends(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())

	module := &meta.Module{Name: "sales"}
	if err := runtimeScope.Session().Create(module).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	decls := []*meta.Model{
		{Name: "A", Path: "/a.ts", ModelTable: "sales_a", ModuleId: module.Id, Extends: "/b.ts"},
		{Name: "B", Path: "/b.ts", ModelTable: "sales_b", ModuleId: module.Id, Extends: "/a.ts"},
	}
	for _, model := range decls {
		if err := meta.PersistModelTreeAsRaw(runtimeScope.Session().DB, model); err != nil {
			t.Fatalf("persist declaration %s: %v", model.Name, err)
		}
	}

	if _, err := getModuleModels(runtimeScope, module); err == nil || !strings.Contains(err.Error(), "expanding model extends") {
		t.Fatalf("expected circular extends error, got %v", err)
	}
}

func TestNewMigrator(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())
	migrated, err := NewMigrator(runtimeScope, &meta.Module{})
	if err != nil {
		t.Fatalf("NewMigrator() error = %v", err)
	}
	if migrated == nil {
		t.Fatal("expected NewMigrator to return migrator instance")
	}
}

func TestNewMigratorPropagatesLoadError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())
	module := &meta.Module{Name: "sales", ApplicationStr: "sales"}
	if err := runtimeScope.Session().Create(module).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	decls := []*meta.Model{
		{Name: "A", Path: "/a.ts", ModelTable: "sales_a", ModuleId: module.Id, Extends: "/b.ts"},
		{Name: "B", Path: "/b.ts", ModelTable: "sales_b", ModuleId: module.Id, Extends: "/a.ts"},
	}
	for _, model := range decls {
		if err := meta.PersistModelTreeAsRaw(runtimeScope.Session().DB, model); err != nil {
			t.Fatalf("persist declaration %s: %v", model.Name, err)
		}
	}
	if _, err := NewMigrator(runtimeScope, module); err == nil || !strings.Contains(err.Error(), "expanding model extends") {
		t.Fatalf("expected expand error from NewMigrator, got %v", err)
	}
}

func TestMigratorMigrateTerminologyEditorAllowsNoopWithoutRoleTable(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	m := &migrator{
		runtimeScope:       runtimeScope,
		module:             &meta.Module{Name: "auth", ApplicationStr: "auth"},
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v, want success when auth_role tables are absent", err)
	}
}

type modelMigratorFunc func() error

func (f modelMigratorFunc) MigrateSchema() error { return f() }

type foreignKeyMigratorFunc func() error

func (f foreignKeyMigratorFunc) MigrateForeignKeys() error { return f() }
