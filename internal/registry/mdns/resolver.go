// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package mdns

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/registry"
	"google.golang.org/grpc/resolver"
)

type mdnsResolver struct {
	registry registry.Registry
	target   resolver.Target
	cc       resolver.ClientConn
}

// ResolveNow is a no-op here.
// It's just a hint, resolver can ignore this if it's not necessary.
func (r *mdnsResolver) ResolveNow(o resolver.ResolveNowOptions) {
	serviceName := strings.TrimPrefix(r.target.URL.Path, "/")
	endpoints, err := r.registry.GetService(serviceName)
	if err != nil {
		r.cc.ReportError(err)
	}
	addrs := make([]resolver.Address, 0, len(endpoints))
	for _, e := range endpoints {
		addrs = append(addrs, *e.Address)
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *mdnsResolver) Close() {}

type mdnsBuilder struct {
	registry registry.Registry
}

func NewMdnsBuilder(r registry.Registry) *mdnsBuilder {
	return &mdnsBuilder{registry: r}
}

func (b *mdnsBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &mdnsResolver{
		registry: b.registry,
		target:   target,
		cc:       cc,
	}
	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}

func (b *mdnsBuilder) Scheme() string {
	return schemeName
}
