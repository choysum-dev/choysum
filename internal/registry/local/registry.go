// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package local

import (
	"sync"

	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/rs/xid"
	"google.golang.org/grpc/resolver"
)

var schemeName = "local"

type localRegistry struct {
	sync.Mutex
	name      string
	builder   resolver.Builder
	endpoints []*registry.Endpoint
}

func (r *localRegistry) Scheme() string {
	return schemeName
}

func (r *localRegistry) Register(serviceName string, addr *resolver.Address) (*registry.Endpoint, error) {
	r.Lock()
	defer r.Unlock()
	endpoint := &registry.Endpoint{
		Id:          xid.New().String(),
		ServiceName: serviceName,
		Address:     addr,
	}
	r.endpoints = append(r.endpoints, endpoint)
	return endpoint, nil
}

func (r *localRegistry) UnRegister(endpoint *registry.Endpoint) error {
	r.Lock()
	defer r.Unlock()

	for i, e := range r.endpoints {
		if e.Id == endpoint.Id {
			r.endpoints = append(r.endpoints[:i], r.endpoints[i+1:]...)
			return nil
		}
	}

	return nil
}

func (r *localRegistry) UnRegisterAll() error {
	r.Lock()
	defer r.Unlock()
	r.endpoints = nil
	return nil
}

func (r *localRegistry) ListServices() ([]*registry.Endpoint, error) {
	r.Lock()
	defer r.Unlock()
	return r.endpoints, nil
}

func (r *localRegistry) GetService(serviceName string) ([]*registry.Endpoint, error) {
	r.Lock()
	defer r.Unlock()
	endpoints := []*registry.Endpoint{}
	for _, e := range r.endpoints {
		if e.ServiceName == serviceName {
			endpoints = append(endpoints, e)
		}
	}
	return endpoints, nil
}

func (r *localRegistry) Resolver() resolver.Builder {
	return r.builder
}

func NewLocalRegistry() registry.Registry {
	r := &localRegistry{
		name: schemeName,
	}
	builder := NewLocalBuilder(r)
	r.builder = builder
	return r
}
