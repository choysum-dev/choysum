// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type opsTestScope struct {
	ctx        context.Context
	logger     *slog.Logger
	cfg        *config.Config
	session    *scope.Session
	transactor scope.Transactor
}

func (s *opsTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *opsTestScope) Transactor() scope.Transactor {
	if s.transactor != nil {
		return s.transactor
	}
	return scopetest.NewPassthroughTransactor(s)
}
func (s *opsTestScope) Session() *scope.Session { return s.session }
func (s *opsTestScope) WithContext(ctx context.Context) scope.Scope {
	cp := *s
	cp.ctx = ctx
	return &cp
}
func (s *opsTestScope) Context() context.Context { return s.ctx }
func (s *opsTestScope) Logger() *slog.Logger     { return s.logger }
func (s *opsTestScope) Config() *config.Config   { return s.cfg }
func (s *opsTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.cfg)
}

type testLocker struct{}

func (testLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (testLocker) Renew(context.Context, string, string, time.Duration) error {
	return nil
}
func (testLocker) Release(context.Context, string, string) error { return nil }

func newOpsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "ops.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&modmeta.ModuleOperation{},
		&modmeta.ModuleCatalog{},
		&modmeta.ModuleCatalogDepend{},
		&modmeta.ModuleSourceSnapshot{},
		&pkgmeta.Module{},
		&metaAppSetting{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func newOpsTestScope(t *testing.T, db *gorm.DB) *opsTestScope {
	t.Helper()
	return &opsTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: t.TempDir()},
		session: &scope.Session{DB: db},
	}
}

type recordingBus struct {
	events []bus.Event
}

func (b *recordingBus) Publish(_ context.Context, ev bus.Event) error {
	b.events = append(b.events, ev)
	return nil
}
func (b *recordingBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return nopSub{}, nil
}

type nopSub struct{}

func (nopSub) Close() error { return nil }

type mapResolver struct {
	peek    map[string]*pkgmeta.Module
	load    map[string]*pkgmeta.Module
	peekErr error
	loadErr error
}

func (r *mapResolver) Peek(_ context.Context, name string) (*pkgmeta.Module, error) {
	if r.peekErr != nil {
		return nil, r.peekErr
	}
	if r.peek == nil {
		return nil, nil
	}
	return r.peek[name], nil
}
func (r *mapResolver) Load(name string) (*pkgmeta.Module, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	if r.load == nil {
		return nil, nil
	}
	return r.load[name], nil
}

type stubLifecycle struct {
	installErr          error
	upgradeErr          error
	uninstallErr        error
	installs            []string
	upgrades            []string
	uninstalls          []string
	lastWithDemo        bool
	lastSkipWebShell    bool
	lastWebShellProfile plan.WebShellProfile
}

func (s *stubLifecycle) Install(_ context.Context, req lifecycle.InstallRequest) error {
	s.installs = append(s.installs, req.Name)
	s.lastWithDemo = req.WithDemo
	s.lastSkipWebShell = req.SkipWebShell
	s.lastWebShellProfile = req.WebShellProfile
	return s.installErr
}
func (s *stubLifecycle) Upgrade(_ context.Context, req lifecycle.UpgradeRequest) error {
	s.upgrades = append(s.upgrades, req.Input)
	s.lastWithDemo = req.WithDemo
	s.lastSkipWebShell = req.SkipWebShell
	s.lastWebShellProfile = req.WebShellProfile
	return s.upgradeErr
}
func (s *stubLifecycle) Uninstall(_ context.Context, req lifecycle.UninstallRequest) error {
	s.uninstalls = append(s.uninstalls, req.Name)
	return s.uninstallErr
}
func (s *stubLifecycle) SchemaPlan(context.Context, string) (schema.SchemaPlan, error) {
	return schema.SchemaPlan{}, nil
}

var (
	_ plan.Resolver     = (*mapResolver)(nil)
	_ bus.EventBus      = (*recordingBus)(nil)
	_ lifecycle.Service = (*stubLifecycle)(nil)
	_ statepkg.Locker   = testLocker{}
)
