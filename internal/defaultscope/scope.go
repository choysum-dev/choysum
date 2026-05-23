// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"log/slog"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type defaultScope struct {
	ctx    context.Context
	input  scope.FactoryInput
	logger *slog.Logger

	db      *gorm.DB
	session *scope.Session
	tx      scope.Transaction
}

const (
	defaultScopeUnavailableReasonDatabaseInputUnavailable = "database input unavailable"
	defaultScopeUnavailableReasonMissingDatabaseConfig    = "missing database dialect or dsn"
)

func (e *defaultScope) Run(fn func(scope scope.Scope) error) error {
	return e.Transactor().Required(e.ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		return fn(txScope)
	})
}

func (e *defaultScope) Session() *scope.Session {
	if e.session != nil {
		return e.session
	}
	if e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db.WithContext(e.ctx)}
}

func (e *defaultScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	newScope := &defaultScope{
		ctx:     ctx,
		input:   e.input,
		logger:  e.logger,
		db:      e.db,
		session: nil,
		tx:      nil,
	}
	if tx, ok := scope.TransactionFromContext(ctx); ok {
		newScope.tx = tx
		newScope.session = tx.Session()
		return newScope
	}

	return newScope
}

func (e *defaultScope) Context() context.Context {
	return e.ctx
}

func (e *defaultScope) Logger() *slog.Logger {
	return e.logger
}

func (e *defaultScope) FactoryInput() scope.FactoryInput {
	if e == nil {
		return nil
	}
	return e.input
}

func newDefaultScopeFromInputWithReason(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) (scope.Scope, string) {
	dbOpts, ok := scope.DatabaseRuntimeOptionsFromInput(input)
	if !ok {
		return nil, defaultScopeUnavailableReasonDatabaseInputUnavailable
	}
	if strings.TrimSpace(dbOpts.Dialect) == "" || strings.TrimSpace(dbOpts.DSN) == "" {
		return nil, defaultScopeUnavailableReasonMissingDatabaseConfig
	}
	db := newDb(ctx, dbOpts, logger)
	return &defaultScope{
		ctx:     ctx,
		logger:  logger,
		input:   input,
		db:      db,
		session: nil,
		tx:      nil,
	}, ""
}

func newDefaultScopeFromInput(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
	runtimeScope, _ := newDefaultScopeFromInputWithReason(ctx, input, logger)
	return runtimeScope
}

// NewDefaultScope builds the default scope implementation from a factory input.
func NewDefaultScope(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
	return newDefaultScopeFromInput(ctx, input, logger)
}
