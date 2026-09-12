// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gschema "gorm.io/gorm/schema"
)

type fakeDialector struct {
	name string
}

func (d fakeDialector) Name() string                                          { return d.name }
func (d fakeDialector) Initialize(*gorm.DB) error                             { return nil }
func (d fakeDialector) Migrator(*gorm.DB) gorm.Migrator                       { return nil }
func (d fakeDialector) DataTypeOf(*gschema.Field) string                      { return "" }
func (d fakeDialector) DefaultValueOf(*gschema.Field) clause.Expression       { return nil }
func (d fakeDialector) BindVarTo(clause.Writer, *gorm.Statement, interface{}) {}
func (d fakeDialector) QuoteTo(clause.Writer, string)                         {}
func (d fakeDialector) Explain(sql string, vars ...interface{}) string        { return sql }

func TestModelMigratorRuntimePaths(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	disabledAutoMigrate := false
	active := &meta.Model{
		Name:       "Order",
		Path:       "sales/order.ts",
		ModelTable: "sales_order",
		Fields: []*meta.Field{
			newFieldWithOptions(t, "Status", `{"type":"selection","column":{"checkConstraint":" status in ('draft','done') "}}`),
		},
	}
	readonly := &meta.Model{Name: "Readonly", Path: "sales/readonly.ts", ModelTable: "sales_readonly", Readonly: true, Fields: []*meta.Field{newFieldWithOptions(t, "Ignored", `{"type":"selection"}`)}}
	disabled := &meta.Model{Name: "Disabled", Path: "sales/disabled.ts", ModelTable: "sales_disabled", AutoMigrate: &disabledAutoMigrate, Fields: []*meta.Field{newFieldWithOptions(t, "Ignored", `{"type":"selection"}`)}}

	migrator := newModelMigrator(runtimeScope, nil, []*meta.Model{active, readonly, disabled})
	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema() error = %v", err)
	}
	if !runtimeScope.Session().Migrator().HasTable("sales_order") {
		t.Fatal("expected active model table to be created")
	}
	if runtimeScope.Session().Migrator().HasTable("sales_readonly") {
		t.Fatal("expected readonly model table to be skipped")
	}
	if runtimeScope.Session().Migrator().HasTable("sales_disabled") {
		t.Fatal("expected automigrate=false model table to be skipped")
	}
	if !runtimeScope.Session().Migrator().HasTable(&taskJobExecution{}) {
		t.Fatal("expected task_job_execution table to be ensured")
	}
}

func TestModelMigratorErrorPaths(t *testing.T) {
	t.Run("field metadata errors bubble up", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		broken := &meta.Model{
			Name:       "Broken",
			Path:       "sales/broken.ts",
			ModelTable: "sales_broken",
			Fields: []*meta.Field{
				newFieldWithOptions(t, "Broken", `{invalid}`),
			},
		}

		migrator := newModelMigrator(runtimeScope, nil, []*meta.Model{broken})
		if err := migrator.MigrateSchema(); err == nil || !strings.Contains(err.Error(), "error unmarshal field resolved spec") {
			t.Fatalf("MigrateSchema() error = %v", err)
		}
		if err := migrator.applyTableCheckConstraints("sales_broken", broken); err == nil || !strings.Contains(err.Error(), "error unmarshal field resolved spec") {
			t.Fatalf("applyTableCheckConstraints() error = %v", err)
		}
	})

	t.Run("unknown dialect skips check constraints", func(t *testing.T) {
		model := &meta.Model{
			Name:       "Order",
			Path:       "sales/order.ts",
			ModelTable: "sales_order",
			Fields: []*meta.Field{
				newFieldWithOptions(t, "Status", `{"type":"selection","column":{"checkConstraint":"status <> ''"}}`),
			},
		}

		fakeRuntimeScope := &schemaTestScope{session: &scope.Session{DB: &gorm.DB{Config: &gorm.Config{Dialector: fakeDialector{name: "oracle"}}}}}
		if err := newModelMigrator(fakeRuntimeScope, nil, nil).applyTableCheckConstraints("sales_order", model); err != nil {
			t.Fatalf("applyTableCheckConstraints(unknown dialect) error = %v", err)
		}
	})

	t.Run("closed database surfaces migration errors", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		sqlDB, err := runtimeScope.Session().DB.DB()
		if err != nil {
			t.Fatalf("DB() error = %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		active := &meta.Model{
			Name:       "Order",
			Path:       "sales/order.ts",
			ModelTable: "sales_order",
			Fields: []*meta.Field{
				newFieldWithOptions(t, "Status", `{"type":"selection"}`),
			},
		}

		migrator := newModelMigrator(runtimeScope, nil, []*meta.Model{active})
		if err := migrator.MigrateSchema(); err == nil || !strings.Contains(err.Error(), "database is closed") {
			t.Fatalf("MigrateSchema(closed DB) error = %v", err)
		}
		if err := newModelMigrator(runtimeScope, nil, nil).MigrateSchema(); err == nil {
			t.Fatal("expected MigrateSchema() to fail when task_job_execution migration uses closed DB")
		}
	})
}

