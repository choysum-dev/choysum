// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"time"
)

const (
	JobStatusQueued      = "queued"
	JobStatusDispatching = "dispatching"
	JobStatusSucceeded   = "succeeded"
	JobStatusFailed      = "failed"
	JobStatusCancelled   = "cancelled"
)

// QueueJob is the stable queue record exchanged across task runtime seams.
type QueueJob struct {
	ID                 string
	TargetApp          string
	FullMethod         string
	PayloadJSON        json.RawMessage
	SchedulerUserID    string
	TriggeredByUserID  string
	Status             string
	RunAfter           time.Time
	Attempt            int
	MaxAttempts        int
	TimeoutMs          int64
	CancelRequestedAt  *time.Time
	CancelledAt        *time.Time
	FinishedAt         *time.Time
	LastErrorJSON      json.RawMessage
	LastErrorHash      string
	LastErrorTruncated bool
	ResultJSON         json.RawMessage
	ResultHash         string
	ResultTruncated    bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// QueueSuccess captures a terminal success update for a queued job.
type QueueSuccess struct {
	FinishedAt      time.Time
	UpdatedAt       time.Time
	ResultJSON      json.RawMessage
	ResultHash      string
	ResultTruncated bool
}

// QueueFailure captures a terminal failure update for a queued job.
type QueueFailure struct {
	FinishedAt      time.Time
	UpdatedAt       time.Time
	ErrorJSON       json.RawMessage
	ErrorHash       string
	ErrorTruncated  bool
	ResultJSON      json.RawMessage
	ResultHash      string
	ResultTruncated bool
}

// QueueRetry captures a retry update for a queued job.
type QueueRetry struct {
	RunAfter       time.Time
	UpdatedAt      time.Time
	ErrorJSON      json.RawMessage
	ErrorHash      string
	ErrorTruncated bool
}

// QueueCancellation captures a terminal cancellation update for a queued job.
type QueueCancellation struct {
	CancelledAt time.Time
	FinishedAt  time.Time
	UpdatedAt   time.Time
	ErrorJSON   json.RawMessage
}

// TaskQueue defines the minimum stable queue semantics needed by the default
// dispatcher and scheduler runtimes.
type TaskQueue interface {
	Enqueue(context.Context, QueueJob) error
	ListReady(context.Context, time.Time, int) ([]QueueJob, error)
	TryClaim(context.Context, string, time.Time) (bool, error)
	Get(context.Context, string) (*QueueJob, error)
	UpdateAttempt(context.Context, string, int, time.Time) error
	MarkSucceeded(context.Context, string, QueueSuccess) error
	MarkFailed(context.Context, string, QueueFailure) error
	Retry(context.Context, string, QueueRetry) error
	MarkCancelled(context.Context, string, QueueCancellation) error
}