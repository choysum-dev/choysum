// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metadata

import (
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

type ModuleManagementLog struct {
	meta.BaseModel
	JobId            string         `gorm:"column:job_id;size:64;uniqueIndex"`
	ModuleName       string         `gorm:"column:module_name;size:255;index"`
	Action           string         `gorm:"column:action;size:32;index"`
	OperatorUserId   string         `gorm:"column:operator_user_id;size:64;index"`
	ResultStatus     string         `gorm:"column:result_status;size:32;index"`
	JobCreatedAt     *time.Time     `gorm:"column:job_created_at;index"`
	JobFinishedAt    *time.Time     `gorm:"column:job_finished_at;index"`
	ErrorDomain      string         `gorm:"column:error_domain;size:128"`
	ErrorCode        string         `gorm:"column:error_code;size:128"`
	SummaryJson      datatypes.JSON `gorm:"column:summary_json"`
	LastErrorJson    datatypes.JSON `gorm:"column:last_error_json"`
	ServerInstanceId string         `gorm:"column:server_instance_id;size:128"`
	Attempt          int            `gorm:"column:attempt"`
	MaxAttempts      int            `gorm:"column:max_attempts"`
}

func (ModuleManagementLog) TableName() string {
	return "meta_module_management_log"
}
