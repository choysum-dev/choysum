// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/internal/server/transport"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
)

func TestServerStopReloadCleansResources(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	reg := &trackingRegistry{}
	authn := &fakeAuthenticator{}
	tel := &fakeMetricTelemetry{}
	trackingGC := &trackingTaskGarbageCollector{}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Close()
	watchDir := t.TempDir()
	if err := watcher.Add(watchDir); err != nil {
		t.Fatalf("watcher.Add() error = %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	httpSrv := &http.Server{Handler: http.NotFoundHandler()}
	go func() { _ = httpSrv.Serve(ln) }()

	srv := &GRPCWebServer{
		runtimeScope:   runtimeScope,
		listener:       ln,
		httpServer:     httpSrv,
		hotreload:      hotreloadState{watcher: watcher},
		registry:       reg,
		authenticator:  authn,
		telemetry:      tel,
		server:         grpc.NewServer(),
		grpcClientPool: &transport.GRPCClientPool{},
		runState:       runState{applicationNames: []string{"task"}},
		taskRuntime: taskRuntimeState{hostRuntimeProvider: func(scope.Scope) taskcontract.Runtime {
			return taskcontract.Runtime{Collector: trackingGC}
		}},
	}
	srv.ready.Store(true)
	srv.startTaskRuntime()
	if err := srv.stop(true); err != nil {
		t.Fatalf("stop(true) error = %v", err)
	}
	assertServerReadyState(t, srv, false, "stop(true)")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{}, "stop(true)")
	assertTaskRuntimeStopped(t, srv, "stop(true)")
	if trackingGC.stopCalls != 1 {
		t.Fatalf("task collector stop calls = %d, want 1", trackingGC.stopCalls)
	}
	if srv.authenticator != nil || srv.telemetry != nil {
		t.Fatalf("expected authenticator and telemetry to be cleared, got auth=%v telemetry=%v", srv.authenticator, srv.telemetry)
	}
	if reg.unregisterAllCnt != 1 {
		t.Fatalf("UnRegisterAll calls = %d, want 1", reg.unregisterAllCnt)
	}
	if authn.closed != 1 {
		t.Fatalf("authenticator Close calls = %d, want 1", authn.closed)
	}
	if tel.shutdowns != 1 || tel.metricShutdowns != 1 {
		t.Fatalf("telemetry shutdowns = %d/%d, want 1/1", tel.shutdowns, tel.metricShutdowns)
	}
	if got := watcher.WatchList(); len(got) != 0 {
		t.Fatalf("expected watcher list to be empty after stop, got %#v", got)
	}
}

func TestServerStartAndRestartLifecycle(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) error = %v", err)
	}
	assertServerReadyState(t, srv, true, "start(false)")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{HTTPServer: true, Listener: true, GRPCServer: true, GRPCClientPool: true}, "start(false)")
	if srv.jsExecutor == nil {
		t.Fatal("expected start(false) to initialize js executor")
	}

	readyRec := httptest.NewRecorder()
	srv.newProtocolRouter().ServeHTTP(readyRec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyRec.Code != http.StatusOK || readyRec.Body.String() != "ready" {
		t.Fatalf("readyz status/body = %d/%q, want 200/ready", readyRec.Code, readyRec.Body.String())
	}
	healthRec := httptest.NewRecorder()
	srv.newProtocolRouter().ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRec.Code != http.StatusOK || healthRec.Body.String() != "ok" {
		t.Fatalf("healthz status/body = %d/%q, want 200/ok", healthRec.Code, healthRec.Body.String())
	}

	if err := srv.Restart(); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	assertServerReadyState(t, srv, true, "Restart()")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{HTTPServer: true, Listener: true, GRPCServer: true, GRPCClientPool: true}, "Restart()")

	if err := srv.restart(); err != nil {
		t.Fatalf("restart() error = %v", err)
	}
	assertServerReadyState(t, srv, true, "restart()")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{HTTPServer: true, Listener: true, GRPCServer: true, GRPCClientPool: true}, "restart()")
	if err := srv.stop(false); err != nil {
		t.Fatalf("stop(false) error = %v", err)
	}
	assertServerReadyState(t, srv, false, "stop(false)")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{}, "stop(false)")
	if srv.jsExecutor == nil {
		t.Fatal("expected stop(false) to leave jsExecutor allocated for potential reuse")
	}
}

