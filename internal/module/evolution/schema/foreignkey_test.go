// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"reflect"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

func TestForeignKeySQLBuilders(t *testing.T) {
	fk := ForeignKeyInfo{TableName: "order", ColumnName: "user_id", ReferTableName: "user", ReferColumnName: "id", OnDelete: "CASCADE", OnUpdate: "RESTRICT"}

	pgCreate := (&PostgresForeignKeyBuilder{}).BuildForeignKeySQL(fk)
	if !strings.Contains(pgCreate, `ALTER TABLE "order" ADD CONSTRAINT "fk_order_user_id"`) || !strings.Contains(pgCreate, `ON DELETE CASCADE`) || !strings.Contains(pgCreate, `ON UPDATE RESTRICT`) {
		t.Fatalf("unexpected postgres create sql: %s", pgCreate)
	}
	pgDrop := (&PostgresForeignKeyBuilder{}).BuildDropForeignKeySQL(fk)
	if pgDrop != `ALTER TABLE "order" DROP CONSTRAINT IF EXISTS "fk_order_user_id"` {
		t.Fatalf("unexpected postgres drop sql: %s", pgDrop)
	}

	myCreate := (&MySQLForeignKeyBuilder{}).BuildForeignKeySQL(fk)
	if !strings.Contains(myCreate, "ALTER TABLE `order` ADD CONSTRAINT `fk_order_user_id`") || !strings.Contains(myCreate, "ON DELETE CASCADE") || !strings.Contains(myCreate, "ON UPDATE RESTRICT") {
		t.Fatalf("unexpected mysql create sql: %s", myCreate)
	}
	myDrop := (&MySQLForeignKeyBuilder{}).BuildDropForeignKeySQL(fk)
	if myDrop != "ALTER TABLE `order` DROP FOREIGN KEY IF EXISTS `fk_order_user_id`" {
		t.Fatalf("unexpected mysql drop sql: %s", myDrop)
	}
}

func TestForeignKeyDiscoveryAndConstructors(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrateSchemaMetaTables(t, runtimeScope.Session())
	module := &meta.Module{}
	targetModel := &meta.Model{Path: "app/user.ts", ModelTable: "app_user"}
	disabledAutoMigrate := false
	userModel := &meta.Model{Path: "app/order.ts", ModelTable: "app_order", Fields: []*meta.Field{newRelationField("OwnerId", "app/user", `{"type":"ManyToOne","relation":{"onDelete":"CASCADE","onUpdate":"RESTRICT"}}`), newRelationField("ComputedOwner", "app/user", `{"type":"ManyToOne","select":"expr"}`)}}
	readonlyModel := &meta.Model{Path: "app/readonly.ts", ModelTable: "readonly", Readonly: true, Fields: []*meta.Field{newRelationField("Ignored", "app/user", `{"type":"ManyToOne"}`)}}
	disabledModel := &meta.Model{Path: "app/disabled.ts", ModelTable: "disabled", AutoMigrate: &disabledAutoMigrate, Fields: []*meta.Field{newRelationField("Ignored", "app/user", `{"type":"ManyToOne"}`)}}

	fkMigrator := newForeignKeyMigrator(runtimeScope, module, []*meta.Model{userModel, targetModel, readonlyModel, disabledModel}).(*foreignKeyMigrator)
	fks, err := fkMigrator.getForeignKeys()
	if err != nil {
		t.Fatalf("getForeignKeys() error = %v", err)
	}
	if len(fks) != 1 {
		t.Fatalf("expected 1 foreign key, got %#v", fks)
	}
	if fks[0].TableName != "app_order" || fks[0].ColumnName != "owner_id" || fks[0].ReferTableName != "app_user" || fks[0].OnDelete != "CASCADE" || fks[0].OnUpdate != "RESTRICT" {
		t.Fatalf("unexpected foreign key info: %#v", fks[0])
	}

	if builder := getForeignKeyBuilder(runtimeScope.Session()); reflect.TypeOf(builder).Elem().Name() != "PostgresForeignKeyBuilder" {
		t.Fatalf("expected default sqlite builder to be PostgresForeignKeyBuilder, got %T", builder)
	}

	postgresSession := &scope.Session{DB: &gorm.DB{Config: &gorm.Config{Dialector: fakeDialector{name: "postgres"}}}}
	if builder := getForeignKeyBuilder(postgresSession); reflect.TypeOf(builder).Elem().Name() != "PostgresForeignKeyBuilder" {
		t.Fatalf("expected postgres builder, got %T", builder)
	}

	mySQLSession := &scope.Session{DB: &gorm.DB{Config: &gorm.Config{Dialector: fakeDialector{name: "mysql"}}}}
	if builder := getForeignKeyBuilder(mySQLSession); reflect.TypeOf(builder).Elem().Name() != "MySQLForeignKeyBuilder" {
		t.Fatalf("expected mysql builder, got %T", builder)
	}
}

