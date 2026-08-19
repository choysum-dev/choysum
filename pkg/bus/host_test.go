// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

type valueStubBus struct{ name string }

func (valueStubBus) Publish(context.Context, Event) error { return nil }

func (valueStubBus) Subscribe(string, EventHandler) (Subscription, error) {
	return stubSub{}, nil
}

func TestEnsureHostReusesSingleton(t *testing.T) {
	ClearHostForTest()
	t.Cleanup(ClearHostForTest)

	Register("inprocess", func() EventBus { return &stubBus{name: "host"} })
	t.Cleanup(func() {
		mu.Lock()
		delete(factories, "inprocess")
		mu.Unlock()
	})

	runtimeScope := &stubScope{cfg: &config.Config{}}
	first := EnsureHost(runtimeScope)
	if first == nil {
		t.Fatal("EnsureHost returned nil")
	}
	second := EnsureHost(runtimeScope)
	if second != first {
		t.Fatalf("EnsureHost second = %p, want first %p", second, first)
	}
	if Host() != first {
		t.Fatal("Host() did not return EnsureHost instance")
	}
}

func TestSetHostOverridesEnsureHost(t *testing.T) {
	ClearHostForTest()
	t.Cleanup(ClearHostForTest)

	injected := &stubBus{name: "injected"}
	SetHost(injected)
	SetHost(nil) // ignored
	if Host() != injected {
		t.Fatal("SetHost(nil) should leave the host unchanged")
	}

	Register("inprocess", func() EventBus { return &stubBus{name: "created"} })
	t.Cleanup(func() {
		mu.Lock()
		delete(factories, "inprocess")
		mu.Unlock()
	})

	got := EnsureHost(&stubScope{cfg: &config.Config{}})
	if got != injected {
		t.Fatalf("EnsureHost = %p, want injected %p", got, injected)
	}
}

func TestIsUsableRejectsTypedNilEventBus(t *testing.T) {
	var typedNil *stubBus
	var events EventBus = typedNil
	if IsUsable(events) {
		t.Fatal("typed-nil EventBus should not be usable")
	}
	if IsUsable(nil) {
		t.Fatal("nil EventBus should not be usable")
	}
	if !IsUsable(&stubBus{name: "ok"}) {
		t.Fatal("concrete EventBus should be usable")
	}
}

func TestIsUsableAcceptsStructBackedEventBus(t *testing.T) {
	var events EventBus = valueStubBus{name: "struct"}
	if !IsUsable(events) {
		t.Fatal("struct-backed EventBus should be usable")
	}
}

func TestSetHostAndEnsureHostRejectTypedNil(t *testing.T) {
	ClearHostForTest()
	t.Cleanup(ClearHostForTest)

	var typedNil *stubBus
	var events EventBus = typedNil
	SetHost(events)
	if Host() != nil {
		t.Fatal("SetHost(typed-nil) should leave host unset")
	}

	Register("inprocess", func() EventBus { return &stubBus{name: "fallback"} })
	t.Cleanup(func() {
		mu.Lock()
		delete(factories, "inprocess")
		mu.Unlock()
	})

	// Simulate a stale typed-nil host slot and ensure EnsureHost replaces it.
	hostMu.Lock()
	host = events
	hostMu.Unlock()

	got := EnsureHost(&stubScope{cfg: &config.Config{}})
	if !IsUsable(got) {
		t.Fatal("EnsureHost should replace typed-nil host")
	}
	if Host() != got {
		t.Fatal("EnsureHost should bind the replacement host")
	}
}
