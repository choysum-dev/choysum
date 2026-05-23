// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metadata

import "github.com/choysum-dev/choysum/pkg/meta"

type IrModelData struct {
	meta.BaseModel `gorm:"embedded"`

	Module     string `gorm:"type:varchar(255);not null;index:idx_model_data_module_external_id,unique"`
	ExternalID string `gorm:"type:varchar(255);not null;index:idx_model_data_module_external_id,unique"`
	Model      string `gorm:"type:varchar(255);not null;index"`
	ResID      string `gorm:"type:char(20);not null;index"`
	NoUpdate   bool   `gorm:"not null;default:false"`
}

func (md *IrModelData) TableName() string {
	return "meta_ir_model_data"
}
