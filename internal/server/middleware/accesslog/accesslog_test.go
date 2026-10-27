// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
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

func TestUnaryNilInfoAndNilLogger(t *testing.T) {
	t.Parallel()
	interceptor := UnaryInterceptor(Options{Access: true}, nil)
	_, err := interceptor(context.Background(), nil, nil,
		func(context.Context, any) (any, error) { return nil, nil })
	if err != nil {
		t.Fatalf("nil logger must not panic: %v", err)
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

func TestHTTPMiddlewareAccessWithObserveIdentity(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := Middleware(Options{Access: true}, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ObserveIdentity(r.Context(), testIdentity{userID: "u1", valid: true})
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
	if payload["user"] != "u1" {
		t.Fatalf("expected user u1, got %#v", payload["user"])
	}
}

func TestHTTPMiddlewareNilNextAndQuiet(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Middleware(Options{}, logger)(nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/x", nil))
	if buf.Len() != 0 {
		t.Fatalf("quiet success should not log: %s", buf.String())
	}
}

func TestHTTPMiddlewareNilRequest(t *testing.T) {
	t.Parallel()
	h := Middleware(Options{Access: true}, slog.Default())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	h.ServeHTTP(httptest.NewRecorder(), nil)
}

func TestStatusRecorderFlushHijackUnwrapPush(t *testing.T) {
	t.Parallel()
	flushCalled := false
	pushCalled := false
	inner := &flushHijackPushWriter{
		ResponseRecorder: httptest.NewRecorder(),
		onFlush:          func() { flushCalled = true },
		onPush:           func() { pushCalled = true },
	}
	rec := &statusRecorder{ResponseWriter: inner, status: http.StatusOK}
	rec.Flush()
	if !flushCalled {
		t.Fatal("Flush should forward")
	}
	if rec.Unwrap() != inner {
		t.Fatal("Unwrap mismatch")
	}
	_, _, err := rec.Hijack()
	if err == nil {
		t.Fatal("expected Hijack error from stub")
	}
	if err := rec.Push("/x.js", nil); err != nil {
		t.Fatalf("Push forward: %v", err)
	}
	if !pushCalled {
		t.Fatal("Push should forward")
	}

	plain := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	plain.Flush() // no-op without Flusher
	_, _, err = plain.Hijack()
	if !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("Hijack without Hijacker: %v", err)
	}
	if err := plain.Push("/x.js", nil); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("Push without Pusher: %v", err)
	}
	n, err := plain.Write([]byte("hi"))
	if err != nil || n != 2 || plain.status != http.StatusOK {
		t.Fatalf("Write without prior header: n=%d err=%v status=%d", n, err, plain.status)
	}
	plain.WriteHeader(http.StatusCreated) // ignored after first write
	if plain.status != http.StatusOK {
		t.Fatalf("second WriteHeader should be ignored, status=%d", plain.status)
	}
}

type flushHijackPushWriter struct {
	*httptest.ResponseRecorder
	onFlush func()
	onPush  func()
}

func (f *flushHijackPushWriter) Flush() {
	if f.onFlush != nil {
		f.onFlush()
	}
	f.ResponseRecorder.Flush()
}

func (f *flushHijackPushWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("hijack stub")
}

func (f *flushHijackPushWriter) Push(string, *http.PushOptions) error {
	if f.onPush != nil {
		f.onPush()
	}
	return nil
}

func TestStreamAccessStartEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := StreamInterceptor(Options{Access: true}, logger)
	ss := &fakeServerStream{ctx: context.Background()}
	err := interceptor(nil, ss, &grpc.StreamServerInfo{FullMethod: "/tip.TipHub/Subscribe"},
		func(_ any, stream grpc.ServerStream) error {
			ObserveIdentity(stream.Context(), testIdentity{userID: "s1", valid: true})
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
	if !strings.Contains(out, `"user":"s1"`) {
		t.Fatalf("expected end user: %s", out)
	}
}

func TestStreamCanceledLogsEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := StreamInterceptor(Options{Access: true}, logger)
	err := interceptor(nil, &fakeServerStream{}, &grpc.StreamServerInfo{FullMethod: "/tip.TipHub/Subscribe"},
		func(any, grpc.ServerStream) error {
			return status.Error(codes.Canceled, "client gone")
		})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"stream":"start"`) || !strings.Contains(out, `"stream":"end"`) {
		t.Fatalf("canceled must log start/end: %s", out)
	}
	if !strings.Contains(out, `"code":"Canceled"`) {
		t.Fatalf("expected Canceled code: %s", out)
	}
}

func TestStreamQuietFailureStillLogs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := StreamInterceptor(Options{}, logger)
	_ = interceptor(nil, &fakeServerStream{}, nil,
		func(any, grpc.ServerStream) error {
			return status.Error(codes.Internal, "boom")
		})
	out := buf.String()
	if strings.Contains(out, `"stream":"start"`) {
		t.Fatalf("quiet mode should not log start: %s", out)
	}
	if !strings.Contains(out, `"stream":"end"`) || !strings.Contains(out, `"level":"ERROR"`) {
		t.Fatalf("expected ERROR end: %s", out)
	}
}

func TestStreamCanceledQuietNoLog(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := StreamInterceptor(Options{}, logger)
	_ = interceptor(nil, &fakeServerStream{}, &grpc.StreamServerInfo{FullMethod: "/x"},
		func(any, grpc.ServerStream) error {
			return status.Error(codes.Canceled, "gone")
		})
	if buf.Len() != 0 {
		t.Fatalf("quiet cancel should not log: %s", buf.String())
	}
}

func TestIdentityCaptureHelpers(t *testing.T) {
	t.Parallel()
	ctx := ContextWithIdentityCapture(nil)
	ctx2 := ContextWithIdentityCapture(ctx)
	if ctx2 != ctx {
		t.Fatal("second capture attach should reuse slot")
	}
	ObserveIdentity(ctx, nil)
	ObserveIdentity(ctx, testIdentity{userID: "  bob  ", valid: true})
	if got := observedUserID(ctx); got != "bob" {
		t.Fatalf("observedUserID=%q", got)
	}
	ObserveIdentity(context.Background(), testIdentity{userID: "x", valid: true}) // no-op
	if identityCaptureFrom(nil) != nil {
		t.Fatal("nil ctx capture")
	}
	authCtx := auth.ContextWithIdentity(context.Background(), testIdentity{userID: "from-auth", valid: true})
	if got := observedUserID(authCtx); got != "from-auth" {
		t.Fatalf("fallback IdentityFromContext: %q", got)
	}
	invalidAuth := auth.ContextWithIdentity(context.Background(), testIdentity{userID: "bad", valid: false})
	if got := observedUserID(invalidAuth); got != "" {
		t.Fatalf("invalid IdentityFromContext should be empty: %q", got)
	}
	if got := observedUserID(context.Background()); got != "" {
		t.Fatalf("empty ctx user: %q", got)
	}
	ObserveIdentity(ctx, testIdentity{userID: "skip", valid: false})
	if got := observedUserID(ctx); got != "bob" {
		t.Fatalf("invalid identity must not overwrite: %q", got)
	}
}

func TestLogRequestVariants(t *testing.T) {
	t.Parallel()
	LogRequest(nil, nil, Record{}) // nil logger
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	LogRequest(nil, logger, Record{
		Protocol: "http",
		Method:   "GET",
		Path:     "/p",
		Code:     "200",
		User:     "u",
		Slow:     true,
		Stream:   "end",
		Level:    slog.LevelWarn,
	})
	out := buf.String()
	if !strings.Contains(out, `"slow":true`) || !strings.Contains(out, `"path":"/p"`) {
		t.Fatalf("attrs missing: %s", out)
	}
	if OptionsFromLogConfig(true, 500) != (Options{Access: true, SlowMs: 500}) {
		t.Fatal("OptionsFromLogConfig mismatch")
	}
}

func TestCodeFromError(t *testing.T) {
	t.Parallel()
	if CodeFromError(nil) != codes.OK {
		t.Fatal("nil")
	}
	if CodeFromError(status.Error(codes.NotFound, "x")) != codes.NotFound {
		t.Fatal("status")
	}
	if CodeFromError(errors.New("plain")) != codes.Unknown {
		t.Fatal("unknown")
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
