// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"time"

	"gorm.io/datatypes"
)

type Job struct {
	Id                 string         `gorm:"column:id;primaryKey"`
	TargetApp          string         `gorm:"column:target_app"`
	FullMethod         string         `gorm:"column:full_method"`
	PayloadJson        datatypes.JSON `gorm:"column:payload_json"`
	SchedulerUserId    string         `gorm:"column:scheduler_user_id"`
	TriggeredByUserId  string         `gorm:"column:triggered_by_user_id"`
	Status             string         `gorm:"column:status"`
	RunAfter           time.Time      `gorm:"column:run_after"`
	Attempt            int            `gorm:"column:attempt"`
	MaxAttempts        int            `gorm:"column:max_attempts"`
	TimeoutMs          int64          `gorm:"column:timeout_ms"`
	CancelRequestedAt  *time.Time     `gorm:"column:cancel_requested_at"`
	CancelledAt        *time.Time     `gorm:"column:cancelled_at"`
	FinishedAt         *time.Time     `gorm:"column:finished_at"`
	LastErrorJson      datatypes.JSON `gorm:"column:last_error_json"`
	LastErrorHash      string         `gorm:"column:last_error_hash"`
	LastErrorTruncated bool           `gorm:"column:last_error_truncated"`
	ResultJson         datatypes.JSON `gorm:"column:result_json"`
	ResultHash         string         `gorm:"column:result_hash"`
	ResultTruncated    bool           `gorm:"column:result_truncated"`
	CreatedAt          time.Time      `gorm:"column:created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at"`
}

func (Job) TableName() string {
	return "task_job"
}

type Execution struct {
	JobId             string         `gorm:"column:job_id;primaryKey"`
	Status            string         `gorm:"column:status"`
	LeaseOwner        string         `gorm:"column:lease_owner"`
	LeaseUntil        *time.Time     `gorm:"column:lease_until"`
	Attempt           int            `gorm:"column:attempt"`
	SchedulerUserId   string         `gorm:"column:scheduler_user_id"`
	TriggeredByUserId string         `gorm:"column:triggered_by_user_id"`
	FullMethod        string         `gorm:"column:full_method"`
	PayloadJson       datatypes.JSON `gorm:"column:payload_json"`
	ResultJson        datatypes.JSON `gorm:"column:result_json"`
	ResultHash        string         `gorm:"column:result_hash"`
	ResultTruncated   bool           `gorm:"column:result_truncated"`
	ErrorJson         datatypes.JSON `gorm:"column:error_json"`
	ErrorHash         string         `gorm:"column:error_hash"`
	ErrorTruncated    bool           `gorm:"column:error_truncated"`
	CancelledAt       *time.Time     `gorm:"column:cancelled_at"`
	StartedAt         *time.Time     `gorm:"column:started_at"`
	FinishedAt        *time.Time     `gorm:"column:finished_at"`
	CreatedAt         time.Time      `gorm:"column:created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at"`
}

func (Execution) TableName() string {
	return "task_job_execution"
}

type Schedule struct {
	Id                string         `gorm:"column:id;primaryKey"`
	Active            bool           `gorm:"column:active"`
	Name              string         `gorm:"column:name"`
	TargetApp         string         `gorm:"column:target_app"`
	FullMethod        string         `gorm:"column:full_method"`
	PayloadTemplate   datatypes.JSON `gorm:"column:payload_template_json"`
	SchedulerUserId   string         `gorm:"column:scheduler_user_id"`
	TriggeredByUserId string         `gorm:"column:triggered_by_user_id"`
	CronExpr          string         `gorm:"column:cron_expr"`
	Timezone          string         `gorm:"column:timezone"`
	TimeoutMs         int64          `gorm:"column:timeout_ms"`
	NextRunAt         *time.Time     `gorm:"column:next_run_at"`
	LastRunAt         *time.Time     `gorm:"column:last_run_at"`
	LastTriggeredAt   *time.Time     `gorm:"column:last_triggered_at"`
	CreatedAt         time.Time      `gorm:"column:created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at"`
}

func (Schedule) TableName() string {
	return "task_schedule"
}
