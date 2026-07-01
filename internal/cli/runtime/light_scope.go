// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"context"
	"io"
	"log/slog"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type commandRuntimeScope struct {
	ctx    context.Context
	input  scope.FactoryInput
	logger *slog.Logger
}

func NewScopeWithoutDB(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &commandRuntimeScope{ctx: ctx, input: input, logger: logger}
}

func RebuildScope(runtimeScope scope.Scope, factoryInput scope.FactoryInput, logger *slog.Logger) scope.Scope {
	if runtimeScope == nil {
		return nil
	}
	if _, ok := runtimeScope.(*commandRuntimeScope); ok {
		return NewScopeWithoutDB(runtimeScope.Context(), factoryInput, logger)
	}
	return scope.NewScope(runtimeScope.Context(), factoryInput, logger)
}

func (s *commandRuntimeScope) Run(fn func(scope.Scope) error) error {
	if fn == nil {
		return nil
	}
	return fn(s)
}

func (s *commandRuntimeScope) Session() *scope.Session { return nil }

func (s *commandRuntimeScope) Transactor() scope.Transactor {
	return scope.NewRunSessionTransactor(s)
}

func (s *commandRuntimeScope) WithContext(ctx context.Context) scope.Scope {
	if s == nil {
		return nil
	}
	clone := *s
	if ctx != nil {
		clone.ctx = ctx
	}
	return &clone
}

func (s *commandRuntimeScope) Context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *commandRuntimeScope) Logger() *slog.Logger {
	if s == nil || s.logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return s.logger
}

func (s *commandRuntimeScope) FactoryInput() scope.FactoryInput {
	if s == nil {
		return nil
	}
	return s.input
}
