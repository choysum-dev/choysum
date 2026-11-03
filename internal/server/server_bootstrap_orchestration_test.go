// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/server/runplan"
)

func TestServerStartBootstrapModeServesPendingSplash(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.JsEngineFactory = "missing"

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	_, decision, err := runplan.Plan(runtimeScope.cfg.DistPath, runtimeScope.cfg.Compile.BundleMode, runtimeScope.Logger(), nil)
	if err != nil {
		t.Fatalf("runplan.Plan() error = %v", err)
	}
	if decision.RunMode != runplan.RunModeBootstrap {
		t.Fatalf("runplan.Plan() run mode = %q, want %q", decision.RunMode, runplan.RunModeBootstrap)
	}

	srv.runState.applyPlannedDecision(nil, decision)

	result := srv.runStartupLifecycle(false, srv.resolvedRuntimeOptions())
	if result.errorValue() != nil {
		t.Fatalf("runStartupLifecycle() error = %v", result.errorValue())
	}
	assertModeRuntimeSummaryFields(t, result.ModeRuntime, runplan.RunModeBootstrap, 1, 0, false, false, "bootstrap startup lifecycle")
	assertTaskRuntimeSummaryFields(t, result.TaskRuntime, false, false, false, false, false, "bootstrap startup lifecycle")

	if srv.jsExecutor != nil {
		t.Fatal("bootstrap mode should not initialize js executor")
	}

	h := srv.newProtocolRouter()

	reqRoot := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rwRoot := httptest.NewRecorder()
	h.ServeHTTP(rwRoot, reqRoot)
	if rwRoot.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET / status = %d, want %d", rwRoot.Code, http.StatusServiceUnavailable)
	}
	body := rwRoot.Body.String()
	if !strings.Contains(strings.ToLower(body), "doctype html") {
		t.Fatalf("GET / body does not look like html: %q", body)
	}
	if !strings.Contains(body, "Choysum") {
		t.Fatalf("GET / body missing brand: %q", body)
	}
	if !strings.Contains(body, "Application assets are not ready yet") {
		t.Fatalf("GET / body missing pending message: %q", body)
	}

	reqBootstrap := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap/", nil)
	rwBootstrap := httptest.NewRecorder()
	h.ServeHTTP(rwBootstrap, reqBootstrap)
	if rwBootstrap.Code != http.StatusNotFound {
		t.Fatalf("GET /bootstrap/ status = %d, want %d", rwBootstrap.Code, http.StatusNotFound)
	}
}
