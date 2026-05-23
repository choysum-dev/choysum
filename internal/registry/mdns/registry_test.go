// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package mdns

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

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

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func findEndpoint(endpoints []*registrypkg.Endpoint, serviceName, endpointID string) *registrypkg.Endpoint {
	for _, endpoint := range endpoints {
		if endpoint != nil && endpoint.ServiceName == serviceName && endpoint.Id == endpointID {
			return endpoint
		}
	}
	return nil
}

func TestNewMdnsRegistryBuilderAndFactory(t *testing.T) {
	if !registrypkg.Exists("mdns") {
		t.Fatal("expected mdns registry to be registered in factory")
	}

	r := NewMdnsRegistry().(*mdnsRegistry)
	if r.Scheme() != "mdns" {
		t.Fatalf("Scheme() = %q, want mdns", r.Scheme())
	}
	if r.Resolver() == nil || r.Resolver().Scheme() != "mdns" {
		t.Fatal("expected mdns resolver builder")
	}
	if r.zservers == nil {
		t.Fatal("expected zservers map to be initialized")
	}

	builder := NewMdnsBuilder(&stubRegistry{})
	if builder.Scheme() != "mdns" {
		t.Fatalf("builder Scheme() = %q, want mdns", builder.Scheme())
	}
}

func TestListMulticastInterfacesOnlyReturnsUsableInterfaces(t *testing.T) {
	for _, iface := range listMulticastInterfaces() {
		if (iface.Flags & net.FlagUp) == 0 {
			t.Fatalf("interface %s is not up", iface.Name)
		}
		if (iface.Flags & net.FlagMulticast) == 0 {
			t.Fatalf("interface %s does not support multicast", iface.Name)
		}
	}
}

func TestGetCurrentInterfacesEmptyAndUnknownHost(t *testing.T) {
	all := listMulticastInterfaces()
	got, err := getCurrentInterfaces("")
	if err != nil {
		t.Fatalf("getCurrentInterfaces empty host: %v", err)
	}
	if len(got) != len(all) {
		t.Fatalf("empty host interfaces len = %d, want %d", len(got), len(all))
	}

	got, err = getCurrentInterfaces("203.0.113.250")
	if err == nil {
		t.Fatalf("expected unknown host error, got interfaces %#v", got)
	}
}

func TestMdnsResolverResolveNowUpdatesAddressesWithoutEmptyEntries(t *testing.T) {
	registry := &stubRegistry{endpoints: []*registrypkg.Endpoint{
		{Id: "1", ServiceName: "svc", Address: &resolver.Address{Addr: "127.0.0.1:8080"}},
		{Id: "2", ServiceName: "svc", Address: &resolver.Address{Addr: "127.0.0.1:8081"}},
	}}
	cc := &stubClientConn{}
	r := &mdnsResolver{
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

func TestMdnsResolverReportsRegistryError(t *testing.T) {
	wantErr := errors.New("lookup failed")
	cc := &stubClientConn{}
	r := &mdnsResolver{
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
}

func TestMdnsBuilderBuildAndResolverClose(t *testing.T) {
	registry := &stubRegistry{endpoints: []*registrypkg.Endpoint{{Id: "1", ServiceName: "svc", Address: &resolver.Address{Addr: "127.0.0.1:8080"}}}}
	cc := &stubClientConn{}
	builder := NewMdnsBuilder(registry)

	r, err := builder.Build(resolver.Target{URL: url.URL{Path: "/svc"}}, cc, resolver.BuildOptions{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(cc.state.Addresses) != 1 || cc.state.Addresses[0].Addr != "127.0.0.1:8080" {
		t.Fatalf("unexpected resolver state after Build: %#v", cc.state.Addresses)
	}
	r.Close()
}

func TestMdnsRegistryRuntimeLifecycle(t *testing.T) {
	r := NewMdnsRegistry().(*mdnsRegistry)

	endpointA, err := r.Register("copilot-runtime-a", &resolver.Address{Addr: "0.0.0.0:19091"})
	if err != nil {
		t.Fatalf("Register(service A) error = %v", err)
	}
	endpointB, err := r.Register("copilot-runtime-b", &resolver.Address{Addr: "0.0.0.0:19092"})
	if err != nil {
		_ = r.UnRegisterAll()
		t.Fatalf("Register(service B) error = %v", err)
	}
	t.Cleanup(func() { _ = r.UnRegisterAll() })

	waitForCondition(t, 5*time.Second, func() bool {
		endpoints, err := r.GetService("copilot-runtime-a")
		if err != nil {
			return false
		}
		ep := findEndpoint(endpoints, "copilot-runtime-a", endpointA.Id)
		return ep != nil && strings.HasSuffix(ep.Address.Addr, ":19091")
	})

	waitForCondition(t, 5*time.Second, func() bool {
		services, err := r.ListServices()
		if err != nil {
			return false
		}
		return findEndpoint(services, "copilot-runtime-a", endpointA.Id) != nil && findEndpoint(services, "copilot-runtime-b", endpointB.Id) != nil
	})

	if err := r.UnRegister(endpointA); err != nil {
		t.Fatalf("UnRegister() error = %v", err)
	}
	waitForCondition(t, 5*time.Second, func() bool {
		endpoints, err := r.GetService("copilot-runtime-a")
		if err != nil {
			return false
		}
		return findEndpoint(endpoints, "copilot-runtime-a", endpointA.Id) == nil
	})

	if err := r.UnRegisterAll(); err != nil {
		t.Fatalf("UnRegisterAll() error = %v", err)
	}
	if len(r.zservers) != 0 {
		t.Fatalf("expected zservers map to be reset, got %#v", r.zservers)
	}
	waitForCondition(t, 5*time.Second, func() bool {
		services, err := r.ListServices()
		if err != nil {
			return false
		}
		return findEndpoint(services, "copilot-runtime-b", endpointB.Id) == nil
	})
}
