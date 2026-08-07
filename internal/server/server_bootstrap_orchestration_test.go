// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	if rwRoot.Code != http.StatusFound {
		t.Fatalf("GET / status = %d, want %d", rwRoot.Code, http.StatusFound)
	}
	if loc := rwRoot.Header().Get("Location"); loc != "/bootstrap/" {
		t.Fatalf("GET / Location = %q, want /bootstrap/", loc)
	}

	reqBootstrap := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap", nil)
	rwBootstrap := httptest.NewRecorder()
	h.ServeHTTP(rwBootstrap, reqBootstrap)
	if rwBootstrap.Code != http.StatusFound {
		t.Fatalf("GET /bootstrap status = %d, want %d", rwBootstrap.Code, http.StatusFound)
	}
	if loc := rwBootstrap.Header().Get("Location"); loc != "/bootstrap/" {
		t.Fatalf("GET /bootstrap Location = %q, want /bootstrap/", loc)
	}

	reqBootstrapCanonical := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap/", nil)
	rwBootstrapCanonical := httptest.NewRecorder()
	h.ServeHTTP(rwBootstrapCanonical, reqBootstrapCanonical)
	if rwBootstrapCanonical.Code != http.StatusOK {
		t.Fatalf("GET /bootstrap/ status = %d, want %d", rwBootstrapCanonical.Code, http.StatusOK)
	}
	body := rwBootstrapCanonical.Body.String()
	if !strings.Contains(strings.ToLower(body), "doctype html") {
		t.Fatalf("GET /bootstrap/ body does not look like html: %q", body)
	}
	if !strings.Contains(body, `src="/bootstrap/assets/`) {
		t.Fatalf("GET /bootstrap/ body missing bootstrap script asset prefix: %q", body)
	}
	if !strings.Contains(body, `href="/bootstrap/assets/`) {
		t.Fatalf("GET /bootstrap/ body missing bootstrap stylesheet asset prefix: %q", body)
	}
	if strings.Contains(body, `src="assets/`) || strings.Contains(body, `href="assets/`) {
		t.Fatalf("GET /bootstrap/ body contains relative assets path that breaks /bootstrap: %q", body)
	}

	scriptAttr := `src="/bootstrap/assets/`
	scriptAttrIndex := strings.Index(body, scriptAttr)
	if scriptAttrIndex < 0 {
		t.Fatalf("GET /bootstrap/ body missing bootstrap script src attribute: %q", body)
	}
	scriptPathStart := scriptAttrIndex + len(`src="`)
	scriptPathEnd := strings.Index(body[scriptPathStart:], `"`)
	if scriptPathEnd < 0 {
		t.Fatalf("GET /bootstrap/ body script src attribute is not closed: %q", body)
	}
	scriptPath := body[scriptPathStart : scriptPathStart+scriptPathEnd]

	reqBootstrapScript := httptest.NewRequest(http.MethodGet, "http://example.com"+scriptPath, nil)
	rwBootstrapScript := httptest.NewRecorder()
	h.ServeHTTP(rwBootstrapScript, reqBootstrapScript)
	if rwBootstrapScript.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", scriptPath, rwBootstrapScript.Code, http.StatusOK)
	}

	scriptBody := rwBootstrapScript.Body.String()
	if hasCrossOriginImport(scriptBody) {
		t.Fatalf("GET %s body contains cross-origin import statements that violate CSP self", scriptPath)
	}

	reqMissingAsset := httptest.NewRequest(http.MethodGet, "http://example.com/bootstrap/assets/not-found.js", nil)
	rwMissingAsset := httptest.NewRecorder()
	h.ServeHTTP(rwMissingAsset, reqMissingAsset)
	if rwMissingAsset.Code != http.StatusNotFound {
		t.Fatalf("GET /bootstrap/assets/not-found.js status = %d, want %d", rwMissingAsset.Code, http.StatusNotFound)
	}
}

func hasCrossOriginImport(js string) bool {
	for _, prefix := range []string{`import "https://`, `import 'https://`, `import "http://`, `import 'http://`, `from "https://`, `from 'https://`, `from "http://`, `from 'http://`} {
		if strings.Contains(js, prefix) {
			return true
		}
	}
	return false
}

func TestServerRequestBootstrapModeSwitchTransitionsToApplicationAndRestarts(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"

	seedBundleModeWebReadyDist(t, runtimeScope.cfg.DistPath)

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
	var logBuf bytes.Buffer
	runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	seedBundleModeWebReadyDist(t, runtimeScope.cfg.DistPath)

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
	logs := logBuf.String()
	if !strings.Contains(logs, "server restarting") {
		t.Fatalf("expected mode-switch restart wording in logs, got %q", logs)
	}
	if strings.Contains(logs, "server stopped") {
		t.Fatalf("did not expect stop wording during mode-switch restart, got %q", logs)
	}
	if !strings.Contains(logs, "application server ready") || !strings.Contains(logs, "access_url=") {
		t.Fatalf("expected application server ready log with access_url, got %q", logs)
	}
	if strings.Contains(logs, "http server listening") {
		t.Fatalf("expected reload startup to suppress duplicate http server listening log, got %q", logs)
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

	seedBundleModeWebReadyDist(t, runtimeScope.cfg.DistPath)

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