func TestServerStartWithAuthAndGrpcWebProxyRegistersJobToken(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Server.EnableGrpcWebProxy = true
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.Type = "jwt"
	keyDir := t.TempDir()
	runtimeScope.cfg.Auth.JWT.PrivateKeyFile = filepath.Join(keyDir, "private.pem")
	runtimeScope.cfg.Auth.JWT.PublicKeyFile = filepath.Join(keyDir, "public.pem")
	runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
	runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = false

	reg := &trackingRegistry{}
	srv := &GRPCWebServer{runtimeScope: runtimeScope, registry: reg}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) with auth+proxy error = %v", err)
	}
	if srv.authenticator == nil {
		t.Fatal("expected start(false) to initialize authenticator when auth is enabled")
	}
	if srv.proxy == nil {
		t.Fatal("expected start(false) to initialize grpc-web proxy when enabled")
	}

	registeredJobToken := false
	for _, serviceName := range reg.registerCalls {
		if serviceName == "auth.JobTokenService" {
			registeredJobToken = true
			break
		}
	}
	if !registeredJobToken {
		t.Fatalf("expected JobTokenService registration, got %#v", reg.registerCalls)
	}
}

func TestServerRestartClearsProxyWhenGrpcWebProxyDisabled(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Server.EnableGrpcWebProxy = true
	runtimeScope.cfg.Server.HotReload = false

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) with grpc-web proxy error = %v", err)
	}
	if srv.proxy == nil {
		t.Fatal("expected start(false) to initialize grpc-web proxy when enabled")
	}

	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	if err := srv.restart(); err != nil {
		t.Fatalf("restart() after disabling grpc-web proxy error = %v", err)
	}
	if srv.proxy != nil {
		t.Fatal("expected restart() to clear grpc-web proxy when disabled")
	}
}

func TestServerStartContinuesWhenJobTokenRegistryRegistrationFails(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.Type = "jwt"
	keyDir := t.TempDir()
	runtimeScope.cfg.Auth.JWT.PrivateKeyFile = filepath.Join(keyDir, "private.pem")
	runtimeScope.cfg.Auth.JWT.PublicKeyFile = filepath.Join(keyDir, "public.pem")
	runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
	runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = false

	reg := &trackingRegistry{registerErr: errors.New("register failed")}
	srv := &GRPCWebServer{runtimeScope: runtimeScope, registry: reg}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) with registry registration warning error = %v", err)
	}
	assertServerReadyState(t, srv, true, "start(false) with registry registration warning")
	if len(reg.registerCalls) == 0 || reg.registerCalls[0] != "auth.JobTokenService" {
		t.Fatalf("expected JobTokenService registration attempt, got %#v", reg.registerCalls)
	}
}

func TestServerStartReturnsHTTPListenError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.BindAddress = "256.0.0.1"
	runtimeScope.cfg.Server.Port = 8080
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	t.Cleanup(func() {
		if srv.jsExecutor != nil || srv.grpcClientPool != nil || srv.server != nil {
			_ = srv.stop(false)
		}
	})

	err := srv.start(false)
	if err == nil {
		t.Fatal("expected start(false) to fail when HTTP listener cannot bind")
	}
	if !strings.Contains(err.Error(), "Failed to listen") {
		t.Fatalf("start(false) error = %v, want failed listen", err)
	}
}

func TestServerStartReturnsGrpcClientPoolError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnabledTLS = true
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.TLSCaFile = filepath.Join(t.TempDir(), "missing-ca.pem")

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	err := srv.start(false)
	if err == nil {
		t.Fatal("expected start(false) to fail when grpc client pool TLS CA file is missing")
	}
	if !strings.Contains(err.Error(), "failed to create grpc client pool") {
		t.Fatalf("start(false) error = %v, want grpc client pool creation failure", err)
	}
}

func TestServerStartReturnsRegisterApplicationServicesError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()

	srv := &GRPCWebServer{runtimeScope: runtimeScope, runState: runState{applicationNames: []string{"missing"}}}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	err := srv.start(false)
	if err == nil {
		t.Fatal("expected start(false) to fail when application service dist path is missing")
	}
	if !strings.Contains(err.Error(), "Failed to create application service") {
		t.Fatalf("start(false) error = %v, want register application services failure", err)
	}
	assertServerReadyState(t, srv, false, "failed start(false) register application services")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{}, "failed start(false) register application services")
	if srv.proxy != nil {
		t.Fatal("expected failed start(false) to clean grpc-web proxy during startup recovery")
	}
	if srv.jsExecutor != nil {
		t.Fatal("expected failed start(false) to clear js executor during startup recovery")
	}
}

