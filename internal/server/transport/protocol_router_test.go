// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

type protocolRouterScope struct {
	logger *slog.Logger
}

func (s *protocolRouterScope) Run(func(scope.Scope) error) error { return nil }
func (s *protocolRouterScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *protocolRouterScope) Session() *scope.Session { return nil }
func (s *protocolRouterScope) WithContext(context.Context) scope.Scope {
	return s
}
func (s *protocolRouterScope) Context() context.Context { return context.Background() }
func (s *protocolRouterScope) Logger() *slog.Logger     { return s.logger }
func (s *protocolRouterScope) Config() *config.Config   { return nil }
func (s *protocolRouterScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.Config())
}

func TestProtocolRouterDoesNotGzipGRPCWeb(t *testing.T) {
	runtimeScope := &protocolRouterScope{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	grpcServer := grpc.NewServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("http-body-for-gzip"))
	})
	handler := NewProtocolRouter(ProtocolRouterOptions{
		RuntimeScope: runtimeScope,
		Mux:          mux,
		GRPCServer:   grpcServer,
		GRPCWebProxy: grpcweb.WrapServer(grpcServer),
		EnableGzip:   true,
	})

	grpcWebReq := httptest.NewRequest(http.MethodPost, "/tip.TipHub/SubscribeThread", nil)
	grpcWebReq.Header.Set("Content-Type", "application/grpc-web+proto")
	grpcWebReq.Header.Set("Accept-Encoding", "gzip")
	grpcWebRR := httptest.NewRecorder()
	handler.ServeHTTP(grpcWebRR, grpcWebReq)
	if grpcWebRR.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("grpc-web responses must not be gzip-wrapped")
	}

	httpReq := httptest.NewRequest(http.MethodGet, "/", nil)
	httpReq.Header.Set("Accept-Encoding", "gzip")
	httpRR := httptest.NewRecorder()
	handler.ServeHTTP(httpRR, httpReq)
	if httpRR.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("HTTP Content-Encoding = %q, want gzip", httpRR.Header().Get("Content-Encoding"))
	}
	gzReader, err := gzip.NewReader(bytes.NewReader(httpRR.Body.Bytes()))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gzReader.Close()
	decoded, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(decoded) != "http-body-for-gzip" {
		t.Fatalf("decoded HTTP body = %q", decoded)
	}
}
