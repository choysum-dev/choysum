// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

import (
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/cache"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// JWTIdentityCacheStoreFactory creates the backend cache store used by the JWT
// identity cache wrapper.
type JWTIdentityCacheStoreFactory func(runtimeScope scope.Scope) (cache.Cache, error)

var (
	jwtIdentityCacheStoreDriversMu sync.RWMutex
	jwtIdentityCacheStoreDrivers   = make(map[string]JWTIdentityCacheStoreFactory)
)

// RegisterJWTIdentityCacheStoreDriver registers a host-visible JWT identity
// cache backend driver so external repositories can contribute replacements
// without importing internal/jwtauth.
func RegisterJWTIdentityCacheStoreDriver(name string, factory JWTIdentityCacheStoreFactory) {
	name = strings.TrimSpace(name)
	if name == "" {
		panic("auth: RegisterJWTIdentityCacheStoreDriver name is empty")
	}
	if factory == nil {
		panic("auth: RegisterJWTIdentityCacheStoreDriver factory is nil")
	}

	jwtIdentityCacheStoreDriversMu.Lock()
	defer jwtIdentityCacheStoreDriversMu.Unlock()
	if _, dup := jwtIdentityCacheStoreDrivers[name]; dup {
		panic("auth: RegisterJWTIdentityCacheStoreDriver called twice for driver " + name)
	}
	jwtIdentityCacheStoreDrivers[name] = factory
}

// NewJWTIdentityCacheStore resolves and creates the named JWT identity cache
// backend. An empty backend name falls back to the default memory driver.
func NewJWTIdentityCacheStore(backend string, runtimeScope scope.Scope) (cache.Cache, error) {
	backend = strings.TrimSpace(backend)
	if backend == "" {
		backend = "memory"
	}

	jwtIdentityCacheStoreDriversMu.RLock()
	factory, ok := jwtIdentityCacheStoreDrivers[backend]
	jwtIdentityCacheStoreDriversMu.RUnlock()
	if !ok {
		return nil, autherrors.NewAuthErrorf(autherrors.ErrDriverNotRegistered, "JWT identity cache backend %q is not registered", backend)
	}
	return factory(runtimeScope)
}
