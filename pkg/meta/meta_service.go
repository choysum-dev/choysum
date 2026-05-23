// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

type IrService struct {
	BaseModel `gorm:"embedded"`
	Name      string `gorm:"type:varchar(255);not null;index:idx_model_service_name,unique" json:"name"`

	OriginModelPath string `gorm:"type:varchar(512);" json:"origin_model_path"`

	AccessibilityModifier string `gorm:"type:varchar(255);" json:"accessibility_modifier"`
	TsTypeAnnotation      string `gorm:"type:varchar(255);" json:"ts_type_annotation"` // return type
	ProtobufType          string `gorm:"type:varchar(255);" json:"protobuf_type"`
	IsStatic              bool   `json:"is_static"`

	TypeParameters []*IrTypeParameter `gorm:"foreignKey:ServiceId;constraint:OnDelete:CASCADE;" json:"type_parameters"`
	Parameters     []*IrParameter     `gorm:"foreignKey:ServiceId;constraint:OnDelete:CASCADE;" json:"parameters"`
	ModelId        sql.NullString     `gorm:"type:char(20);index:idx_model_service_name,unique" json:"model_id"`
	Model          *IrModel           `gorm:"foreignKey:ModelId" json:"model"`

	Decorators []*IrDecorator `gorm:"foreignKey:ServiceId;constraint:OnDelete:CASCADE;" json:"decorators"`
}

func (svc *IrService) TableName() string {
	return "meta_ir_service"
}
