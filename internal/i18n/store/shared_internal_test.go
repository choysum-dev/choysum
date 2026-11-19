// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDurableRuntimeScopeStripsTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "durable.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	base := &edgeScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	if got := durableRuntimeScope(nil); got != nil {
		t.Fatal("nil scope should stay nil")
	}
	if got := durableRuntimeScope(base); got != base {
		t.Fatal("non-tx scope should be unchanged")
	}

	tx := &fakeTx{session: &scope.Session{DB: db}}
	txCtx := scope.ContextWithTransaction(context.Background(), tx)
	txScope := base.WithContext(txCtx)
	durable := durableRuntimeScope(txScope)
	if _, ok := scope.TransactionFromContext(durable.Context()); ok {
		t.Fatal("durable scope must not carry a transaction marker")
	}

	ResetSharedRegistryForTests()
	t.Cleanup(ResetSharedRegistryForTests)
	reg := RegistryFor(txScope)
	if _, ok := scope.TransactionFromContext(reg.runtimeScope.Context()); ok {
		t.Fatal("RegistryFor must store a non-transaction runtime scope")
	}
	if RegistryFor(base) != reg {
		t.Fatal("root scope must reuse registry created via tx wrapper")
	}
}

type fakeTx struct {
	session *scope.Session
}

func (t *fakeTx) Context() context.Context                 { return context.Background() }
func (t *fakeTx) Session() *scope.Session                  { return t.session }
func (t *fakeTx) Savepoint(string) error                   { return nil }
func (t *fakeTx) RollbackToSavepoint(string) error         { return nil }
func (t *fakeTx) ReleaseSavepoint(string) error            { return nil }

func TestScopesShareRuntimeDBEdges(t *testing.T) {
	if !scopesShareRuntimeDB(nil, nil) {
		t.Fatal("nil,nil should share")
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "edge.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s := &edgeScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	if scopesShareRuntimeDB(nil, s) {
		t.Fatal("nil vs scope must not share")
	}
	if !scopesShareRuntimeDB(s, s) {
		t.Fatal("same scope pointer must share")
	}
	noSess := &edgeScope{ctx: context.Background(), logger: s.logger}
	if scopesShareRuntimeDB(s, noSess) {
		t.Fatal("missing session must not share")
	}
	nilDB := &edgeScope{
		ctx: s.ctx, logger: s.logger,
		session: &scope.Session{},
	}
	if scopesShareRuntimeDB(s, nilDB) {
		t.Fatal("nil gorm DB must not share")
	}

	broken := &edgeScope{
		ctx: s.ctx, logger: s.logger,
		session: &scope.Session{DB: &gorm.DB{}},
	}
	if scopesShareRuntimeDB(s, broken) {
		t.Fatal("gorm.DB without Config must not share")
	}
	if scopesShareRuntimeDB(broken, s) {
		t.Fatal("gorm.DB without Config (left) must not share")
	}
	noPool := &edgeScope{
		ctx: s.ctx, logger: s.logger,
		session: &scope.Session{DB: &gorm.DB{Config: &gorm.Config{}}},
	}
	if scopesShareRuntimeDB(s, noPool) {
		t.Fatal("gorm.DB with nil ConnPool must not share")
	}
	if scopesShareRuntimeDB(noPool, s) {
		t.Fatal("gorm.DB with nil ConnPool (left) must not share")
	}
}

type edgeScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *edgeScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *edgeScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *edgeScope) Session() *scope.Session { return s.session }
func (s *edgeScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &edgeScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *edgeScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *edgeScope) Logger() *slog.Logger { return s.logger }
