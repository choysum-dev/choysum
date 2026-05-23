// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"errors"
	"fmt"

	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
)

type defaultTransactor struct {
	scope *defaultScope
}

type transactionState struct {
	rollbackCause error
}

type defaultTransaction struct {
	ctx     context.Context
	session *scope.Session
	state   *transactionState
}

func (e *defaultScope) Transactor() scope.Transactor {
	return &defaultTransactor{scope: e}
}

func (u *defaultTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	if fn == nil {
		return scope.ErrTransactionHandlerRequired
	}
	if ctx == nil {
		ctx = u.scope.Context()
	}

	propagation := opts.Propagation
	if propagation == "" {
		propagation = scope.PropagationRequired
	}

	currentTx, hasCurrent := resolveCurrentTransaction(ctx, u.scope)
	switch propagation {
	case scope.PropagationRequired:
		if hasCurrent {
			return u.runJoined(ctx, currentTx, fn)
		}
		return u.runFresh(ctx, fn)
	case scope.PropagationRequiresNew:
		return u.runFresh(ctx, fn)
	case scope.PropagationNested:
		if !hasCurrent {
			return u.runFresh(ctx, fn)
		}
		return u.runNested(ctx, currentTx, opts, fn)
	default:
		return fmt.Errorf("%w: %q", scope.ErrInvalidTransactionPropagation, propagation)
	}
}

func (u *defaultTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (u *defaultTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (u *defaultTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func resolveCurrentTransaction(ctx context.Context, currentScope *defaultScope) (scope.Transaction, bool) {
	if tx, ok := scope.TransactionFromContext(ctx); ok {
		return tx, true
	}
	if currentScope != nil && currentScope.tx != nil {
		return currentScope.tx, true
	}
	return nil, false
}

func (u *defaultTransactor) runFresh(ctx context.Context, fn scope.TxFunc) error {
	if u.scope == nil || u.scope.db == nil {
		return scope.ErrSessionUnavailable
	}
	session := &scope.Session{DB: u.scope.db.WithContext(ctx).Begin()}
	if session.DB == nil {
		return scope.ErrSessionUnavailable
	}
	if session.Error != nil {
		return session.Error
	}

	tx := &defaultTransaction{session: session, state: &transactionState{}}
	tx.ctx = scope.ContextWithTransaction(ctx, tx)
	txScope := u.scope.WithContext(tx.ctx)

	err := fn(txScope, tx)
	if err != nil {
		_ = session.Rollback()
		return err
	}
	if cause := tx.rollbackCause(); cause != nil {
		_ = session.Rollback()
		return cause
	}
	return session.Commit().Error
}

func (u *defaultTransactor) runJoined(ctx context.Context, currentTx scope.Transaction, fn scope.TxFunc) error {
	joinCtx := ctx
	if joinCtx == nil {
		joinCtx = currentTx.Context()
	}
	joinCtx = scope.ContextWithTransaction(joinCtx, currentTx)
	txScope := u.scope.WithContext(joinCtx)
	err := fn(txScope, currentTx)
	if err != nil {
		if localTx, ok := currentTx.(*defaultTransaction); ok {
			localTx.markRollback(err)
		}
	}
	return err
}

func (u *defaultTransactor) runNested(ctx context.Context, currentTx scope.Transaction, opts scope.TransactionOptions, fn scope.TxFunc) error {
	savepointName := opts.SavepointName
	if savepointName == "" {
		savepointName = "sp_" + xid.New().String()
	}
	if err := currentTx.Savepoint(savepointName); err != nil {
		return err
	}

	nestedTx := &defaultTransaction{session: currentTx.Session(), state: &transactionState{}}
	nestedCtx := ctx
	if nestedCtx == nil {
		nestedCtx = currentTx.Context()
	}
	nestedTx.ctx = scope.ContextWithTransaction(nestedCtx, nestedTx)
	txScope := u.scope.WithContext(nestedTx.ctx)

	err := fn(txScope, nestedTx)
	if err != nil {
		rollbackErr := currentTx.RollbackToSavepoint(savepointName)
		if rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	if cause := nestedTx.rollbackCause(); cause != nil {
		rollbackErr := currentTx.RollbackToSavepoint(savepointName)
		if rollbackErr != nil {
			return errors.Join(cause, rollbackErr)
		}
		return cause
	}
	return currentTx.ReleaseSavepoint(savepointName)
}

func (tx *defaultTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *defaultTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *defaultTransaction) Savepoint(name string) error {
	if tx == nil {
		return scope.ErrSessionUnavailable
	}
	return tx.session.Savepoint(name)
}

func (tx *defaultTransaction) RollbackToSavepoint(name string) error {
	if tx == nil {
		return scope.ErrSessionUnavailable
	}
	return tx.session.RollbackToSavepoint(name)
}

func (tx *defaultTransaction) ReleaseSavepoint(name string) error {
	if tx == nil {
		return scope.ErrSessionUnavailable
	}
	return tx.session.ReleaseSavepoint(name)
}

func (tx *defaultTransaction) markRollback(err error) {
	if tx == nil || tx.state == nil || err == nil {
		return
	}
	if tx.state.rollbackCause == nil {
		tx.state.rollbackCause = err
	}
}

func (tx *defaultTransaction) rollbackCause() error {
	if tx == nil || tx.state == nil {
		return nil
	}
	return tx.state.rollbackCause
}
