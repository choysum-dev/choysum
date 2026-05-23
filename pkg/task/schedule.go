// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"time"
)

// ScheduleEntry is the stable schedule record exchanged across scheduler seams.
type ScheduleEntry struct {
	ID                  string
	Active              bool
	Name                string
	TargetApp           string
	FullMethod          string
	PayloadTemplateJSON json.RawMessage
	SchedulerUserID     string
	TriggeredByUserID   string
	CronExpr            string
	Timezone            string
	TimeoutMs           int64
	NextRunAt           *time.Time
	LastRunAt           *time.Time
	LastTriggeredAt     *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ScheduleStore defines the minimum stable persistence semantics needed by the
// default scheduler loop.
type ScheduleStore interface {
	ListDue(context.Context, time.Time, int) ([]ScheduleEntry, error)
	TryAdvanceDue(context.Context, string, time.Time, time.Time, time.Time) (bool, error)
	UpdateNextRun(context.Context, string, time.Time, time.Time) error
	MarkTriggered(context.Context, string, time.Time, time.Time) error
	Disable(context.Context, string, time.Time) error
}

// Scheduler is the stable runtime control seam for schedule polling loops.
type Scheduler interface {
	Start()
	Stop()
}
