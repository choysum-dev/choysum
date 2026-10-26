// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

type cancelAwareTaskQueue struct {
	stubTaskQueue
	listReadyErr error
}

func (q cancelAwareTaskQueue) ListReady(ctx context.Context, now time.Time, limit int) ([]taskcontract.QueueJob, error) {
	if q.listReadyErr != nil {
		return nil, q.listReadyErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

type cancelAwareScheduleStore struct {
	stubScheduleStore
	listDueErr error
}

func (s cancelAwareScheduleStore) ListDue(ctx context.Context, now time.Time, limit int) ([]taskcontract.ScheduleEntry, error) {
	if s.listDueErr != nil {
		return nil, s.listDueErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func TestIsExpectedPollShutdownError(t *testing.T) {
	if !isExpectedPollShutdownError(context.Canceled) {
		t.Fatal("expected context.Canceled to be treated as shutdown poll error")
	}
	if isExpectedPollShutdownError(context.DeadlineExceeded) {
		t.Fatal("did not expect DeadlineExceeded to be treated as shutdown poll error")
	}
	if isExpectedPollShutdownError(nil) {
		t.Fatal("did not expect nil to be treated as shutdown poll error")
	}
}

func TestDispatcherPollOnce_ContextCanceledDoesNotWarn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	dispatcher := NewDispatcherWithRuntime(
		&testScope{ctx: ctx, logger: logger, cfg: &config.Config{Task: config.NewDefaultTaskConfig()}},
		nil,
		taskcontract.Runtime{Queue: cancelAwareTaskQueue{}},
	)
	dispatcher.pollOnce("poll")

	if strings.Contains(buf.String(), "task dispatch poll failed") {
		t.Fatalf("expected no WARN for canceled poll, got logs: %s", buf.String())
	}
}

func TestDispatcherPollOnce_OtherErrorsStillWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	dispatcher := NewDispatcherWithRuntime(
		&testScope{ctx: context.Background(), logger: logger, cfg: &config.Config{Task: config.NewDefaultTaskConfig()}},
		nil,
		taskcontract.Runtime{Queue: cancelAwareTaskQueue{listReadyErr: errTestPoll}},
	)
	dispatcher.pollOnce("poll")

	if !strings.Contains(buf.String(), "task dispatch poll failed") {
		t.Fatalf("expected WARN for non-cancel poll error, got logs: %s", buf.String())
	}
}

func TestDispatcherLoop_ExitsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := NewDispatcherWithRuntime(
		&testScope{
			ctx:    ctx,
			logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			cfg:    &config.Config{Task: config.NewDefaultTaskConfig()},
		},
		nil,
		taskcontract.Runtime{Queue: cancelAwareTaskQueue{}},
	)
	dispatcher.interval = time.Hour
	dispatcher.Start()

	cancel()

	done := make(chan struct{})
	go func() {
		dispatcher.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Dispatcher.Stop did not return after context cancel")
	}
}

func TestSchedulerTick_ContextCanceledDoesNotWarn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	scheduler := NewSchedulerWithRuntime(
		&testScope{ctx: ctx, logger: logger, cfg: &config.Config{Task: config.NewDefaultTaskConfig()}},
		taskcontract.Runtime{
			Queue: cancelAwareTaskQueue{},
			Store: cancelAwareScheduleStore{},
		},
	)
	scheduler.tick()

	if strings.Contains(buf.String(), "task schedule poll failed") {
		t.Fatalf("expected no WARN for canceled schedule poll, got logs: %s", buf.String())
	}
}

func TestSchedulerLoop_ExitsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	scheduler := NewSchedulerWithRuntime(
		&testScope{
			ctx:    ctx,
			logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			cfg:    &config.Config{Task: config.NewDefaultTaskConfig()},
		},
		taskcontract.Runtime{
			Queue: cancelAwareTaskQueue{},
			Store: cancelAwareScheduleStore{},
		},
	)
	scheduler.interval = time.Hour
	scheduler.Start()

	cancel()

	done := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Scheduler.Stop did not return after context cancel")
	}
}

var errTestPoll = errString("poll boom")

type errString string

func (e errString) Error() string { return string(e) }
