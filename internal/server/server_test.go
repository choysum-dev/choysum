// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	_ "github.com/choysum-dev/choysum/internal/jwtauth"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/internal/server/transport"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"github.com/choysum-dev/choysum/pkg/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

func TestServerHelpers(t *testing.T) {
	srv := &GRPCWebServer{address: &resolver.Address{Addr: "127.0.0.1:9527"}}
	if srv.hasGrpcMethod("/auth.User/Get") {
		t.Fatal("expected hasGrpcMethod to be false for nil index")
	}
	srv.registration.storeBindings(nil, map[string]struct{}{"/auth.User/Get": {}})
	if !srv.hasGrpcMethod("/auth.User/Get") {
		t.Fatal("expected hasGrpcMethod to find registered method")
	}

	runtimeScope := newServerTestScope()
	srv.runtimeScope = runtimeScope
	if err := srv.initGRPCClientPool(); err != nil {
		t.Fatalf("initGRPCClientPool() error = %v", err)
	}
	if srv.grpcClientPool == nil || srv.grpcClientPool.MaxConns() != runtimeScope.cfg.Server.GrpcClient.MaxCachedConns {
		t.Fatalf("unexpected grpc client pool after init: %#v", srv.grpcClientPool)
	}
	if len(srv.baseUnaryInterceptors()) != 1 {
		t.Fatalf("expected base unary interceptors to include grpc pool interceptor")
	}
}

func TestServerMoreInterceptorsAndTelemetry(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Auth.Enabled = false
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	serverOpts := []grpc.ServerOption{}
	baseUnary := grpc.UnaryServerInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	})
	unary := []grpc.UnaryServerInterceptor{baseUnary}
	authResult := srv.setupAuthInterceptors(&serverOpts, &unary)
	assertAuthRuntimeSetupResult(t, authResult, false, false, false, false, "auth-disabled setup")
	if len(serverOpts) != 0 || len(unary) != 1 {
		t.Fatalf("expected auth-disabled setup to be a no-op, got opts=%d unary=%d", len(serverOpts), len(unary))
	}

	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.Type = "missing"
	executor := jsexecutortest.NewUninitializedExecutor()
	srv.jsExecutor = executor
	serverOpts = nil
	unary = []grpc.UnaryServerInterceptor{baseUnary}
	authResult = srv.setupAuthInterceptors(&serverOpts, &unary)
	assertAuthRuntimeSetupResult(t, authResult, true, false, true, true, "auth setup error path")
	if srv.jsExecutor != executor {
		t.Fatal("expected setupAuthInterceptors error path to leave js executor ownership unchanged")
	}
	if srv.authenticator != nil || len(serverOpts) != 0 || len(unary) != 1 {
		t.Fatalf("expected setupAuthInterceptors error path to leave grpc auth state unchanged, got authenticator=%v opts=%d unary=%d", srv.authenticator != nil, len(serverOpts), len(unary))
	}

	runtimeScope.cfg.Auth.Type = "jwt"
	keyDir := t.TempDir()
	runtimeScope.cfg.Auth.JWT.PrivateKeyFile = filepath.Join(keyDir, "private.pem")
	runtimeScope.cfg.Auth.JWT.PublicKeyFile = filepath.Join(keyDir, "public.pem")
	runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
	runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = false
	serverOpts = nil
	unary = []grpc.UnaryServerInterceptor{baseUnary}
	authResult = srv.setupAuthInterceptors(&serverOpts, &unary)
	assertAuthRuntimeSetupResult(t, authResult, true, true, false, false, "auth setup success path")
	if srv.authenticator == nil || len(serverOpts) != 1 || len(unary) != 2 {
		t.Fatalf("expected auth interceptors to be installed, got authenticator=%v opts=%d unary=%d", srv.authenticator != nil, len(serverOpts), len(unary))
	}

	tel := fakeTelemetry{}
	srv.telemetry = tel
	serverOpts = nil
	srv.ensureTelemetry(&serverOpts)
	if len(serverOpts) != 1 {
		t.Fatalf("expected ensureTelemetry to append server options, got %d", len(serverOpts))
	}

	srv.telemetry = nil
	serverOpts = nil
	srv.ensureTelemetry(&serverOpts)
	if srv.telemetry == nil || len(serverOpts) == 0 {
		t.Fatalf("expected ensureTelemetry to auto-initialize telemetry, got telemetry=%v opts=%d", srv.telemetry != nil, len(serverOpts))
	}
	t.Cleanup(func() {
		if srv.telemetry != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.telemetry.Shutdown(ctx)
			if tm, ok := srv.telemetry.(interface{ MetricShutdown(context.Context) error }); ok {
				_ = tm.MetricShutdown(ctx)
			}
		}
	})

	srv.jsExecutor = jsexecutortest.NewUninitializedExecutor()
	if err := srv.ensureJsExecutor(true); err != nil {
		t.Fatalf("ensureJsExecutor(reload=true) error = %v", err)
	}
	runtimeScope.cfg.Server.JsEngineFactory = "missing"
	srv.jsExecutor = nil
	if err := srv.ensureJsExecutor(false); err == nil {
		t.Fatal("expected ensureJsExecutor to fail when js engine factory is missing")
	}
}

