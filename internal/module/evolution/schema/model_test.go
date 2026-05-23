// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"reflect"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gschema "gorm.io/gorm/schema"

	"github.com/choysum-dev/choysum/pkg/meta"
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
	active := &meta.IrModel{
		Name:       "Order",
		Path:       "sales/order.ts",
		ModelTable: "sales_order",
		Fields: []*meta.IrField{
			newFieldWithOptions(t, "Status", `{"type":"selection","column":{"checkConstraint":" status in ('draft','done') "}}`),
		},
	}
	readonly := &meta.IrModel{Name: "Readonly", Path: "sales/readonly.ts", ModelTable: "sales_readonly", Readonly: true, Fields: []*meta.IrField{newFieldWithOptions(t, "Ignored", `{"type":"selection"}`)}}
	disabled := &meta.IrModel{Name: "Disabled", Path: "sales/disabled.ts", ModelTable: "sales_disabled", AutoMigrate: &disabledAutoMigrate, Fields: []*meta.IrField{newFieldWithOptions(t, "Ignored", `{"type":"selection"}`)}}

	migrator := newModelMigrator(runtimeScope, nil, []*meta.IrModel{active, readonly, disabled})
	if err := migrator.migrateTableSchema([]*meta.IrModel{active, readonly, disabled}); err != nil {
		t.Fatalf("migrateTableSchema() error = %v", err)
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

	if err := migrator.MigrateSchema(); err != nil {
		t.Fatalf("MigrateSchema() error = %v", err)
	}
	if !runtimeScope.Session().Migrator().HasTable(&taskJobExecution{}) {
		t.Fatal("expected task_job_execution table to be ensured")
	}
}

func TestModelMigratorErrorPaths(t *testing.T) {
	t.Run("field metadata errors bubble up", func(t *testing.T) {
		runtimeScope := newSchemaTestScope(t)
		broken := &meta.IrModel{
			Name:       "Broken",
			Path:       "sales/broken.ts",
			ModelTable: "sales_broken",
			Fields: []*meta.IrField{
				newFieldWithOptions(t, "Broken", `{invalid}`),
			},
		}

		migrator := newModelMigrator(runtimeScope, nil, []*meta.IrModel{broken})
		if err := migrator.migrateTableSchema([]*meta.IrModel{broken}); err == nil || !strings.Contains(err.Error(), "error unmarshal @Field options") {
			t.Fatalf("migrateTableSchema() error = %v", err)
		}
		if err := migrator.applyTableCheckConstraints("sales_broken", broken); err == nil || !strings.Contains(err.Error(), "error unmarshal @Field options") {
			t.Fatalf("applyTableCheckConstraints() error = %v", err)
		}
	})

	t.Run("unknown dialect skips check constraints", func(t *testing.T) {
		model := &meta.IrModel{
			Name:       "Order",
			Path:       "sales/order.ts",
			ModelTable: "sales_order",
			Fields: []*meta.IrField{
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

		active := &meta.IrModel{
			Name:       "Order",
			Path:       "sales/order.ts",
			ModelTable: "sales_order",
			Fields: []*meta.IrField{
				newFieldWithOptions(t, "Status", `{"type":"selection"}`),
			},
		}

		migrator := newModelMigrator(runtimeScope, nil, []*meta.IrModel{active})
		if err := migrator.migrateTableSchema([]*meta.IrModel{active}); err == nil || !strings.Contains(err.Error(), "migrate table sales_order") {
			t.Fatalf("migrateTableSchema(closed DB) error = %v", err)
		}
		if err := newModelMigrator(runtimeScope, nil, nil).MigrateSchema(); err == nil {
			t.Fatal("expected MigrateSchema() to fail when task_job_execution migration uses closed DB")
		}
	})
}

func TestModelMigratorFieldParsingAndStructTags(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrator := newModelMigrator(runtimeScope, nil, nil)

	manyToOneField := newFieldWithOptions(t, "OwnerId", `{"type":"ManyToOne","relation":{"onDelete":"CASCADE"}}`)
	manyToOneField.NotNull = true
	metaMap, err := migrator.getFieldColumnMeta(manyToOneField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(ManyToOne) error = %v", err)
	}
	if metaMap["type"] != "char" || metaMap["size"] != 20 || metaMap["notNull"] != true || metaMap["index"] != true {
		t.Fatalf("unexpected ManyToOne meta: %#v", metaMap)
	}

	selectionField := newFieldWithOptions(t, "Status", `{"type":"selection"}`)
	selectionMeta, err := migrator.getFieldColumnMeta(selectionField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(selection) error = %v", err)
	}
	if selectionMeta["type"] != "varchar" || selectionMeta["size"] != 255 {
		t.Fatalf("unexpected selection meta: %#v", selectionMeta)
	}

	refField := newFieldWithOptions(t, "RemoteId", `{"type":"ManyToOneRef"}`)
	refMeta, err := migrator.getFieldColumnMeta(refField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(ManyToOneRef) error = %v", err)
	}
	if refMeta["type"] != "char" || refMeta["index"] != true {
		t.Fatalf("unexpected ManyToOneRef meta: %#v", refMeta)
	}

	manyRefField := newFieldWithOptions(t, "RemoteIds", `{"type":"ManyToManyRef"}`)
	manyRefMeta, err := migrator.getFieldColumnMeta(manyRefField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(ManyToManyRef) error = %v", err)
	}
	if manyRefMeta["type"] != "jsonobject" {
		t.Fatalf("unexpected ManyToManyRef meta: %#v", manyRefMeta)
	}

	binaryField := newFieldWithOptions(t, "Payload", `{"type":"binary"}`)
	binaryMeta, err := migrator.getFieldColumnMeta(binaryField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(binary) error = %v", err)
	}
	if binaryMeta["type"] != "blob" {
		t.Fatalf("unexpected binary meta: %#v", binaryMeta)
	}

	ownerModel := &meta.IrModel{Name: "User", Application: "auth", ModelTable: "auth_user"}
	ownerBinaryMeta, err := migrator.getFieldColumnMeta(binaryField, ownerModel)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(binary, owner model) error = %v", err)
	}
	if ownerBinaryMeta != nil {
		t.Fatalf("expected owner model binary field to be skipped, got %#v", ownerBinaryMeta)
	}

	imageField := newFieldWithOptions(t, "Avatar", `{"type":"image"}`)
	imageMeta, err := migrator.getFieldColumnMeta(imageField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(image) error = %v", err)
	}
	if imageMeta["type"] != "blob" {
		t.Fatalf("unexpected image meta: %#v", imageMeta)
	}

	documentModel := &meta.IrModel{Name: "AttachmentObject", Application: "document", ModelTable: "document_attachment_object"}
	documentImageMeta, err := migrator.getFieldColumnMeta(imageField, documentModel)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(image, document model) error = %v", err)
	}
	if documentImageMeta == nil || documentImageMeta["type"] != "blob" {
		t.Fatalf("expected document model image field to remain blob, got %#v", documentImageMeta)
	}

	virtualField := newFieldWithOptions(t, "DisplayName", `{"type":"varchar","select":"expr"}`)
	virtualMeta, err := migrator.getFieldColumnMeta(virtualField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(virtual) error = %v", err)
	}
	if virtualMeta != nil {
		t.Fatalf("expected virtual field to be skipped, got %#v", virtualMeta)
	}

	oneToManyField := newFieldWithOptions(t, "Items", `{"type":"OneToMany"}`)
	oneToManyMeta, err := migrator.getFieldColumnMeta(oneToManyField)
	if err != nil {
		t.Fatalf("getFieldColumnMeta(OneToMany) error = %v", err)
	}
	if oneToManyMeta != nil {
		t.Fatalf("expected OneToMany field to be skipped, got %#v", oneToManyMeta)
	}

	builder := dynamicStructBuilder()
	if err := migrator.addFieldToStruct(&builder, selectionField, selectionMeta); err != nil {
		t.Fatalf("addFieldToStruct() error = %v", err)
	}
	instance := builder.Build().New()
	field, ok := reflect.TypeOf(instance).Elem().FieldByName("Status")
	if !ok {
		t.Fatal("expected Status field in dynamic struct")
	}
	if tag := string(field.Tag); !strings.Contains(tag, `gorm:"type:varchar(255)"`) || !strings.Contains(tag, `json:"status"`) {
		t.Fatalf("unexpected struct tag: %s", tag)
	}

	unknownMeta := map[string]interface{}{"type": "unsupported"}
	if err := migrator.addFieldToStruct(&builder, selectionField, unknownMeta); err != nil {
		t.Fatalf("addFieldToStruct(unsupported) error = %v", err)
	}
}

