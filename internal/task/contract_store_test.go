// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type contractStoreTestScope struct {
	ctx context.Context
	db  *gorm.DB
}

func (e *contractStoreTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *contractStoreTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *contractStoreTestScope) Session() *scope.Session {
	if e == nil || e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db}
}
func (e *contractStoreTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *contractStoreTestScope) Context() context.Context { return e.ctx }
func (e *contractStoreTestScope) Logger() *slog.Logger     { return nil }
func (e *contractStoreTestScope) Config() *config.Config   { return nil }
func (e *contractStoreTestScope) FactoryInput() scope.FactoryInput {
	return nil
}

func newContractStoreTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestTaskDBUsesContextScopeBeforeRuntimeScopeSession(t *testing.T) {
	baseDB := newContractStoreTestDB(t, "base.db")
	ctxDB := newContractStoreTestDB(t, "ctx.db")
	runtimeScope := &contractStoreTestScope{ctx: context.Background(), db: baseDB}
	ctx := scope.ContextWithScope(context.Background(), &contractStoreTestScope{ctx: context.Background(), db: ctxDB})

	db, err := taskDB(runtimeScope, ctx)
	if err != nil {
		t.Fatalf("taskDB() error = %v", err)
	}
	if err := db.Exec("CREATE TABLE ctx_only (id integer)").Error; err != nil {
		t.Fatalf("create ctx_only: %v", err)
	}
	if !ctxDB.Migrator().HasTable("ctx_only") {
		t.Fatal("expected context-bound session to receive write")
	}
	if baseDB.Migrator().HasTable("ctx_only") {
		t.Fatal("expected env session not to be used when context session exists")
	}
}

func TestTaskDBFallsBackToRuntimeScopeSession(t *testing.T) {
	baseDB := newContractStoreTestDB(t, "base.db")
	runtimeScope := &contractStoreTestScope{ctx: context.Background(), db: baseDB}

	db, err := taskDB(runtimeScope, nil)
	if err != nil {
		t.Fatalf("taskDB() error = %v", err)
	}
	if err := db.Exec("CREATE TABLE env_only (id integer)").Error; err != nil {
		t.Fatalf("create env_only: %v", err)
	}
	if !baseDB.Migrator().HasTable("env_only") {
		t.Fatal("expected env session to receive write")
	}
}

func TestTaskDBReturnsUnavailableWithoutSession(t *testing.T) {
	runtimeScope := &contractStoreTestScope{ctx: context.Background()}

	_, err := taskDB(runtimeScope, context.Background())
	if err == nil {
		t.Fatal("expected taskDB to fail when no session is available")
	}
	if err != errTaskStoreUnavailable {
		t.Fatalf("taskDB() error = %v, want %v", err, errTaskStoreUnavailable)
	}
}
