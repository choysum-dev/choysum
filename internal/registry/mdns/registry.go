// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package mdns

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/grandcat/zeroconf"
	"github.com/rs/xid"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc/resolver"
)

var schemeName = "mdns"

type mdnsRegistry struct {
	sync.Mutex

	name      string
	zservers  map[string][]*zeroconf.Server
	endpoints map[string]*registry.Endpoint
	builder   resolver.Builder
}

func getCurrentInterfaces(host string) ([]net.Interface, error) {
	ifaces := listMulticastInterfaces()
	if host == "" || host == "0.0.0.0" {
		return ifaces, nil
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, xfmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
		}
		for _, addr := range addrs {
			if strings.HasPrefix(addr.String(), host) {
				return []net.Interface{iface}, nil
			}
		}
	}
	return nil, xfmt.Errorf("no interface found for address %s", host)
}

func listMulticastInterfaces() []net.Interface {
	var interfaces []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, ifi := range ifaces {
		if (ifi.Flags & net.FlagUp) == 0 {
			continue
		}
		if (ifi.Flags & net.FlagMulticast) > 0 {
			interfaces = append(interfaces, ifi)
		}
	}

	return interfaces
}

func (r *mdnsRegistry) Register(serviceName string, addr *resolver.Address) (*registry.Endpoint, error) {
	r.Lock()
	defer r.Unlock()

	host, port, _ := net.SplitHostPort(addr.Addr)
	portInt, _ := strconv.Atoi(port)

	ifaces, err := getCurrentInterfaces(host)
	if err != nil {
		return nil, xfmt.Errorf("failed to get interfaces: %w", err)
	}

	endpointId := xid.New().String()

	s, err := zeroconf.Register(endpointId, serviceName+"._choysum._tcp", "local.", portInt, nil, ifaces)
	if err != nil {
		return nil, xfmt.Errorf("failed to register service: %w", err)
	}

	// for list services
	ls, err := zeroconf.Register(fmt.Sprintf("%s:%s:%s", "choysum", serviceName, endpointId), "_services._choysum._tcp", "local.", portInt, nil, ifaces)
	if err != nil {
		s.Shutdown()
		return nil, xfmt.Errorf("failed to register service: %w", err)
	}

	r.zservers[endpointId] = []*zeroconf.Server{s, ls}

	endpoint := &registry.Endpoint{
		Id:          endpointId,
		ServiceName: serviceName,
		Address:     addr,
	}
	r.endpoints[endpointId] = endpoint
	return endpoint, nil

}

func (r *mdnsRegistry) UnRegister(endpoint *registry.Endpoint) error {
	r.Lock()
	defer r.Unlock()

	if endpoint == nil {
		return nil
	}
	endpointId := endpoint.Id
	servers, ok := r.zservers[endpointId]
	if !ok {
		return nil
	}
	for _, server := range servers {
		if server != nil {
			server.Shutdown()
		}
	}

	delete(r.zservers, endpointId)
	delete(r.endpoints, endpointId)

	return nil
}

func (r *mdnsRegistry) UnRegisterAll() error {
	r.Lock()
	defer r.Unlock()

	for _, servers := range r.zservers {
		servers[0].Shutdown()
		servers[1].Shutdown()
	}

	r.zservers = make(map[string][]*zeroconf.Server)
	r.endpoints = make(map[string]*registry.Endpoint)

	return nil

}

func (r *mdnsRegistry) GetService(serviceName string) ([]*registry.Endpoint, error) {
	// Snapshot under the registry lock, then release before network browse so
	// Register/UnRegister are not blocked on mDNS I/O.
	r.Lock()
	endpoints := r.localEndpointsLocked(serviceName)
	r.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zr, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, xfmt.Errorf("failed to create zeroconf resolver: %w", err)
	}

	seen := endpointSet(endpoints)
	var collectMu sync.Mutex
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		t := time.NewTimer(time.Millisecond * 100) // wait 100ms at first time
		for {
			select {
			case entry, ok := <-results:
				if !ok {
					return
				}
				t.Stop()
				if len(entry.AddrIPv4) == 0 {
					entry.AddrIPv4 = []net.IP{net.ParseIP("127.0.0.1")}
				}
				for _, ipv4 := range entry.AddrIPv4 {
					endpoint := &registry.Endpoint{
						Id:          entry.Instance,
						ServiceName: serviceName,
						Address: &resolver.Address{
							Addr: fmt.Sprintf("%s:%d", ipv4.String(), entry.Port),
						},
					}
					collectMu.Lock()
					appendEndpointIfNew(&endpoints, seen, endpoint)
					collectMu.Unlock()
				}

				t.Reset(time.Millisecond * 1) // wait 1ms for next entry
			case <-t.C:
				cancel()
			}
		}
	}(entries)

	err = zr.Browse(ctx, serviceName+"._choysum._tcp", "local.", entries)
	if err != nil {
		return nil, xfmt.Errorf("failed to browse service: %w", err)
	}

	<-ctx.Done()

	collectMu.Lock()
	defer collectMu.Unlock()
	return endpoints, nil
}

