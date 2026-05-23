// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"testing"
	"time"

	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

func TestDispatchWakeupPublishesToDefaultEventBus(t *testing.T) {
	bus := DispatchEventBus()
	if bus == nil {
		t.Fatal("expected default dispatch event bus")
	}

	called := 0
	lastSource := ""
	sub, err := bus.Subscribe(taskcontract.EventTopicDispatchWakeup, func(ctx context.Context, event taskcontract.Event) {
		called++
		lastSource = event.Source
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(func() {
		_ = sub.Close()
	})

	WakeDispatch("enqueue")
	if called != 1 {
		t.Fatalf("wakeup call count = %d, want 1", called)
	}
	if lastSource != "enqueue" {
		t.Fatalf("wakeup source = %q, want enqueue", lastSource)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	lastSource = ""
	called = 0
	WakeDispatch("ignored")
	time.Sleep(10 * time.Millisecond)
	if called != 0 {
		t.Fatalf("wakeup call count after close = %d, want 0", called)
	}
}
