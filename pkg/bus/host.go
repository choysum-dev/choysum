// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"reflect"
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
)

var (
	hostMu sync.RWMutex
	host   EventBus
)

// IsUsable reports whether events can safely receive Publish/Subscribe calls.
// It rejects both untyped nil and typed-nil interface values (nil pointers
// stored in EventBus), which pass `== nil` checks but panic on method calls.
func IsUsable(events EventBus) bool {
	if events == nil {
		return false
	}
	value := reflect.ValueOf(events)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !value.IsNil()
	default:
		return true
	}
}

// SetHost installs the process-wide host EventBus singleton shared by task,
// TipHub, and JS-module Publish. Nil and typed-nil values are ignored.
func SetHost(events EventBus) {
	if !IsUsable(events) {
		return
	}
	hostMu.Lock()
	host = events
	hostMu.Unlock()
}

// Host returns the process-wide host EventBus, or nil when unset.
func Host() EventBus {
	hostMu.RLock()
	defer hostMu.RUnlock()
	return host
}

// EnsureHost returns the host EventBus, creating one from scope when unset or
// unusable. Concurrent callers share the first created instance.
func EnsureHost(runtimeScope scope.Scope) EventBus {
	hostMu.Lock()
	defer hostMu.Unlock()
	if IsUsable(host) {
		return host
	}
	host = NewBus(runtimeScope)
	return host
}

// ClearHostForTest clears the process-wide host EventBus. Tests only.
func ClearHostForTest() {
	hostMu.Lock()
	host = nil
	hostMu.Unlock()
}
