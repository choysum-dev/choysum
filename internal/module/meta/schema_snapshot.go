// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"time"

	"gorm.io/datatypes"
)

// SchemaSnapshot stores the last successfully applied DesiredSchema slice per model_table.
type SchemaSnapshot struct {
	ModelTable      string         `gorm:"column:model_table;type:varchar(255);primaryKey"`
	Application     string         `gorm:"column:application;type:varchar(255)"`
	ModelName       string         `gorm:"column:model_name;type:varchar(255)"`
	DesiredJSON     datatypes.JSON `gorm:"column:desired_json;type:json"`
	UpdatedByModule string         `gorm:"column:updated_by_module;type:varchar(255)"`
	ModuleVersion   string         `gorm:"column:module_version;type:varchar(64)"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (SchemaSnapshot) TableName() string {
	return "meta_schema_snapshot"
}
