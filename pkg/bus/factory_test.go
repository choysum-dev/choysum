// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"context"
	"log/slog"
	"reflect"
	"sort"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubBus struct{ name string }

func (b *stubBus) Publish(context.Context, Event) error { return nil }

func (b *stubBus) Subscribe(string, EventHandler) (Subscription, error) {
	return stubSub{}, nil
}

type stubSub struct{}

func (stubSub) Close() error { return nil }

type stubScope struct {
	cfg *config.Config
}

func (e *stubScope) Run(func(scope.Scope) error) error { return nil }

func (e *stubScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *stubScope) Session() *scope.Session { return nil }

func (e *stubScope) WithContext(ctx context.Context) scope.Scope { return e }

func (e *stubScope) Context() context.Context { return context.Background() }

func (e *stubScope) Logger() *slog.Logger { return nil }

func (e *stubScope) Config() *config.Config { return e.cfg }

func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func snapshotFactories() map[string]Factory {
	mu.RLock()
	defer mu.RUnlock()
	clone := make(map[string]Factory, len(factories))
	for name, factory := range factories {
		clone[name] = factory
	}
	return clone
}

func restoreFactories(snapshot map[string]Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories = snapshot
}

func TestFactoryRegisterExistsKeysAndNewByName(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	Register("alpha", func() EventBus { return &stubBus{name: "alpha"} })
	Register("beta", func() EventBus { return &stubBus{name: "beta"} })

	if !Exists("alpha") || !Exists("beta") {
		t.Fatal("expected registered factories to exist")
	}
	if Exists("missing") {
		t.Fatal("did not expect missing factory to exist")
	}

	keys := Keys()
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"alpha", "beta"}) {
		t.Fatalf("Keys() = %#v, want alpha/beta", keys)
	}

	got := NewByName("alpha")
	if got == nil || got.(*stubBus).name != "alpha" {
		t.Fatalf("NewByName(alpha) = %#v", got)
	}
	if NewByName("missing") != nil {
		t.Fatal("expected missing factory lookup to return nil")
	}
}

func TestNewBusDefaultsToInprocess(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	Register(defaultDriver, func() EventBus { return &stubBus{name: defaultDriver} })

	got := NewBus(nil)
	if got == nil || got.(*stubBus).name != defaultDriver {
		t.Fatalf("NewBus(nil) = %#v, want inprocess stub", got)
	}

	got = NewBus(&stubScope{cfg: &config.Config{Server: &config.ServerConfig{}}})
	if got == nil || got.(*stubBus).name != defaultDriver {
		t.Fatalf("NewBus(scope) = %#v, want inprocess stub", got)
	}

	factories = make(map[string]Factory)
	if NewBus(nil) != nil {
		t.Fatal("expected nil bus when default driver is unregistered")
	}
}
