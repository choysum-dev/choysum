// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type schedulerNoSessionScope struct {
	*testScope
}

func (e *schedulerNoSessionScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *schedulerNoSessionScope) Session() *scope.Session { return nil }

func TestParseCronExpr_Invalid(t *testing.T) {
	if _, err := parseCronExpr("invalid"); err == nil {
		t.Fatalf("expected error for invalid cron expr")
	}
}

func TestParseCronFieldPaths(t *testing.T) {
	t.Run("wildcard covers full range", func(t *testing.T) {
		got, err := parseCronField("*", 1, 5)
		if err != nil {
			t.Fatalf("parseCronField(*) error = %v", err)
		}
		for value := 1; value <= 5; value++ {
			if !got[value] {
				t.Fatalf("expected wildcard to include %d, got %#v", value, got)
			}
		}
		if len(got) != 5 {
			t.Fatalf("wildcard size = %d, want 5", len(got))
		}
	})

	t.Run("supports steps ranges singles and empty segments", func(t *testing.T) {
		got, err := parseCronField("*/15, 10-12, , 58", 0, 59)
		if err != nil {
			t.Fatalf("parseCronField(composite) error = %v", err)
		}
		for _, value := range []int{0, 10, 11, 12, 15, 30, 45, 58} {
			if !got[value] {
				t.Fatalf("expected composite field to include %d, got %#v", value, got)
			}
		}
		if got[59] {
			t.Fatalf("did not expect composite field to include 59, got %#v", got)
		}
		if len(got) != 8 {
			t.Fatalf("composite field size = %d, want 8", len(got))
		}
	})

	t.Run("rejects invalid step", func(t *testing.T) {
		if _, err := parseCronField("*/0", 0, 59); err == nil || err.Error() != "invalid step" {
			t.Fatalf("expected invalid step error, got %v", err)
		}
	})

	t.Run("rejects invalid range and bounds", func(t *testing.T) {
		if _, err := parseCronField("10-5", 0, 59); err == nil || err.Error() != "range out of bounds" {
			t.Fatalf("expected reversed range error, got %v", err)
		}
		if _, err := parseCronField("1-70", 0, 59); err == nil || err.Error() != "range out of bounds" {
			t.Fatalf("expected out of bounds range error, got %v", err)
		}
	})

	t.Run("rejects invalid numeric input", func(t *testing.T) {
		if _, err := parseCronField("abc", 0, 59); err == nil {
			t.Fatal("expected invalid number error")
		}
		if _, err := parseCronField("60", 0, 59); err == nil || err.Error() != "value out of bounds" {
			t.Fatalf("expected value out of bounds error, got %v", err)
		}
	})
}

func TestParseCronExpr_Complex(t *testing.T) {
	fields, err := parseCronExpr("*/20 6,18 1-3 2 1-5")
	if err != nil {
		t.Fatalf("parseCronExpr(complex) error = %v", err)
	}

	for _, minute := range []int{0, 20, 40} {
		if !fields.minutes[minute] {
			t.Fatalf("expected minute %d in %#v", minute, fields.minutes)
		}
	}
	for _, hour := range []int{6, 18} {
		if !fields.hours[hour] {
			t.Fatalf("expected hour %d in %#v", hour, fields.hours)
		}
	}
	for _, day := range []int{1, 2, 3} {
		if !fields.dom[day] {
			t.Fatalf("expected day-of-month %d in %#v", day, fields.dom)
		}
	}
	if !fields.month[2] || fields.month[1] {
		t.Fatalf("unexpected month field contents: %#v", fields.month)
	}
	for _, weekday := range []int{1, 2, 3, 4, 5} {
		if !fields.dow[weekday] {
			t.Fatalf("expected weekday %d in %#v", weekday, fields.dow)
		}
	}
}

func TestNextCronTime_EveryFiveMinutes(t *testing.T) {
	fields, err := parseCronExpr("*/5 * * * *")
	if err != nil {
		t.Fatalf("parse cron failed: %v", err)
	}
	from := time.Date(2026, 1, 19, 10, 2, 30, 0, time.UTC)
	next, ok := nextCronTime(from, fields)
	if !ok {
		t.Fatalf("expected next cron time")
	}
	if next.Minute() != 5 || next.Hour() != 10 {
		t.Fatalf("unexpected next time: %v", next)
	}
}

func TestSchedulerMaxEnqueuesPerMinute(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, 1, 20, 12, 0, 30, 0, time.UTC)

	dsn := "file:task_scheduler_rate_limit?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Schedule{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Schedule.MaxScheduleEnqueuesPerMinute = 1

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	s := NewScheduler(runtimeScope)

	older := fixedNow.Add(-2 * time.Minute)
	newer := fixedNow.Add(-1 * time.Minute)
	sch1 := Schedule{
		Id:                "sch-1",
		Active:            true,
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		PayloadTemplate:   []byte(`{"email":"a@b.com"}`),
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "* * * * *",
		Timezone:          "UTC",
		NextRunAt:         &older,
		CreatedAt:         fixedNow,
		UpdatedAt:         fixedNow,
	}
	sch2 := Schedule{
		Id:                "sch-2",
		Active:            true,
		TargetApp:         "auth",
		FullMethod:        "auth.User/Logout",
		PayloadTemplate:   []byte(`{"userId":"u1"}`),
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "* * * * *",
		Timezone:          "UTC",
		NextRunAt:         &newer,
		CreatedAt:         fixedNow,
		UpdatedAt:         fixedNow,
	}

	if err := db.Create(&sch1).Error; err != nil {
		t.Fatalf("create schedule1: %v", err)
	}
	if err := db.Create(&sch2).Error; err != nil {
		t.Fatalf("create schedule2: %v", err)
	}

	s.processSchedule(db, &sch1, fixedNow)
	s.processSchedule(db, &sch2, fixedNow)

	var jobs []Job
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	var out1, out2 Schedule
	if err := db.First(&out1, "id = ?", "sch-1").Error; err != nil {
		t.Fatalf("get schedule1: %v", err)
	}
	if err := db.First(&out2, "id = ?", "sch-2").Error; err != nil {
		t.Fatalf("get schedule2: %v", err)
	}
	if out1.LastTriggeredAt == nil {
		t.Fatalf("expected schedule1 to be triggered")
	}
	if out2.LastTriggeredAt != nil {
		t.Fatalf("expected schedule2 to be skipped by rate limit")
	}
	if out2.NextRunAt == nil {
		t.Fatalf("expected schedule2 next_run_at to advance")
	}
}

