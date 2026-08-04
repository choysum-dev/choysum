// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import "database/sql"

// RawDecorator is a declaration-layer decorator under Raw Model/Field/Service.
type RawDecorator struct {
	BaseModel `gorm:"embedded"`

	Name string `gorm:"type:varchar(255);not null" json:"name"`

	ModuleSpecPath string `gorm:"type:varchar(255);" json:"module_spec_path"`
	ReferenceIdent string `gorm:"type:varchar(255);" json:"reference_ident"`

	Arguments []*RawArgument `gorm:"foreignKey:DecoratorId;constraint:OnDelete:CASCADE;" json:"arguments"`

	ModelId sql.NullString `gorm:"type:char(20)" json:"model_id"`
	Model   *RawModel      `gorm:"foreignKey:ModelId" json:"model"`

	ServiceId sql.NullString `gorm:"type:char(20)" json:"service_id"`
	Service   *RawService    `gorm:"foreignKey:ServiceId" json:"service"`

	FieldId sql.NullString `gorm:"type:char(20)" json:"field_id"`
	Field   *RawField      `gorm:"foreignKey:FieldId" json:"field"`
}

func (dec *RawDecorator) TableName() string {
	return "meta_raw_decorator"
}
