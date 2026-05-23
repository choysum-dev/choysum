// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var errTaskStoreUnavailable = errors.New("task store is unavailable")

var _ taskcontract.TaskQueue = (*dbTaskQueue)(nil)
var _ taskcontract.ScheduleStore = (*dbScheduleStore)(nil)

type dbTaskQueue struct {
	runtimeScope scope.Scope
}

type dbScheduleStore struct {
	runtimeScope scope.Scope
}

func newTaskRuntimeQueue(runtimeScope scope.Scope) taskcontract.TaskQueue {
	return &dbTaskQueue{runtimeScope: runtimeScope}
}

func newTaskRuntimeScheduleStore(runtimeScope scope.Scope) taskcontract.ScheduleStore {
	return &dbScheduleStore{runtimeScope: runtimeScope}
}

func taskDB(runtimeScope scope.Scope, ctx context.Context) (*gorm.DB, error) {
	if db, ok := scope.DBForScope(ctx, runtimeScope); ok {
		return db, nil
	}
	if runtimeScope == nil || runtimeScope.Session() == nil || runtimeScope.Session().DB == nil {
		return nil, errTaskStoreUnavailable
	}
	db := runtimeScope.Session().DB
	if ctx != nil {
		return db.WithContext(ctx), nil
	}
	return db, nil
}

func (q *dbTaskQueue) Enqueue(ctx context.Context, job taskcontract.QueueJob) error {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Create(queueJobToModel(job)).Error
}

func (q *dbTaskQueue) ListReady(ctx context.Context, now time.Time, limit int) ([]taskcontract.QueueJob, error) {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	where := "status = ? AND run_after <= ?"
	if isSQLiteConfig(q.runtimeScope) {
		where = "status = ? AND datetime(run_after) <= datetime(?)"
	}
	query := db.Where(where, taskcontract.JobStatusQueued, now).Order("run_after asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}
	result := make([]taskcontract.QueueJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, queueJobFromModel(job))
	}
	return result, nil
}

func (q *dbTaskQueue) TryClaim(ctx context.Context, jobID string, now time.Time) (bool, error) {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return false, err
	}
	res := db.Model(&Job{}).
		Where("id = ? AND status = ?", jobID, taskcontract.JobStatusQueued).
		Updates(map[string]any{"status": taskcontract.JobStatusDispatching, "updated_at": now})
	return res.RowsAffected > 0, res.Error
}

func (q *dbTaskQueue) Get(ctx context.Context, jobID string) (*taskcontract.QueueJob, error) {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return nil, err
	}
	var job Job
	if err := db.Where("id = ?", jobID).First(&job).Error; err != nil {
		return nil, err
	}
	contractJob := queueJobFromModel(job)
	return &contractJob, nil
}

func (q *dbTaskQueue) UpdateAttempt(ctx context.Context, jobID string, attempt int, updatedAt time.Time) error {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Job{}).Where("id = ?", jobID).
		Updates(map[string]any{"attempt": attempt, "updated_at": updatedAt}).Error
}

func (q *dbTaskQueue) MarkSucceeded(ctx context.Context, jobID string, update taskcontract.QueueSuccess) error {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Job{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":           taskcontract.JobStatusSucceeded,
		"finished_at":      update.FinishedAt,
		"updated_at":       update.UpdatedAt,
		"result_json":      datatypes.JSON(update.ResultJSON),
		"result_hash":      update.ResultHash,
		"result_truncated": update.ResultTruncated,
	}).Error
}

func (q *dbTaskQueue) MarkFailed(ctx context.Context, jobID string, update taskcontract.QueueFailure) error {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Job{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":               taskcontract.JobStatusFailed,
		"finished_at":          update.FinishedAt,
		"updated_at":           update.UpdatedAt,
		"last_error_json":      datatypes.JSON(update.ErrorJSON),
		"last_error_hash":      update.ErrorHash,
		"last_error_truncated": update.ErrorTruncated,
		"result_json":          datatypes.JSON(update.ResultJSON),
		"result_hash":          update.ResultHash,
		"result_truncated":     update.ResultTruncated,
	}).Error
}

