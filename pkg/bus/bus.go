// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"context"
	"time"
)

// Event carries best-effort tip / wakeup notifications through the platform
// event bus. Payload must stay thin; authoritative state lives in tables.
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
