// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metadata

import "github.com/choysum-dev/choysum/pkg/meta"

// IrSetting stores key/value settings for stable system state.
type IrSetting struct {
	meta.BaseModel `gorm:"embedded"`

	Key   string `gorm:"type:varchar(255);not null;uniqueIndex"`
	Value string `gorm:"type:text;not null"`
}

func (s *IrSetting) TableName() string {
	return "meta_ir_setting"
}
