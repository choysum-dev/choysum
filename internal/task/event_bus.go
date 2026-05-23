// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"sync"
	"time"

	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

var _ taskcontract.EventBus = (*inProcessEventBus)(nil)
var _ taskcontract.Subscription = (*eventSubscription)(nil)

var (
	legacyWakeMu  sync.Mutex
	legacyWakeSub taskcontract.Subscription
)

type inProcessEventBus struct {
	mu       sync.RWMutex
	nextID   int
	handlers map[string]map[int]taskcontract.EventHandler
}

type eventSubscription struct {
	closeOnce sync.Once
	closeFn   func()
}

var dispatchEventBus taskcontract.EventBus = newInProcessEventBus()

func newInProcessEventBus() taskcontract.EventBus {
	return &inProcessEventBus{handlers: map[string]map[int]taskcontract.EventHandler{}}
}

func DispatchEventBus() taskcontract.EventBus {
	return dispatchEventBus
}

// WakeDispatch triggers an in-process dispatcher wakeup through the default
// event bus. The delivery is best-effort.
func WakeDispatch(source string) {
	bus := DispatchEventBus()
	if bus == nil {
		return
	}
	_ = bus.Publish(context.Background(), taskcontract.Event{
		Topic:  taskcontract.EventTopicDispatchWakeup,
		Source: source,
		At:     time.Now().UTC(),
	})
}

func subscribeDispatchWakeup(handler func(source string)) (taskcontract.Subscription, error) {
	bus := DispatchEventBus()
	if bus == nil {
		return nil, nil
	}
	return bus.Subscribe(taskcontract.EventTopicDispatchWakeup, func(ctx context.Context, event taskcontract.Event) {
		if handler == nil {
			return
		}
		handler(event.Source)
	})
}

func registerDispatchWakeup(fn func(source string)) {
	legacyWakeMu.Lock()
	defer legacyWakeMu.Unlock()
	if legacyWakeSub != nil {
		_ = legacyWakeSub.Close()
		legacyWakeSub = nil
	}
	sub, err := subscribeDispatchWakeup(fn)
	if err == nil {
		legacyWakeSub = sub
	}
}

func clearDispatchWakeup() {
	legacyWakeMu.Lock()
	defer legacyWakeMu.Unlock()
	if legacyWakeSub != nil {
		_ = legacyWakeSub.Close()
		legacyWakeSub = nil
	}
}

func (b *inProcessEventBus) Publish(ctx context.Context, event taskcontract.Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	b.mu.RLock()
	registered := b.handlers[event.Topic]
	handlers := make([]taskcontract.EventHandler, 0, len(registered))
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

func (b *inProcessEventBus) Subscribe(topic string, handler taskcontract.EventHandler) (taskcontract.Subscription, error) {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	if _, ok := b.handlers[topic]; !ok {
		b.handlers[topic] = map[int]taskcontract.EventHandler{}
	}
	b.handlers[topic][id] = handler
	b.mu.Unlock()
	return &eventSubscription{closeFn: func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		registered := b.handlers[topic]
		delete(registered, id)
		if len(registered) == 0 {
			delete(b.handlers, topic)
		}
	}}, nil
}

func (s *eventSubscription) Close() error {
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
