// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

type IrModel struct {
	BaseModel `gorm:"embedded"`

	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Path        string `gorm:"type:varchar(512);not null" json:"path"`
	Application string `gorm:"type:varchar(255);" json:"application"`

	ClassName   string `gorm:"type:varchar(255);" json:"class_name"`
	ModelTable  string `gorm:"type:varchar(255);" json:"model_table"`
	Abstract    bool   `gorm:"type:boolean;" json:"abstract"`
	AutoMigrate *bool  `gorm:"type:boolean;default:true" json:"auto_migrate"`
	Readonly    bool   `gorm:"type:boolean;default:false" json:"readonly"`

	RawExtends string `gorm:"type:varchar(255);" json:"raw_extends"`
	Extends    string `gorm:"type:varchar(255);" json:"extends"`

	CompanyField *string `gorm:"type:varchar(255);" json:"company_field"`

	Decorators []*IrDecorator `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"decorators"`
	Services   []*IrService   `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"services"`
	Fields     []*IrField     `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"fields"`
	ModuleId   sql.NullString `gorm:"type:char(20)" json:"module_id"`
	Module     *IrModule      `gorm:"foreignKey:ModuleId" json:"module"`
}

func (model *IrModel) TableName() string {
	return "meta_ir_model"
}