func TestSchedulerDefaultsAppliedToJob(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, 1, 20, 12, 0, 30, 0, time.UTC)

	dsn := "file:task_scheduler_defaults?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Schedule{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.DefaultMaxAttempts = 3
	cfg.Task.Dispatch.DefaultJobTimeoutMs = 5000

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	s := NewScheduler(runtimeScope)
	older := fixedNow.Add(-2 * time.Minute)
	sch := Schedule{
		Id:                "sch-defaults",
		Active:            true,
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		PayloadTemplate:   []byte(`{"email":"a@b.com"}`),
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "* * * * *",
		Timezone:          "UTC",
		NextRunAt:         &older,
		CreatedAt:         fixedNow,
		UpdatedAt:         fixedNow,
	}

	if err := db.Create(&sch).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	s.processSchedule(db, &sch, fixedNow)

	var jobs []Job
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].MaxAttempts != 3 {
		t.Fatalf("unexpected max attempts: %d", jobs[0].MaxAttempts)
	}
	if jobs[0].TimeoutMs != 5000 {
		t.Fatalf("unexpected timeout ms: %d", jobs[0].TimeoutMs)
	}
}

func TestSchedulerScheduleTimeoutOverridesDefault(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, 1, 20, 12, 0, 30, 0, time.UTC)

	dsn := "file:task_scheduler_timeout_override?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Schedule{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.DefaultJobTimeoutMs = 5000

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	s := NewScheduler(runtimeScope)
	older := fixedNow.Add(-2 * time.Minute)
	sch := Schedule{
		Id:                "sch-timeout-override",
		Active:            true,
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		PayloadTemplate:   []byte(`{"email":"a@b.com"}`),
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "* * * * *",
		Timezone:          "UTC",
		TimeoutMs:         12000,
		NextRunAt:         &older,
		CreatedAt:         fixedNow,
		UpdatedAt:         fixedNow,
	}

	if err := db.Create(&sch).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	s.processSchedule(db, &sch, fixedNow)

	var jobs []Job
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].TimeoutMs != 12000 {
		t.Fatalf("unexpected timeout ms: %d", jobs[0].TimeoutMs)
	}
}

func TestSchedulerTickNoopWhenSessionMissing(t *testing.T) {
	runtimeScope := &schedulerNoSessionScope{testScope: &testScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{Task: config.NewDefaultTaskConfig()},
	}}

	s := NewScheduler(runtimeScope)
	s.tick()
}

func TestSchedulerStartStopProcessesDueSchedule(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	dsn := "file:task_scheduler_loop?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Schedule{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.DefaultMaxAttempts = 2
	cfg.Task.Dispatch.DefaultJobTimeoutMs = 4000

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	older := now.Add(-2 * time.Minute)
	sch := Schedule{
		Id:                "sch-loop",
		Active:            true,
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		PayloadTemplate:   []byte(`{"email":"loop@choysum.dev"}`),
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "* * * * *",
		Timezone:          "UTC",
		NextRunAt:         &older,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := db.Create(&sch).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	s := NewScheduler(runtimeScope)
	s.interval = 10 * time.Millisecond
	s.Start()
	t.Cleanup(func() {
		select {
		case <-s.stopCh:
		default:
			s.Stop()
		}
	})

	deadline := time.After(2 * time.Second)
	for {
		var jobs []Job
		if err := db.Find(&jobs).Error; err != nil {
			t.Fatalf("list jobs: %v", err)
		}
		if len(jobs) > 0 {
			if jobs[0].Status != "queued" {
				t.Fatalf("job status = %s, want queued", jobs[0].Status)
			}
			if jobs[0].MaxAttempts != 2 {
				t.Fatalf("job max attempts = %d, want 2", jobs[0].MaxAttempts)
			}
			if jobs[0].TimeoutMs != 4000 {
				t.Fatalf("job timeout = %d, want 4000", jobs[0].TimeoutMs)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for scheduler loop to enqueue a job")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	s.Stop()

	var out Schedule
	if err := db.First(&out, "id = ?", sch.Id).Error; err != nil {
		t.Fatalf("load schedule: %v", err)
	}
	if out.LastTriggeredAt == nil {
		t.Fatal("expected scheduler loop to record last_triggered_at")
	}
	if out.NextRunAt == nil || !out.NextRunAt.After(now) {
		t.Fatalf("expected scheduler loop to advance next_run_at, got %v", out.NextRunAt)
	}
}
