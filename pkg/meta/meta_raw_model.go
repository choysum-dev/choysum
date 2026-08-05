// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

// rawModel is a per-module/file declaration row (IMD may have many rows per logical name).
// Effective projection lives in Model / meta_model (see MergeSameNameModelsByExtensionChain).
type rawModel struct {
	BaseModel `gorm:"embedded"`

	Name        string `gorm:"type:varchar(255);not null;index:idx_raw_model_app_name" json:"name"`
	Path        string `gorm:"type:varchar(512);not null;uniqueIndex:uidx_raw_model_module_path" json:"path"`
	Application string `gorm:"type:varchar(255);index:idx_raw_model_app_name" json:"application"`

	ClassName   string `gorm:"type:varchar(255);" json:"class_name"`
	ModelTable  string `gorm:"type:varchar(255);" json:"model_table"`
	Abstract    bool   `gorm:"type:boolean;" json:"abstract"`
	AutoMigrate *bool  `gorm:"type:boolean;default:true" json:"auto_migrate"`
	Readonly    bool   `gorm:"type:boolean;default:false" json:"readonly"`

	RawExtends string `gorm:"type:varchar(255);" json:"raw_extends"`
	Extends    string `gorm:"type:varchar(255);" json:"extends"`

	CompanyField *string `gorm:"type:varchar(255);" json:"company_field"`

	Decorators []*rawDecorator `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"decorators"`
	Services   []*rawService   `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"services"`
	Fields     []*rawField     `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"fields"`
	ModuleId   sql.NullString  `gorm:"type:char(20);uniqueIndex:uidx_raw_model_module_path" json:"module_id"`
	Module     *Module         `gorm:"foreignKey:ModuleId" json:"module"`
}

func (model *rawModel) TableName() string {
	return "meta_raw_model"
}
