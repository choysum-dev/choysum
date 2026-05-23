// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bootstrapweb "github.com/choysum-dev/choysum/internal/bootstrap/web"
	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/server/runplan"
)

func TestServerStartBootstrapModeServesBootstrapRoutes(t *testing.T) {
	t.Setenv(bootstrapweb.EnvBootstrapWebSource, "embed")

	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	runtimeScope.cfg.Server.Port = 0
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
	if rwRoot.Code != http.StatusFound {
		t.Fatalf("GET / status = %d, want %d", rwRoot.Code, http.StatusFound)
	}
	if loc := rwRoot.Header().Get("Location"); loc != "/bootstrap" {
		t.Fatalf("GET / Location = %q, want /bootstrap", loc)
	}

	reqBootstrap := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap", nil)
	rwBootstrap := httptest.NewRecorder()
	h.ServeHTTP(rwBootstrap, reqBootstrap)
	if rwBootstrap.Code != http.StatusOK {
		t.Fatalf("GET /bootstrap status = %d, want %d", rwBootstrap.Code, http.StatusOK)
	}
	if !strings.Contains(strings.ToLower(rwBootstrap.Body.String()), "doctype html") {
		t.Fatalf("GET /bootstrap body does not look like html: %q", rwBootstrap.Body.String())
	}

	reqMissingAsset := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap/assets/not-found.js", nil)
	rwMissingAsset := httptest.NewRecorder()
	h.ServeHTTP(rwMissingAsset, reqMissingAsset)
	if rwMissingAsset.Code != http.StatusNotFound {
		t.Fatalf("GET /bootstrap/assets/not-found.js status = %d, want %d", rwMissingAsset.Code, http.StatusNotFound)
	}
}

func TestServerRequestBootstrapModeSwitchTransitionsToApplicationAndRestarts(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"

	if err := os.MkdirAll(filepath.Join(runtimeScope.cfg.DistPath, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(web) error = %v", err)
	}

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	restoreRunStateForTest(srv, runStateSnapshot{
		runMode:           runplan.RunModeBootstrap,
		runModeReason:     "default target resolution failed",
		compileBundleMode: runtimeScope.cfg.Compile.BundleMode,
	})

	restartCalled := 0
	srv.runtimeRecovery.modeSwitchRestartExecutor = func() error {
		restartCalled++
		assertRunStateMode(t, srv, runplan.RunModeApplication, "mode-switch restart executor")
		return nil
	}

	if err := srv.requestBootstrapModeSwitch(context.Background()); err != nil {
		t.Fatalf("requestBootstrapModeSwitch() error = %v", err)
	}
	assertRunStateMode(t, srv, runplan.RunModeApplication, "after successful bootstrap mode switch")
	if restartCalled != 1 {
		t.Fatalf("restart hook call count = %d, want 1", restartCalled)
	}
	assertRunStateTargets(t, srv, []string{"web"}, "after successful bootstrap mode switch")
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srv), recoveryActionModeSwitchRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 0}, "successful bootstrap mode-switch diagnostics")

	if err := srv.requestBootstrapModeSwitch(context.Background()); err != nil {
		t.Fatalf("second requestBootstrapModeSwitch() error = %v", err)
	}
	if restartCalled != 1 {
		t.Fatalf("restart hook call count after second switch = %d, want 1", restartCalled)
	}
}

func TestServerRequestBootstrapModeSwitchDefaultRestartUsesColdStart(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	if err := os.MkdirAll(filepath.Join(runtimeScope.cfg.DistPath, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(web) error = %v", err)
	}

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	restoreRunStateForTest(srv, runStateSnapshot{
		runMode:           runplan.RunModeBootstrap,
		runModeReason:     "no app is ready to serve yet",
		compileBundleMode: runtimeScope.cfg.Compile.BundleMode,
	})

	if err := srv.requestBootstrapModeSwitch(context.Background()); err != nil {
		t.Fatalf("requestBootstrapModeSwitch() error = %v", err)
	}

	assertRunStateMode(t, srv, runplan.RunModeApplication, "after default bootstrap mode switch restart")
	assertRunStateTargets(t, srv, []string{"web"}, "after default bootstrap mode switch restart")
	assertServerReadyState(t, srv, true, "after default bootstrap mode switch restart")
	if srv.jsExecutor == nil {
		t.Fatal("expected bootstrap mode switch cold start to initialize js executor")
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srv), recoveryActionModeSwitchRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 0}, "default bootstrap mode switch restart diagnostics")
}

func TestServerRequestBootstrapModeSwitchRestoresStateWhenRestartFails(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"

	if err := os.MkdirAll(filepath.Join(runtimeScope.cfg.DistPath, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(web) error = %v", err)
	}

	previousManifest := &distmanifest.DistManifestV2{}
	previousTargets := []string{"bootstrap"}
	srv := NewServer(runtimeScope).(*GRPCWebServer)
	restoreRunStateForTest(srv, runStateSnapshot{
		distManifest:      previousManifest,
		runMode:           runplan.RunModeBootstrap,
		runModeReason:     "default target resolution failed",
		compileBundleMode: "bootstrap-bundle",
		applicationNames:  append([]string{}, previousTargets...),
	})

	restartErr := errors.New("restart failed")
	srv.runtimeRecovery.modeSwitchRestartExecutor = func() error {
		assertRunStateMode(t, srv, runplan.RunModeApplication, "failed mode-switch restart executor")
		assertRunStateTargets(t, srv, []string{"web"}, "failed mode-switch restart executor")
		return restartErr
	}

	err := srv.requestBootstrapModeSwitch(context.Background())
	if err == nil {
		t.Fatal("expected requestBootstrapModeSwitch() to fail when restart hook fails")
	}
	if !strings.Contains(err.Error(), "bootstrap switch restart failed") || !strings.Contains(err.Error(), restartErr.Error()) {
		t.Fatalf("requestBootstrapModeSwitch() error = %v, want wrapped restart failure", err)
	}
	assertRunStateSnapshot(t, srv, runStateSnapshot{
		distManifest:      previousManifest,
		runMode:           runplan.RunModeBootstrap,
		runModeReason:     "default target resolution failed",
		compileBundleMode: "bootstrap-bundle",
		applicationNames:  previousTargets,
	}, "after failed bootstrap mode switch")
	recoverySnapshot := recoveryDiagnosticsForTest(srv)
	assertRecoveryActionDiagnostics(t, recoverySnapshot, recoveryActionModeSwitchRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "failed bootstrap mode-switch diagnostics")
	assertRecoveryModeSwitchRollbacks(t, recoverySnapshot, 1, "failed bootstrap mode-switch diagnostics")
}