func TestServerMoreHTTPAndProtocolRouterPaths(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	startSrv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		address:      &resolver.Address{Addr: "127.0.0.1:0"},
		mux:          http.NewServeMux(),
		server:       grpc.NewServer(),
	}

	startSrv.mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	if err := startSrv.startHTTPServer(http.NotFoundHandler()); err != nil {
		t.Fatalf("startHTTPServer() error = %v", err)
	}
	if startSrv.listener == nil || startSrv.httpServer == nil {
		t.Fatal("expected startHTTPServer to initialize listener and http server")
	}
	t.Cleanup(func() {
		if startSrv.httpServer != nil {
			_ = startSrv.httpServer.Shutdown(context.Background())
		}
		if startSrv.listener != nil {
			_ = startSrv.listener.Close()
		}
	})

	handler := startSrv.newProtocolRouter()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("newProtocolRouter non-grpc status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	tlsRuntimeScope := newRichServerTestScope(t)
	tlsRuntimeScope.cfg.Server.EnabledTLS = true
	caPath, certPath, keyPath := writeTestTLSFiles(t)
	tlsRuntimeScope.cfg.Server.TLSCaFile = caPath
	tlsRuntimeScope.cfg.Server.TLSCertFile = certPath
	tlsRuntimeScope.cfg.Server.TLSKeyFile = keyPath
	tlsListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	tlsSrv := &GRPCWebServer{
		runtimeScope: tlsRuntimeScope,
		address:      &resolver.Address{Addr: tlsListener.Addr().String()},
		listener:     tlsListener,
	}
	if err := tlsSrv.startHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})); err != nil {
		t.Fatalf("startHTTPServer() with TLS error = %v", err)
	}
	if tlsSrv.listener != tlsListener {
		t.Fatal("expected startHTTPServer to reuse existing listener")
	}
	t.Cleanup(func() {
		if tlsSrv.httpServer != nil {
			_ = tlsSrv.httpServer.Shutdown(context.Background())
		}
		if tlsSrv.listener != nil {
			_ = tlsSrv.listener.Close()
		}
	})

	tlsClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	respTLS, err := tlsClient.Get("https://" + tlsListener.Addr().String())
	if err != nil {
		t.Fatalf("TLS GET error = %v", err)
	}
	defer respTLS.Body.Close()
	if respTLS.StatusCode != http.StatusNoContent {
		t.Fatalf("TLS status = %d, want %d", respTLS.StatusCode, http.StatusNoContent)
	}

	proxySrv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		mux:          http.NewServeMux(),
		server:       grpc.NewServer(),
	}
	proxySrv.proxy = proxySrv.newGRPCWebProxy()
	blocked := httptest.NewRequest(http.MethodPost, "/auth.JobTokenService/IssueTaskJobToken", nil)
	blocked.Header.Set("content-type", "application/grpc-web+proto")
	blockedRR := httptest.NewRecorder()
	proxySrv.newProtocolRouter().ServeHTTP(blockedRR, blocked)
	if blockedRR.Code != http.StatusNotFound {
		t.Fatalf("newProtocolRouter grpc-web internal route status = %d, want %d", blockedRR.Code, http.StatusNotFound)
	}

	interceptor := proxySrv.taskRuntimeWakeInterceptor()
	resp, err := interceptor(context.Background(), nil, nil, func(ctx context.Context, req any) (any, error) { return "ok", nil })
	if err != nil || resp != "ok" {
		t.Fatalf("taskRuntimeWakeInterceptor nil-info result = %#v err=%v", resp, err)
	}
	for _, method := range []string{"/task.Job/EnqueueJob", "/task.Schedule/TriggerSchedule"} {
		resp, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, func(ctx context.Context, req any) (any, error) { return method, nil })
		if err != nil || resp != method {
			t.Fatalf("taskRuntimeWakeInterceptor(%q) result = %#v err=%v", method, resp, err)
		}
	}
}

