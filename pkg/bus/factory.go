// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// Factory creates an EventBus instance.
type Factory func() EventBus

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register registers a named EventBus factory.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = f
}

// Exists reports whether an EventBus factory is registered.
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

// NewByName creates an EventBus by name. It returns nil when missing.
func NewByName(name string) EventBus {
	mu.RLock()
	f := factories[name]
	mu.RUnlock()
	if f != nil {
		return f()
	}
	return nil
}

// NewBus creates an EventBus from scope configuration. Missing drivers yield nil.
func NewBus(runtimeScope scope.Scope) EventBus {
	return NewByName(runtimeOptionsFromScope(runtimeScope).driver)
}

// UnregisterFactoryForTest removes a registered factory until restore runs. Tests only.
func UnregisterFactoryForTest(name string) (restore func()) {
	mu.Lock()
	old, ok := factories[name]
	if ok {
		delete(factories, name)
	}
	mu.Unlock()
	return func() {
		mu.Lock()
		if ok {
			factories[name] = old
		}
		mu.Unlock()
	}
}
