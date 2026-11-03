// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc"
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
	if !strings.Contains(body, "restart the server") {
		t.Fatalf("GET / body missing restart recovery instruction: %q", body)
	}

	reqBootstrap := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap/", nil)
	rwBootstrap := httptest.NewRecorder()
	h.ServeHTTP(rwBootstrap, reqBootstrap)
	if rwBootstrap.Code != http.StatusNotFound {
		t.Fatalf("GET /bootstrap/ status = %d, want %d", rwBootstrap.Code, http.StatusNotFound)
	}

	// Exact /{$} must not swallow base routes registered on the same mux.
	reqHealth := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rwHealth := httptest.NewRecorder()
	h.ServeHTTP(rwHealth, reqHealth)
	if rwHealth.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", rwHealth.Code, http.StatusOK)
	}
}

func TestServePendingSplashMethods(t *testing.T) {
	post := httptest.NewRecorder()
	servePendingSplash(post, httptest.NewRequest(http.MethodPost, "/", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", post.Code, http.StatusMethodNotAllowed)
	}
	if allow := post.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", allow)
	}

	head := httptest.NewRecorder()
	servePendingSplash(head, httptest.NewRequest(http.MethodHead, "/", nil))
	if head.Code != http.StatusServiceUnavailable {
		t.Fatalf("HEAD status = %d, want %d", head.Code, http.StatusServiceUnavailable)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", head.Body.Len())
	}
	if ct := head.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("HEAD Content-Type = %q", ct)
	}

	get := httptest.NewRecorder()
	servePendingSplash(get, httptest.NewRequest(http.MethodGet, "/", nil))
	if get.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET status = %d, want %d", get.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(get.Body.String(), "Choysum") {
		t.Fatalf("GET body missing brand: %q", get.Body.String())
	}
}

func TestRegisterBootstrapServicePropagatesBindingError(t *testing.T) {
	prev := pendingSplashBinding
	pendingSplashBinding = func() registrationBinding { return failPendingSplashBinding{} }
	t.Cleanup(func() { pendingSplashBinding = prev })

	srv := &GRPCWebServer{mux: http.NewServeMux()}
	_, err := srv.registerBootstrapService()
	if err == nil || !strings.Contains(err.Error(), "pending splash") {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(err.Error(), "desc boom") {
		t.Fatalf("error=%v, want wrapped desc boom", err)
	}
}

type failPendingSplashBinding struct{}

func (failPendingSplashBinding) Name() string { return "bootstrap" }
func (failPendingSplashBinding) ServiceDescs() ([]*grpc.ServiceDesc, error) {
	return nil, errors.New("desc boom")
}
func (failPendingSplashBinding) ServiceScripts() []*jsengine.JsScript { return nil }
func (failPendingSplashBinding) WebHandlers() (map[string]http.Handler, error) {
	return nil, nil
}
