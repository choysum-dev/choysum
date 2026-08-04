// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
)

// Model is the effective (E2) logical-model projection stored in meta_model.
// After dual-store (EDS), each live (application, name) has at most one row;
// per-module/file declarations live in RawModel / meta_raw_model.
// UNIQUE(application, name) on live rows is applied by dual-store migrate/backfill
// (not via a GORM tag yet — IMD multi-row persists until EDS-2 rewires Persist).
type Model struct {
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

	Decorators []*Decorator `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"decorators"`
	Services   []*Service   `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"services"`
	Fields     []*Field     `gorm:"foreignKey:ModelId;constraint:OnDelete:CASCADE;" json:"fields"`
	// ModuleId is legacy for IMD declaration rows. Effective projections leave it empty
	// after dual-store recompute (EDS-2+); prefer RawModel.ModuleId for module ownership.
	ModuleId sql.NullString `gorm:"type:char(20)" json:"module_id"`
	Module   *Module        `gorm:"foreignKey:ModuleId" json:"module"`
}

func (model *Model) TableName() string {
	return "meta_model"
}
