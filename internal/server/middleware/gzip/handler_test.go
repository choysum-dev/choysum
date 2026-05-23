// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gzipmiddleware

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
)

type stubScope struct {
	logger *slog.Logger
}

func (e *stubScope) Run(func(scope.Scope) error) error { return nil }

func (e *stubScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *stubScope) Session() *scope.Session { return nil }

func (e *stubScope) WithContext(ctx context.Context) scope.Scope { return e }

func (e *stubScope) Context() context.Context { return context.Background() }

func (e *stubScope) Logger() *slog.Logger { return e.logger }

func (e *stubScope) Config() *config.Config { return nil }

func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func testHandlerScope() scope.Scope {
	return &stubScope{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestGzipResponseWriterWriteHeaderAndWrite(t *testing.T) {
	recorder := httptest.NewRecorder()
	buffer := bytes.NewBuffer(nil)
	writer := &gzipResponseWriter{Writer: buffer, ResponseWriter: recorder, statusCode: http.StatusOK}

	writer.WriteHeader(http.StatusCreated)
	if writer.statusCode != http.StatusCreated {
		t.Fatalf("statusCode = %d, want %d", writer.statusCode, http.StatusCreated)
	}
	if recorder.Code != http.StatusCreated {
		t.Fatalf("recorder status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	if _, err := writer.Write([]byte("payload")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if buffer.String() != "payload" {
		t.Fatalf("buffer = %q, want payload", buffer.String())
	}
}

func TestGzipHandlerPassesThroughWhenClientDoesNotAcceptGzip(t *testing.T) {
	handler := NewGzipHandler(testHandlerScope())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "plain")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("plain-response"))
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if recorder.Header().Get("Content-Encoding") != "" {
		t.Fatalf("unexpected Content-Encoding header: %q", recorder.Header().Get("Content-Encoding"))
	}
	if recorder.Body.String() != "plain-response" {
		t.Fatalf("body = %q, want plain-response", recorder.Body.String())
	}
	if recorder.Header().Get("X-Test") != "plain" {
		t.Fatalf("expected downstream header to be preserved, got %q", recorder.Header().Get("X-Test"))
	}
}

func TestGzipHandlerCompressesResponseWhenRequested(t *testing.T) {
	handler := NewGzipHandler(testHandlerScope())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")

	handler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("compressed-response"))
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", recorder.Header().Get("Content-Encoding"))
	}
	if recorder.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want Accept-Encoding", recorder.Header().Get("Vary"))
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gzReader.Close()
	decoded, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("ReadAll gzip body: %v", err)
	}
	if string(decoded) != "compressed-response" {
		t.Fatalf("decoded body = %q, want compressed-response", string(decoded))
	}
}
