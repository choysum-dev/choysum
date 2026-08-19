// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

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
