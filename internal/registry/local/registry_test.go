// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package local

import (
	"errors"
	"net/url"
	"testing"

	registrypkg "github.com/choysum-dev/choysum/pkg/registry"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

type stubClientConn struct {
	state resolver.State
	err   error
}

func (c *stubClientConn) UpdateState(state resolver.State) error {
	c.state = state
	return nil
}

func (c *stubClientConn) ReportError(err error) {
	c.err = err
}

func (c *stubClientConn) NewAddress([]resolver.Address) {}

func (c *stubClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult {
	return nil
}

type stubRegistry struct {
	endpoints []*registrypkg.Endpoint
	err       error
}

func (r *stubRegistry) Scheme() string { return "stub" }

func (r *stubRegistry) Register(serviceName string, addr *resolver.Address) (*registrypkg.Endpoint, error) {
	return nil, nil
}

func (r *stubRegistry) UnRegister(endpoint *registrypkg.Endpoint) error { return nil }

func (r *stubRegistry) UnRegisterAll() error { return nil }

func (r *stubRegistry) ListServices() ([]*registrypkg.Endpoint, error) { return r.endpoints, r.err }

func (r *stubRegistry) GetService(serviceName string) ([]*registrypkg.Endpoint, error) {
	if r.err != nil {
		return nil, r.err
	}
	filtered := make([]*registrypkg.Endpoint, 0, len(r.endpoints))
	for _, endpoint := range r.endpoints {
		if endpoint.ServiceName == serviceName {
			filtered = append(filtered, endpoint)
		}
	}
	return filtered, nil
}

func (r *stubRegistry) Resolver() resolver.Builder { return nil }

func TestNewLocalRegistryLifecycle(t *testing.T) {
	if !registrypkg.Exists("local") {
		t.Fatal("expected local registry to be registered in factory")
	}

	r := NewLocalRegistry().(*localRegistry)
	if r.Scheme() != "local" {
		t.Fatalf("Scheme() = %q, want local", r.Scheme())
	}
	if r.Resolver() == nil || r.Resolver().Scheme() != "local" {
		t.Fatal("expected resolver builder with local scheme")
	}

	first, err := r.Register("svc", &resolver.Address{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("Register first: %v", err)
	}
	second, err := r.Register("other", &resolver.Address{Addr: "127.0.0.1:8081"})
	if err != nil {
		t.Fatalf("Register second: %v", err)
	}
	if first.Id == "" || second.Id == "" || first.Id == second.Id {
		t.Fatalf("expected unique non-empty endpoint ids: first=%q second=%q", first.Id, second.Id)
	}

	services, err := r.ListServices()
	if err != nil || len(services) != 2 {
		t.Fatalf("ListServices = %d, %v; want 2, nil", len(services), err)
	}

	filtered, err := r.GetService("svc")
	if err != nil || len(filtered) != 1 || filtered[0].Id != first.Id {
		t.Fatalf("GetService(svc) = %#v, %v", filtered, err)
	}

	if err := r.UnRegister(first); err != nil {
		t.Fatalf("UnRegister: %v", err)
	}
	filtered, err = r.GetService("svc")
	if err != nil || len(filtered) != 0 {
		t.Fatalf("GetService after unregister = %#v, %v", filtered, err)
	}

	if err := r.UnRegisterAll(); err != nil {
		t.Fatalf("UnRegisterAll: %v", err)
	}
	services, err = r.ListServices()
	if err != nil || len(services) != 0 {
		t.Fatalf("ListServices after clear = %#v, %v", services, err)
	}

	if err := r.UnRegister(first); err != nil {
		t.Fatalf("UnRegister missing endpoint: %v", err)
	}
	if err := r.UnRegister(second); err != nil {
		t.Fatalf("UnRegister removed endpoint: %v", err)
	}
}

func TestLocalResolverResolveNowUpdatesAddressesWithoutEmptyEntries(t *testing.T) {
	registry := &stubRegistry{endpoints: []*registrypkg.Endpoint{
		{Id: "1", ServiceName: "svc", Address: &resolver.Address{Addr: "127.0.0.1:8080"}},
		{Id: "2", ServiceName: "svc", Address: &resolver.Address{Addr: "127.0.0.1:8081"}},
		{Id: "3", ServiceName: "other", Address: &resolver.Address{Addr: "127.0.0.1:8082"}},
	}}
	cc := &stubClientConn{}
	r := &localResolver{
		registry: registry,
		target:   resolver.Target{URL: url.URL{Path: "/svc"}},
		cc:       cc,
	}

	r.ResolveNow(resolver.ResolveNowOptions{})

	if cc.err != nil {
		t.Fatalf("ReportError = %v, want nil", cc.err)
	}
	if len(cc.state.Addresses) != 2 {
		t.Fatalf("UpdateState addresses len = %d, want 2 (%#v)", len(cc.state.Addresses), cc.state.Addresses)
	}
	if cc.state.Addresses[0].Addr != "127.0.0.1:8080" || cc.state.Addresses[1].Addr != "127.0.0.1:8081" {
		t.Fatalf("unexpected addresses: %#v", cc.state.Addresses)
	}
}

func TestLocalResolverReportsRegistryError(t *testing.T) {
	wantErr := errors.New("lookup failed")
	cc := &stubClientConn{}
	r := &localResolver{
		registry: &stubRegistry{err: wantErr},
		target:   resolver.Target{URL: url.URL{Path: "/svc"}},
		cc:       cc,
	}

	r.ResolveNow(resolver.ResolveNowOptions{})

	if !errors.Is(cc.err, wantErr) {
		t.Fatalf("ReportError = %v, want %v", cc.err, wantErr)
	}
	if len(cc.state.Addresses) != 0 {
		t.Fatalf("unexpected addresses after error: %#v", cc.state.Addresses)
	}
	if builder := NewLocalBuilder(&stubRegistry{}); builder.Scheme() != "local" {
		t.Fatalf("builder Scheme() = %q, want local", builder.Scheme())
	}
}
