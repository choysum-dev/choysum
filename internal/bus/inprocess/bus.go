// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package inprocess

import (
	"context"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
)

var _ bus.EventBus = (*EventBus)(nil)
var _ bus.Subscription = (*subscription)(nil)

// EventBus is the CE in-process tip / wakeup fan-out implementation.
type EventBus struct {
	mu       sync.RWMutex
	nextID   int
	handlers map[string]map[int]bus.EventHandler
}

type subscription struct {
	closeOnce sync.Once
	closeFn   func()
}

// NewInProcessBus creates an empty in-process EventBus.
func NewInProcessBus() bus.EventBus {
	return &EventBus{handlers: map[string]map[int]bus.EventHandler{}}
}

func (b *EventBus) Publish(ctx context.Context, event bus.Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	b.mu.RLock()
	registered := b.handlers[event.Topic]
	handlers := make([]bus.EventHandler, 0, len(registered))
	for _, handler := range registered {
		handlers = append(handlers, handler)
	}
	b.mu.RUnlock()
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		handler(ctx, event)
	}
	return nil
}

func (b *EventBus) Subscribe(topic string, handler bus.EventHandler) (bus.Subscription, error) {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	if _, ok := b.handlers[topic]; !ok {
		b.handlers[topic] = map[int]bus.EventHandler{}
	}
	b.handlers[topic][id] = handler
	b.mu.Unlock()
	return &subscription{closeFn: func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		registered := b.handlers[topic]
		delete(registered, id)
		if len(registered) == 0 {
			delete(b.handlers, topic)
		}
	}}, nil
}

func (s *subscription) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.closeFn != nil {
			s.closeFn()
		}
	})
	return nil
}
