// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package inprocess

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
)

func TestInProcessBusFanOutAndClose(t *testing.T) {
	b := NewInProcessBus()

	var (
		mu   sync.Mutex
		gotA []string
		gotB []string
	)

	subA, err := b.Subscribe("topic.a", func(ctx context.Context, event bus.Event) {
		mu.Lock()
		gotA = append(gotA, event.Source)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Subscribe A: %v", err)
	}
	subB, err := b.Subscribe("topic.a", func(ctx context.Context, event bus.Event) {
		mu.Lock()
		gotB = append(gotB, event.Source)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Subscribe B: %v", err)
	}

	if err := b.Publish(context.Background(), bus.Event{Topic: "topic.a", Source: "one"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := b.Publish(context.Background(), bus.Event{Topic: "topic.missing", Source: "noop"}); err != nil {
		t.Fatalf("Publish with no subscribers: %v", err)
	}

	mu.Lock()
	if len(gotA) != 1 || gotA[0] != "one" || len(gotB) != 1 || gotB[0] != "one" {
		t.Fatalf("fan-out gotA=%v gotB=%v", gotA, gotB)
	}
	mu.Unlock()

	if err := subA.Close(); err != nil {
		t.Fatalf("Close A: %v", err)
	}
	if err := b.Publish(context.Background(), bus.Event{Topic: "topic.a", Source: "two"}); err != nil {
		t.Fatalf("Publish after close: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(gotA) != 1 {
		t.Fatalf("gotA after close = %v, want [one]", gotA)
	}
	if len(gotB) != 2 || gotB[0] != "one" || gotB[1] != "two" {
		t.Fatalf("gotB after second publish = %v, want [one two]", gotB)
	}
	if err := subB.Close(); err != nil {
		t.Fatalf("Close B: %v", err)
	}
}

func TestInProcessBusFillsZeroAt(t *testing.T) {
	b := NewInProcessBus()
	var at time.Time
	sub, err := b.Subscribe("t", func(ctx context.Context, event bus.Event) {
		at = event.At
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	before := time.Now().UTC().Add(-time.Second)
	if err := b.Publish(context.Background(), bus.Event{Topic: "t", Source: "s"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if at.IsZero() || at.Before(before) {
		t.Fatalf("At = %v, want filled near now", at)
	}
}

func TestInProcessBusSubscribeDuringHandler(t *testing.T) {
	b := NewInProcessBus()
	done := make(chan struct{})

	_, err := b.Subscribe("t", func(ctx context.Context, event bus.Event) {
		_, subErr := b.Subscribe("t", func(context.Context, bus.Event) {})
		if subErr != nil {
			t.Errorf("nested Subscribe: %v", subErr)
		}
		close(done)
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	finished := make(chan struct{})
	go func() {
		_ = b.Publish(context.Background(), bus.Event{Topic: "t", Source: "nested"})
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish deadlocked while handler Subscribe ran")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not complete")
	}
}
