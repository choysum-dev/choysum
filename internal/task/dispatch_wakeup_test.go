// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
)

func TestDispatchWakeupPublishesOnInjectedEventBus(t *testing.T) {
	events := bus.NewBus(nil)
	if events == nil {
		t.Fatal("expected inprocess event bus")
	}

	d := &Dispatcher{events: events}

	called := 0
	lastSource := ""
	sub, err := events.Subscribe(bus.TopicDispatchWakeup, func(ctx context.Context, event bus.Event) {
		called++
		lastSource = event.Source
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(func() {
		_ = sub.Close()
	})

	d.publishWakeup("enqueue")
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
	d.publishWakeup("ignored")
	time.Sleep(10 * time.Millisecond)
	if called != 0 {
		t.Fatalf("wakeup call count after close = %d, want 0", called)
	}
}
