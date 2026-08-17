// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

type stubTaskQueue struct{}

func (stubTaskQueue) Enqueue(context.Context, taskcontract.QueueJob) error { return nil }
func (stubTaskQueue) ListReady(context.Context, time.Time, int) ([]taskcontract.QueueJob, error) {
	return nil, nil
}
func (stubTaskQueue) TryClaim(context.Context, string, time.Time) (bool, error)   { return false, nil }
func (stubTaskQueue) Get(context.Context, string) (*taskcontract.QueueJob, error) { return nil, nil }
func (stubTaskQueue) UpdateAttempt(context.Context, string, int, time.Time) error { return nil }
func (stubTaskQueue) MarkSucceeded(context.Context, string, taskcontract.QueueSuccess) error {
	return nil
}
func (stubTaskQueue) MarkFailed(context.Context, string, taskcontract.QueueFailure) error { return nil }
func (stubTaskQueue) Retry(context.Context, string, taskcontract.QueueRetry) error        { return nil }
func (stubTaskQueue) MarkCancelled(context.Context, string, taskcontract.QueueCancellation) error {
	return nil
}

type stubScheduleStore struct{}

func (stubScheduleStore) ListDue(context.Context, time.Time, int) ([]taskcontract.ScheduleEntry, error) {
	return nil, nil
}
func (stubScheduleStore) TryAdvanceDue(context.Context, string, time.Time, time.Time, time.Time) (bool, error) {
	return false, nil
}
func (stubScheduleStore) UpdateNextRun(context.Context, string, time.Time, time.Time) error {
	return nil
}
func (stubScheduleStore) MarkTriggered(context.Context, string, time.Time, time.Time) error {
	return nil
}
func (stubScheduleStore) Disable(context.Context, string, time.Time) error { return nil }

type stubSubscription struct{}

func (stubSubscription) Close() error { return nil }

type stubEventBus struct{}

func (stubEventBus) Publish(context.Context, bus.Event) error { return nil }
func (stubEventBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return stubSubscription{}, nil
}

type stubGarbageCollector struct{}

func (stubGarbageCollector) Start() {}
func (stubGarbageCollector) Stop()  {}

func TestNewDispatcherWithRuntimeUsesInjectedComponents(t *testing.T) {
	runtimeScope := &testScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{Task: config.NewDefaultTaskConfig()},
	}
	queue := stubTaskQueue{}
	events := stubEventBus{}

	dispatcher := NewDispatcherWithRuntime(runtimeScope, nil, taskcontract.Runtime{
		Queue:  queue,
		Events: events,
	})

	if dispatcher.queue != queue {
		t.Fatal("expected injected task queue")
	}
	if dispatcher.events != events {
		t.Fatal("expected injected event bus")
	}
}

func TestNewSchedulerWithRuntimeUsesInjectedComponents(t *testing.T) {
	runtimeScope := &testScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{Task: config.NewDefaultTaskConfig()},
	}
	queue := stubTaskQueue{}
	store := stubScheduleStore{}
	events := stubEventBus{}

	scheduler := NewSchedulerWithRuntime(runtimeScope, taskcontract.Runtime{
		Queue:  queue,
		Store:  store,
		Events: events,
	})

	if scheduler.queue != queue {
		t.Fatal("expected injected task queue")
	}
	if scheduler.store != store {
		t.Fatal("expected injected schedule store")
	}
	if scheduler.events != events {
		t.Fatal("expected injected event bus")
	}
}
