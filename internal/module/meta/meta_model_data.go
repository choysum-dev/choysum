// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import pkgmeta "github.com/choysum-dev/choysum/pkg/meta"

type ModelData struct {
	pkgmeta.BaseModel `gorm:"embedded"`
	Application       string `gorm:"type:varchar(255);not null;index"`
	Module            string `gorm:"type:varchar(255);not null;index:idx_model_data_module_name,unique"`
	Name              string `gorm:"type:varchar(255);not null;index:idx_model_data_module_name,unique"`
	ModelId           string `gorm:"column:model_id;type:char(20);not null;index"`
	ModelName         string `gorm:"column:model_name;type:varchar(255);not null;index"`
	ResID             string `gorm:"type:char(20);not null;index"`
	NoUpdate          bool   `gorm:"not null;default:false"`
}

func (md *ModelData) TableName() string {
	return "meta_model_data"
}
