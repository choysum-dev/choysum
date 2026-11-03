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

type noFactoryInputScope struct{}

func (e *noFactoryInputScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *noFactoryInputScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *noFactoryInputScope) Session() *scope.Session             { return nil }
func (e *noFactoryInputScope) WithContext(context.Context) scope.Scope {
	return e
}
func (e *noFactoryInputScope) Context() context.Context            { return context.Background() }
func (e *noFactoryInputScope) Logger() *slog.Logger                { return nil }
func (e *noFactoryInputScope) FactoryInput() scope.FactoryInput    { return nil }
func (e *noFactoryInputScope) Config() *config.Config              { return &config.Config{} }

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
	for _, name := range provision.MinimalClosureModules() {
		dir := filepath.Join(cfg.ModulesPath, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s module: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644); err != nil {
			t.Fatalf("write %s package.json: %v", name, err)
		}
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
	prev := hostProvisionPeerWait()
	setHostProvisionPeerWait(20 * time.Millisecond)
	t.Cleanup(func() { setHostProvisionPeerWait(prev) })
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

func TestEnsureHostProvisionNilServerAndNilContext(t *testing.T) {
	var srv *GRPCWebServer
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("nil server: %v", err)
	}
	runtimeScope, _ := newHostProvisionDB(t)
	srv = &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.applyPlannedDecision(nil, runplan.RunDecision{RunMode: runplan.RunModeBootstrap, Reason: "seeded"})
	if err := srv.ensureHostProvision(nil); err != nil {
		t.Fatalf("nil ctx: %v", err)
	}
}

func TestEnsureHostProvisionResolveErrorWithSession(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	// Incomplete meta_module schema makes the soft-delete query fail closed.
	if err := db.Exec(`DROP TABLE meta_module`).Error; err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`).Error; err != nil {
		t.Fatalf("create broken meta_module: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.ensureHostProvision(context.Background()); err == nil {
		t.Fatal("expected resolve error")
	}
}

func TestEnsureHostProvisionUsesRealProvisionMinimal(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	// No hostProvisionFn — exercises runHostProvisionMinimal default path.
	// Real install fails quickly without a full module tree; soft-fallback.
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		t.Fatalf("expected bootstrap soft-fallback, got %q", srv.runState.mode())
	}
}

func TestEnsureHostProvisionPeerReplanSoftFallback(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.setServeRequestArgs([]string{"missing-target-xyz"})
	if err := srv.planServe(nil); err != nil {
		// empty plan may still succeed in bootstrap; force a target that breaks replan
		_ = err
	}
	srv.runState.setServeRequestArgs([]string{"missing-target-xyz"})
	prev := hostProvisionPeerWait()
	setHostProvisionPeerWait(time.Minute)
	t.Cleanup(func() { setHostProvisionPeerWait(prev) })

	// Peer wait succeeds immediately by inserting auth, then replan with missing target fails soft.
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		go func() {
			time.Sleep(20 * time.Millisecond)
			_ = runtimeScope.db.Exec(
				`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
				"mod-auth", "auth", string(meta.Installed),
			)
		}()
		return provision.Result{}, provision.ErrLeaseBusy
	}
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
}

