// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

// ModuleOperation is one Plan→Run record (install/upgrade/uninstall/sync/load-demo).
type ModuleOperation struct {
	pkgmeta.BaseModel `gorm:"embedded"`

	Action           string         `gorm:"column:action;type:varchar(32);not null;index"`
	ModuleName       string         `gorm:"column:module_name;type:varchar(255);index"`
	PlanJson         datatypes.JSON `gorm:"column:plan_json"`
	Status           string         `gorm:"column:status;type:varchar(32);not null;index"`
	Phase            string         `gorm:"column:phase;type:varchar(32);index"`
	ErrorDomain      string         `gorm:"column:error_domain;type:varchar(128)"`
	ErrorCode        string         `gorm:"column:error_code;type:varchar(128)"`
	ErrorMessage     string         `gorm:"column:error_message;type:text"`
	ReloadTriggered  bool           `gorm:"column:reload_triggered;not null;default:false"`
	ReloadFailed     bool           `gorm:"column:reload_failed;not null;default:false"`
	OperatorUserId   string         `gorm:"column:operator_user_id;type:varchar(64);index"`
	JobId            sql.NullString `gorm:"column:job_id;type:varchar(64);uniqueIndex"`
	StartedAt        *time.Time     `gorm:"column:started_at;index"`
	FinishedAt       *time.Time     `gorm:"column:finished_at;index"`
}

func (ModuleOperation) TableName() string {
	return "meta_module_operation"
}
