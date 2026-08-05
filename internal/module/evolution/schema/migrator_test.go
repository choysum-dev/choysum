// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"errors"
	"strings"
	"testing"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
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

func TestMigratorMigrateEnsuresTranslationTermTable(t *testing.T) {
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
	if !runtimeScope.Session().Migrator().HasTable("auth_translation_term") {
		t.Fatal("expected auth_translation_term after Migrate")
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

func TestMigratorMigrateWrapsEnsureI18nMetaError(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(runtimeScope, "auth"); err != nil {
		t.Fatalf("EnsureTranslationTermTable() error = %v", err)
	}
	if err := meta.EnsureDualStoreTables(runtimeScope.Session().DB); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	if err := runtimeScope.Session().Exec(`CREATE TRIGGER IF NOT EXISTS block_i18n_raw_model_insert
		BEFORE INSERT ON meta_raw_model
		WHEN NEW.name = 'I18n'
		BEGIN
			SELECT RAISE(ABORT, 'i18n blocked');
		END`).Error; err != nil {
		t.Fatalf("create i18n raw insert trigger: %v", err)
	}

	m := &migrator{
		runtimeScope:       runtimeScope,
		module:             &meta.Module{Name: "auth", ApplicationStr: "auth"},
		modelMigrator:      modelMigratorFunc(func() error { return nil }),
		foreignKeyMigrator: foreignKeyMigratorFunc(func() error { return nil }),
	}
	if err := m.Migrate(); err == nil || !strings.Contains(err.Error(), "ensure i18n ir meta") {
		t.Fatalf("Migrate() error = %v, want wrapped ensure i18n meta failure", err)
	}
}

type modelMigratorFunc func() error

func (f modelMigratorFunc) MigrateSchema() error { return f() }

type foreignKeyMigratorFunc func() error

func (f foreignKeyMigratorFunc) MigrateForeignKeys() error { return f() }
