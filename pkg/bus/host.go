// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
)

var (
	hostMu sync.RWMutex
	host   EventBus
)

// SetHost installs the process-wide host EventBus singleton shared by task,
// TipHub, and JS-module Publish. Nil is ignored.
func SetHost(events EventBus) {
	if events == nil {
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

// EnsureHost returns the host EventBus, creating one from scope when unset.
// Concurrent callers share the first created instance.
func EnsureHost(runtimeScope scope.Scope) EventBus {
	hostMu.Lock()
	defer hostMu.Unlock()
	if host != nil {
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
