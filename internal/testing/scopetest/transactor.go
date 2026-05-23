// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scopetest

import (
	"context"
	"fmt"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type passthroughTransactor struct {
	rootScope scope.Scope
}

type passthroughTransaction struct {
	ctx     context.Context
	session *scope.Session
}

func NewPassthroughTransactor(rootScope scope.Scope) scope.Transactor {
	return passthroughTransactor{rootScope: rootScope}
}

func (t passthroughTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	propagation := opts.Propagation
	if propagation == "" {
		propagation = scope.PropagationRequired
	}

	switch propagation {
	case scope.PropagationRequired:
		if existingTx, ok := scope.TransactionFromContext(ctx); ok && existingTx != nil {
			return fn(t.rootScope.WithContext(ctx), existingTx)
		}

		txScope := t.rootScope
		if ctx != nil {
			txScope = t.rootScope.WithContext(ctx)
		}

		tx := &passthroughTransaction{session: txScope.Session()}
		tx.ctx = scope.ContextWithTransaction(txScope.Context(), tx)
		return fn(txScope.WithContext(tx.ctx), tx)
	case scope.PropagationRequiresNew:
		return scope.ErrRequiresNewUnsupported
	case scope.PropagationNested:
		return scope.ErrNestedUnsupported
	default:
		return fmt.Errorf("%w: %q", scope.ErrInvalidTransactionPropagation, propagation)
	}
}

func (t passthroughTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (t passthroughTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (t passthroughTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (tx *passthroughTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *passthroughTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *passthroughTransaction) Savepoint(name string) error {
	if tx == nil || tx.session == nil {
		return scope.ErrSessionUnavailable
	}
	return tx.session.Savepoint(name)
}

func (tx *passthroughTransaction) RollbackToSavepoint(name string) error {
	if tx == nil || tx.session == nil {
		return scope.ErrSessionUnavailable
	}
	return tx.session.RollbackToSavepoint(name)
}

func (tx *passthroughTransaction) ReleaseSavepoint(name string) error {
	if tx == nil || tx.session == nil {
		return scope.ErrSessionUnavailable
	}
	return tx.session.ReleaseSavepoint(name)
}
