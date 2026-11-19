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
	// Nil / unusable input must not replace a warmed registry with a cold one.
	newPool, newOK := runtimeConnPool(durable)
	if !newOK {
		return sharedRegistry
	}
	oldPool, oldOK := runtimeConnPool(sharedRegistry.runtimeScope)
	if oldOK && oldPool == newPool {
		return sharedRegistry
	}
	// Cold replacement: callers (startup / finishApplicationModeStartup) must
	// WarmRuntimeCache again. Do not warm under sharedMu (DB I/O while locked).
	if durable.Logger() != nil {
		durable.Logger().Warn("terminology registry replaced with cold cache; WarmRuntimeCache required")
	}
	sharedRegistry = NewRegistry(durable)
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

// runtimeConnPool returns the shared SQL ConnPool for a scope, if usable.
func runtimeConnPool(runtimeScope scope.Scope) (pool any, ok bool) {
	if runtimeScope == nil {
		return nil, false
	}
	session := runtimeScope.Session()
	if session == nil || session.DB == nil || session.DB.Config == nil || session.DB.ConnPool == nil {
		return nil, false
	}
	return session.DB.ConnPool, true
}

// scopesShareRuntimeDB reports whether two scopes use the same SQL pool.
func scopesShareRuntimeDB(a, b scope.Scope) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a == b {
		return true
	}
	pa, oka := runtimeConnPool(a)
	pb, okb := runtimeConnPool(b)
	if !oka || !okb {
		return false
	}
	return pa == pb
}