func (r *mdnsRegistry) ListServices() ([]*registry.Endpoint, error) {
	r.Lock()
	endpoints := r.localEndpointsLocked("")
	r.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zr, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, xfmt.Errorf("failed to create zeroconf resolver: %w", err)
	}

	seen := endpointSet(endpoints)
	var collectMu sync.Mutex
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		t := time.NewTimer(time.Millisecond * 100) // wait 100ms at first time
		for {
			select {
			case entry, ok := <-results:
				if !ok {
					return
				}
				t.Stop()
				if len(entry.AddrIPv4) == 0 {
					entry.AddrIPv4 = []net.IP{net.ParseIP("127.0.0.1")}
				}
				// sr_name = fmt.Sprintf("%s:%s:%s", "choysum", name, instance)
				parts := strings.SplitN(entry.Instance, ":", 3)
				if len(parts) != 3 || parts[0] != "choysum" {
					continue
				}
				svcName := parts[1]
				endpointId := parts[2]

				for _, ipv4 := range entry.AddrIPv4 {
					endpoint := &registry.Endpoint{
						Id:          endpointId,
						ServiceName: svcName,
						Address: &resolver.Address{
							Addr: fmt.Sprintf("%s:%d", ipv4.String(), entry.Port),
						},
					}
					collectMu.Lock()
					appendEndpointIfNew(&endpoints, seen, endpoint)
					collectMu.Unlock()
				}

				t.Reset(time.Millisecond * 1) // wait 1ms for next entry
			case <-t.C:
				cancel()
			}
		}
	}(entries)

	err = zr.Browse(ctx, "_services._choysum._tcp", "local.", entries)
	if err != nil {
		return nil, xfmt.Errorf("failed to browse service: %w", err)
	}

	<-ctx.Done()

	collectMu.Lock()
	defer collectMu.Unlock()
	return endpoints, nil
}

func (r *mdnsRegistry) Scheme() string {
	return schemeName
}

func (r *mdnsRegistry) Resolver() resolver.Builder {
	return r.builder
}

func (r *mdnsRegistry) localEndpointsLocked(serviceName string) []*registry.Endpoint {
	endpoints := make([]*registry.Endpoint, 0, len(r.endpoints))
	for _, endpoint := range r.endpoints {
		if endpoint == nil || (serviceName != "" && endpoint.ServiceName != serviceName) {
			continue
		}
		cloned := *endpoint
		if endpoint.Address != nil {
			address := *endpoint.Address
			host, port, err := net.SplitHostPort(address.Addr)
			if err == nil && (host == "" || host == "0.0.0.0" || host == "::") {
				address.Addr = net.JoinHostPort("127.0.0.1", port)
			}
			cloned.Address = &address
		}
		endpoints = append(endpoints, &cloned)
	}
	return endpoints
}

func endpointSet(endpoints []*registry.Endpoint) map[string]struct{} {
	seen := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint != nil {
			seen[endpoint.Id] = struct{}{}
		}
	}
	return seen
}

func appendEndpointIfNew(endpoints *[]*registry.Endpoint, seen map[string]struct{}, endpoint *registry.Endpoint) {
	if endpoint == nil {
		return
	}
	if _, ok := seen[endpoint.Id]; ok {
		return
	}
	seen[endpoint.Id] = struct{}{}
	*endpoints = append(*endpoints, endpoint)
}

func NewMdnsRegistry() registry.Registry {
	r := &mdnsRegistry{
		name:      schemeName,
		zservers:  make(map[string][]*zeroconf.Server),
		endpoints: make(map[string]*registry.Endpoint),
	}
	builder := NewMdnsBuilder(r)
	r.builder = builder
	return r
}
