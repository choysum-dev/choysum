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

type registryCoverageScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *registryCoverageScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *registryCoverageScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *registryCoverageScope) Session() *scope.Session { return s.session }
func (s *registryCoverageScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &registryCoverageScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *registryCoverageScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *registryCoverageScope) Logger() *slog.Logger { return s.logger }

func TestRegistryLoadModuleApplicationWithoutMetaModuleTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "registry-no-meta.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := NewRegistry(rs)

	if app, ok := reg.loadModuleApplication("auth"); ok || app != "" {
		t.Fatalf("loadModuleApplication() = %q ok=%v, want empty without meta_module table", app, ok)
	}
}

func TestRegistryExistingStore(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "existing-store.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := NewRegistry(rs)

	if _, ok := reg.ExistingStore(""); ok {
		t.Fatal("ExistingStore('') should be false")
	}
	if _, ok := reg.ExistingStore("auth"); ok {
		t.Fatal("ExistingStore before StoreFor should be false")
	}
	created := reg.StoreFor("auth")
	got, ok := reg.ExistingStore("auth")
	if !ok || got != created {
		t.Fatalf("ExistingStore(auth) ok=%v store=%p want %p", ok, got, created)
	}
	got, ok = reg.ExistingStore("  auth  ")
	if !ok || got != created {
		t.Fatalf("ExistingStore trimmed ok=%v", ok)
	}
}

func TestRegistryListHostApplicationsWithoutMetaModuleTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "registry-hosts-no-meta.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := NewRegistry(rs)
	if hosts := reg.listHostApplications(); len(hosts) != 0 {
		t.Fatalf("listHostApplications() = %#v, want empty without meta_module table", hosts)
	}
}
