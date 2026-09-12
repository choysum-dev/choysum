// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestColumnSpecFromField_PrimaryKeyImpliesNotNull(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}
	field := &meta.Field{Name: "Id"}
	pk := true
	spec := &meta.FieldResolvedSpec{
		FieldName: "Id",
		Structural: meta.FieldStructuralSpec{
			Name:      "Id",
			FieldType: "varchar",
			StorageHints: &meta.FieldStructuralStorageHints{
				PrimaryKey: &pk,
				Size:       intPtrValue(20),
			},
		},
		Migration: meta.FieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: "varchar",
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	col, err := columnSpecFromField(field, model)
	if err != nil || col == nil {
		t.Fatalf("columnSpecFromField: %#v %v", col, err)
	}
	if !col.PrimaryKey || !col.NotNull {
		t.Fatalf("primary key must imply NotNull, got %#v", col)
	}

	// Live PK columns report Nullable=false; Desired must not flag loosen-nullability.
	liveNullable := false
	if mismatch, reason := columnMismatch(*col, LiveColumn{
		DatabaseTypeName: "varchar",
		Nullable:         &liveNullable,
		Length:           int64Ptr(20),
	}, "postgres"); mismatch {
		t.Fatalf("unexpected mismatch for PK column: %q", reason)
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestColumnSpecFromField_SelectionVarchar255(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}
	field := newFieldWithOptions(t, "Status", `{"type":"selection"}`)
	col, err := columnSpecFromField(field, model)
	if err != nil {
		t.Fatalf("columnSpecFromField() error = %v", err)
	}
	if col == nil {
		t.Fatal("expected column spec for selection")
	}
	if col.PhysicalType != "varchar" || col.Size == nil || *col.Size != 255 {
		t.Fatalf("unexpected selection spec: %#v", col)
	}
}

func TestColumnSpecFromField_SkipsVirtualAndRelationOnly(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}

	virtualField := newFieldWithOptions(t, "DisplayName", `{"type":"varchar","select":"expr"}`)
	col, err := columnSpecFromField(virtualField, model)
	if err != nil || col != nil {
		t.Fatalf("virtual sql compute = (%v, %v)", col, err)
	}

	oneToManyField := newFieldWithOptions(t, "Items", `{"type":"OneToMany"}`)
	col, err = columnSpecFromField(oneToManyField, model)
	if err != nil || col != nil {
		t.Fatalf("OneToMany = (%v, %v)", col, err)
	}

	manyToManyField := newFieldWithOptions(t, "Tags", `{"type":"ManyToMany"}`)
	col, err = columnSpecFromField(manyToManyField, model)
	if err != nil || col != nil {
		t.Fatalf("ManyToMany = (%v, %v)", col, err)
	}
}

func TestColumnSpecFromField_TranslateJSONMap(t *testing.T) {
	model := &meta.Model{Name: "Language", ModelTable: "base_language"}
	trueVal := true
	trigram := "trigram"
	size := 100

	field := &meta.Field{Name: "Name"}
	spec := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Name:      "Name",
			FieldType: "varchar",
			Translate: &trueVal,
			StorageHints: &meta.FieldStructuralStorageHints{
				Size:   &size,
				Unique: &trueVal,
				Index:  &trigram,
			},
			ColumnType: "jsonobject",
		},
		Migration: meta.FieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: "jsonobject",
			ReasonCode:         "TRANSLATE_LANG_MAP",
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec error = %v", err)
	}

	col, err := columnSpecFromField(field, model)
	if err != nil {
		t.Fatalf("columnSpecFromField() error = %v", err)
	}
	if col == nil || col.PhysicalType != "jsonobject" || col.StorageKind != StorageJSONMap {
		t.Fatalf("unexpected translate spec: %#v", col)
	}
	if col.Size != nil || col.Unique || col.Indexed || col.IndexName != "" {
		t.Fatalf("translate must strip size/index/unique, got %#v", col)
	}
	if !col.Trigram {
		t.Fatalf("expected trigram marker, got %#v", col)
	}

	typeTag := buildColumnTypeTag("postgres", col.PhysicalType, map[string]interface{}{})
	if typeTag != "type:jsonb" {
		t.Fatalf("expected postgres jsonb tag, got %q", typeTag)
	}
}

