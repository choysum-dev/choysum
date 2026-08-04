// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

// RawTypeParameter is a declaration-layer type parameter under RawService.
type RawTypeParameter struct {
	BaseModel      `gorm:"embedded"`
	Name           string `gorm:"type:varchar(255);not null" json:"name"`
	ModuleSpecPath string `gorm:"type:varchar(255);" json:"module_spec_path"`
	ReferenceIdent string `gorm:"type:varchar(255);" json:"reference_ident"`

	ServiceId sql.NullString `gorm:"type:char(20)" json:"service_id"`
	Service   *RawService    `gorm:"foreignKey:ServiceId" json:"service"`
}

func (tp *RawTypeParameter) TableName() string {
	return "meta_raw_type_parameter"
}
