// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import "database/sql"

// RawArgument is a declaration-layer decorator argument.
type RawArgument struct {
	BaseModel `gorm:"embedded"`

	Type  string `gorm:"type:varchar" json:"ts_type"`
	Value string `gorm:"type:varchar" json:"value"`

	ReferenceIdent string `gorm:"type:varchar" json:"reference_ident"`
	ModuleSpecPath string `gorm:"type:varchar" json:"module_spec_path"`

	DecoratorId sql.NullString `gorm:"type:char(20)" json:"decorator_id"`
	Decorator   *RawDecorator  `gorm:"foreignKey:DecoratorId" json:"decorator"`
}

func (arg *RawArgument) TableName() string {
	return "meta_raw_argument"
}