func (q *dbTaskQueue) Retry(ctx context.Context, jobID string, update taskcontract.QueueRetry) error {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Job{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":               taskcontract.JobStatusQueued,
		"run_after":            update.RunAfter,
		"updated_at":           update.UpdatedAt,
		"last_error_json":      datatypes.JSON(update.ErrorJSON),
		"last_error_hash":      update.ErrorHash,
		"last_error_truncated": update.ErrorTruncated,
	}).Error
}

func (q *dbTaskQueue) MarkCancelled(ctx context.Context, jobID string, update taskcontract.QueueCancellation) error {
	db, err := taskDB(q.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Job{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":          taskcontract.JobStatusCancelled,
		"cancelled_at":    update.CancelledAt,
		"finished_at":     update.FinishedAt,
		"updated_at":      update.UpdatedAt,
		"last_error_json": datatypes.JSON(update.ErrorJSON),
	}).Error
}

func (s *dbScheduleStore) ListDue(ctx context.Context, now time.Time, limit int) ([]taskcontract.ScheduleEntry, error) {
	db, err := taskDB(s.runtimeScope, ctx)
	if err != nil {
		return nil, err
	}
	var schedules []Schedule
	where := "active = ? AND (next_run_at IS NULL OR next_run_at <= ?)"
	if isSQLiteConfig(s.runtimeScope) {
		where = "active = ? AND (next_run_at IS NULL OR datetime(next_run_at) <= datetime(?))"
	}
	query := db.Where(where, true, now).Order("next_run_at asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&schedules).Error; err != nil {
		return nil, err
	}
	result := make([]taskcontract.ScheduleEntry, 0, len(schedules))
	for _, schedule := range schedules {
		result = append(result, scheduleEntryFromModel(schedule))
	}
	return result, nil
}

func (s *dbScheduleStore) TryAdvanceDue(ctx context.Context, scheduleID string, dueAt time.Time, nextRunAt time.Time, updatedAt time.Time) (bool, error) {
	db, err := taskDB(s.runtimeScope, ctx)
	if err != nil {
		return false, err
	}
	where := "id = ? AND active = ? AND (next_run_at IS NULL OR next_run_at <= ?)"
	if isSQLiteConfig(s.runtimeScope) {
		where = "id = ? AND active = ? AND (next_run_at IS NULL OR datetime(next_run_at) <= datetime(?))"
	}
	res := db.Model(&Schedule{}).
		Where(where, scheduleID, true, dueAt).
		Updates(map[string]any{"next_run_at": nextRunAt, "last_run_at": dueAt, "updated_at": updatedAt})
	return res.RowsAffected > 0, res.Error
}

func (s *dbScheduleStore) UpdateNextRun(ctx context.Context, scheduleID string, nextRunAt time.Time, updatedAt time.Time) error {
	db, err := taskDB(s.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Schedule{}).Where("id = ?", scheduleID).
		Updates(map[string]any{"next_run_at": nextRunAt, "updated_at": updatedAt}).Error
}

func (s *dbScheduleStore) MarkTriggered(ctx context.Context, scheduleID string, triggeredAt time.Time, updatedAt time.Time) error {
	db, err := taskDB(s.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Schedule{}).Where("id = ?", scheduleID).
		Updates(map[string]any{"last_triggered_at": triggeredAt, "updated_at": updatedAt}).Error
}

func (s *dbScheduleStore) Disable(ctx context.Context, scheduleID string, updatedAt time.Time) error {
	db, err := taskDB(s.runtimeScope, ctx)
	if err != nil {
		return err
	}
	return db.Model(&Schedule{}).Where("id = ?", scheduleID).
		Updates(map[string]any{"active": false, "updated_at": updatedAt}).Error
}