func TestEnsureHostProvisionReplanRequiredFailure(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.setServeRequestArgs([]string{"missing-target-xyz"})
	if err := srv.planServe(nil); err == nil {
		// When empty, explicit missing target may error; ignore either outcome.
	}
	srv.runState.setServeRequestArgs([]string{"missing-target-xyz"})
	srv.hostProvisionFn = func(context.Context) (provision.Result, error) {
		if err := db.Exec(
			`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
			"mod-auth", "auth", string(meta.Installed),
		).Error; err != nil {
			return provision.Result{}, err
		}
		return provision.Result{Skipped: false, Phase: provision.PhaseSetupPending}, nil
	}
	if err := srv.ensureHostProvision(context.Background()); err == nil {
		t.Fatal("expected replan failure after provision")
	}
}

func TestEnsureHostProvisionSkippedReplanSoftOptional(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	seedBundleModeWebReadyDist(t, runtimeScope.config.DistPath)
	seedApplicationModeWebBackendDist(t, runtimeScope.config.DistPath)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	// Force bootstrap + pending so non-empty path hits optional replan.
	if err := runtimeScope.db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv.runState.applyPlannedDecision(nil, runplan.RunDecision{RunMode: runplan.RunModeBootstrap, Reason: "forced"})
	// Break dist so optional replan logs and soft-returns.
	_ = os.RemoveAll(runtimeScope.config.DistPath)
	if err := srv.ensureHostProvision(context.Background()); err != nil {
		t.Fatalf("ensureHostProvision: %v", err)
	}
}

func TestWaitForHostProvisionPeerContextCancel(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := srv.waitForHostProvisionPeer(ctx); err == nil {
		t.Fatal("expected ctx cancel")
	}
}

func TestWaitForHostProvisionPeerContextCancelDuringWait(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	prev := hostProvisionPeerWait()
	setHostProvisionPeerWait(time.Minute)
	t.Cleanup(func() { setHostProvisionPeerWait(prev) })
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	if err := srv.waitForHostProvisionPeer(ctx); err == nil {
		t.Fatal("expected ctx cancel during wait")
	}
}

func TestWaitForHostProvisionPeerResolveError(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	if err := db.Exec(`DROP TABLE meta_module`).Error; err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`).Error; err != nil {
		t.Fatalf("create broken meta_module: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.waitForHostProvisionPeer(context.Background()); err == nil {
		t.Fatal("expected resolve error")
	}
}

func TestLogHostProvisionWarnNil(t *testing.T) {
	var srv *GRPCWebServer
	srv.logHostProvisionWarn("noop")
	srv = &GRPCWebServer{}
	srv.logHostProvisionWarn("noop")
}

func TestHasLocalProvisionModules(t *testing.T) {
	if hasLocalProvisionModules(nil) {
		t.Fatal("nil scope")
	}
	if hasLocalProvisionModules(&noFactoryInputScope{}) {
		t.Fatal("nil factory input must be false")
	}
	runtimeScope, _ := newHostProvisionDB(t)
	if hasLocalProvisionModules(runtimeScope) {
		t.Fatal("empty ModulesPath must be false")
	}
	runtimeScope.config.ModulesPath = filepath.Join(t.TempDir(), "modules")
	if err := os.MkdirAll(filepath.Join(runtimeScope.config.ModulesPath, "web"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeScope.config.ModulesPath, "web", "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if hasLocalProvisionModules(runtimeScope) {
		t.Fatal("web-only must be false; full minimal closure required")
	}
	seedLocalWebModule(t, runtimeScope.config)
	if !hasLocalProvisionModules(runtimeScope) {
		t.Fatal("full minimal closure package.json must be true")
	}
	// A directory named package.json must not count as present.
	dirPkg := filepath.Join(runtimeScope.config.ModulesPath, "web", "package.json")
	if err := os.Remove(dirPkg); err != nil {
		t.Fatalf("remove web package.json: %v", err)
	}
	if err := os.Mkdir(dirPkg, 0o755); err != nil {
		t.Fatalf("mkdir package.json: %v", err)
	}
	if hasLocalProvisionModules(runtimeScope) {
		t.Fatal("directory named package.json must be false")
	}
	if err := os.RemoveAll(dirPkg); err != nil {
		t.Fatalf("remove dir package.json: %v", err)
	}
	if err := os.WriteFile(dirPkg, nil, 0o644); err != nil {
		t.Fatalf("write empty package.json: %v", err)
	}
	if hasLocalProvisionModules(runtimeScope) {
		t.Fatal("empty package.json must be false")
	}
}

func TestServeOrdersHostProvisionBeforePlan(t *testing.T) {
	runtimeScope, db := newHostProvisionDB(t)
	if err := db.Exec(`DROP TABLE meta_module`).Error; err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := db.Exec(`CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`).Error; err != nil {
		t.Fatalf("create broken meta_module: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	err := srv.Serve(context.Background())
	if err == nil || !strings.Contains(err.Error(), "resolve setup phase") {
		t.Fatalf("error=%v", err)
	}
}

func TestReplanAfterHostProvisionOptional(t *testing.T) {
	runtimeScope, _ := newHostProvisionDB(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.setServeRequestArgs([]string{"missing-target-xyz"})
	if err := srv.replanAfterHostProvision(false); err != nil {
		t.Fatalf("optional replan: %v", err)
	}
	if err := srv.replanAfterHostProvision(true); err == nil {
		t.Fatal("required replan should fail")
	}
}
