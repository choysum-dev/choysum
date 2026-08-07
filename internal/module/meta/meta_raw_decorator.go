// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

// rawDecorator is a declaration-layer decorator under Raw pkgmeta.Model/pkgmeta.Field/pkgmeta.Service.
type rawDecorator struct {
	pkgmeta.BaseModel `gorm:"embedded"`

	Name string `gorm:"type:varchar(255);not null" json:"name"`

	ModuleSpecPath string `gorm:"type:varchar(255);" json:"module_spec_path"`
	ReferenceIdent string `gorm:"type:varchar(255);" json:"reference_ident"`

	Arguments []*rawArgument `gorm:"foreignKey:DecoratorId;constraint:OnDelete:CASCADE;" json:"arguments"`

	ModelId sql.NullString `gorm:"type:char(20)" json:"model_id"`
	Model   *rawModel      `gorm:"foreignKey:ModelId" json:"model"`

	ServiceId sql.NullString `gorm:"type:char(20)" json:"service_id"`
	Service   *rawService    `gorm:"foreignKey:ServiceId" json:"service"`

	FieldId sql.NullString `gorm:"type:char(20)" json:"field_id"`
	Field   *rawField      `gorm:"foreignKey:FieldId" json:"field"`
}

func (dec *rawDecorator) TableName() string {
	return "meta_raw_decorator"
}
