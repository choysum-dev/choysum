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
//
// Identity is the underlying *sql.DB, not the scope pointer: StaticScopeProvider
// / WithContext allocate new scope wrappers, and pointer equality would replace a
// warmed registry with a cold one when the JS bridge installs.
func RegistryFor(runtimeScope scope.Scope) *Registry {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedRegistry == nil {
		sharedRegistry = NewRegistry(runtimeScope)
		return sharedRegistry
	}
	if !scopesShareRuntimeDB(sharedRegistry.runtimeScope, runtimeScope) {
		// Cold replacement: callers (startup / finishApplicationModeStartup) must
		// WarmRuntimeCache again. Do not warm under sharedMu (DB I/O while locked).
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

// scopesShareRuntimeDB reports whether two scopes use the same SQL pool.
func scopesShareRuntimeDB(a, b scope.Scope) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a == b {
		return true
	}
	sa, sb := a.Session(), b.Session()
	if sa == nil || sb == nil || sa.DB == nil || sb.DB == nil {
		return false
	}
	// Uninitialized *gorm.DB has nil Config; reading ConnPool can panic.
	if sa.DB.Config == nil || sb.DB.Config == nil {
		return false
	}
	// WithContext clones *gorm.DB but keeps the same ConnPool (SQL pool).
	if sa.DB.ConnPool == nil || sb.DB.ConnPool == nil {
		return false
	}
	return sa.DB.ConnPool == sb.DB.ConnPool
}
