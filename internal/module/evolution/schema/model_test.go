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
		if err := migrator.migrateTableSchema([]*meta.IrModel{broken}); err == nil || !strings.Contains(err.Error(), "error unmarshal field resolved spec") {
			t.Fatalf("migrateTableSchema() error = %v", err)
		}
		if err := migrator.applyTableCheckConstraints("sales_broken", broken); err == nil || !strings.Contains(err.Error(), "error unmarshal field resolved spec") {
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
	metaMap, err := migrator.getResolvedFieldColumnMeta(manyToOneField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(ManyToOne) error = %v", err)
	}
	if metaMap["type"] != "char" || metaMap["size"] != 20 || metaMap["notNull"] != true || metaMap["index"] != true {
		t.Fatalf("unexpected ManyToOne meta: %#v", metaMap)
	}

	// ManyToOne with uniqueIndex should not also default an ordinary index.
	manyToOneUniqueField := newFieldWithOptions(t, "UniqueOwnerId", `{"type":"ManyToOne","uniqueIndex":true,"relation":{"onDelete":"CASCADE"}}`)
	manyToOneUniqueMeta, err := migrator.getResolvedFieldColumnMeta(manyToOneUniqueField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(ManyToOne+uniqueIndex) error = %v", err)
	}
	if manyToOneUniqueMeta["type"] != "char" || manyToOneUniqueMeta["index"] != nil || manyToOneUniqueMeta["uniqueIndex"] == nil {
		t.Fatalf("expected ManyToOne uniqueIndex to suppress redundant index, got %#v", manyToOneUniqueMeta)
	}

	// ManyToOne with uniqueIndex:false should still default an ordinary index.
	manyToOneNoUniqueField := newFieldWithOptions(t, "RefOwnerId", `{"type":"ManyToOne","uniqueIndex":false,"relation":{"onDelete":"CASCADE"}}`)
	manyToOneNoUniqueMeta, err := migrator.getResolvedFieldColumnMeta(manyToOneNoUniqueField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(ManyToOne+uniqueIndex=false) error = %v", err)
	}
	if manyToOneNoUniqueMeta["type"] != "char" || manyToOneNoUniqueMeta["index"] != true {
		t.Fatalf("expected ManyToOne uniqueIndex=false to keep default index, got %#v", manyToOneNoUniqueMeta)
	}

	selectionField := newFieldWithOptions(t, "Status", `{"type":"selection"}`)
	selectionMeta, err := migrator.getResolvedFieldColumnMeta(selectionField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(selection) error = %v", err)
	}
	if selectionMeta["type"] != "varchar" || selectionMeta["size"] != 255 {
		t.Fatalf("unexpected selection meta: %#v", selectionMeta)
	}

	refField := newFieldWithOptions(t, "RemoteId", `{"type":"ManyToOneRef"}`)
	refMeta, err := migrator.getResolvedFieldColumnMeta(refField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(ManyToOneRef) error = %v", err)
	}
	if refMeta["type"] != "char" || refMeta["index"] != true {
		t.Fatalf("unexpected ManyToOneRef meta: %#v", refMeta)
	}

	manyRefField := newFieldWithOptions(t, "RemoteIds", `{"type":"ManyToManyRef"}`)
	manyRefMeta, err := migrator.getResolvedFieldColumnMeta(manyRefField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(ManyToManyRef) error = %v", err)
	}
	if manyRefMeta["type"] != "jsonobject" {
		t.Fatalf("unexpected ManyToManyRef meta: %#v", manyRefMeta)
	}

	binaryField := newFieldWithOptions(t, "Payload", `{"type":"binary"}`)
	binaryMeta, err := migrator.getResolvedFieldColumnMeta(binaryField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(binary) error = %v", err)
	}
	if binaryMeta["type"] != "blob" {
		t.Fatalf("unexpected binary meta: %#v", binaryMeta)
	}

	ownerModel := &meta.IrModel{Name: "User", Application: "auth", ModelTable: "auth_user"}
	ownerBinaryMeta, err := migrator.getResolvedFieldColumnMeta(binaryField, ownerModel)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(binary, owner model) error = %v", err)
	}
	if ownerBinaryMeta != nil {
		t.Fatalf("expected owner model binary field to be skipped, got %#v", ownerBinaryMeta)
	}

	imageField := newFieldWithOptions(t, "Avatar", `{"type":"image"}`)
	imageMeta, err := migrator.getResolvedFieldColumnMeta(imageField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(image) error = %v", err)
	}
	if imageMeta["type"] != "blob" {
		t.Fatalf("unexpected image meta: %#v", imageMeta)
	}

	documentModel := &meta.IrModel{Name: "AttachmentObject", Application: "document", ModelTable: "document_attachment_object"}
	documentImageMeta, err := migrator.getResolvedFieldColumnMeta(imageField, documentModel)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(image, document model) error = %v", err)
	}
	if documentImageMeta == nil || documentImageMeta["type"] != "blob" {
		t.Fatalf("expected document model image field to remain blob, got %#v", documentImageMeta)
	}

	virtualField := newFieldWithOptions(t, "DisplayName", `{"type":"varchar","select":"expr"}`)
	virtualMeta, err := migrator.getResolvedFieldColumnMeta(virtualField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(virtual) error = %v", err)
	}
	if virtualMeta != nil {
		t.Fatalf("expected virtual field to be skipped, got %#v", virtualMeta)
	}

	oneToManyField := newFieldWithOptions(t, "Items", `{"type":"OneToMany"}`)
	oneToManyMeta, err := migrator.getResolvedFieldColumnMeta(oneToManyField)
	if err != nil {
		t.Fatalf("getResolvedFieldColumnMeta(OneToMany) error = %v", err)
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
		metaMap, err := migrator.getResolvedFieldColumnMeta(fieldWithoutDecorators)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(no decorators) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected nil meta for no decorators, got %#v", metaMap)
		}

		nonObject := &meta.IrField{Name: "Plain", Decorators: []*meta.IrDecorator{{Name: "Field", Arguments: []*meta.IrArgument{{Type: "StringLiteral", Value: `"ignored"`}}}}}
		metaMap, err = migrator.getResolvedFieldColumnMeta(nonObject)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(non object) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected nil meta for non-object decorator, got %#v", metaMap)
		}

		missingType := newFieldWithOptions(t, "MissingType", `{"column":{"size":32}}`)
		metaMap, err = migrator.getResolvedFieldColumnMeta(missingType)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(missing type) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected nil meta for missing type, got %#v", metaMap)
		}

		manyToManyField := newFieldWithOptions(t, "Tags", `{"type":"ManyToMany"}`)
		metaMap, err = migrator.getResolvedFieldColumnMeta(manyToManyField)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(ManyToMany) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected ManyToMany field to be skipped, got %#v", metaMap)
		}

		invalidJSON := newFieldWithOptions(t, "Broken", `{invalid}`)
		if _, err := migrator.getResolvedFieldColumnMeta(invalidJSON); err == nil || !strings.Contains(err.Error(), "error unmarshal field resolved spec") {
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

func TestModelMigratorResolvedSpecMigrationDecisions(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	migrator := newModelMigrator(runtimeScope, nil, nil)

	t.Run("sql compute fields do not create columns", func(t *testing.T) {
		field := &meta.IrField{Name: "DisplayName"}
		spec := &meta.IrFieldResolvedSpec{
			FieldName: "DisplayName",
			Structural: meta.IrFieldStructuralSpec{
				Name:      "DisplayName",
				FieldType: "varchar",
			},
			Migration: meta.IrFieldMigrationDecision{
				StorageKind:        "virtualSql",
				ShouldCreateColumn: false,
				ReasonCode:         "SQL_COMPUTE",
			},
		}
		if err := field.SetResolvedSpec(spec); err != nil {
			t.Fatalf("SetResolvedSpec error = %v", err)
		}

		metaMap, err := migrator.getResolvedFieldColumnMeta(field)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(sql compute) error = %v", err)
		}
		if metaMap != nil {
			t.Fatalf("expected sql compute field to skip column, got %#v", metaMap)
		}
	})

	t.Run("compute store false does not create columns while store true does", func(t *testing.T) {
		virtualField := &meta.IrField{Name: "VirtualTotal"}
		virtualSpec := &meta.IrFieldResolvedSpec{
			FieldName: "VirtualTotal",
			Structural: meta.IrFieldStructuralSpec{
				Name:      "VirtualTotal",
				FieldType: "decimal",
				StorageHints: &meta.IrFieldStructuralStorageHints{
					Precision: intPtr(16),
					Scale:     intPtr(2),
				},
			},
			Migration: meta.IrFieldMigrationDecision{
				StorageKind:        "virtualRuntime",
				ShouldCreateColumn: false,
				ReasonCode:         "COMPUTE_STORE_FALSE",
			},
		}
		if err := virtualField.SetResolvedSpec(virtualSpec); err != nil {
			t.Fatalf("SetResolvedSpec(virtual) error = %v", err)
		}

		persistedField := &meta.IrField{Name: "PersistedTotal"}
		persistedSpec := &meta.IrFieldResolvedSpec{
			FieldName: "PersistedTotal",
			Structural: meta.IrFieldStructuralSpec{
				Name:      "PersistedTotal",
				FieldType: "decimal",
				StorageHints: &meta.IrFieldStructuralStorageHints{
					Precision: intPtr(18),
					Scale:     intPtr(4),
				},
			},
			Migration: meta.IrFieldMigrationDecision{
				StorageKind:        "physical",
				ShouldCreateColumn: true,
				ResolvedColumnType: "decimal",
				ReasonCode:         "COMPUTE_STORE_TRUE",
			},
		}
		if err := persistedField.SetResolvedSpec(persistedSpec); err != nil {
			t.Fatalf("SetResolvedSpec(persisted) error = %v", err)
		}

		virtualMeta, err := migrator.getResolvedFieldColumnMeta(virtualField)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(virtual) error = %v", err)
		}
		if virtualMeta != nil {
			t.Fatalf("expected virtual compute field to skip column, got %#v", virtualMeta)
		}

		persistedMeta, err := migrator.getResolvedFieldColumnMeta(persistedField)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(persisted) error = %v", err)
		}
		if persistedMeta == nil || persistedMeta["type"] != "decimal" || persistedMeta["precision"] != 18 || persistedMeta["scale"] != 4 {
			t.Fatalf("unexpected persisted compute meta: %#v", persistedMeta)
		}
	})

	t.Run("related store true and flat hints map to physical column params", func(t *testing.T) {
		field := newFieldWithOptions(t, "PartnerName", `{"type":"varchar","related":{"path":"PartnerId.Name","store":true},"required":true,"indexed":true,"size":120}`)
		metaMap, err := migrator.getResolvedFieldColumnMeta(field)
		if err != nil {
			t.Fatalf("getResolvedFieldColumnMeta(related store=true) error = %v", err)
		}
		if metaMap == nil {
			t.Fatal("expected related store=true field to create a column")
		}
		if metaMap["type"] != "varchar" || metaMap["size"] != 120 || metaMap["notNull"] != true || metaMap["index"] != true {
			t.Fatalf("unexpected related+storage-hints column meta: %#v", metaMap)
		}
	})
}

func intPtr(v int) *int {
	value := v
	return &value
}