func TestColumnSpecFromField_CompanyDependent(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}

	selectionField := newFieldWithOptions(t, "CompanyStatus", `{"type":"selection","companyDependent":true}`)
	col, err := columnSpecFromField(selectionField, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(selection+companyDependent) error = %v", err)
	}
	if col == nil || col.PhysicalType != "jsonobject" || col.StorageKind != StorageJSONMap {
		t.Fatalf("unexpected companyDependent selection: %#v", col)
	}
	if col.Size != nil || col.Indexed {
		t.Fatalf("companyDependent must drop size/index, got %#v", col)
	}

	m2oField := newFieldWithOptions(t, "CompanyPartnerId", `{"type":"ManyToOne","companyDependent":true,"relation":{"onDelete":"SET NULL"}}`)
	col, err = columnSpecFromField(m2oField, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(ManyToOne+companyDependent) error = %v", err)
	}
	if col == nil || col.PhysicalType != "jsonobject" || col.StorageKind != StorageJSONMap {
		t.Fatalf("unexpected companyDependent ManyToOne: %#v", col)
	}
	if col.Size != nil || col.Indexed {
		t.Fatalf("companyDependent ManyToOne must drop size/index, got %#v", col)
	}
}

func TestColumnSpecFromField_BinarySkipVsDocumentCarrier(t *testing.T) {
	binaryField := newFieldWithOptions(t, "Payload", `{"type":"binary"}`)

	ownerModel := &meta.Model{Name: "User", Application: "auth", ModelTable: "auth_user"}
	col, err := columnSpecFromField(binaryField, ownerModel)
	if err != nil || col != nil {
		t.Fatalf("owner model binary skip = (%v, %v)", col, err)
	}

	documentModel := &meta.Model{Name: "AttachmentObject", Application: "document", ModelTable: "document_attachment_object"}
	col, err = columnSpecFromField(binaryField, documentModel)
	if err != nil {
		t.Fatalf("document carrier binary error = %v", err)
	}
	if col == nil || col.PhysicalType != "blob" {
		t.Fatalf("expected document carrier blob column, got %#v", col)
	}

	imageField := newFieldWithOptions(t, "Avatar", `{"type":"image"}`)
	col, err = columnSpecFromField(imageField, documentModel)
	if err != nil || col == nil || col.PhysicalType != "blob" {
		t.Fatalf("document image blob = (%v, %v)", col, err)
	}
}

func TestColumnSpecFromField_ManyToOneDefaults(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}

	manyToOneField := newFieldWithOptions(t, "OwnerId", `{"type":"ManyToOne","relation":{"onDelete":"CASCADE"}}`)
	manyToOneField.NotNull = true
	col, err := columnSpecFromField(manyToOneField, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(ManyToOne) error = %v", err)
	}
	if col == nil || col.PhysicalType != "char" || col.Size == nil || *col.Size != 20 || !col.NotNull || !col.Indexed {
		t.Fatalf("unexpected ManyToOne defaults: %#v", col)
	}

	uniqueIndexField := newFieldWithOptions(t, "UniqueOwnerId", `{"type":"ManyToOne","uniqueIndex":true,"relation":{"onDelete":"CASCADE"}}`)
	col, err = columnSpecFromField(uniqueIndexField, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(ManyToOne+uniqueIndex) error = %v", err)
	}
	if col == nil || !col.UniqueIndex || col.Indexed {
		t.Fatalf("expected uniqueIndex without redundant index, got %#v", col)
	}

	noUniqueIndexField := newFieldWithOptions(t, "RefOwnerId", `{"type":"ManyToOne","uniqueIndex":false,"relation":{"onDelete":"CASCADE"}}`)
	col, err = columnSpecFromField(noUniqueIndexField, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(ManyToOne+uniqueIndex=false) error = %v", err)
	}
	if col == nil || !col.Indexed {
		t.Fatalf("expected default index when uniqueIndex=false, got %#v", col)
	}

	uniqueField := newFieldWithOptions(t, "UniqueRefId", `{"type":"ManyToOne","unique":true}`)
	col, err = columnSpecFromField(uniqueField, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(ManyToOne+unique) error = %v", err)
	}
	if col == nil || !col.Unique || col.Indexed {
		t.Fatalf("expected unique without default index, got %#v", col)
	}
}