func TestGetDialect(t *testing.T) {
	cases := []struct {
		name      string
		dialector string
		want      string
	}{
		{name: "postgres alias", dialector: "postgresql", want: "postgres"},
		{name: "mysql alias", dialector: "mariadb", want: "mysql"},
		{name: "sqlite", dialector: "sqlite", want: "sqlite"},
		{name: "sqlserver", dialector: "sqlserver", want: "sqlserver"},
		{name: "unknown", dialector: "oracle", want: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeRuntimeScope := &schemaTestScope{session: &scope.Session{DB: &gorm.DB{Config: &gorm.Config{Dialector: fakeDialector{name: tc.dialector}}}}}
			if got := newModelMigrator(fakeRuntimeScope, nil, nil).getDialect(); got != tc.want {
				t.Fatalf("getDialect() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestColumnSpecFromFieldEdgeCases(t *testing.T) {
	t.Run("nil field and missing resolved spec", func(t *testing.T) {
		if col, err := columnSpecFromField(nil, nil); err != nil || col != nil {
			t.Fatalf("nil field = (%v, %v)", col, err)
		}
		fieldWithoutDecorators := &meta.Field{Name: "Plain"}
		col, err := columnSpecFromField(fieldWithoutDecorators, nil)
		if err != nil || col != nil {
			t.Fatalf("no decorators = (%v, %v)", col, err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		invalidJSON := newFieldWithOptions(t, "Broken", `{invalid}`)
		if _, err := columnSpecFromField(invalidJSON, nil); err == nil || !strings.Contains(err.Error(), "error unmarshal field resolved spec") {
			t.Fatalf("expected invalid JSON error, got %v", err)
		}
	})

	t.Run("isStorageBlobCarrierModel covers all document carriers", func(t *testing.T) {
		carriers := []struct {
			app   string
			name  string
			table string
		}{
			{app: "document", name: "AttachmentObject", table: ""},
			{app: "document", name: "UploadSession", table: ""},
			{app: "document", name: "AttachmentContent", table: ""},
			{app: "document", name: "AttachmentUploadSession", table: ""},
			{app: "document", name: "StoredContent", table: ""},
			{app: "other", name: "User", table: "document_attachment_object"},
			{app: "other", name: "User", table: "document_upload_session"},
			{app: "other", name: "User", table: "document_attachment_content"},
			{app: "other", name: "User", table: "document_attachment_upload_session"},
			{app: "other", name: "User", table: "document_stored_content"},
		}
		for _, c := range carriers {
			model := &meta.Model{Application: c.app, Name: c.name, ModelTable: c.table}
			if !isStorageBlobCarrierModel(model) {
				t.Fatalf("expected isStorageBlobCarrierModel=true for app=%q name=%q table=%q", c.app, c.name, c.table)
			}
		}
		notCarrier := &meta.Model{Application: "auth", Name: "User", ModelTable: "auth_user"}
		if isStorageBlobCarrierModel(notCarrier) {
			t.Fatal("expected auth.User not to be a blob carrier")
		}
		if isStorageBlobCarrierModel(nil) {
			t.Fatal("expected nil model not to be a blob carrier")
		}
	})
}

func TestMigrateSchema_EnsureTaskJobExecutionTableFailure(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db, _ := runtimeScope.Session().DB.DB()
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	migrator := newModelMigrator(runtimeScope, nil, nil)
	if err := migrator.MigrateSchema(); err == nil {
		t.Fatal("expected MigrateSchema to fail with closed database")
	}
}
