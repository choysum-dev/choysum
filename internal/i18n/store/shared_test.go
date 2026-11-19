// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRegistryForNilScopeKeepsExisting(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "nil-keep.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	base := &sharedScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := store.RegistryFor(base)
	reg.RememberModuleApplication("auth", "auth")
	if got := store.RegistryFor(nil); got != reg {
		t.Fatal("RegistryFor(nil) must keep the existing warmed registry")
	}
	unusable := &sharedScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if got := store.RegistryFor(unusable); got != reg {
		t.Fatal("RegistryFor with unusable pool must keep the existing warmed registry")
	}
	app, ok := reg.ApplicationForModule("auth")
	if !ok || app != "auth" {
		t.Fatalf("cache after RegistryFor(nil) = %q ok=%v", app, ok)
	}
}

func TestRegistryForKeepsCacheAcrossWithContext(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shared-reg.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	base := &sharedScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg1 := store.RegistryFor(base)
	reg1.RememberModuleApplication("auth", "auth")

	wrapped := base.WithContext(context.Background())
	if wrapped == base {
		t.Fatal("expected WithContext to allocate a distinct scope wrapper")
	}
	reg2 := store.RegistryFor(wrapped)
	if reg1 != reg2 {
		t.Fatal("RegistryFor must keep the warmed registry across WithContext wrappers")
	}
	app, ok := reg2.ApplicationForModule("auth")
	if !ok || app != "auth" {
		t.Fatalf("ApplicationForModule after WithContext = %q ok=%v", app, ok)
	}
}

func TestRegistryForReplacesOnDistinctDB(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db1, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "a.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open a: %v", err)
	}
	db2, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "b.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open b: %v", err)
	}
	reg1 := store.RegistryFor(&sharedScope{
		ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db1},
	})
	reg2 := store.RegistryFor(&sharedScope{
		ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db2},
	})
	if reg1 == reg2 {
		t.Fatal("distinct SQL pools must not share a Registry")
	}
}

type sharedScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *sharedScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *sharedScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *sharedScope) Session() *scope.Session { return s.session }
func (s *sharedScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &sharedScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *sharedScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *sharedScope) Logger() *slog.Logger { return s.logger }