func TestServerRunStartupLifecycleResultSuccess(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()

	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "auth", "globalThis.__startupLifecycleResultOk = true;")

	srv := &GRPCWebServer{runtimeScope: runtimeScope, runState: runState{applicationNames: []string{"auth"}}}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	result := srv.runStartupLifecycle(false, srv.resolvedRuntimeOptions())
	if result.errorValue() != nil {
		t.Fatalf("runStartupLifecycle() error = %v", result.errorValue())
	}
	assertStartupLifecycleStatus(t, result, true, true, "", "successful startup lifecycle")
	assertAuthRuntimeSetupResult(t, result.AuthRuntime, false, false, false, false, "successful startup lifecycle")
	assertModeRuntimeSummaryFields(t, result.ModeRuntime, runplan.RunModeApplication, 1, 1, true, true, "successful startup lifecycle")
	assertTaskRuntimeSummaryFields(t, result.TaskRuntime, false, false, false, false, false, "successful startup lifecycle")
	assertStartupCleanupState(t, result, "", startupRecoveryPlan{}, startupRecoveryReport{}, "successful startup lifecycle")
	assertServerReadyState(t, srv, true, "successful startup lifecycle")
}

func TestServerRunStartupLifecycleLogsStartedServicesOnExecutorStart(t *testing.T) {
	var logBuf bytes.Buffer
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.logger = slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	portListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	runtimeScope.cfg.Server.BindAddress = "127.0.0.1"
	runtimeScope.cfg.Server.Port = portListener.Addr().(*net.TCPAddr).Port
	_ = portListener.Close()
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()

	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "auth", "globalThis.__startupLifecycleLogsServices = true;")

	srv := &GRPCWebServer{runtimeScope: runtimeScope, runState: runState{applicationNames: []string{"auth"}}}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	result := srv.runStartupLifecycle(false, srv.resolvedRuntimeOptions())
	if result.errorValue() != nil {
		t.Fatalf("runStartupLifecycle() error = %v", result.errorValue())
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "\"msg\":\"js executor started\"") {
		t.Fatalf("expected js executor started info log, got %q", logs)
	}
	if !strings.Contains(logs, "\"service_count\":1") {
		t.Fatalf("expected started service count in js executor log, got %q", logs)
	}
	if !strings.Contains(logs, "\"services\":[\"auth\"]") {
		t.Fatalf("expected started service names in js executor log, got %q", logs)
	}
	if !strings.Contains(logs, "\"duration_ms\":") {
		t.Fatalf("expected js executor startup duration in log, got %q", logs)
	}
	if !strings.Contains(logs, "\"min_pool_size\":") {
		t.Fatalf("expected js executor min pool size in log, got %q", logs)
	}
	if !strings.Contains(logs, "\"max_pool_size\":") {
		t.Fatalf("expected js executor max pool size in log, got %q", logs)
	}
	if strings.Contains(logs, "application service registered") {
		t.Fatalf("expected fragmented service registration info log to stay hidden, got %q", logs)
	}
}

func TestServerRunStartupLifecycleCarriesAuthDegradeSummary(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.Type = "missing"
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()

	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "auth", "globalThis.__startupLifecycleAuthDegraded = true;")

	reg := &trackingRegistry{}
	srv := &GRPCWebServer{runtimeScope: runtimeScope, registry: reg, runState: runState{applicationNames: []string{"auth"}}}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	result := srv.runStartupLifecycle(false, srv.resolvedRuntimeOptions())
	if result.errorValue() != nil {
		t.Fatalf("runStartupLifecycle() with degraded auth error = %v", result.errorValue())
	}
	assertStartupLifecycleStatus(t, result, true, true, "", "successful startup lifecycle with degraded auth")
	assertAuthRuntimeSetupResult(t, result.AuthRuntime, true, false, true, true, "successful startup lifecycle with degraded auth")
	assertModeRuntimeSummaryFields(t, result.ModeRuntime, runplan.RunModeApplication, 1, 1, true, true, "successful startup lifecycle with degraded auth")
	assertTaskRuntimeSummaryFields(t, result.TaskRuntime, false, false, false, false, false, "successful startup lifecycle with degraded auth")
	if srv.authenticator != nil {
		t.Fatal("expected degraded auth startup to continue without authenticator")
	}
	for _, serviceName := range reg.registerCalls {
		if serviceName == "auth.JobTokenService" {
			t.Fatalf("expected degraded auth startup to skip JobTokenService registration, got %#v", reg.registerCalls)
		}
	}
	assertServerReadyState(t, srv, true, "successful startup lifecycle with degraded auth")
}

