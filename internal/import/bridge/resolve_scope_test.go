// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"log/slog"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestResolveImportScope_PrefersContextScope(t *testing.T) {
	base := &resolveScopeStub{name: "base"}
	tx := &resolveScopeStub{name: "tx"}
	execCtx := scope.ContextWithScope(context.Background(), tx)

	got := resolveImportScope(jsengine.StaticScopeProvider(base), execCtx)
	if got != tx {
		t.Fatalf("got %#v, want tx scope from context", got)
	}

	got = resolveImportScope(jsengine.StaticScopeProvider(base), context.Background())
	if stub, ok := got.(*resolveScopeStub); !ok || stub.name != "base" {
		t.Fatalf("fallback got %#v, want base scope", got)
	}
}

type resolveScopeStub struct {
	name string
	ctx  context.Context
}

func (s *resolveScopeStub) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *resolveScopeStub) Session() *scope.Session              { return nil }
func (s *resolveScopeStub) Transactor() scope.Transactor         { return nil }
func (s *resolveScopeStub) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *resolveScopeStub) WithContext(ctx context.Context) scope.Scope {
	out := *s
	out.ctx = ctx
	return &out
}
func (s *resolveScopeStub) Logger() *slog.Logger {
	return slog.Default()
}
