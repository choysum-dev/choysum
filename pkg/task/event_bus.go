// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"time"
)

const EventTopicDispatchWakeup = "task.dispatch.wakeup"

// Event carries async notifications through a stable event bus seam.
type Event struct {
	Topic   string
	Source  string
	At      time.Time
	Payload map[string]any
}

// EventHandler receives one published event.
type EventHandler func(context.Context, Event)

// Subscription closes a live event bus subscription.
type Subscription interface {
	Close() error
}

// EventBus defines the minimum stable async publish and subscribe semantics.
type EventBus interface {
	Publish(context.Context, Event) error
	Subscribe(string, EventHandler) (Subscription, error)
}
