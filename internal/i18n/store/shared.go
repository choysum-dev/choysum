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
// Identity is the underlying SQL ConnPool, not the scope pointer: StaticScopeProvider
// / WithContext allocate new scope wrappers, and pointer equality would replace a
// warmed registry with a cold one when the JS bridge installs.
//
// The stored runtime scope is always non-transactional. Writer paths may call
// RegistryFor(txScope); keeping a committed tx session would break later warms.
func RegistryFor(runtimeScope scope.Scope) *Registry {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	durable := durableRuntimeScope(runtimeScope)
	if sharedRegistry == nil {
		sharedRegistry = NewRegistry(durable)
		return sharedRegistry
	}
	// Nil input must not replace a warmed registry with a cold nil-scope one.
	if durable == nil {
		return sharedRegistry
	}
	if !scopesShareRuntimeDB(sharedRegistry.runtimeScope, durable) {
		// Cold replacement: callers (startup / finishApplicationModeStartup) must
		// WarmRuntimeCache again. Do not warm under sharedMu (DB I/O while locked).
		if durable.Logger() != nil {
			durable.Logger().Warn("terminology registry replaced with cold cache; WarmRuntimeCache required")
		}
		sharedRegistry = NewRegistry(durable)
	}
	return sharedRegistry
}

// ResetSharedRegistryForTests clears the process-shared registry (tests only).
func ResetSharedRegistryForTests() {
	sharedMu.Lock()
	sharedRegistry = nil
	sharedMu.Unlock()
}

// durableRuntimeScope strips a transaction marker so TermStore / WarmRuntimeCache
// keep a live pool session after the caller's tx commits.
func durableRuntimeScope(runtimeScope scope.Scope) scope.Scope {
	if runtimeScope == nil {
		return nil
	}
	ctx := runtimeScope.Context()
	if ctx == nil {
		return runtimeScope
	}
	if _, ok := scope.TransactionFromContext(ctx); !ok {
		return runtimeScope
	}
	return runtimeScope.WithContext(scope.ContextWithoutTransaction(ctx))
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
