// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type IrResolvedValue[T any] struct {
	Value  T      `json:"value"`
	Source string `json:"source,omitempty"`
}

type IrFieldStructuralStorageHints struct {
	Required           *bool   `json:"required,omitempty"`
	Indexed            *bool   `json:"indexed,omitempty"`
	Index              *string `json:"index,omitempty"`
	Size               *int    `json:"size,omitempty"`
	Precision          *int    `json:"precision,omitempty"`
	Scale              *int    `json:"scale,omitempty"`
	PrimaryKey         *bool   `json:"primaryKey,omitempty"`
	Unique             *bool   `json:"unique,omitempty"`
	UniqueIndex        *string `json:"uniqueIndex,omitempty"`
	UniqueIndexEnabled *bool   `json:"uniqueIndexEnabled,omitempty"`
	Default            *string `json:"default,omitempty"`
}

type IrFieldRelatedSpec struct {
	Path  string   `json:"path"`
	Store bool     `json:"store"`
	Deps  []string `json:"deps,omitempty"`
}

type TermReference struct {
	Key    string `json:"key"`
	Module string `json:"module"`
	Scope  string `json:"scope"`
	Src    string `json:"src"`
	Kind   string `json:"kind"`
}

const TermReferenceNamespace = "__terms"

func TermReferenceKey(module, scope, src, kind string) string {
	values := []string{module, scope, src, kind}
	var identity strings.Builder
	for _, value := range values {
		identity.WriteString(fmt.Sprintf("%d:", len([]byte(value))))
		identity.WriteString(value)
	}
	return TermReferenceNamespace + "." + hex.EncodeToString([]byte(identity.String()))
}

func NewTermReference(module, scope, src, kind string) TermReference {
	return TermReference{
		Key:    TermReferenceKey(module, scope, src, kind),
		Module: module,
		Scope:  scope,
		Src:    src,
		Kind:   kind,
	}
}

type IrFieldSelectionItem struct {
	Value     string         `json:"value"`
	Label     string         `json:"label"`
	LabelText *TermReference `json:"labelText,omitempty"`
}

type IrFieldStructuralSpec struct {
	Name            string                         `json:"name"`
	FieldType       string                         `json:"fieldType"`
	String          string                         `json:"string,omitempty"`
	StringText      *TermReference                 `json:"stringText,omitempty"`
	Relation        map[string]any                 `json:"relation,omitempty"`
	Selection       []IrFieldSelectionItem         `json:"selection,omitempty"`
	SelectionKind   string                         `json:"selectionKind,omitempty"`
	SelectionMethod string                         `json:"selectionMethod,omitempty"`
	Related         *IrFieldRelatedSpec            `json:"related,omitempty"`
	StorageHints    *IrFieldStructuralStorageHints `json:"storageHints,omitempty"`
	ColumnType      string                         `json:"columnType,omitempty"`
	CheckConstraint string                         `json:"checkConstraint,omitempty"`
	// Translate marks data-i18n fields stored as JSON/JSONB lang maps (see data-i18n-design.md).
	Translate *bool `json:"translate,omitempty"`
}

type IrFieldBehaviorComputeSpec struct {
	Method     string   `json:"method"`
	Deps       []string `json:"deps"`
	Store      bool     `json:"store"`
	Searchable *bool    `json:"searchable,omitempty"`
}

type IrFieldBehaviorSqlComputeSpec struct {
	Method     string `json:"method"`
	CtxType    string `json:"ctxType"`
	ReturnType string `json:"returnType"`
}

type IrFieldBehaviorMethodRef struct {
	Method string `json:"method"`
}

type IrFieldBehaviorSpec struct {
	Compute    *IrFieldBehaviorComputeSpec    `json:"compute,omitempty"`
	SqlCompute *IrFieldBehaviorSqlComputeSpec `json:"sqlCompute,omitempty"`
	Inverse    *IrFieldBehaviorMethodRef      `json:"inverse,omitempty"`
	Search     *IrFieldBehaviorMethodRef      `json:"search,omitempty"`
}

type IrFieldMigrationDecision struct {
	StorageKind        string `json:"storageKind"`
	ShouldCreateColumn bool   `json:"shouldCreateColumn"`
	ResolvedColumnType string `json:"resolvedColumnType,omitempty"`
	ReasonCode         string `json:"reasonCode"`
}

type IrFieldDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type IrFieldResolvedSpec struct {
	FieldName  string                `json:"fieldName"`
	Structural IrFieldStructuralSpec `json:"structural"`
	Behavior   IrFieldBehaviorSpec   `json:"behavior"`
	Resolved   struct {
		Store      IrResolvedValue[bool]  `json:"store"`
		Searchable IrResolvedValue[*bool] `json:"searchable"`
	} `json:"resolved"`
	Migration   IrFieldMigrationDecision `json:"migration"`
	Diagnostics []IrFieldDiagnostic      `json:"diagnostics,omitempty"`
}

type IrField struct {
	BaseModel `gorm:"embedded"`

	Name             string `gorm:"type:varchar(255);not null;index:idx_model_field_name,unique" json:"name"`
	TsTypeAnnotation string `gorm:"type:varchar(1000);" json:"ts_type_annotation"`
	TsTypeReference  string `gorm:"type:varchar(1000);" json:"ts_type_reference"`

	OriginModelPath string `gorm:"type:varchar(512);" json:"origin_model_path"`

	FieldType      string `gorm:"type:varchar(255);" json:"field_type"`
	Relation       string `gorm:"type:varchar(255);" json:"relation"`
	RelationModel  string `gorm:"type:varchar(255);" json:"relation_model"`
	RelationFilter string `gorm:"type:varchar(255);" json:"relation_filter"`

	RelationInverseField     string `gorm:"type:varchar(255);" json:"relation_inverse_field"`      // Reverse foreign-key field name for OneToMany on the child model.
	RelationJoinModel        string `gorm:"type:varchar(255);" json:"relation_join_model"`         // Join model name for ManyToMany.
	RelationJoinField        string `gorm:"type:varchar(255);" json:"relation_join_field"`         // Join-table field pointing to the current model.
	RelationInverseJoinField string `gorm:"type:varchar(255);" json:"relation_inverse_join_field"` // Join-table field pointing to the target model.
	RelationModelParentField string `gorm:"type:varchar(255);" json:"relation_model_parent_field"`

	// Selection stores the option list for selection fields as a JSON string.
	// Storage format keeps the existing labelText wire name for database and API compatibility.
	// Its value is a TermReference; renaming the property requires a separate migration.
	// The parser writes it directly and the frontend consumes it without intermediate conversion.
	Selection string `gorm:"type:text" json:"selection,omitempty"`
	// SelectionKind is "static" or "dynamic" (P3). Dynamic fields omit inline Selection.
	SelectionKind string `gorm:"type:varchar(32)" json:"selection_kind,omitempty"`
	// SelectionMethod is the static model method name for dynamic selection when authored as a string.
	SelectionMethod string `gorm:"type:varchar(255)" json:"selection_method,omitempty"`

	// FieldString is the field title msgid (English fallback). JSON wire name remains "string".
	FieldString string `gorm:"type:varchar(512)" json:"string,omitempty"`
	// StringText stores the field title TermReference as JSON when authored with reference `_t(...)`.
	StringText string `gorm:"type:text" json:"string_text,omitempty"`

	ReferenceIdent string `gorm:"type:varchar" json:"reference_ident"`
	ModuleSpecPath string `gorm:"type:varchar" json:"module_spec_path"`

	AccessibilityModifier string `gorm:"type:varchar(255);" json:"accessibility_modifier"`
	IsStatic              bool   `json:"is_static"`
	IsReadonly            bool   `json:"is_readonly"`

	// ColumnOptions
	Indexed    bool    `json:"indexed"`
	NotNull    bool    `json:"not_null"`
	Size       int     `gorm:"type:int" json:"size"`
	Precision  int     `json:"precision,omitempty"`
	Scale      int     `json:"scale,omitempty"`
	ScaleField string  `gorm:"type:varchar(255);" json:"scale_field,omitempty"`
	Round      *string `json:"round,omitempty"`

	ResolvedSpec string `gorm:"type:text" json:"resolved_spec,omitempty"`

	ModelId sql.NullString `gorm:"type:char(20);index:idx_model_field_name,unique" json:"model_id"`
	Model   *IrModel       `gorm:"foreignKey:ModelId" json:"model"`

	Decorators []*IrDecorator `gorm:"foreignKey:FieldId;constraint:OnDelete:CASCADE;" json:"decorators"`
}

func (field *IrField) TableName() string {
	return "meta_ir_field"
}

func (field *IrField) SetResolvedSpec(spec *IrFieldResolvedSpec) error {
	if field == nil {
		return nil
	}
	if spec == nil {
		field.ResolvedSpec = ""
		return nil
	}
	b, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	field.ResolvedSpec = string(b)
	return nil
}

func (field *IrField) GetResolvedSpec() (*IrFieldResolvedSpec, error) {
	if field == nil {
		return nil, nil
	}
	raw := field.ResolvedSpec
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var spec IrFieldResolvedSpec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
