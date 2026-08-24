// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub

// Row is a minimal table used by stub writer integration tests.
type Row struct {
	ID        uint `gorm:"primaryKey"`
	UnitIndex int  `gorm:"column:unit_index;not null"`
}

// TableName implements schema.Tabler.
func (Row) TableName() string {
	return "import_stub_row"
}
