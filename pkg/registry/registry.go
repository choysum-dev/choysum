// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"google.golang.org/grpc/resolver"
)

type Endpoint struct {
	Id          string
	ServiceName string
	Address     *resolver.Address
}

type Registry interface {
	Scheme() string
	Register(serviceName string, addr *resolver.Address) (endpoint *Endpoint, err error)
	UnRegister(endpoint *Endpoint) error
	UnRegisterAll() error
	ListServices() ([]*Endpoint, error)
	GetService(serviceName string) ([]*Endpoint, error)
	Resolver() resolver.Builder
}
