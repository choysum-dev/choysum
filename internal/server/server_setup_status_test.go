// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestServeSetupStatusUsesHostUIWhenEmpty(t *testing.T) {
	t.Parallel()
	srv := &GRPCWebServer{}
	srv.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)

	rec := httptest.NewRecorder()
	srv.serveSetupStatus(rec, httptest.NewRequest(http.MethodGet, "/-/setup/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var payload setupStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Phase != "starting" {
		t.Fatalf("phase = %q, want starting", payload.Phase)
	}
	if !strings.Contains(payload.Message, "Preparing setup") {
		t.Fatalf("message = %q", payload.Message)
	}
}

func TestSetupStatusKeepsStartingUntilHostUIIdle(t *testing.T) {
	t.Parallel()
	runtimeScope, db := newProvisionMinimalDB(t)
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	// DB is already SetupPending, but splash must wait until restart clears UI.
	srv.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	payload := srv.setupStatusPayload()
	if payload.Phase != "starting" {
		t.Fatalf("phase = %q, want starting while splash UI in-flight", payload.Phase)
	}

	srv.setSetupSplashUI(setupSplashUIIdle, "")
	payload = srv.setupStatusPayload()
	if payload.Phase != "ready" {
		t.Fatalf("phase = %q, want ready after splash UI idle", payload.Phase)
	}
}

func TestRedirectWebToSplash(t *testing.T) {
	t.Parallel()
	splash := newSetupSplashService(&GRPCWebServer{})
	rec := httptest.NewRecorder()
	splash.redirectWebToSplash(rec, httptest.NewRequest(http.MethodGet, "/web/setup", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("Location = %q, want /", loc)
	}
}

func TestServeSetupSplashStartingOKAndFailedUnavailable(t *testing.T) {
	t.Parallel()
	srv := &GRPCWebServer{}
	splash := newSetupSplashService(srv)

	srv.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	recOK := httptest.NewRecorder()
	splash.serveSetupSplash(recOK, httptest.NewRequest(http.MethodGet, "/", nil))
	if recOK.Code != http.StatusOK {
		t.Fatalf("starting splash status = %d, want 200", recOK.Code)
	}
	body := recOK.Body.String()
	if !strings.Contains(body, "Choysum") || !strings.Contains(body, "/-/setup/status") {
		t.Fatalf("expected brand + status poll in splash HTML, got %q", body[:min(200, len(body))])
	}
	if !strings.Contains(body, `class="brand-mark"`) || !strings.Contains(body, `fill="#10b981"`) {
		t.Fatal("expected inlined sprout mark in splash HTML")
	}
	if !strings.Contains(body, "flex-direction: row") || !strings.Contains(body, "2.5rem") {
		t.Fatal("expected one-line brand lockup with enlarged mark")
	}
	if strings.Contains(body, "brand-breathe") {
		t.Fatal("brand breathe motion should be removed for quieter splash")
	}
	if !strings.Contains(body, "indeterminate") {
		t.Fatal("expected indeterminate progress motion in splash HTML")
	}
	if !strings.Contains(body, "aria-live") {
		t.Fatal("expected aria-live on splash status")
	}
	if !strings.Contains(body, "Refresh") {
		t.Fatal("expected failed-state Refresh control in splash HTML")
	}
	if !strings.Contains(body, "#ffffff") || !strings.Contains(body, "#059669") {
		t.Fatal("expected shell-aligned white + emerald progress colors")
	}
	if !strings.Contains(body, "rgba(16, 185, 129, 0.16)") {
		t.Fatal("expected strengthened emerald background wash")
	}
	if !strings.Contains(body, "Preparing setup") {
		t.Fatalf("expected Preparing setup copy, got %q", body[:min(300, len(body))])
	}
	if !strings.Contains(body, `phase === "ready"`) || !strings.Contains(body, "goSetup()") {
		t.Fatal("expected ready → goSetup without Opening flash")
	}

	srv.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
	recFail := httptest.NewRecorder()
	splash.serveSetupSplash(recFail, httptest.NewRequest(http.MethodGet, "/", nil))
	if recFail.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed splash status = %d, want 503", recFail.Code)
	}
}

