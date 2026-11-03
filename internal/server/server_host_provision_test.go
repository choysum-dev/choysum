// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/internal/server/runplan"
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

func newHostProvisionDB(t *testing.T) (*hostProvisionTestScope, *gorm.DB) {
	t.Helper()
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
	cfg := &config.Config{
		DistPath: t.TempDir(),
		Compile:  &config.CompileConfig{BundleMode: "bundle"},
		Db:       &config.DbConfig{Dialect: "sqlite"},
	}
	return &hostProvisionTestScope{ctx: context.Background(), db: db, config: cfg}, db
}

func TestEnsureHostProvisionSkipsWhenSetupPending(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}

	phase, err := provision.Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != provision.PhaseSetupPending {
		t.Fatalf("phase=%q want setup_pending", phase)
	}

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.applyPlannedDecision(nil, runplan.RunDecision{RunMode: runplan.RunModeApplication, Reason: "seeded"})
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

func TestEnsureHostProvisionFallsBackToBootstrapWhenInstallFails(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		t.Fatalf("expected bootstrap before failed provision, got %q", srv.runState.mode())
	}
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		return provision.Result{}, errors.New("forced provision failure")
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		t.Fatalf("expected bootstrap fallback after failed provision, got %q", srv.runState.mode())
	}
}

func TestEnsureHostProvisionSkipsWhenModulesPathEmpty(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	called := false
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		called = true
		return provision.Result{}, errors.New("must not run")
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if called {
		t.Fatal("host provision must not run without local modules")
	}
}

func seedLocalWebModule(t *testing.T, cfg *config.Config) {
	t.Helper()
	if cfg == nil {
		t.Fatal("nil config")
	}
	if strings.TrimSpace(cfg.ModulesPath) == "" {
		cfg.ModulesPath = filepath.Join(t.TempDir(), "modules")
	}
	webDir := filepath.Join(cfg.ModulesPath, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("mkdir web module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name":"web"}`), 0o644); err != nil {
		t.Fatalf("write web package.json: %v", err)
	}
}

func TestEnsureHostProvisionReplansAfterSuccess(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	seedBundleModeWebReadyDist(t, runtimeScope.config.DistPath)
	seedApplicationModeWebBackendDist(t, runtimeScope.config.DistPath)

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}

	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		if err := db.Exec(
			`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
			"mod-auth", "auth", string(meta.Installed),
		).Error; err != nil {
			return provision.Result{}, err
		}
		return provision.Result{Skipped: false, Phase: provision.PhaseSetupPending}, nil
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if srv.runState.mode() != runplan.RunModeApplication {
		t.Fatalf("mode=%q want application after provision+replan (reason=%q)", srv.runState.mode(), srv.runState.reason())
	}
}

func TestEnsureHostProvisionKeepsBootstrapWhenDistStillMissing(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		return provision.Result{Skipped: false, Phase: provision.PhaseSetupPending}, nil
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		t.Fatalf("expected bootstrap when dist still missing, got %q", srv.runState.mode())
	}
}

func TestEnsureHostProvisionLeaseBusyWaitsForPeer(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	seedBundleModeWebReadyDist(t, runtimeScope.config.DistPath)
	seedApplicationModeWebBackendDist(t, runtimeScope.config.DistPath)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	calls := 0
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		calls++
		if calls == 1 {
			go func() {
				time.Sleep(50 * time.Millisecond)
				_ = db.Exec(
					`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
					"mod-auth", "auth", string(meta.Installed),
				)
			}()
			return provision.Result{}, provision.ErrLeaseBusy
		}
		return provision.Result{}, errors.New("unexpected second provision call")
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if srv.runState.mode() != runplan.RunModeApplication {
		t.Fatalf("mode=%q want application after peer wait", srv.runState.mode())
	}
}

func TestEnsureHostProvisionReplansWhenSkipped(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	seedBundleModeWebReadyDist(t, runtimeScope.config.DistPath)
	seedApplicationModeWebBackendDist(t, runtimeScope.config.DistPath)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		return provision.Result{Skipped: true, Phase: provision.PhaseSetupPending}, nil
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if srv.runState.mode() != runplan.RunModeApplication {
		t.Fatalf("mode=%q want application after skipped replan", srv.runState.mode())
	}
}

func TestEnsureHostProvisionLeaseBusyTimeoutFallsBack(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	prev := hostProvisionPeerWait
	hostProvisionPeerWait = 20 * time.Millisecond
	t.Cleanup(func() { hostProvisionPeerWait = prev })
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		return provision.Result{}, provision.ErrLeaseBusy
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		t.Fatalf("expected bootstrap after lease-busy timeout, got %q", srv.runState.mode())
	}
}

func TestEnsureHostProvisionReplansBootstrapWhenAlreadyPending(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	seedBundleModeWebReadyDist(t, runtimeScope.config.DistPath)
	seedApplicationModeWebBackendDist(t, runtimeScope.config.DistPath)
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		// Dist may already be application-ready from seeds; force bootstrap first.
		srv.runState.applyPlannedDecision(nil, runplan.RunDecision{RunMode: runplan.RunModeBootstrap, Reason: "forced"})
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if srv.runState.mode() != runplan.RunModeApplication {
		t.Fatalf("mode=%q want application after pending replan", srv.runState.mode())
	}
}
