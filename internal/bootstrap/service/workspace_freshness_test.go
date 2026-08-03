// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type freshnessTestScope struct {
	ctx    context.Context
	db     *gorm.DB
	config *config.Config
	logger *slog.Logger
}

func (e *freshnessTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *freshnessTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *freshnessTestScope) Session() *scope.Session {
	if e == nil || e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db}
}

func (e *freshnessTestScope) WithContext(ctx context.Context) scope.Scope {
	return &freshnessTestScope{ctx: ctx, db: e.db, config: e.config, logger: e.logger}
}

func (e *freshnessTestScope) Context() context.Context {
	if e != nil && e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *freshnessTestScope) Logger() *slog.Logger {
	if e != nil && e.logger != nil {
		return e.logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *freshnessTestScope) Config() *config.Config {
	if e != nil && e.config != nil {
		return e.config
	}
	return &config.Config{}
}

func (e *freshnessTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newFreshnessTestCoordinator(t *testing.T) (*coordinator, *gorm.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "workspace_freshness.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		db:     db,
		config: &config.Config{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	return &coordinator{runtimeScope: runtimeScope}, db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec sql %q failed: %v", sql, err)
	}
}

func TestDefaultCheckWorkspaceFreshnessFreshWhenAnchorsMissing(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)

	err := c.defaultCheckWorkspaceFreshness(context.Background())
	if err != nil {
		t.Fatalf("defaultCheckWorkspaceFreshness() error = %v, want nil", err)
	}
}

func TestDefaultCheckWorkspaceFreshnessNonFreshWhenModuleAnchorExists(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)

	mustExec(t, db, "CREATE TABLE meta_module (id TEXT, deleted_at DATETIME)")
	mustExec(t, db, "INSERT INTO meta_module(id, deleted_at) VALUES(?, NULL)", "mod-1")

	err := c.defaultCheckWorkspaceFreshness(context.Background())
	if err == nil {
		t.Fatal("expected non-fresh error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeWorkspaceNotFresh {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeWorkspaceNotFresh)
	}
}

func TestDefaultCheckWorkspaceFreshnessNonFreshWhenAuthModelAnchorExists(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)

	mustExec(t, db, "CREATE TABLE meta_model (id TEXT, application TEXT, name TEXT, deleted_at DATETIME)")
	mustExec(t, db, "INSERT INTO meta_model(id, application, name, deleted_at) VALUES(?, ?, ?, NULL)", "model-1", "auth", "User")

	err := c.defaultCheckWorkspaceFreshness(context.Background())
	if err == nil {
		t.Fatal("expected non-fresh error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeWorkspaceNotFresh {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeWorkspaceNotFresh)
	}
}

func TestDefaultCheckWorkspaceFreshnessNonFreshWhenAuthAdminAnchorExists(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)

	mustExec(t, db, "CREATE TABLE meta_model_data (id TEXT, module TEXT, name TEXT, deleted_at DATETIME)")
	mustExec(t, db, "INSERT INTO meta_model_data(id, module, name, deleted_at) VALUES(?, ?, ?, NULL)", "data-1", "auth", "user_admin")

	err := c.defaultCheckWorkspaceFreshness(context.Background())
	if err == nil {
		t.Fatal("expected non-fresh error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeWorkspaceNotFresh {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeWorkspaceNotFresh)
	}
}

func TestDefaultCheckWorkspaceFreshnessReturnsGateErrorOnQueryFailure(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)

	// Intentionally omit deleted_at so GORM soft-delete condition causes a query error.
	mustExec(t, db, "CREATE TABLE meta_module (id TEXT)")

	err := c.defaultCheckWorkspaceFreshness(context.Background())
	if err == nil {
		t.Fatal("expected gate error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeGateError {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeGateError)
	}
}

func TestRejectAlreadyInstalledAuthModuleReturnsWorkspaceNotFresh(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)

	mustExec(t, db, "CREATE TABLE meta_module (id TEXT, name TEXT, deleted_at DATETIME)")
	mustExec(t, db, "INSERT INTO meta_module(id, name, deleted_at) VALUES(?, ?, NULL)", "mod-1", "auth")

	err := c.rejectAlreadyInstalledAuthModule(context.Background())
	if err == nil {
		t.Fatal("expected non-fresh error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeWorkspaceNotFresh {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeWorkspaceNotFresh)
	}
}

func TestRejectAlreadyInstalledAuthModuleReturnsRuntimePrepareOnQueryFailure(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)

	// Intentionally omit deleted_at so GORM soft-delete condition causes a query error.
	mustExec(t, db, "CREATE TABLE meta_module (id TEXT, name TEXT)")

	err := c.rejectAlreadyInstalledAuthModule(context.Background())
	if err == nil {
		t.Fatal("expected runtime prepare error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeRuntimePrepare {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeRuntimePrepare)
	}
}
