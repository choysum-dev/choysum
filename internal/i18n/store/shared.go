// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
)

var (
	sharedMu       sync.Mutex
	sharedRegistry *Registry
)

// RegistryFor returns the process-shared terminology Registry for this runtime.
// Install/upgrade/uninstall and the JS $choysum.i18n bridge must share one cache.
func RegistryFor(runtimeScope scope.Scope) *Registry {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedRegistry == nil || sharedRegistry.runtimeScope != runtimeScope {
		sharedRegistry = NewRegistry(runtimeScope)
	}
	return sharedRegistry
}

// ResetSharedRegistryForTests clears the process-shared registry (tests only).
func ResetSharedRegistryForTests() {
	sharedMu.Lock()
	sharedRegistry = nil
	sharedMu.Unlock()
}