func TestModelMigratorFieldParsingEdgeCasesAndDialectHelpers(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrator := newModelMigrator(runtimeScope, nil, nil)

	t.Run("field metadata parsing edge cases", func(t *testing.T) {
		fieldWithoutDecorators := &meta.IrField{Name: "Plain"}
		metaMap, err := migrator.getFieldColumnMeta(fieldWithoutDecorators)
		if err != nil {
			t.Fatalf("getFieldColumnMeta(no decorators) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected nil meta for no decorators, got %#v", metaMap)
		}

		nonObject := &meta.IrField{Name: "Plain", Decorators: []*meta.IrDecorator{{Name: "Field", Arguments: []*meta.IrArgument{{Type: "StringLiteral", Value: `"ignored"`}}}}}
		metaMap, err = migrator.getFieldColumnMeta(nonObject)
		if err != nil {
			t.Fatalf("getFieldColumnMeta(non object) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected nil meta for non-object decorator, got %#v", metaMap)
		}

		missingType := newFieldWithOptions(t, "MissingType", `{"column":{"size":32}}`)
		metaMap, err = migrator.getFieldColumnMeta(missingType)
		if err != nil {
			t.Fatalf("getFieldColumnMeta(missing type) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected nil meta for missing type, got %#v", metaMap)
		}

		manyToManyField := newFieldWithOptions(t, "Tags", `{"type":"ManyToMany"}`)
		metaMap, err = migrator.getFieldColumnMeta(manyToManyField)
		if err != nil {
			t.Fatalf("getFieldColumnMeta(ManyToMany) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected ManyToMany field to be skipped, got %#v", metaMap)
		}

		invalidJSON := newFieldWithOptions(t, "Broken", `{invalid}`)
		if _, err := migrator.getFieldColumnMeta(invalidJSON); err == nil || !strings.Contains(err.Error(), "error unmarshal @Field options") {
			t.Fatalf("expected invalid JSON error, got %v", err)
		}
	})

	t.Run("nil metadata and unsupported dialect helpers", func(t *testing.T) {
		builder := dynamicStructBuilder()
		if err := migrator.addFieldToStruct(&builder, &meta.IrField{Name: "Skipped"}, nil); err != nil {
			t.Fatalf("addFieldToStruct(nil meta) error = %v", err)
		}

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
	})
}
