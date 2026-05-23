// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import "context"

type scopeContextKey struct{}

// ContextWithScope stores a scope in ctx for trusted (Go-side) propagation.
func ContextWithScope(ctx context.Context, scope Scope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

// ScopeFromContext loads a scope from ctx.
func ScopeFromContext(ctx context.Context) (Scope, bool) {
	if ctx == nil {
		return nil, false
	}
	scope, ok := ctx.Value(scopeContextKey{}).(Scope)
	return scope, ok && scope != nil
}

// SessionFromContext loads a DB session from ctx-derived transaction or scope.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	if ctx == nil {
		return nil, false
	}
	if tx, ok := TransactionFromContext(ctx); ok {
		sess := tx.Session()
		return sess, sess != nil
	}
	if scope, ok := ScopeFromContext(ctx); ok {
		sess := scope.Session()
		return sess, sess != nil
	}
	return nil, false
}
