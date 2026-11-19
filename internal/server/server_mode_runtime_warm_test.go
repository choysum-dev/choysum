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

func TestFinishApplicationModeStartupRequiresRuntimeScope(t *testing.T) {
	srv := &GRPCWebServer{jsExecutor: &readyHooksMockExec{}}
	_, err := srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: true})
	if err == nil {
		t.Fatal("expected error when runtimeScope is nil")
	}
}

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

func TestFinishApplicationModeStartupNilContextFallsBack(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-nil-ctx.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &nilContextWarmScope{
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	srv := &GRPCWebServer{runtimeScope: rs, jsExecutor: &readyHooksMockExec{}}
	if _, err := srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: true}); err != nil {
		t.Fatalf("nil Context should fall back to Background: %v", err)
	}
}

// errAfterNCtx returns context.Canceled once Err has been observed n times.
type errAfterNCtx struct {
	n, calls int
}

func (c *errAfterNCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *errAfterNCtx) Done() <-chan struct{}       { return nil }
func (c *errAfterNCtx) Value(any) any               { return nil }
func (c *errAfterNCtx) Err() error {
	c.calls++
	if c.calls >= c.n {
		return context.Canceled
	}
	return nil
}

func TestFinishApplicationModeStartupCancelsAfterWarm(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-after-cancel.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Broken schema → WarmRuntimeCache fails after its own entry Err check.
	if err := db.Exec(`CREATE TABLE meta_module (id text primary key, name text, status text)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}

	// pre-check(1) + WarmRuntimeCache entry(2) + post-warm check(3) → cancel.
	rs := &warmFailScope{
		ctx:     &errAfterNCtx{n: 3},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	srv := &GRPCWebServer{runtimeScope: rs, jsExecutor: &readyHooksMockExec{}}
	_, err = srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: true})
	if err == nil {
		t.Fatal("expected cancellation after warm attempt")
	}
}

func TestFinishApplicationModeStartupCancelsDuringBackoff(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-backoff-cancel.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_module (id text primary key, name text, status text)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	rs := &warmFailScope{
		ctx:     ctx,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	srv := &GRPCWebServer{runtimeScope: rs, jsExecutor: &readyHooksMockExec{}}
	// First warm fails instantly; cancel mid 100ms backoff so waitWarmBackoff returns err.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	started := time.Now()
	_, err = srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: true})
	if err == nil {
		t.Fatal("expected cancel during warm backoff")
	}
	if time.Since(started) > 200*time.Millisecond {
		t.Fatalf("backoff cancel took too long: %v", time.Since(started))
	}
}

func TestWaitWarmBackoffCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitWarmBackoff(ctx, time.Hour); err == nil {
		t.Fatal("expected cancelled backoff error")
	}
	if err := waitWarmBackoff(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("uncancelled backoff: %v", err)
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

// nilContextWarmScope always reports a nil Context so finishApplicationModeStartup
// exercises the Background fallback.
type nilContextWarmScope struct {
	logger  *slog.Logger
	session *scope.Session
}

func (s *nilContextWarmScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *nilContextWarmScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *nilContextWarmScope) Session() *scope.Session           { return s.session }
func (s *nilContextWarmScope) Context() context.Context          { return nil }
func (s *nilContextWarmScope) Logger() *slog.Logger              { return s.logger }
func (s *nilContextWarmScope) WithContext(ctx context.Context) scope.Scope {
	return &warmFailScope{ctx: ctx, logger: s.logger, session: s.session}
}
