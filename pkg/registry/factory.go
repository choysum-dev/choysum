// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// Factory creates a registry instance.
type Factory func() Registry

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register registers a named registry factory.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = f
}

// Exists reports whether a registry factory is registered.
func Exists(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := factories[name]
	return ok
}

// Keys returns a copy of all registered factory names.
func Keys() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}

// NewByName creates a registry by name. It returns nil when missing.
func NewByName(name string) Registry {
	mu.RLock()
	f := factories[name]
	mu.RUnlock()
	if f != nil {
		return f()
	}
	return nil
}

// NewRegistry creates a registry from scope configuration.
func NewRegistry(runtimeScope scope.Scope) Registry {
	return NewByName(runtimeOptionsFromScope(runtimeScope).serverRegister)
}
