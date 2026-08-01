// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

type Application struct {
	BaseModel `gorm:"embedded"`

	Name     string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Revision int64

	ProtoDir  string `gorm:"type:varchar(512);"`
	ProtoFile string `gorm:"type:varchar(512);"`

	Modules []*Module `gorm:"foreignKey:ApplicationId;constraint:OnDelete:CASCADE;"`
	Models  []*Model  `gorm:"-"`
}

func (app *Application) TableName() string {
	return "meta_application"
}
