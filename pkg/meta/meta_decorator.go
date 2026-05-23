// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import "database/sql"

type IrDecorator struct {
	BaseModel `gorm:"embedded"`

	Name string `gorm:"type:varchar(255);not null" json:"name"`

	ModuleSpecPath string `gorm:"type:varchar(255);" json:"module_spec_path"`
	ReferenceIdent string `gorm:"type:varchar(255);" json:"reference_ident"`

	Arguments []*IrArgument `gorm:"foreignKey:DecoratorId;constraint:OnDelete:CASCADE;" json:"arguments"`

	ModelId sql.NullString `gorm:"type:char(20)" json:"model_id"`
	Model   *IrModel       `gorm:"foreignKey:ModelId" json:"model"`

	ServiceId sql.NullString `gorm:"type:char(20)" json:"service_id"`
	Service   *IrService     `gorm:"foreignKey:ServiceId" json:"service"`

	FieldId sql.NullString `gorm:"type:char(20)" json:"field_id"`
	Field   *IrField       `gorm:"foreignKey:FieldId" json:"field"`

	ComponentId sql.NullString `gorm:"type:char(20)" json:"component_id"`
	Component   *IrComponent   `gorm:"foreignKey:ComponentId" json:"component"`
}

func (dec *IrDecorator) TableName() string {
	return "meta_ir_decorator"
}