func TestPrepareProvisionMinimalForServeDefersWhenLocalClosurePresent(t *testing.T) {
	t.Parallel()
	// No runtime scope → skipped (same as missing DB fixtures).
	srv := &GRPCWebServer{}
	deferred, err := srv.prepareProvisionMinimalForServe(nil)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if deferred {
		t.Fatal("expected no deferred provision without runtime scope")
	}
}

func TestPrepareProvisionMinimalForServeDefersEmptyWithModules(t *testing.T) {
	t.Parallel()
	runtimeScope, _ := newProvisionMinimalDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope, mux: http.NewServeMux()}

	deferred, err := srv.prepareProvisionMinimalForServe(context.Background())
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !deferred {
		t.Fatal("Empty + local closure should defer provision until after splash listening")
	}
	phase, msg := srv.setupSplashUISnapshot()
	if phase != setupSplashUIProvisioning {
		t.Fatalf("ui phase = %q, want provisioning", phase)
	}
	if !strings.Contains(msg, "Preparing setup") {
		t.Fatalf("ui message = %q", msg)
	}

	if _, err := srv.registerBootstrapService(); err != nil {
		t.Fatalf("registerBootstrapService: %v", err)
	}
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / during deferred provision = %d, want 200", rec.Code)
	}

	statusRec := httptest.NewRecorder()
	srv.serveSetupStatus(statusRec, httptest.NewRequest(http.MethodGet, "/-/setup/status", nil))
	var payload setupStatusResponse
	if err := json.Unmarshal(statusRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if payload.Phase != "starting" {
		t.Fatalf("status phase = %q, want starting", payload.Phase)
	}
}

func TestDeferredProvisionMinimalSetsFailedWithoutRestartOnInstallError(t *testing.T) {
	t.Parallel()
	runtimeScope, _ := newProvisionMinimalDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe: %v", err)
	}
	srv.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	srv.provisionMinimalFn = func(ctx context.Context) (provision.Result, error) {
		once.Do(func() { close(started) })
		select {
		case <-release:
		case <-ctx.Done():
			return provision.Result{}, ctx.Err()
		}
		return provision.Result{}, context.Canceled
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.deferredProvisionMinimal(context.Background())
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for deferred provision to start")
	}
	// Splash status stays starting while install is in flight.
	if phase, _ := srv.setupSplashUISnapshot(); phase != setupSplashUIProvisioning {
		t.Fatalf("in-flight ui phase = %q", phase)
	}
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for deferred provision to finish")
	}
	if phase, _ := srv.setupSplashUISnapshot(); phase != setupSplashUIFailed {
		t.Fatalf("after failure ui phase = %q, want failed", phase)
	}
}

func TestSetupStatusKeepsStartingWhileBootstrapDespiteSetupPending(t *testing.T) {
	t.Parallel()
	runtimeScope, db := newProvisionMinimalDB(t)
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.applyPlannedDecision(nil, runplan.RunDecision{
		RunMode: runplan.RunModeBootstrap,
		Reason:  "dist missing",
	})
	srv.setSetupSplashUI(setupSplashUIIdle, "")
	payload := srv.setupStatusPayload()
	if payload.Phase != "starting" {
		t.Fatalf("phase = %q, want starting while bootstrap owns /web", payload.Phase)
	}
}