func queueJobToModel(job taskcontract.QueueJob) *Job {
	return &Job{
		Id:                 job.ID,
		TargetApp:          job.TargetApp,
		FullMethod:         job.FullMethod,
		PayloadJson:        datatypes.JSON(job.PayloadJSON),
		SchedulerUserId:    job.SchedulerUserID,
		TriggeredByUserId:  job.TriggeredByUserID,
		Status:             job.Status,
		RunAfter:           job.RunAfter,
		Attempt:            job.Attempt,
		MaxAttempts:        job.MaxAttempts,
		TimeoutMs:          job.TimeoutMs,
		CancelRequestedAt:  job.CancelRequestedAt,
		CancelledAt:        job.CancelledAt,
		FinishedAt:         job.FinishedAt,
		LastErrorJson:      datatypes.JSON(job.LastErrorJSON),
		LastErrorHash:      job.LastErrorHash,
		LastErrorTruncated: job.LastErrorTruncated,
		ResultJson:         datatypes.JSON(job.ResultJSON),
		ResultHash:         job.ResultHash,
		ResultTruncated:    job.ResultTruncated,
		CreatedAt:          job.CreatedAt,
		UpdatedAt:          job.UpdatedAt,
	}
}

func queueJobFromModel(job Job) taskcontract.QueueJob {
	return taskcontract.QueueJob{
		ID:                 job.Id,
		TargetApp:          job.TargetApp,
		FullMethod:         job.FullMethod,
		PayloadJSON:        cloneJSON(job.PayloadJson),
		SchedulerUserID:    job.SchedulerUserId,
		TriggeredByUserID:  job.TriggeredByUserId,
		Status:             job.Status,
		RunAfter:           job.RunAfter,
		Attempt:            job.Attempt,
		MaxAttempts:        job.MaxAttempts,
		TimeoutMs:          job.TimeoutMs,
		CancelRequestedAt:  job.CancelRequestedAt,
		CancelledAt:        job.CancelledAt,
		FinishedAt:         job.FinishedAt,
		LastErrorJSON:      cloneJSON(job.LastErrorJson),
		LastErrorHash:      job.LastErrorHash,
		LastErrorTruncated: job.LastErrorTruncated,
		ResultJSON:         cloneJSON(job.ResultJson),
		ResultHash:         job.ResultHash,
		ResultTruncated:    job.ResultTruncated,
		CreatedAt:          job.CreatedAt,
		UpdatedAt:          job.UpdatedAt,
	}
}

func scheduleEntryFromModel(schedule Schedule) taskcontract.ScheduleEntry {
	return taskcontract.ScheduleEntry{
		ID:                  schedule.Id,
		Active:              schedule.Active,
		Name:                schedule.Name,
		TargetApp:           schedule.TargetApp,
		FullMethod:          schedule.FullMethod,
		PayloadTemplateJSON: cloneJSON(schedule.PayloadTemplate),
		SchedulerUserID:     schedule.SchedulerUserId,
		TriggeredByUserID:   schedule.TriggeredByUserId,
		CronExpr:            schedule.CronExpr,
		Timezone:            schedule.Timezone,
		TimeoutMs:           schedule.TimeoutMs,
		NextRunAt:           schedule.NextRunAt,
		LastRunAt:           schedule.LastRunAt,
		LastTriggeredAt:     schedule.LastTriggeredAt,
		CreatedAt:           schedule.CreatedAt,
		UpdatedAt:           schedule.UpdatedAt,
	}
}

func scheduleEntryToModel(entry taskcontract.ScheduleEntry) *Schedule {
	return &Schedule{
		Id:                entry.ID,
		Active:            entry.Active,
		Name:              entry.Name,
		TargetApp:         entry.TargetApp,
		FullMethod:        entry.FullMethod,
		PayloadTemplate:   datatypes.JSON(entry.PayloadTemplateJSON),
		SchedulerUserId:   entry.SchedulerUserID,
		TriggeredByUserId: entry.TriggeredByUserID,
		CronExpr:          entry.CronExpr,
		Timezone:          entry.Timezone,
		TimeoutMs:         entry.TimeoutMs,
		NextRunAt:         entry.NextRunAt,
		LastRunAt:         entry.LastRunAt,
		LastTriggeredAt:   entry.LastTriggeredAt,
		CreatedAt:         entry.CreatedAt,
		UpdatedAt:         entry.UpdatedAt,
	}
}

func cloneJSON(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make([]byte, len(raw))
	copy(cloned, raw)
	return cloned
}
