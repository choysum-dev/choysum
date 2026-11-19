// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFinishApplicationModeStartupWarnsWhenWarmFails(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-fail.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Broken meta_module schema makes WarmRuntimeCache return an error every attempt.
	if err := db.Exec(`CREATE TABLE meta_module (id text primary key, name text, status text)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	rs := &warmFailScope{
		ctx:     context.Background(),
		logger:  logger,
		session: &scope.Session{DB: db},
	}
	srv := &GRPCWebServer{
		runtimeScope: rs,
		jsExecutor:   &readyHooksMockExec{},
	}
	_, err = srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: true})
	if err != nil {
		t.Fatalf("finishApplicationModeStartup: %v", err)
	}
	out := logBuf.String()
	if !strings.Contains(out, "terminology cache warm retry") {
		t.Fatalf("expected warm retry log, got %q", out)
	}
	if !strings.Contains(out, "terminology cache warm incomplete") {
		t.Fatalf("expected warm incomplete log, got %q", out)
	}
}

func TestFinishApplicationModeStartupAbortsWarmBackoffOnCancel(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-cancel.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_module (id text primary key, name text, status text)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rs := &warmFailScope{
		ctx:     ctx,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	srv := &GRPCWebServer{
		runtimeScope: rs,
		jsExecutor:   &readyHooksMockExec{},
	}
	started := time.Now()
	_, err = srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: true})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if time.Since(started) > 150*time.Millisecond {
		t.Fatalf("cancelled warm backoff took too long: %v", time.Since(started))
	}
}

type warmFailScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *warmFailScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *warmFailScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *warmFailScope) Session() *scope.Session { return s.session }
func (s *warmFailScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &warmFailScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *warmFailScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *warmFailScope) Logger() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
