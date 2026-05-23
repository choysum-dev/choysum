// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type Scheduler struct {
	runtimeScope scope.Scope
	runtimeOpts  runtimeOptions
	queue        taskcontract.TaskQueue
	store        taskcontract.ScheduleStore
	events       taskcontract.EventBus
	stopCh       chan struct{}
	wg           sync.WaitGroup
	interval     time.Duration
	window       time.Time
	count        int
}

func NewScheduler(runtimeScope scope.Scope) *Scheduler {
	return NewSchedulerWithRuntime(runtimeScope, taskcontract.Runtime{})
}

func NewSchedulerWithRuntime(runtimeScope scope.Scope, runtime taskcontract.Runtime) *Scheduler {
	opts := runtimeOptionsFromScope(runtimeScope)
	runtime = runtimeWithDefaultTaskRuntimeDeps(runtimeScope, runtime)
	return &Scheduler{
		runtimeScope: runtimeScope,
		runtimeOpts:  opts,
		queue:        runtime.Queue,
		store:        runtime.Store,
		events:       runtime.Events,
		stopCh:       make(chan struct{}),
		interval:     30 * time.Second,
	}
}

func (s *Scheduler) resolvedRuntimeOptions() runtimeOptions {
	if s != nil && s.runtimeOpts.initialized {
		return s.runtimeOpts
	}
	if s != nil {
		return runtimeOptionsFromScope(s.runtimeScope)
	}
	return newRuntimeOptions(scope.DatabaseRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Scheduler) loop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	if s.runtimeScope == nil || s.store == nil || s.queue == nil {
		return
	}
	ctx := s.runtimeScope.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC()

	schedules, err := s.store.ListDue(ctx, now, 50)
	if err != nil {
		s.runtimeScope.Logger().Warn("task schedule poll failed", "error", err)
		return
	}

	for _, sch := range schedules {
		s.processSchedule(nil, scheduleEntryToModel(sch), now)
	}
}

func (s *Scheduler) processSchedule(db *gorm.DB, sch *Schedule, now time.Time) {
	ctx := context.Background()
	if s.runtimeScope != nil && s.runtimeScope.Context() != nil {
		ctx = s.runtimeScope.Context()
	}
	loc := time.UTC
	if strings.TrimSpace(sch.Timezone) == "" {
		_ = s.store.Disable(ctx, sch.Id, time.Now().UTC())
		return
	}
	if tz, err := time.LoadLocation(sch.Timezone); err == nil {
		loc = tz
	} else {
		_ = s.store.Disable(ctx, sch.Id, time.Now().UTC())
		return
	}

	fields, err := parseCronExpr(strings.TrimSpace(sch.CronExpr))
	if err != nil {
		// Mark inactive on invalid cron to avoid hot loop
		_ = s.store.Disable(ctx, sch.Id, time.Now().UTC())
		return
	}

	nowLoc := now.In(loc)
	next, ok := nextCronTime(nowLoc, fields)
	if !ok {
		return
	}
	// Misfire: if next_run_at <= now, trigger once and advance.
	trigger := sch.NextRunAt == nil || !sch.NextRunAt.After(now)

	if trigger {
		updated, err := s.store.TryAdvanceDue(ctx, sch.Id, now, next.In(time.UTC), time.Now().UTC())
		if err != nil || !updated {
			return
		}
		if !s.allowScheduleEnqueue(now) {
			s.emitMetric("task_schedule_enqueue_skipped", map[string]any{
				"schedule_id": sch.Id,
				"target_app":  sch.TargetApp,
				"full_method": sch.FullMethod,
				"reason":      "rate_limit",
			})
			return
		}

		payload := map[string]any{}
		if len(sch.PayloadTemplate) > 0 {
			_ = json.Unmarshal(sch.PayloadTemplate, &payload)
		}

		job := taskcontract.QueueJob{
			ID:                xid.New().String(),
			TargetApp:         sch.TargetApp,
			FullMethod:        sch.FullMethod,
			PayloadJSON:       mustJSON(payload),
			SchedulerUserID:   sch.SchedulerUserId,
			TriggeredByUserID: sch.TriggeredByUserId,
			Status:            taskcontract.JobStatusQueued,
			RunAfter:          now,
			Attempt:           0,
			MaxAttempts:       s.defaultMaxAttempts(),
			TimeoutMs:         resolveScheduleTimeoutMs(sch, s.defaultJobTimeoutMs()),
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			s.runtimeScope.Logger().Warn("task schedule enqueue failed", "error", err, "schedule_id", sch.Id)
			return
		}
		_ = s.store.MarkTriggered(ctx, sch.Id, now, time.Now().UTC())
		s.emitMetric("task_schedule_triggered", map[string]any{"schedule_id": sch.Id, "target_app": sch.TargetApp, "full_method": sch.FullMethod})
		s.publishDispatchWakeup(ctx, "schedule")
		return
	}

	nextUTC := next.In(time.UTC)
	_ = s.store.UpdateNextRun(ctx, sch.Id, nextUTC, time.Now().UTC())
	s.emitMetric("task_schedule_next_run", map[string]any{"schedule_id": sch.Id, "next_run_at": nextUTC})
}

