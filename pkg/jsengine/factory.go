// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsengine

import (
	"context"
	"sync"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// ScopeProvider resolves the effective scope for a runtime ctx.
type ScopeProvider func(ctx context.Context) scope.Scope

// Factory creates a concrete engine factory from a scope provider.
type Factory func(scopeProvider ScopeProvider, authenticator auth.Authenticator, options ...JsEngineOption) JsEngineFactory

// StaticScopeProvider adapts a concrete scope into a provider that
// rebinds runtime contexts through WithContext when one is available.
func StaticScopeProvider(baseScope scope.Scope) ScopeProvider {
	return func(ctx context.Context) scope.Scope {
		if baseScope == nil {
			return nil
		}
		if ctx == nil {
			return baseScope
		}
		return baseScope.WithContext(ctx)
	}
}

// ResolveScope evaluates the provider for the requested runtime ctx.
func ResolveScope(provider ScopeProvider, ctx context.Context) scope.Scope {
	if provider == nil {
		return nil
	}
	return provider(ctx)
}

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

func Register(name string, p Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = p
}

func Exists(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := factories[name]
	return ok
}

func Keys() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}

func NewByName(name string) Factory {
	mu.RLock()
	p := factories[name]
	mu.RUnlock()
	if p == nil {
		return nil
	}
	return p
}

func NewJsEngineFactory(runtimeScope scope.Scope, authenticator auth.Authenticator) JsEngineFactory {
	f := NewByName(runtimeOptionsFromScope(runtimeScope).serverJsEngineFactory)
	if f == nil {
		return nil
	}
	return f(StaticScopeProvider(runtimeScope), authenticator)
}
