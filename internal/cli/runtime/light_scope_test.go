// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type lightScopeInput struct {
	environment string
}

func (i lightScopeInput) Environment() string {
	return i.environment
}

type lightScopeStub struct {
	ctx    context.Context
	input  scope.FactoryInput
	logger *slog.Logger
}

func (s *lightScopeStub) Run(fn func(scope.Scope) error) error {
	if fn == nil {
		return nil
	}
	return fn(s)
}

func (s *lightScopeStub) Session() *scope.Session {
	return nil
}

func (s *lightScopeStub) Transactor() scope.Transactor {
	return scope.NewRunSessionTransactor(s)
}

func (s *lightScopeStub) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	if ctx != nil {
		clone.ctx = ctx
	}
	return &clone
}

func (s *lightScopeStub) Context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *lightScopeStub) Logger() *slog.Logger {
	if s == nil || s.logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return s.logger
}

func (s *lightScopeStub) FactoryInput() scope.FactoryInput {
	if s == nil {
		return nil
	}
	return s.input
}

func TestNewScopeWithoutDBDefaultsAndMethods(t *testing.T) {
	runtimeScope := NewScopeWithoutDB(context.TODO(), nil, nil)
	if runtimeScope == nil {
		t.Fatal("NewScopeWithoutDB() returned nil")
	}

	if runtimeScope.Context() == nil {
		t.Fatal("Context() should never return nil")
	}
	if runtimeScope.Logger() == nil {
		t.Fatal("Logger() should never return nil")
	}
	if runtimeScope.Session() != nil {
		t.Fatal("Session() should be nil for lightweight scope")
	}
	if runtimeScope.Transactor() == nil {
		t.Fatal("Transactor() should be available")
	}

	ctx := context.WithValue(context.Background(), "trace", "x")
	withCtx := runtimeScope.WithContext(ctx)
	if withCtx.Context().Value("trace") != "x" {
		t.Fatal("WithContext() should replace runtime scope context when ctx is non-nil")
	}
	called := false
	if err := runtimeScope.Run(func(scope.Scope) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("Run() callback was not called")
	}

	var nilScope *commandRuntimeScope
	if nilScope.Context() == nil {
		t.Fatal("nil receiver Context() should return background context")
	}
	if nilScope.Logger() == nil {
		t.Fatal("nil receiver Logger() should return discard logger")
	}
	if nilScope.FactoryInput() != nil {
		t.Fatal("nil receiver FactoryInput() should return nil")
	}
}

func TestRebuildScope(t *testing.T) {
	t.Run("nil runtime scope", func(t *testing.T) {
		if rebuilt := RebuildScope(nil, lightScopeInput{environment: "any"}, nil); rebuilt != nil {
			t.Fatalf("RebuildScope(nil) = %#v, want nil", rebuilt)
		}
	})

	t.Run("command runtime scope", func(t *testing.T) {
		base := NewScopeWithoutDB(context.Background(), lightScopeInput{environment: "base"}, nil)
		rebuilt := RebuildScope(base, lightScopeInput{environment: "rebuilt"}, nil)
		if rebuilt == nil {
			t.Fatal("RebuildScope() returned nil")
		}
		input := scope.FactoryInputFromScope(rebuilt)
		if input == nil || input.Environment() != "rebuilt" {
			t.Fatalf("RebuildScope() input environment = %v, want rebuilt", input)
		}
	})

	t.Run("non-command runtime scope", func(t *testing.T) {
		envName := "runtime-light-scope-test"
		scope.Register(envName, func(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
			return NewScopeWithoutDB(ctx, input, logger)
		})

		base := &lightScopeStub{
			ctx:    context.Background(),
			input:  lightScopeInput{environment: envName},
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		}
		rebuilt := RebuildScope(base, lightScopeInput{environment: envName}, base.Logger())
		if rebuilt == nil {
			t.Fatal("RebuildScope() returned nil for non-command scope")
		}
		if rebuilt == base {
			t.Fatal("RebuildScope() should return a rebuilt scope instance")
		}
	})
}