func TestForeignKeyMigratorRuntimePaths(t *testing.T) {
	t.Run("resolves target model from database and migrates sqlite no-op", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope.Session())
		module := &meta.Module{Name: "sales"}
		if err := runtimeScope.Session().Create(module).Error; err != nil {
			t.Fatalf("create module: %v", err)
		}
		target := &meta.RawModel{Name: "User", Path: "sales/user.ts", ModelTable: "sales_user", ModuleId: module.Id}
		if err := runtimeScope.Session().Create(target).Error; err != nil {
			t.Fatalf("create target raw model: %v", err)
		}
		source := &meta.Model{Name: "Order", Path: "sales/order.ts", ModelTable: "sales_order", ModuleId: module.Id, Fields: []*meta.Field{newRelationField("OwnerId", "sales/user", `{"type":"ManyToOne","relation":{"onDelete":"CASCADE"}}`)}}

		fkMigrator := newForeignKeyMigrator(runtimeScope, module, []*meta.Model{source}).(*foreignKeyMigrator)
		resolved, err := fkMigrator.resolveTargetModelByPath("sales/user.ts")
		if err != nil {
			t.Fatalf("resolveTargetModelByPath() error = %v", err)
		}
		if resolved == nil || resolved.ModelTable != "sales_user" {
			t.Fatalf("resolved target model = %#v", resolved)
		}

		keys, err := fkMigrator.getManyToOneKeys()
		if err != nil {
			t.Fatalf("getManyToOneKeys() error = %v", err)
		}
		if len(keys) != 1 || keys[0].ReferTableName != "sales_user" {
			t.Fatalf("unexpected foreign keys: %#v", keys)
		}
		if err := fkMigrator.createForeignKeys(keys); err != nil {
			t.Fatalf("createForeignKeys() error = %v", err)
		}
		if err := fkMigrator.MigrateForeignKeys(); err != nil {
			t.Fatalf("MigrateForeignKeys() error = %v", err)
		}
	})

	t.Run("reports missing target model", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope.Session())
		module := &meta.Module{Name: "sales"}
		fkMigrator := newForeignKeyMigrator(runtimeScope, module, []*meta.Model{{Name: "Order", Path: "sales/order.ts", ModelTable: "sales_order", Fields: []*meta.Field{newRelationField("OwnerId", "sales/missing", `{"type":"ManyToOne"}`)}}}).(*foreignKeyMigrator)

		if _, err := fkMigrator.getManyToOneKeys(); err == nil || !strings.Contains(err.Error(), "target model sales/missing.ts not found") {
			t.Fatalf("expected missing target error, got %v", err)
		}
	})

	t.Run("rejects duplicate path with conflicting ModelTable", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope.Session())
		modA := &meta.Module{Name: "mod_a"}
		modB := &meta.Module{Name: "mod_b"}
		if err := runtimeScope.Session().Create(modA).Error; err != nil {
			t.Fatalf("create modA: %v", err)
		}
		if err := runtimeScope.Session().Create(modB).Error; err != nil {
			t.Fatalf("create modB: %v", err)
		}
		for _, row := range []*meta.RawModel{
			{Name: "User", Path: "shared/user.ts", ModelTable: "mod_a_user", ModuleId: modA.Id},
			{Name: "User", Path: "shared/user.ts", ModelTable: "mod_b_user", ModuleId: modB.Id},
		} {
			if err := runtimeScope.Session().Create(row).Error; err != nil {
				t.Fatalf("create raw: %v", err)
			}
		}
		fkMigrator := newForeignKeyMigrator(runtimeScope, modA, nil).(*foreignKeyMigrator)
		if _, err := fkMigrator.resolveTargetModelByPath("shared/user.ts"); err == nil || !strings.Contains(err.Error(), "ambiguous model path") {
			t.Fatalf("expected ambiguous path error, got %v", err)
		}
	})

	t.Run("resolves target from effective projection", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope.Session())
		eff := &meta.Model{Name: "User", Path: "crm/user.ts", Application: "crm", ModelTable: "crm_user"}
		if err := runtimeScope.Session().Create(eff).Error; err != nil {
			t.Fatalf("create effective: %v", err)
		}
		fkMigrator := newForeignKeyMigrator(runtimeScope, &meta.Module{Name: "crm"}, nil).(*foreignKeyMigrator)
		resolved, err := fkMigrator.resolveTargetModelByPath("crm/user.ts")
		if err != nil {
			t.Fatalf("resolveTargetModelByPath() error = %v", err)
		}
		if resolved == nil || resolved.ModelTable != "crm_user" {
			t.Fatalf("resolved = %#v", resolved)
		}
	})

	t.Run("rejects ambiguous effective ModelTable", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope.Session())
		for _, row := range []*meta.Model{
			{Name: "UserA", Path: "dup/user.ts", Application: "a", ModelTable: "a_user"},
			{Name: "UserB", Path: "dup/user.ts", Application: "b", ModelTable: "b_user"},
		} {
			if err := runtimeScope.Session().Create(row).Error; err != nil {
				t.Fatalf("create effective: %v", err)
			}
		}
		fkMigrator := newForeignKeyMigrator(runtimeScope, nil, nil).(*foreignKeyMigrator)
		if _, err := fkMigrator.resolveTargetModelByPath("dup/user.ts"); err == nil || !strings.Contains(err.Error(), "ambiguous model path") {
			t.Fatalf("expected ambiguous effective path error, got %v", err)
		}
	})

	t.Run("propagates effective and raw lookup errors", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope.Session())
		fkMigrator := newForeignKeyMigrator(runtimeScope, nil, nil).(*foreignKeyMigrator)
		if err := runtimeScope.Session().Migrator().DropTable(&meta.Model{}); err != nil {
			t.Fatalf("drop meta_model: %v", err)
		}
		if _, err := fkMigrator.resolveTargetModelByPath("any/path.ts"); err == nil {
			t.Fatal("expected effective find error")
		}

		runtimeScope2 := newSchemaTestScope(t)
		migrateSchemaMetaTables(t, runtimeScope2.Session())
		fkMigrator2 := newForeignKeyMigrator(runtimeScope2, nil, nil).(*foreignKeyMigrator)
		if err := runtimeScope2.Session().Migrator().DropTable(&meta.RawModel{}); err != nil {
			t.Fatalf("drop meta_raw_model: %v", err)
		}
		if _, err := fkMigrator2.resolveTargetModelByPath("any/path.ts"); err == nil {
			t.Fatal("expected raw find error")
		}
	})

	t.Run("uses default relation actions and ignores non many-to-one fields", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		target := &meta.Model{Name: "User", Path: "sales/user.ts", ModelTable: "sales_user"}
		source := &meta.Model{
			Name:       "Order",
			Path:       "sales/order.ts",
			ModelTable: "sales_order",
			Fields: []*meta.Field{
				newRelationField("OwnerId", "sales/user", `{"type":"ManyToOne"}`),
				newRelationField("Notes", "sales/user", `{"type":"OneToMany"}`),
			},
		}

		keys, err := newForeignKeyMigrator(runtimeScope, nil, []*meta.Model{source, target}).(*foreignKeyMigrator).getManyToOneKeys()
		if err != nil {
			t.Fatalf("getManyToOneKeys() error = %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("expected 1 foreign key, got %#v", keys)
		}
		if keys[0].OnDelete != "NO ACTION" || keys[0].OnUpdate != "NO ACTION" {
			t.Fatalf("expected default relation actions, got %#v", keys[0])
		}
	})

	t.Run("reports invalid field options json", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		target := &meta.Model{Name: "User", Path: "sales/user.ts", ModelTable: "sales_user"}
		source := &meta.Model{Name: "Order", Path: "sales/order.ts", ModelTable: "sales_order", Fields: []*meta.Field{newRelationField("OwnerId", "sales/user", `{bad json}`)}}

		_, err := newForeignKeyMigrator(runtimeScope, nil, []*meta.Model{source, target}).(*foreignKeyMigrator).getManyToOneKeys()
		if err == nil || !strings.Contains(err.Error(), "parse @Field options failed") {
			t.Fatalf("expected invalid json error, got %v", err)
		}
	})

	t.Run("propagates create errors on non-sqlite dialects", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		runtimeScope.session.DB.Config.Dialector = fakeDialector{name: "postgres"}
		fkMigrator := newForeignKeyMigrator(runtimeScope, nil, nil).(*foreignKeyMigrator)
		fks := []ForeignKeyInfo{{TableName: "sales_order", ColumnName: "owner_id", ReferTableName: "sales_user", ReferColumnName: "id", OnDelete: "CASCADE"}}

		if err := fkMigrator.createForeignKeys(fks); err == nil || !strings.Contains(err.Error(), "create foreign key failed") {
			t.Fatalf("expected create foreign key error, got %v", err)
		}
		if err := fkMigrator.MigrateForeignKeys(); err != nil {
			t.Fatalf("MigrateForeignKeys() with no keys error = %v", err)
		}
	})
}
