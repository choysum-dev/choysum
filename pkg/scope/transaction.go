// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import (
	"context"
	"errors"
	"fmt"
)

type Propagation string

const (
	PropagationRequired    Propagation = "required"
	PropagationRequiresNew Propagation = "requires_new"
	PropagationNested      Propagation = "nested"
)

var (
	ErrSessionUnavailable            = errors.New("db session is unavailable")
	ErrTransactionHandlerRequired    = errors.New("transaction callback is required")
	ErrTransactorUnavailable         = errors.New("transactor is unavailable")
	ErrRequiresNewUnsupported        = errors.New("requires-new propagation is not supported")
	ErrNestedUnsupported             = errors.New("nested propagation is not supported")
	ErrInvalidTransactionPropagation = errors.New("invalid transaction propagation")
)

type Transaction interface {
	Context() context.Context
	Session() *Session
	Savepoint(name string) error
	RollbackToSavepoint(name string) error
	ReleaseSavepoint(name string) error
}

type TxFunc func(Scope, Transaction) error

type TransactionOptions struct {
	Propagation   Propagation
	SavepointName string
}

type Transactor interface {
	Do(ctx context.Context, opts TransactionOptions, fn TxFunc) error
	Required(ctx context.Context, fn TxFunc) error
	RequiresNew(ctx context.Context, fn TxFunc) error
	Nested(ctx context.Context, fn TxFunc) error
}

// NewRunSessionTransactor returns an explicit transactor adapter for scopes
// that only provide Run/Session-backed local transaction behavior.
func NewRunSessionTransactor(scope Scope) Transactor {
	return runSessionTransactor{scope: scope}
}

type transactionContextKey struct{}

func ContextWithTransaction(ctx context.Context, tx Transaction) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

func TransactionFromContext(ctx context.Context) (Transaction, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(transactionContextKey{}).(Transaction)
	return tx, ok && tx != nil
}

type runSessionTransactor struct {
	scope Scope
}

type runSessionTransaction struct {
	ctx     context.Context
	session *Session
}

func (u runSessionTransactor) Do(ctx context.Context, opts TransactionOptions, fn TxFunc) error {
	if fn == nil {
		return ErrTransactionHandlerRequired
	}
	if u.scope == nil {
		return ErrTransactorUnavailable
	}
	propagation := opts.Propagation
	if propagation == "" {
		propagation = PropagationRequired
	}

	switch propagation {
	case PropagationRequired:
		if tx, ok := TransactionFromContext(ctx); ok {
			return fn(u.scope.WithContext(ContextWithTransaction(ctx, tx)), tx)
		}
		rootScope := u.scope
		if ctx != nil {
			rootScope = u.scope.WithContext(ctx)
		}
		return rootScope.Run(func(txScope Scope) error {
			tx := &runSessionTransaction{session: txScope.Session()}
			tx.ctx = ContextWithTransaction(txScope.Context(), tx)
			return fn(txScope.WithContext(tx.ctx), tx)
		})
	case PropagationRequiresNew:
		return ErrRequiresNewUnsupported
	case PropagationNested:
		return ErrNestedUnsupported
	default:
		return fmt.Errorf("%w: %q", ErrInvalidTransactionPropagation, propagation)
	}
}

func (u runSessionTransactor) Required(ctx context.Context, fn TxFunc) error {
	return u.Do(ctx, TransactionOptions{Propagation: PropagationRequired}, fn)
}

func (u runSessionTransactor) RequiresNew(ctx context.Context, fn TxFunc) error {
	return u.Do(ctx, TransactionOptions{Propagation: PropagationRequiresNew}, fn)
}

func (u runSessionTransactor) Nested(ctx context.Context, fn TxFunc) error {
	return u.Do(ctx, TransactionOptions{Propagation: PropagationNested}, fn)
}

func (tx *runSessionTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *runSessionTransaction) Session() *Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *runSessionTransaction) Savepoint(name string) error {
	if tx == nil {
		return ErrSessionUnavailable
	}
	return tx.session.Savepoint(name)
}

func (tx *runSessionTransaction) RollbackToSavepoint(name string) error {
	if tx == nil {
		return ErrSessionUnavailable
	}
	return tx.session.RollbackToSavepoint(name)
}

func (tx *runSessionTransaction) ReleaseSavepoint(name string) error {
	if tx == nil {
		return ErrSessionUnavailable
	}
	return tx.session.ReleaseSavepoint(name)
}
