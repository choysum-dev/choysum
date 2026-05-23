// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

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
	// Storage format: [{"value":"draft","label":"Draft"},{"value":"confirmed","label":"Confirmed"}]
	// The parser writes it directly and the frontend consumes it without intermediate conversion.
	Selection string `gorm:"type:text" json:"selection,omitempty"`

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

	ModelId sql.NullString `gorm:"type:char(20);index:idx_model_field_name,unique" json:"model_id"`
	Model   *IrModel       `gorm:"foreignKey:ModelId" json:"model"`

	Decorators []*IrDecorator `gorm:"foreignKey:FieldId;constraint:OnDelete:CASCADE;" json:"decorators"`
}

func (field *IrField) TableName() string {
	return "meta_ir_field"
}
