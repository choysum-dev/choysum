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
	"sync/atomic"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/jsengine"
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

// armableCancelCtx stays live until Arm; then Err/Done report cancellation.
type armableCancelCtx struct {
	armed atomic.Bool
}

func (c *armableCancelCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *armableCancelCtx) Value(any) any               { return nil }
func (c *armableCancelCtx) Err() error {
	if c.armed.Load() {
		return context.Canceled
	}
	return nil
}
func (c *armableCancelCtx) Done() <-chan struct{} {
	if c.armed.Load() {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return nil
}
func (c *armableCancelCtx) Arm() { c.armed.Store(true) }

// finalWarmCancelScope arms the context on the 3rd WarmRuntimeCache Session() so
// attempt 3's pre-check and warm entry still pass, and the post-warm Err check is
// the first observer of cancellation (guards against removing that check).
type finalWarmCancelScope struct {
	arm              *armableCancelCtx
	logger           *slog.Logger
	session          *scope.Session
	warmSessionCalls int
}

func (s *finalWarmCancelScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *finalWarmCancelScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *finalWarmCancelScope) Session() *scope.Session  { return s.session }
func (s *finalWarmCancelScope) Context() context.Context { return s.arm }
func (s *finalWarmCancelScope) Logger() *slog.Logger     { return s.logger }
func (s *finalWarmCancelScope) WithContext(ctx context.Context) scope.Scope {
	return &finalWarmCancelChild{parent: s, ctx: ctx}
}

type finalWarmCancelChild struct {
	parent *finalWarmCancelScope
	ctx    context.Context
}

func (c *finalWarmCancelChild) Run(fn func(scope.Scope) error) error { return fn(c) }
func (c *finalWarmCancelChild) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(c)
}
func (c *finalWarmCancelChild) Context() context.Context { return c.ctx }
func (c *finalWarmCancelChild) Logger() *slog.Logger     { return c.parent.logger }
func (c *finalWarmCancelChild) WithContext(ctx context.Context) scope.Scope {
	return &finalWarmCancelChild{parent: c.parent, ctx: ctx}
}
func (c *finalWarmCancelChild) Session() *scope.Session {
	c.parent.warmSessionCalls++
	// WarmRuntimeCache: entry Err check, then WithContext(ctx).Session().
	// Arm after the final attempt's Session so only the post-warm check sees cancel.
	if c.parent.warmSessionCalls >= 3 {
		c.parent.arm.Arm()
	}
	return c.parent.session
}

type countingExec struct {
	readyHooksMockExec
	starts  int
	reloads int
}

func (e *countingExec) Start() error {
	e.starts++
	return nil
}
func (e *countingExec) Reload(scripts ...*jsengine.JsScript) error {
	e.reloads++
	return nil
}

func TestFinishApplicationModeStartupCancelsAfterFinalWarm(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-final-cancel.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Broken schema → every WarmRuntimeCache fails after Session() is taken.
	if err := db.Exec(`CREATE TABLE meta_module (id text primary key, name text, status text)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}

	arm := &armableCancelCtx{}
	rs := &finalWarmCancelScope{
		arm:     arm,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	exec := &countingExec{}
	srv := &GRPCWebServer{runtimeScope: rs, jsExecutor: exec}
	// Reload=false exercises Start; without the post-warm cancel guard, attempt 3
	// would log incomplete and proceed to Start.
	_, err = srv.finishApplicationModeStartup(applicationModeStartupResult{Reload: false})
	if err == nil {
		t.Fatal("expected cancellation after final warm attempt")
	}
	if exec.starts != 0 || exec.reloads != 0 {
		t.Fatalf("executor activated after cancel: starts=%d reloads=%d", exec.starts, exec.reloads)
	}
	if rs.warmSessionCalls != 3 {
		t.Fatalf("warm Session calls = %d, want 3 (final attempt)", rs.warmSessionCalls)
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
