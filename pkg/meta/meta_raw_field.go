// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"encoding/json"
	"strings"
)

// RawField is a declaration-layer field row under RawModel (table meta_raw_field).
type RawField struct {
	BaseModel `gorm:"embedded"`

	Name             string `gorm:"type:varchar(255);not null;index:idx_raw_model_field_name,unique" json:"name"`
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
	// FieldHelp is the field help msgid (English fallback). JSON wire name remains "help".
	FieldHelp string `gorm:"type:varchar(1024)" json:"help,omitempty"`
	// HelpText stores the field help TermReference as JSON when authored with reference `_t(...)`.
	HelpText string `gorm:"type:text" json:"help_text,omitempty"`

	ReferenceIdent string `gorm:"type:varchar" json:"reference_ident"`
	ModuleSpecPath string `gorm:"type:varchar" json:"module_spec_path"`

	AccessibilityModifier string `gorm:"type:varchar(255);" json:"accessibility_modifier"`
	IsStatic              bool   `json:"is_static"`
	IsReadonly            bool   `json:"is_readonly"`

	MaxUploadBytes int `gorm:"type:int" json:"maxUploadBytes,omitempty"`
	MaxWidth       int `gorm:"type:int" json:"maxWidth,omitempty"`
	MaxHeight      int `gorm:"type:int" json:"maxHeight,omitempty"`

	// ColumnOptions
	Indexed       bool    `json:"indexed"`
	NotNull       bool    `json:"not_null"`
	Size          int     `gorm:"type:int" json:"size"`
	Precision     int     `json:"precision,omitempty"`
	Scale         int     `json:"scale,omitempty"`
	ScaleField    string  `gorm:"type:varchar(255);" json:"scale_field,omitempty"`
	CurrencyField string  `gorm:"type:varchar(255);" json:"currency_field,omitempty"`
	Round         *string `json:"round,omitempty"`

	ResolvedSpec string `gorm:"type:text" json:"resolved_spec,omitempty"`

	ModelId sql.NullString `gorm:"type:char(20);index:idx_raw_model_field_name,unique" json:"model_id"`
	Model   *RawModel      `gorm:"foreignKey:ModelId" json:"model"`

	Decorators []*RawDecorator `gorm:"foreignKey:FieldId;constraint:OnDelete:CASCADE;" json:"decorators"`
}

func (field *RawField) TableName() string {
	return "meta_raw_field"
}

func (field *RawField) SetResolvedSpec(spec *FieldResolvedSpec) error {
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

func (field *RawField) GetResolvedSpec() (*FieldResolvedSpec, error) {
	if field == nil {
		return nil, nil
	}
	rawSpec := field.ResolvedSpec
	if strings.TrimSpace(rawSpec) == "" {
		return nil, nil
	}
	var spec FieldResolvedSpec
	if err := json.Unmarshal([]byte(rawSpec), &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