func TestServerStartReturnsJsExecutorStartError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()

	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "auth", "function () {")

	srv := &GRPCWebServer{runtimeScope: runtimeScope, runState: runState{applicationNames: []string{"auth"}}}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	err := srv.start(false)
	if err == nil {
		t.Fatal("expected start(false) to fail when init script has invalid JavaScript")
	}
	if !strings.Contains(err.Error(), "Failed to start js executor") {
		t.Fatalf("start(false) error = %v, want js executor start failure", err)
	}
	assertServerReadyState(t, srv, false, "failed start(false) js executor start")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{}, "failed start(false) js executor start")
	if srv.proxy != nil {
		t.Fatal("expected failed start(false) to clean grpc-web proxy during startup recovery")
	}
}

func TestServerRunStartupLifecycleResultFailureCarriesCleanupPlan(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.BindAddress = "256.0.0.1"
	runtimeScope.cfg.Server.Port = 8080
	runtimeScope.cfg.Server.EnableGrpcWebProxy = true
	runtimeScope.cfg.Server.HotReload = false

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	result := srv.runStartupLifecycle(false, srv.resolvedRuntimeOptions())
	if result.errorValue() == nil {
		t.Fatal("expected runStartupLifecycle() to fail when HTTP listener cannot bind")
	}
	assertStartupLifecycleStatus(t, result, false, false, startupPhaseTransportIngress, "failed startup lifecycle")
	assertAuthRuntimeSetupResult(t, result.AuthRuntime, false, false, false, false, "failed startup lifecycle")
	assertModeRuntimeSummaryFields(t, result.ModeRuntime, runplan.RunModeApplication, 0, 0, true, true, "failed startup lifecycle before ingress")
	assertTaskRuntimeSummaryFields(t, result.TaskRuntime, false, false, false, false, false, "failed startup lifecycle before ingress")
	assertStartupCleanupState(t, result, recoveryActionStartupCleanup, startupRecoveryPlan{
		clearJSRuntime: true,
		clearProxy:     true,
		clearTransport: true,
	}, startupRecoveryReport{
		PlannedJSRuntimeRecovery: true,
		PlannedProxyRecovery:     true,
		PlannedTransportRecovery: true,
		JSRuntimeCleared:         true,
		ProxyCleared:             true,
		TransportCleared:         true,
	}, "failed startup lifecycle")
	assertServerReadyState(t, srv, false, "failed startup lifecycle")
}

func TestServerStartReloadReturnsJsExecutorReloadError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = true
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.Port = 0
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()

	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "auth", "globalThis.__choysumReloadOk = true;")

	srv := &GRPCWebServer{runtimeScope: runtimeScope, runState: runState{applicationNames: []string{"auth"}}}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) error = %v", err)
	}
	if srv.proxy == nil {
		t.Fatal("expected start(false) to initialize grpc-web proxy when enabled")
	}
	if err := srv.stop(true); err != nil {
		t.Fatalf("stop(true) error = %v", err)
	}
	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "auth", "function () {")

	err := srv.start(true)
	if err == nil {
		t.Fatal("expected start(true) to fail when reload init script has invalid JavaScript")
	}
	if !strings.Contains(err.Error(), "Failed to reload js executor") {
		t.Fatalf("start(true) error = %v, want js executor reload failure", err)
	}
	assertServerReadyState(t, srv, false, "failed start(true) js executor reload")
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{}, "failed start(true) js executor reload")
	if srv.proxy != nil {
		t.Fatal("expected reload failure to clean grpc-web proxy during startup recovery")
	}
}

func TestServerStartFailureClearsStaleStateAfterCleanupError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.BindAddress = "256.0.0.1"
	runtimeScope.cfg.Server.Port = 8080
	runtimeScope.cfg.Server.EnableGrpcWebProxy = true
	runtimeScope.cfg.Server.HotReload = false

	reg := &trackingRegistry{unregisterAllErr: errors.New("unregister failed")}
	srv := &GRPCWebServer{runtimeScope: runtimeScope, registry: reg}

	err := srv.start(false)
	if err == nil {
		t.Fatal("expected start(false) to fail when HTTP listener cannot bind")
	}
	if !strings.Contains(err.Error(), "Failed to listen") {
		t.Fatalf("start(false) error = %v, want failed listen", err)
	}
	assertServerReadyState(t, srv, false, "failed start(false) cleanup error")
	if srv.jsExecutor != nil {
		t.Fatal("expected failed start(false) cleanup to clear js executor after cleanup error")
	}
	if srv.proxy != nil {
		t.Fatal("expected failed start(false) cleanup to clear grpc-web proxy after cleanup error")
	}
	assertTransportRuntimeState(t, srv, transportRuntimeStateTestSnapshot{}, "failed start(false) cleanup error")
	if reg.unregisterAllCnt != 1 {
		t.Fatalf("UnRegisterAll calls = %d, want 1", reg.unregisterAllCnt)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srv), recoveryActionStartupCleanup, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "failed startup cleanup diagnostics")
}

