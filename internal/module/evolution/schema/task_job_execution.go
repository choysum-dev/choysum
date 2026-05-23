// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/datatypes"
)

type taskJobExecution struct {
	JobId             string         `gorm:"column:job_id;type:varchar(64);uniqueIndex;not null"`
	Status            string         `gorm:"column:status;type:varchar(32);index"`
	LeaseOwner        string         `gorm:"column:lease_owner;type:varchar(128);index"`
	LeaseUntil        *time.Time     `gorm:"column:lease_until;index"`
	Attempt           int            `gorm:"column:attempt"`
	SchedulerUserId   string         `gorm:"column:scheduler_user_id;type:varchar(20);index"`
	TriggeredByUserId string         `gorm:"column:triggered_by_user_id;type:varchar(20);index"`
	FullMethod        string         `gorm:"column:full_method;type:varchar(255);index"`
	PayloadJson       datatypes.JSON `gorm:"column:payload_json;type:json"`
	ResultJson        datatypes.JSON `gorm:"column:result_json;type:json"`
	ResultHash        string         `gorm:"column:result_hash;type:varchar(128)"`
	ResultTruncated   bool           `gorm:"column:result_truncated"`
	ErrorJson         datatypes.JSON `gorm:"column:error_json;type:json"`
	ErrorHash         string         `gorm:"column:error_hash;type:varchar(128)"`
	ErrorTruncated    bool           `gorm:"column:error_truncated"`
	CancelledAt       *time.Time     `gorm:"column:cancelled_at;index"`
	StartedAt         *time.Time     `gorm:"column:started_at;index"`
	FinishedAt        *time.Time     `gorm:"column:finished_at;index"`
	CreatedAt         time.Time      `gorm:"column:created_at;index"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;index"`
}

func (taskJobExecution) TableName() string {
	return "task_job_execution"
}

func ensureTaskJobExecutionTable(runtimeScope scope.Scope) error {
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	db := runtimeScope.Session()
	return db.AutoMigrate(&taskJobExecution{})
}