func TestColumnSpecFromField_DecimalIgnoresPrecision(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}
	field := &meta.Field{Name: "Amount"}
	spec := &meta.FieldResolvedSpec{
		FieldName: "Amount",
		Structural: meta.FieldStructuralSpec{
			Name:      "Amount",
			FieldType: "decimal",
			StorageHints: &meta.FieldStructuralStorageHints{
				Precision: intPtrValue(16),
				Scale:     intPtrValue(2),
			},
		},
		Migration: meta.FieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: "decimal",
			ReasonCode:         "OK",
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec error = %v", err)
	}

	col, err := columnSpecFromField(field, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(decimal) error = %v", err)
	}
	if col == nil || col.PhysicalType != "decimal" {
		t.Fatalf("expected physical decimal, got %#v", col)
	}
	if col.Size != nil {
		t.Fatalf("decimal must not carry size from precision, got %#v", col)
	}

	typeTag := buildColumnTypeTag("sqlite", col.PhysicalType, map[string]interface{}{
		"precision": 16,
		"scale":     2,
	})
	if !strings.Contains(typeTag, "decimal(38,18)") {
		t.Fatalf("expected fixed decimal(38,18) tag, got %q", typeTag)
	}
}

func TestColumnSpecFromField_CheckConstraint(t *testing.T) {
	model := &meta.Model{Name: "Order", ModelTable: "sales_order"}
	field := newFieldWithOptions(t, "Status", `{"type":"selection","column":{"checkConstraint":"status in ('draft','done')"}}`)
	col, err := columnSpecFromField(field, model)
	if err != nil {
		t.Fatalf("columnSpecFromField(check) error = %v", err)
	}
	if col == nil || col.CheckExpr != "status in ('draft','done')" {
		t.Fatalf("expected check constraint, got %#v", col)
	}
}

func TestBuildDesired_FiltersModels(t *testing.T) {
	disabledAutoMigrate := false
	models := []*meta.Model{
		{
			Name: "Order", ModelTable: "sales_order",
			Fields: []*meta.Field{
				newFieldWithOptions(t, "Status", `{"type":"selection"}`),
				newFieldWithOptions(t, "Lines", `{"type":"OneToMany"}`), // nil column, skipped
			},
		},
		{Name: "Readonly", ModelTable: "sales_readonly", Readonly: true, Fields: []*meta.Field{newFieldWithOptions(t, "Ignored", `{"type":"selection"}`)}},
		{Name: "Disabled", ModelTable: "sales_disabled", AutoMigrate: &disabledAutoMigrate, Fields: []*meta.Field{newFieldWithOptions(t, "Ignored", `{"type":"selection"}`)}},
	}
	desired, err := buildDesired(models)
	if err != nil {
		t.Fatalf("buildDesired() error = %v", err)
	}
	if len(desired.Tables) != 1 {
		t.Fatalf("expected one table, got %#v", desired.Tables)
	}
	cols, ok := desired.Tables["sales_order"]
	if !ok || len(cols) != 1 || cols[0].Name != "status" {
		t.Fatalf("unexpected desired columns: %#v", desired.Tables)
	}
}
