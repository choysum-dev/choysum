// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type hostProvisionTestScope struct {
	ctx    context.Context
	db     *gorm.DB
	config *config.Config
	logger *slog.Logger
}

func (e *hostProvisionTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *hostProvisionTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *hostProvisionTestScope) Session() *scope.Session {
	if e == nil || e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db}
}
func (e *hostProvisionTestScope) WithContext(ctx context.Context) scope.Scope {
	return &hostProvisionTestScope{ctx: ctx, db: e.db, config: e.config, logger: e.logger}
}
func (e *hostProvisionTestScope) Context() context.Context {
	if e != nil && e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *hostProvisionTestScope) Logger() *slog.Logger {
	if e != nil && e.logger != nil {
		return e.logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *hostProvisionTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}
func (e *hostProvisionTestScope) Config() *config.Config {
	if e != nil && e.config != nil {
		return e.config
	}
	return &config.Config{}
}

func TestEnsureHostProvisionSkipsWhenSetupPending(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "host_provision.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_module (
		id TEXT PRIMARY KEY, name TEXT, status TEXT, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}
	if err := db.Exec(`CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY, key TEXT NOT NULL UNIQUE, value TEXT, created_at DATETIME, updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create auth_app_setting: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}

	runtimeScope := &hostProvisionTestScope{ctx: context.Background(), db: db}
	phase, err := provision.Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != provision.PhaseSetupPending {
		t.Fatalf("phase=%q want setup_pending", phase)
	}

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
}

func TestEnsureHostProvisionSkipsWithoutSession(t *testing.T) {
	srv := &GRPCWebServer{runtimeScope: &hostProvisionTestScope{ctx: context.Background(), db: nil}}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
}
