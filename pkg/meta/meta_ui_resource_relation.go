// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import "database/sql"

type IrUiResourceMenuRoute struct {
	BaseModel         `gorm:"embedded"`
	MenuUiResourceId  sql.NullString `gorm:"type:char(20);not null;uniqueIndex:idx_ui_resource_menu_route_pair;index" json:"menuUiResourceId"`
	RouteUiResourceId sql.NullString `gorm:"type:char(20);not null;uniqueIndex:idx_ui_resource_menu_route_pair;uniqueIndex;index" json:"routeUiResourceId"`
}

func (r *IrUiResourceMenuRoute) TableName() string {
	return "meta_ir_ui_resource_menu_route"
}

type IrUiResourceRouteAction struct {
	BaseModel          `gorm:"embedded"`
	RouteUiResourceId  sql.NullString `gorm:"type:char(20);not null;uniqueIndex:idx_ui_resource_route_action_pair;index" json:"routeUiResourceId"`
	ActionUiResourceId sql.NullString `gorm:"type:char(20);not null;uniqueIndex:idx_ui_resource_route_action_pair;index" json:"actionUiResourceId"`
}

func (r *IrUiResourceRouteAction) TableName() string {
	return "meta_ir_ui_resource_route_action"
}