func (s *Scheduler) allowScheduleEnqueue(now time.Time) bool {
	limit := s.scheduleEnqueueLimit()
	if limit <= 0 {
		return true
	}
	window := now.UTC().Truncate(time.Minute)
	if s.window.IsZero() || !s.window.Equal(window) {
		s.window = window
		s.count = 0
	}
	if s.count >= limit {
		return false
	}
	s.count++
	return true
}

func (s *Scheduler) scheduleEnqueueLimit() int {
	return s.resolvedRuntimeOptions().scheduleMaxScheduleEnqueuesPerMinute
}

func (s *Scheduler) defaultMaxAttempts() int {
	if attempts := s.resolvedRuntimeOptions().dispatchDefaultMaxAttempts; attempts > 0 {
		return attempts
	}
	return 1
}

func (s *Scheduler) defaultJobTimeoutMs() int64 {
	return s.resolvedRuntimeOptions().dispatchDefaultJobTimeoutMs
}

func resolveScheduleTimeoutMs(sch *Schedule, fallback int64) int64 {
	if sch == nil {
		return fallback
	}
	if sch.TimeoutMs > 0 {
		return sch.TimeoutMs
	}
	return fallback
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (s *Scheduler) emitMetric(name string, fields map[string]any) {
	if s.runtimeScope == nil {
		return
	}
	fields["metric"] = name
	RecordMetric(name, fields)
}

func (s *Scheduler) publishDispatchWakeup(ctx context.Context, source string) {
	if s.events == nil {
		return
	}
	_ = s.events.Publish(ctx, taskcontract.Event{
		Topic:  taskcontract.EventTopicDispatchWakeup,
		Source: source,
		At:     time.Now().UTC(),
	})
}

type cronFields struct {
	minutes map[int]bool
	hours   map[int]bool
	dom     map[int]bool
	month   map[int]bool
	dow     map[int]bool
}

func parseCronExpr(expr string) (*cronFields, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid cron expression")
	}
	min, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return nil, err
	}
	hr, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return nil, err
	}
	dom, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return nil, err
	}
	mon, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return nil, err
	}
	dow, err := parseCronField(parts[4], 0, 6)
	if err != nil {
		return nil, err
	}
	return &cronFields{minutes: min, hours: hr, dom: dom, month: mon, dow: dow}, nil
}

func parseCronField(input string, min, max int) (map[int]bool, error) {
	set := map[int]bool{}
	if input == "*" {
		for i := min; i <= max; i++ {
			set[i] = true
		}
		return set, nil
	}
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "*/") {
			step, err := strconv.Atoi(strings.TrimPrefix(part, "*/"))
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step")
			}
			for i := min; i <= max; i += step {
				set[i] = true
			}
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range")
			}
			start, err := strconv.Atoi(bounds[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(bounds[1])
			if err != nil {
				return nil, err
			}
			if start < min || end > max || start > end {
				return nil, fmt.Errorf("range out of bounds")
			}
			for i := start; i <= end; i++ {
				set[i] = true
			}
			continue
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		if val < min || val > max {
			return nil, fmt.Errorf("value out of bounds")
		}
		set[val] = true
	}
	return set, nil
}

func nextCronTime(from time.Time, fields *cronFields) (time.Time, bool) {
	// Start from next minute
	cur := from.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 525600; i++ { // up to ~1 year
		if fields.month[int(cur.Month())] && fields.dom[cur.Day()] && fields.hours[cur.Hour()] && fields.minutes[cur.Minute()] && fields.dow[int(cur.Weekday())] {
			return cur, true
		}
		cur = cur.Add(time.Minute)
	}
	return time.Time{}, false
}

func isSQLiteConfig(runtimeScope scope.Scope) bool {
	return strings.EqualFold(strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).dbDialect), "sqlite")
}
