// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

type runSessionTestScope struct {
	ctx      context.Context
	session  *Session
	runCalls *int
}

func (e *runSessionTestScope) Run(fn func(Scope) error) error {
	if e.runCalls != nil {
		*e.runCalls = *e.runCalls + 1
	}
	return fn(e)
}

func (e *runSessionTestScope) Session() *Session { return e.session }

func (e *runSessionTestScope) Transactor() Transactor { return NewRunSessionTransactor(e) }

func (e *runSessionTestScope) WithContext(ctx context.Context) Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}

func (e *runSessionTestScope) Context() context.Context { return e.ctx }

func (e *runSessionTestScope) Logger() *slog.Logger { return nil }

func (e *runSessionTestScope) Config() *config.Config { return nil }

func (e *runSessionTestScope) FactoryInput() FactoryInput { return nil }

type contextTestTransaction struct {
	ctx     context.Context
	session *Session
}

func (tx *contextTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *contextTestTransaction) Session() *Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *contextTestTransaction) Savepoint(string) error           { return nil }
func (tx *contextTestTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *contextTestTransaction) ReleaseSavepoint(string) error    { return nil }

func TestContextWithTransactionAndTransactionFromContext(t *testing.T) {
	tx := &contextTestTransaction{ctx: context.Background(), session: &Session{}}
	ctx := ContextWithTransaction(context.Background(), tx)

	loaded, ok := TransactionFromContext(ctx)
	if !ok || loaded != tx {
		t.Fatalf("TransactionFromContext() = %#v, %v", loaded, ok)
	}

	loadedSession, ok := SessionFromContext(ctx)
	if !ok || loadedSession != tx.session {
		t.Fatalf("SessionFromContext() = %#v, %v", loadedSession, ok)
	}

	if loaded, ok = TransactionFromContext(context.Background()); ok || loaded != nil {
		t.Fatalf("TransactionFromContext(background) = %#v, %v", loaded, ok)
	}

	ctx = ContextWithTransaction(context.Background(), nil)
	if loaded, ok = TransactionFromContext(ctx); ok || loaded != nil {
		t.Fatalf("TransactionFromContext(nil tx) = %#v, %v", loaded, ok)
	}
}

func TestContextWithoutTransactionClearsOnlyTransactionMarker(t *testing.T) {
	tx := &contextTestTransaction{ctx: context.Background(), session: &Session{}}
	ctx := ContextWithTransaction(context.WithValue(context.Background(), "marker", "v"), tx)

	cleared := ContextWithoutTransaction(ctx)
	if loaded, ok := TransactionFromContext(cleared); ok || loaded != nil {
		t.Fatalf("TransactionFromContext(cleared) = %#v, %v", loaded, ok)
	}
	if got := cleared.Value("marker"); got != "v" {
		t.Fatalf("context marker = %#v, want v", got)
	}

	backgroundCleared := ContextWithoutTransaction(nil)
	if backgroundCleared == nil {
		t.Fatal("ContextWithoutTransaction(nil) should return non-nil context")
	}
}

func TestNewRunSessionTransactorRequiredUsesRunAndTransactionContext(t *testing.T) {
	runCalls := 0
	scope := &runSessionTestScope{ctx: context.Background(), session: &Session{}, runCalls: &runCalls}

	err := NewRunSessionTransactor(scope).Required(context.WithValue(context.Background(), "trace", "value"), func(txScope Scope, tx Transaction) error {
		if tx == nil {
			t.Fatal("expected transaction")
		}
		if got, ok := TransactionFromContext(txScope.Context()); !ok || got != tx {
			t.Fatalf("TransactionFromContext(txScope.Context()) = %#v, %v", got, ok)
		}
		if got, ok := SessionFromContext(tx.Context()); !ok || got != scope.session {
			t.Fatalf("SessionFromContext(tx.Context()) = %#v, %v", got, ok)
		}
		if tx.Session() != scope.session {
			t.Fatalf("transaction session = %#v, want %#v", tx.Session(), scope.session)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Required() error = %v", err)
	}
	if runCalls != 1 {
		t.Fatalf("Run() call count = %d, want 1", runCalls)
	}
}

func TestNewRunSessionTransactorRequiredJoinsContextTransaction(t *testing.T) {
	existing := &contextTestTransaction{ctx: context.Background(), session: &Session{}}
	runCalls := 0
	scope := &runSessionTestScope{ctx: context.Background(), session: &Session{}, runCalls: &runCalls}
	ctx := ContextWithTransaction(context.Background(), existing)

	err := NewRunSessionTransactor(scope).Required(ctx, func(txScope Scope, tx Transaction) error {
		if tx != existing {
			t.Fatalf("transaction = %#v, want %#v", tx, existing)
		}
		if got, ok := TransactionFromContext(txScope.Context()); !ok || got != existing {
			t.Fatalf("TransactionFromContext(txScope.Context()) = %#v, %v", got, ok)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Required() error = %v", err)
	}
	if runCalls != 0 {
		t.Fatalf("Run() call count = %d, want 0", runCalls)
	}
}

func TestNewRunSessionTransactorUnsupportedPropagation(t *testing.T) {
	scope := &runSessionTestScope{ctx: context.Background(), session: &Session{}}
	transactor := NewRunSessionTransactor(scope)

	if err := transactor.RequiresNew(context.Background(), func(Scope, Transaction) error { return nil }); !errors.Is(err, ErrRequiresNewUnsupported) {
		t.Fatalf("RequiresNew() error = %v", err)
	}
	if err := transactor.Nested(context.Background(), func(Scope, Transaction) error { return nil }); !errors.Is(err, ErrNestedUnsupported) {
		t.Fatalf("Nested() error = %v", err)
	}
	if err := transactor.Do(context.Background(), TransactionOptions{Propagation: Propagation("invalid")}, func(Scope, Transaction) error { return nil }); !errors.Is(err, ErrInvalidTransactionPropagation) {
		t.Fatalf("Do(invalid propagation) error = %v", err)
	}
}