func TestServerProtocolRouterAppliesHTTPMiddleware(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.EnabledTLS = true
	runtimeScope.cfg.Server.EnableGzip = true
	runtimeScope.cfg.Auth.HttpAuth.ExcludedPaths = []string{"/health", "/healthz", "/readyz"}

	authn := &fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		if token != "header-token" {
			t.Fatalf("ValidateToken token = %q, want header-token", token)
		}
		if tokenType != auth.AccessToken {
			t.Fatalf("ValidateToken tokenType = %q, want access", tokenType)
		}
		if !checkRevoked {
			t.Fatal("ValidateToken checkRevoked = false, want true")
		}
		return serverTestIdentity{userID: "u1", tokenID: "tok1", valid: true}, nil
	}}

	var gotIdentity auth.Identity
	var gotToken string
	var gotTokenOK bool
	srv := &GRPCWebServer{
		runtimeScope:  runtimeScope,
		mux:           http.NewServeMux(),
		authenticator: authn,
		server:        grpc.NewServer(),
	}
	srv.mux.HandleFunc("/secure", func(w http.ResponseWriter, r *http.Request) {
		gotIdentity = auth.IdentityFromContext(r.Context())
		gotToken, gotTokenOK = auth.AccessTokenFromContext(r.Context())
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/secure", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	srv.newProtocolRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("newProtocolRouter GET status = %d, want %d", rr.Code, http.StatusOK)
	}
	if gotIdentity == nil || gotIdentity.GetUserID() != "u1" {
		t.Fatalf("identity in context = %#v, want user u1", gotIdentity)
	}
	if !gotTokenOK || gotToken != "header-token" {
		t.Fatalf("access token in context = %q ok=%v, want header-token", gotToken, gotTokenOK)
	}
	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := rr.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}
	if got := rr.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("Content-Security-Policy = %q, want default-src", got)
	}
	if got := rr.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=") {
		t.Fatalf("Strict-Transport-Security = %q, want max-age", got)
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}

	result := rr.Result()
	defer result.Body.Close()
	var hasCSRFCookie bool
	for _, cookie := range result.Cookies() {
		if cookie.Name == runtimeScope.cfg.Server.Security.CSRF.CookieName && cookie.Value != "" {
			hasCSRFCookie = true
			break
		}
	}
	if !hasCSRFCookie {
		t.Fatalf("expected CSRF cookie %q to be set", runtimeScope.cfg.Server.Security.CSRF.CookieName)
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(rr.Body.Bytes()))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer gzReader.Close()
	body, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("ReadAll(gzip body) error = %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("gzip body = %q, want ok", string(body))
	}
}