func TestSetupStatusHTTPMethodAndDefaultMessages(t *testing.T) {
	t.Parallel()
	srv := &GRPCWebServer{}

	rec := httptest.NewRecorder()
	srv.serveSetupStatus(rec, httptest.NewRequest(http.MethodPost, "/-/setup/status", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", rec.Code)
	}

	srv.setSetupSplashUI(setupSplashUIProvisioning, "")
	head := httptest.NewRecorder()
	srv.serveSetupStatus(head, httptest.NewRequest(http.MethodHead, "/-/setup/status", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", head.Code)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty, got %q", head.Body.String())
	}

	if p := srv.setupStatusPayload(); p.Phase != "starting" || p.Message != setupStatusMsgStarting {
		t.Fatalf("provisioning defaults = %+v", p)
	}
	srv.setSetupSplashUI(setupSplashUIWaitingPeer, "")
	if p := srv.setupStatusPayload(); p.Phase != "waiting" || p.Message != setupStatusMsgWaiting {
		t.Fatalf("waiting defaults = %+v", p)
	}
	srv.setSetupSplashUI(setupSplashUIFailed, "")
	if p := srv.setupStatusPayload(); p.Phase != "failed" || p.Message != setupStatusMsgFailed {
		t.Fatalf("failed defaults = %+v", p)
	}
}

func TestSetupSplashUINilServerAndBindingRestore(t *testing.T) {
	t.Parallel()
	var nilSrv *GRPCWebServer
	nilSrv.setSetupSplashUI(setupSplashUIFailed, "x")
	if phase, msg := nilSrv.setupSplashUISnapshot(); phase != setupSplashUIIdle || msg != "" {
		t.Fatalf("nil snapshot = %q %q", phase, msg)
	}

	restore := setSetupSplashBindingForTest(nil)
	defer restore()
	if binding := setupSplashBinding(&GRPCWebServer{}); binding == nil {
		t.Fatal("nil override should restore default splash binding")
	}
}

func TestRedirectWebToSplashMethodNotAllowed(t *testing.T) {
	t.Parallel()
	splash := newSetupSplashService(&GRPCWebServer{})
	rec := httptest.NewRecorder()
	splash.redirectWebToSplash(rec, httptest.NewRequest(http.MethodPost, "/web/setup", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestPrepareProvisionMinimalForServeBranches(t *testing.T) {
	t.Parallel()

	runtimeScope, _ := newProvisionMinimalDB(t)
	srvEmpty := &GRPCWebServer{runtimeScope: runtimeScope}
	deferred, err := srvEmpty.prepareProvisionMinimalForServe(nil)
	if err != nil {
		t.Fatalf("Empty without modules: %v", err)
	}
	if deferred {
		t.Fatal("Empty without local closure must not defer")
	}
	if phase, _ := srvEmpty.setupSplashUISnapshot(); phase != setupSplashUIFailed {
		t.Fatalf("ui phase = %q, want failed", phase)
	}

	runtimeScope2, db := newProvisionMinimalDB(t)
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	seedLocalWebModule(t, runtimeScope2.config)
	seedBundleModeWebReadyDist(t, runtimeScope2.config.DistPath)
	seedApplicationModeWebBackendDist(t, runtimeScope2.config.DistPath)
	srvPending := &GRPCWebServer{runtimeScope: runtimeScope2}
	deferred, err = srvPending.prepareProvisionMinimalForServe(context.Background())
	if err != nil {
		t.Fatalf("SetupPending prepare: %v", err)
	}
	if deferred {
		t.Fatal("SetupPending must not defer ProvisionMinimal")
	}
}

func TestDeferredProvisionMinimalNilAndRestartPaths(t *testing.T) {
	t.Parallel()
	var nilSrv *GRPCWebServer
	nilSrv.deferredProvisionMinimal(nil)
	nilSrv.kickDeferredProvisionMinimalIfNeeded()

	runtimeScope, db := newProvisionMinimalDB(t)
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed),
	).Error; err != nil {
		t.Fatalf("insert auth: %v", err)
	}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.applyPlannedDecision(nil, runplan.RunDecision{
		RunMode: runplan.RunModeApplication,
		Reason:  "test",
	})
	srv.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	// Real restart() on a bare server fails (covers deferredRestartFn == nil path).
	srv.deferredProvisionMinimal(nil)
	if phase, _ := srv.setupSplashUISnapshot(); phase != setupSplashUIFailed {
		t.Fatalf("ui phase = %q, want failed after restart error", phase)
	}

	srv.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	srv.deferredRestartFn = func() error { return nil }
	srv.deferredProvisionMinimal(context.Background())
	if phase, _ := srv.setupSplashUISnapshot(); phase != setupSplashUIIdle {
		t.Fatalf("ui phase = %q, want idle after successful restart", phase)
	}

	runtimeScope2, _ := newProvisionMinimalDB(t)
	seedLocalWebModule(t, runtimeScope2.config)
	srvFail := &GRPCWebServer{runtimeScope: runtimeScope2}
	srvFail.runState.setServeRequestArgs([]string{"missing-target-xyz"})
	srvFail.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	srvFail.provisionMinimalFn = func(context.Context) (provision.Result, error) {
		return provision.Result{Phase: provision.PhaseSetupPending}, nil
	}
	srvFail.deferredProvisionMinimal(context.Background())
	if phase, _ := srvFail.setupSplashUISnapshot(); phase != setupSplashUIFailed {
		t.Fatalf("ui phase = %q, want failed when ensure returns replan error", phase)
	}

	runtimeEmpty, _ := newProvisionMinimalDB(t)
	seedLocalWebModule(t, runtimeEmpty.config)
	kick := &GRPCWebServer{runtimeScope: runtimeEmpty}
	kick.startDeferredProvisionMinimal = true
	provisioned := make(chan struct{})
	kick.provisionMinimalFn = func(context.Context) (provision.Result, error) {
		close(provisioned)
		return provision.Result{}, context.Canceled
	}
	kick.kickDeferredProvisionMinimalIfNeeded()
	select {
	case <-provisioned:
	case <-time.After(2 * time.Second):
		t.Fatal("kickDeferredProvisionMinimalIfNeeded did not start worker")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if phase, _ := kick.setupSplashUISnapshot(); phase == setupSplashUIFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if phase, _ := kick.setupSplashUISnapshot(); phase != setupSplashUIFailed {
		t.Fatalf("ui phase=%q want failed after kicked provision error", phase)
	}
	if kick.startDeferredProvisionMinimal {
		t.Fatal("flag should clear after kick")
	}
}

func TestArmDeferredProvisionMinimal(t *testing.T) {
	t.Parallel()
	var nilSrv *GRPCWebServer
	nilSrv.armDeferredProvisionMinimal(true)

	srv := &GRPCWebServer{}
	srv.armDeferredProvisionMinimal(true)
	if !srv.startDeferredProvisionMinimal {
		t.Fatal("expected deferred flag set")
	}
	srv.armDeferredProvisionMinimal(false)
	if srv.startDeferredProvisionMinimal {
		t.Fatal("expected deferred flag cleared")
	}

	if got := planNamesForDeferredProvision(true, []string{"web"}); got != nil {
		t.Fatalf("deferred plan names=%v, want nil", got)
	}
	if got := planNamesForDeferredProvision(false, []string{"web"}); len(got) != 1 || got[0] != "web" {
		t.Fatalf("non-deferred plan names=%v", got)
	}
}

func TestDeferredInitialPlanIgnoresExplicitTargets(t *testing.T) {
	t.Parallel()
	runtimeScope, _ := newProvisionMinimalDB(t)
	seedLocalWebModule(t, runtimeScope.config)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.runState.setServeRequestArgs([]string{"web"})
	deferred, err := srv.prepareProvisionMinimalForServe(context.Background())
	if err != nil || !deferred {
		t.Fatalf("deferred=%v err=%v", deferred, err)
	}
	if err := srv.planServe([]string{"web"}); err == nil {
		t.Fatal("explicit web plan should fail without dist")
	}
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("nil plan for deferred bootstrap: %v", err)
	}
	if !srv.runState.isBootstrapMode() {
		t.Fatal("want bootstrap after deferred initial plan")
	}
	if got := srv.runState.serveRequest(); len(got) != 1 || got[0] != "web" {
		t.Fatalf("serveRequest=%v, want preserved [web]", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
