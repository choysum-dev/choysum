// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

func TestGRPCWebProxyRejectsTaskWorkerRequests(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	proxySrv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		server:       grpc.NewServer(),
	}
	proxy := proxySrv.newGRPCWebProxy()
	req := httptest.NewRequest(http.MethodPost, "/foo.TaskWorker/Run", nil)
	req.Header.Set("content-type", "application/grpc-web+proto")
	req.Header.Set("x-grpc-web", "1")
	req.Header.Set("te", "trailers")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("proxy status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Grpc-Status"); got != "12" {
		t.Fatalf("grpc status = %q, want 12", got)
	}
	if got := rec.Header().Get("Grpc-Message"); got != "TaskWorker is not available via gRPC-Web" {
		t.Fatalf("grpc message = %q, want TaskWorker is not available via gRPC-Web", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/grpc-web+proto" {
		t.Fatalf("content-type = %q, want application/grpc-web+proto", got)
	}
}

func TestGRPCWebProxyHandlesNonTaskWorkerRequests(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	proxySrv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		address:      &resolver.Address{Addr: "127.0.0.1:1"},
		server:       grpc.NewServer(),
	}
	proxy := proxySrv.newGRPCWebProxy()

	req := httptest.NewRequest(http.MethodPost, "/auth.User/Get", bytes.NewReader([]byte{0, 0, 0, 0, 0}))
	req.Header.Set("content-type", "application/grpc-web+proto")
	req.Header.Set("x-grpc-web", "1")
	req.Header.Set("te", "trailers")
	req.Header.Set("user-agent", "grpc-web-javascript/0.1")
	req.Header.Set("connection", "keep-alive")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("proxy status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/grpc-web+proto" {
		t.Fatalf("content-type = %q, want application/grpc-web+proto", got)
	}
	if got := rec.Header().Get("Grpc-Status"); got == "" {
		t.Fatal("expected grpc-web response to include grpc status")
	}
}

func TestGRPCWebProxyHandlesRegistryAndTLSBackends(t *testing.T) {
	caPath, certPath, keyPath := writeTestTLSFiles(t)
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	backendCreds, err := credentials.NewServerTLSFromFile(certPath, keyPath)
	if err != nil {
		t.Fatalf("NewServerTLSFromFile() error = %v", err)
	}
	backendServer := grpc.NewServer(
		grpc.Creds(backendCreds),
		grpc.UnknownServiceHandler(func(srv any, stream grpc.ServerStream) error {
			return status.Error(codes.Unimplemented, "backend unavailable")
		}),
	)
	go func() { _ = backendServer.Serve(backendListener) }()
	t.Cleanup(func() {
		backendServer.Stop()
		_ = backendListener.Close()
	})

	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.EnabledTLS = true
	runtimeScope.cfg.Server.TLSCaFile = caPath
	runtimeScope.cfg.Server.TLSServerName = "example.internal"

	builder := fixedResolverBuilder{scheme: "fixed", addr: backendListener.Addr().String()}
	proxySrv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		address:      &resolver.Address{Addr: "127.0.0.1:1"},
		registry:     fixedResolverRegistry{scheme: "fixed", builder: builder},
		server:       grpc.NewServer(),
	}
	proxy := proxySrv.newGRPCWebProxy()

	req := httptest.NewRequest(http.MethodPost, "/auth.User/Get", bytes.NewReader([]byte{0, 0, 0, 0, 0}))
	req.Header.Set("content-type", "application/grpc-web+proto")
	req.Header.Set("x-grpc-web", "1")
	req.Header.Set("te", "trailers")
	req.Header.Set("user-agent", "grpc-web-javascript/0.1")
	req.Header.Set("connection", "keep-alive")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("registry+tls proxy status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/grpc-web+proto" {
		t.Fatalf("content-type = %q, want application/grpc-web+proto", got)
	}
	if got := rec.Header().Get("Grpc-Status"); got == "" {
		t.Fatal("expected registry+tls grpc-web response to include grpc status")
	}
}
