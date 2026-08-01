// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

type Component struct {
	BaseModel  `gorm:"embedded"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	Path       string         `gorm:"type:varchar(512);not null" json:"path"`
	RawExtends string         `gorm:"type:varchar(255);" json:"raw_extends"`
	Extends    string         `gorm:"type:varchar(255);" json:"extends"`
	ModuleId   sql.NullString `gorm:"type:char(20)" json:"module_id"`
}

func (comp *Component) TableName() string {
	return "meta_component"
}
