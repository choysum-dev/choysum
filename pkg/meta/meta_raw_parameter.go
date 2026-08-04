// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

// RawParameter is a declaration-layer parameter under RawService.
type RawParameter struct {
	BaseModel        `gorm:"embedded"`
	Name             string `gorm:"type:varchar(255);not null" json:"name"`
	TsTypeAnnotation string `gorm:"type:varchar(255);" json:"ts_type_annotation"`
	ProtobufType     string `gorm:"type:varchar(255);" json:"protobuf_type"`

	ServiceId sql.NullString `gorm:"type:char(20)" json:"service_id"`
	Service   *RawService    `gorm:"foreignKey:ServiceId" json:"service"`
}

func (param *RawParameter) TableName() string {
	return "meta_raw_parameter"
}
