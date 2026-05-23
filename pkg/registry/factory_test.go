// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"log/slog"
	"reflect"
	"sort"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/resolver"
)

type stubRegistry struct {
	scheme string
}

func (r *stubRegistry) Scheme() string { return r.scheme }

func (r *stubRegistry) Register(serviceName string, addr *resolver.Address) (*Endpoint, error) {
	return &Endpoint{ServiceName: serviceName, Address: addr}, nil
}

func (r *stubRegistry) UnRegister(endpoint *Endpoint) error { return nil }

func (r *stubRegistry) UnRegisterAll() error { return nil }

func (r *stubRegistry) ListServices() ([]*Endpoint, error) { return nil, nil }

func (r *stubRegistry) GetService(serviceName string) ([]*Endpoint, error) { return nil, nil }

func (r *stubRegistry) Resolver() resolver.Builder { return nil }

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

	alpha := func() Registry { return &stubRegistry{scheme: "alpha"} }
	beta := func() Registry { return &stubRegistry{scheme: "beta"} }
	Register("alpha", alpha)
	Register("beta", beta)

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

	alphaRegistry := NewByName("alpha")
	if alphaRegistry == nil || alphaRegistry.Scheme() != "alpha" {
		t.Fatalf("NewByName(alpha) = %#v", alphaRegistry)
	}
	if NewByName("missing") != nil {
		t.Fatal("expected missing factory lookup to return nil")
	}
}

func TestNewRegistryUsesEnvironmentConfig(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	Register("chosen", func() Registry { return &stubRegistry{scheme: "chosen"} })
	runtimeScope := &stubScope{cfg: &config.Config{Server: &config.ServerConfig{Register: "chosen"}}}

	created := NewRegistry(runtimeScope)
	if created == nil || created.Scheme() != "chosen" {
		t.Fatalf("NewRegistry() = %#v", created)
	}

	runtimeScope.cfg.Server.Register = "missing"
	if created = NewRegistry(runtimeScope); created != nil {
		t.Fatalf("expected nil registry for missing factory, got %#v", created)
	}
}
