// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import "database/sql"

type Decorator struct {
	BaseModel `gorm:"embedded"`

	Name string `gorm:"type:varchar(255);not null" json:"name"`

	ModuleSpecPath string `gorm:"type:varchar(255);" json:"module_spec_path"`
	ReferenceIdent string `gorm:"type:varchar(255);" json:"reference_ident"`

	Arguments []*Argument `gorm:"foreignKey:DecoratorId;constraint:OnDelete:CASCADE;" json:"arguments"`

	ModelId sql.NullString `gorm:"type:char(20)" json:"model_id"`
	Model   *Model         `gorm:"foreignKey:ModelId" json:"model"`

	ServiceId sql.NullString `gorm:"type:char(20)" json:"service_id"`
	Service   *Service       `gorm:"foreignKey:ServiceId" json:"service"`

	FieldId sql.NullString `gorm:"type:char(20)" json:"field_id"`
	Field   *Field         `gorm:"foreignKey:FieldId" json:"field"`

	ComponentId sql.NullString `gorm:"type:char(20)" json:"component_id"`
	Component   *Component     `gorm:"foreignKey:ComponentId" json:"component"`
}

func (dec *Decorator) TableName() string {
	return "meta_decorator"
}