func TestServerProtocolRouterRejectsInvalidCSRFBeforeAuth(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Auth.HttpAuth.ExcludedPaths = []string{"/health", "/healthz", "/readyz"}
	authCalls := 0
	authn := &fakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
		authCalls++
		return serverTestIdentity{userID: "u1", tokenID: "tok1", valid: true}, nil
	}}

	srv := &GRPCWebServer{
		runtimeScope:  runtimeScope,
		mux:           http.NewServeMux(),
		authenticator: authn,
		server:        grpc.NewServer(),
	}
	srv.mux.HandleFunc("/secure", func(http.ResponseWriter, *http.Request) {
		t.Fatal("expected CSRF middleware to block request before handler")
	})

	req := httptest.NewRequest(http.MethodPost, "http://example.com/secure", strings.NewReader(`{"name":"choysum"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer header-token")
	rr := httptest.NewRecorder()

	srv.newProtocolRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("newProtocolRouter POST status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if authCalls != 1 {
		t.Fatalf("ValidateToken calls = %d, want 1", authCalls)
	}
	if got := rr.Header().Get("X-CSRF-Error"); got != "true" {
		t.Fatalf("X-CSRF-Error = %q, want true", got)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if body := rr.Body.String(); !strings.Contains(body, "csrf_error") {
		t.Fatalf("response body = %q, want csrf_error payload", body)
	}
	result := rr.Result()
	defer result.Body.Close()
	var hasCSRFCookie bool
	for _, cookie := range result.Cookies() {
		if cookie.Name == runtimeScope.cfg.Server.Security.CSRF.CookieName && cookie.Value != "" {
			hasCSRFCookie = true
			break
		}
	}
	if !hasCSRFCookie {
		t.Fatalf("expected CSRF cookie %q to be set on rejected request", runtimeScope.cfg.Server.Security.CSRF.CookieName)
	}
}

func TestServerOptionsAndConstructor(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	reg := fakeRegistry{}
	exec := jsexecutortest.NewUninitializedExecutor()
	tel := fakeTelemetry{}

	created := NewServer(runtimeScope, WithRegistry(reg), WithExecutor(exec), WithTelemetry(tel))
	srv, ok := created.(*GRPCWebServer)
	if !ok {
		t.Fatalf("expected *GRPCWebServer, got %T", created)
	}
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
	})

	if srv.registry == nil || srv.jsExecutor != exec || srv.telemetry == nil {
		t.Fatalf("unexpected constructor state: registry=%v jsExecutor=%v telemetry=%v", srv.registry, srv.jsExecutor, srv.telemetry)
	}
	if srv.hotreloadWatcher() != nil || srv.hotreloadQueue() != nil {
		t.Fatal("expected NewServer not to allocate hotreload watcher or queue before hotreload lifecycle starts")
	}

	var telemetryIface trace.Telemetry = tel
	WithTelemetry(telemetryIface).apply(srv)
	if srv.telemetry == nil {
		t.Fatal("expected WithTelemetry option to store telemetry")
	}
	WithRegistry(reg).apply(srv)
	if srv.registry == nil {
		t.Fatal("expected WithRegistry option to store registry")
	}
	WithExecutor(exec).apply(srv)
	if srv.jsExecutor != exec {
		t.Fatal("expected WithExecutor option to store executor")
	}
}

func TestServerTaskRuntimeHelpers(t *testing.T) {
	noSessionRuntimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	nilPoolSrv := &GRPCWebServer{runtimeScope: noSessionRuntimeScope, runState: runState{applicationNames: []string{"task"}}}
	nilPoolSrv.startTaskRuntime()
	assertTaskRuntimeStopped(t, nilPoolSrv, "nil grpc client pool")

	taskSrv := &GRPCWebServer{runtimeScope: noSessionRuntimeScope, grpcClientPool: &transport.GRPCClientPool{}, runState: runState{applicationNames: []string{"auth"}}}
	taskSrv.startTaskRuntime()
	assertTaskRuntimeStopped(t, taskSrv, "non-task applications")

	restoreRunStateForTest(taskSrv, runStateSnapshot{applicationNames: []string{"task"}})
	taskSrv.startTaskRuntime()
	startedTaskRuntime := assertTaskRuntimeStarted(t, taskSrv, "task application startup")
	taskSrv.startTaskRuntime()
	assertTaskRuntimeSnapshot(t, taskSrv.taskRuntimeTestSnapshot(), startedTaskRuntime, "task runtime start no-op")
	t.Cleanup(func() {
		stopTaskRuntimeForTest(taskSrv)
	})
	var runtimeFactoryCalls atomic.Int32
	trackingGC := &trackingTaskGarbageCollector{}
	customRuntimeSrv := &GRPCWebServer{
		runtimeScope:   noSessionRuntimeScope,
		grpcClientPool: &transport.GRPCClientPool{},
		runState:       runState{applicationNames: []string{"task"}},
		taskRuntime: taskRuntimeState{hostRuntimeProvider: func(scope.Scope) taskcontract.Runtime {
			runtimeFactoryCalls.Add(1)
			return taskcontract.Runtime{Collector: trackingGC}
		}},
	}
	customRuntimeSrv.startTaskRuntime()
	customTaskRuntime := assertTaskRuntimeStarted(t, customRuntimeSrv, "custom task runtime startup")
	if got := runtimeFactoryCalls.Load(); got != 1 {
		t.Fatalf("expected task host runtime provider to be called once, got %d", got)
	}
	if customTaskRuntime.GarbageCollector != trackingGC {
		t.Fatal("expected custom task collector from task host runtime provider")
	}
	if trackingGC.startCalls != 1 {
		t.Fatalf("expected custom task collector to start once, got %d", trackingGC.startCalls)
	}
	customRuntimeSrv.startTaskRuntime()
	if got := runtimeFactoryCalls.Load(); got != 1 {
		t.Fatalf("expected startTaskRuntime no-op to avoid rerunning task host runtime provider, got %d", got)
	}
	t.Cleanup(func() {
		stopTaskRuntimeForTest(customRuntimeSrv)
		if trackingGC.stopCalls == 0 {
			t.Fatal("expected custom task collector stop to be callable during cleanup")
		}
	})
}

func TestServerTaskRuntimeStartDoesNotEmitInfoComponentLogs(t *testing.T) {
	var logBuf bytes.Buffer
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	srv := &GRPCWebServer{
		runtimeScope:   runtimeScope,
		grpcClientPool: &transport.GRPCClientPool{},
		runState:       runState{applicationNames: []string{"task"}},
	}
	srv.startTaskRuntime()
	t.Cleanup(func() {
		stopTaskRuntimeForTest(srv)
	})

	logs := logBuf.String()
	if strings.Contains(logs, "task dispatcher started") || strings.Contains(logs, "task scheduler started") || strings.Contains(logs, "task garbage collector started") {
		t.Fatalf("expected task runtime component startup logs to be hidden at info level, got %q", logs)
	}
}

func TestServerBootstrapModeStartupDoesNotEmitInfoDetailLogs(t *testing.T) {
	var logBuf bytes.Buffer
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		runState: runState{
			runMode:       runplan.RunModeBootstrap,
			runModeReason: "no app is ready to serve yet",
		},
	}

	srv.finishBootstrapModeStartup(bootstrapModeStartupResult{ServiceName: "bootstrap"})

	if logs := logBuf.String(); strings.Contains(logs, "bootstrap service registered") {
		t.Fatalf("expected bootstrap service registration detail log to be hidden at info level, got %q", logs)
	}
	if got := srv.runState.serviceTargets(); len(got) != 1 || got[0] != "bootstrap" {
		t.Fatalf("bootstrap service targets = %#v, want [bootstrap]", got)
	}
}
