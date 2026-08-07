// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import "github.com/choysum-dev/choysum/pkg/meta"

// Setting stores key/value settings for stable system state.
type Setting struct {
	meta.BaseModel `gorm:"embedded"`

	Key   string `gorm:"type:varchar(255);not null;uniqueIndex"`
	Value string `gorm:"type:text;not null"`
}

func (s *Setting) TableName() string {
	return "meta_setting"
}