func TestServerRestartVariantsSurfaceReloadStartErrors(t *testing.T) {
	newStartedServer := func(t *testing.T) (*GRPCWebServer, *noSessionServerScope) {
		t.Helper()
		runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
		runtimeScope.cfg.Auth.Enabled = false
		runtimeScope.cfg.Server.Port = 0
		runtimeScope.cfg.Server.EnableGrpcWebProxy = false
		runtimeScope.cfg.Server.HotReload = false

		srv := &GRPCWebServer{runtimeScope: runtimeScope}
		if err := srv.start(false); err != nil {
			t.Fatalf("start(false) error = %v", err)
		}
		return srv, runtimeScope
	}

	srvRestart, envRestart := newStartedServer(t)
	envRestart.cfg.Server.JsEngineFactory = "missing"
	srvRestart.jsExecutor = nil
	err := srvRestart.Restart()
	if err == nil {
		t.Fatal("expected Restart() to fail when reload cannot rebuild js executor")
	}
	if strings.Contains(err.Error(), "Failed to start server") {
		t.Fatalf("Restart() error = %v, want direct reload failure without restart() wrapper", err)
	}
	if !strings.Contains(err.Error(), "Failed to create runtime executor") {
		t.Fatalf("Restart() error = %v, want runtime executor failure", err)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srvRestart), recoveryActionRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "Restart() reload start failure diagnostics")

	srvRestartHelper, envRestartHelper := newStartedServer(t)
	t.Cleanup(func() {
		if srvRestartHelper.httpServer != nil || srvRestartHelper.server != nil || srvRestartHelper.listener != nil || srvRestartHelper.grpcClientPool != nil {
			_ = srvRestartHelper.stop(false)
		}
	})
	envRestartHelper.cfg.Server.JsEngineFactory = "missing"
	srvRestartHelper.jsExecutor = nil
	err = srvRestartHelper.restart()
	if err == nil {
		t.Fatal("expected restart() to fail when reload cannot rebuild js executor")
	}
	if !strings.Contains(err.Error(), "Failed to start server") {
		t.Fatalf("restart() error = %v, want wrapped start failure", err)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srvRestartHelper), recoveryActionRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "restart() reload start failure diagnostics")
}

func TestServerRestartVariantsSurfaceStopErrors(t *testing.T) {
	newStartedServerWithRegistry := func(t *testing.T) (*GRPCWebServer, *trackingRegistry) {
		t.Helper()
		runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
		runtimeScope.cfg.Auth.Enabled = false
		runtimeScope.cfg.Server.Port = 0
		runtimeScope.cfg.Server.EnableGrpcWebProxy = false
		runtimeScope.cfg.Server.HotReload = false

		reg := &trackingRegistry{}
		srv := &GRPCWebServer{runtimeScope: runtimeScope, registry: reg}
		if err := srv.start(false); err != nil {
			t.Fatalf("start(false) error = %v", err)
		}
		return srv, reg
	}

	srvRestart, regRestart := newStartedServerWithRegistry(t)
	regRestart.unregisterAllErr = errors.New("unregister failed")
	err := srvRestart.Restart()
	if err == nil {
		t.Fatal("expected Restart() to fail when stop(true) cannot unregister services")
	}
	if strings.Contains(err.Error(), "Failed to stop server") {
		t.Fatalf("Restart() error = %v, want direct stop failure without restart() wrapper", err)
	}
	if !strings.Contains(err.Error(), "Failed to unregister all services") {
		t.Fatalf("Restart() error = %v, want unregister failure", err)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srvRestart), recoveryActionRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "Restart() stop failure diagnostics")

	srvRestartHelper, regRestartHelper := newStartedServerWithRegistry(t)
	t.Cleanup(func() {
		if srvRestartHelper.httpServer != nil || srvRestartHelper.server != nil || srvRestartHelper.listener != nil || srvRestartHelper.grpcClientPool != nil {
			_ = srvRestartHelper.stop(false)
		}
	})
	regRestartHelper.unregisterAllErr = errors.New("unregister failed")
	err = srvRestartHelper.restart()
	if err == nil {
		t.Fatal("expected restart() to fail when stop(true) cannot unregister services")
	}
	if !strings.Contains(err.Error(), "Failed to stop server") {
		t.Fatalf("restart() error = %v, want wrapped stop failure", err)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srvRestartHelper), recoveryActionRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "restart() stop failure diagnostics")
}
