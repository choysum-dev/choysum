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
	live := context.Background()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	if !isExpectedPollShutdownError(canceled, context.Canceled) {
		t.Fatal("expected context.Canceled to be treated as shutdown poll error when runtime ctx is done")
	}
	if isExpectedPollShutdownError(live, context.Canceled) {
		t.Fatal("did not expect context.Canceled to be suppressed while runtime ctx is live")
	}
	if isExpectedPollShutdownError(canceled, context.DeadlineExceeded) {
		t.Fatal("did not expect DeadlineExceeded to be treated as shutdown poll error")
	}
	if isExpectedPollShutdownError(nil, context.Canceled) {
		t.Fatal("did not expect nil runtime ctx to suppress cancel errors")
	}
	if isExpectedPollShutdownError(canceled, nil) {
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

func TestDispatcherPollOnce_StoreCanceledWhileLiveStillWarns(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	dispatcher := NewDispatcherWithRuntime(
		&testScope{ctx: context.Background(), logger: logger, cfg: &config.Config{Task: config.NewDefaultTaskConfig()}},
		nil,
		taskcontract.Runtime{Queue: cancelAwareTaskQueue{listReadyErr: context.Canceled}},
	)
	dispatcher.pollOnce("poll")

	if !strings.Contains(buf.String(), "task dispatch poll failed") {
		t.Fatalf("expected WARN for store cancel while runtime ctx is live, got logs: %s", buf.String())
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
	// Start only the loop so Stop()'s stopCh close cannot mask a missing ctx.Done case.
	dispatcher.wg.Add(1)
	go dispatcher.loop()

	cancel()

	done := make(chan struct{})
	go func() {
		dispatcher.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatcher loop did not exit after context cancel")
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

func TestSchedulerTick_StoreCanceledWhileLiveStillWarns(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	scheduler := NewSchedulerWithRuntime(
		&testScope{ctx: context.Background(), logger: logger, cfg: &config.Config{Task: config.NewDefaultTaskConfig()}},
		taskcontract.Runtime{
			Queue: cancelAwareTaskQueue{},
			Store: cancelAwareScheduleStore{listDueErr: context.Canceled},
		},
	)
	scheduler.tick()

	if !strings.Contains(buf.String(), "task schedule poll failed") {
		t.Fatalf("expected WARN for store cancel while runtime ctx is live, got logs: %s", buf.String())
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
	scheduler.wg.Add(1)
	go scheduler.loop()

	cancel()

	done := make(chan struct{})
	go func() {
		scheduler.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler loop did not exit after context cancel")
	}
}

var errTestPoll = errString("poll boom")

type errString string

func (e errString) Error() string { return string(e) }
