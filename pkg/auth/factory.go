// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

import (
	"fmt"
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// Factory is the shared authenticator factory function type.
type Factory func(runtimeScope scope.Scope) (Authenticator, error)

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register registers the authenticator factory for a type.
func Register(typeName string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[typeName] = factory
}

// Exists reports whether the type has been registered.
func Exists(typeName string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := factories[typeName]
	return ok
}

// Keys returns a copy of all registered type names.
func Keys() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}

// NewByName creates an authenticator by name and returns an error when it is missing.
func NewByName(typeName string, runtimeScope scope.Scope) (Authenticator, error) {
	mu.RLock()
	factory, exists := factories[typeName]
	mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unsupported authenticator type: %s", typeName)
	}
	return factory(runtimeScope)
}

// NewAuthenticator creates an authenticator from runtime configuration.
func NewAuthenticator(runtimeScope scope.Scope) (Authenticator, error) {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if !runtimeOpts.authEnabled {
		runtimeScope.Logger().Warn("authentication disabled")
		return nil, nil
	}
	return NewByName(runtimeOpts.authType, runtimeScope)
}
