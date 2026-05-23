// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

type IrApplication struct {
	BaseModel `gorm:"embedded"`

	Name     string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Revision int64

	ProtoDir  string `gorm:"type:varchar(512);"`
	ProtoFile string `gorm:"type:varchar(512);"`

	Modules []*IrModule `gorm:"foreignKey:ApplicationId;constraint:OnDelete:CASCADE;"`
	Models  []*IrModel  `gorm:"-"`
}

func (app *IrApplication) TableName() string {
	return "meta_ir_application"
}
