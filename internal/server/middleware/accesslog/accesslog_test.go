// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testIdentity struct {
	userID string
	valid  bool
}

func (t testIdentity) GetUserID() string           { return t.userID }
func (t testIdentity) GetTokenID() string          { return "" }
func (t testIdentity) GetMetadata() map[string]any { return nil }
func (t testIdentity) IsValid() bool               { return t.valid }

func TestObserveIdentityAndUnaryAccess(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := UnaryInterceptor(Options{Access: true, SlowMs: 1000}, logger)

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Svc/Search"},
		func(ctx context.Context, _ any) (any, error) {
			ObserveIdentity(ctx, testIdentity{userID: "admin", valid: true})
			return "ok", nil
		})
	if err != nil {
		t.Fatalf("interceptor error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"msg":"request"`) || !strings.Contains(out, `"method":"/svc.Svc/Search"`) {
		t.Fatalf("missing access log: %s", out)
	}
	if !strings.Contains(out, `"user":"admin"`) || !strings.Contains(out, `"level":"INFO"`) {
		t.Fatalf("expected INFO with user: %s", out)
	}
}

func TestUnaryQuietSuccess(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	interceptor := UnaryInterceptor(Options{}, logger)
	_, _ = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x"},
		func(context.Context, any) (any, error) { return nil, nil })
	if buf.Len() != 0 {
		t.Fatalf("expected no log, got %s", buf.String())
	}
}

func TestUnaryFailureWarn(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := UnaryInterceptor(Options{}, logger)
	_, _ = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x"},
		func(context.Context, any) (any, error) {
			return nil, status.Error(codes.PermissionDenied, "no")
		})
	if !strings.Contains(buf.String(), `"level":"WARN"`) {
		t.Fatalf("expected WARN: %s", buf.String())
	}
}

func TestHTTPMiddlewareSkipAssets(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Middleware(Options{Access: true}, logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/assets/a.js", nil))
	if buf.Len() != 0 {
		t.Fatalf("assets should skip: %s", buf.String())
	}
}

func TestHTTPMiddlewareAccess(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := Middleware(Options{Access: true}, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.ContextWithIdentity(r.Context(), testIdentity{userID: "u1", valid: true})
		*r = *r.WithContext(ctx)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/web/sandbox/records", nil)
	h.ServeHTTP(rec, req)
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v body=%s", err, buf.String())
	}
	if payload["msg"] != "request" || payload["path"] != "/web/sandbox/records" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestStreamAccessStartEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := StreamInterceptor(Options{Access: true}, logger)
	ss := &fakeServerStream{ctx: context.Background()}
	err := interceptor(nil, ss, &grpc.StreamServerInfo{FullMethod: "/tip.TipHub/Subscribe"},
		func(any, grpc.ServerStream) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"stream":"start"`) || !strings.Contains(out, `"stream":"end"`) {
		t.Fatalf("expected start/end: %s", out)
	}
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}
func (f *fakeServerStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeServerStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeServerStream) SetTrailer(metadata.MD)       {}
func (f *fakeServerStream) SendMsg(any) error            { return nil }
func (f *fakeServerStream) RecvMsg(any) error            { return nil }
