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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
