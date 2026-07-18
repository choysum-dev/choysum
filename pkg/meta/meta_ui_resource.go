// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"

	"gorm.io/datatypes"
)

type UiResourceType string

const (
	UiResourceTypeRoute  UiResourceType = "ROUTE"
	UiResourceTypeMenu   UiResourceType = "MENU"
	UiResourceTypeAction UiResourceType = "ACTION"
)

type IrUiResource struct {
	BaseModel          `gorm:"embedded"`
	Name               string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Type               UiResourceType `gorm:"type:varchar(16);not null" json:"type"`
	Title              string         `gorm:"type:varchar(255)" json:"title"`
	TitleText          datatypes.JSON `gorm:"column:title_text" json:"titleText,omitempty"`
	Sequence           int            `gorm:"default:0" json:"sequence"`
	Requires           datatypes.JSON `json:"requires"`
	Module             string         `gorm:"type:varchar(255);index" json:"module"`
	Path               string         `gorm:"type:varchar(1024);index" json:"path"`
	ParentResourceName string         `gorm:"-" json:"-"`
	ParentId           sql.NullString `gorm:"type:char(20);index" json:"parentId"`
	ParentPath         string         `gorm:"type:varchar(1000);index" json:"parentPath"`
	IrApplicationId    string         `gorm:"type:varchar(255);index" json:"irApplicationId"`
	UiPath             string         `gorm:"type:varchar(512)" json:"uiPath"`
	DefaultRoles       datatypes.JSON `gorm:"column:default_roles" json:"defaultRoles"`
	ModuleId           sql.NullString `gorm:"type:char(20);index" json:"module_id"`
}

func (r *IrUiResource) TableName() string {
	return "meta_ir_ui_resource"
}
