// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"log/slog"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
)

func TestClassifyGRPC(t *testing.T) {
	t.Parallel()
	optsAccess := Options{Access: true, SlowMs: 1000}
	optsQuiet := Options{Access: false, SlowMs: 1000}

	tests := []struct {
		name     string
		opts     Options
		code     codes.Code
		duration time.Duration
		wantLog  bool
		wantLvl  slog.Level
		wantSlow bool
	}{
		{name: "ok quiet", opts: optsQuiet, code: codes.OK, duration: 10 * time.Millisecond},
		{name: "ok access", opts: optsAccess, code: codes.OK, duration: 10 * time.Millisecond, wantLog: true, wantLvl: slog.LevelInfo},
		{name: "ok slow", opts: optsQuiet, code: codes.OK, duration: 1500 * time.Millisecond, wantLog: true, wantLvl: slog.LevelWarn, wantSlow: true},
		{name: "ok slow access prefers warn", opts: optsAccess, code: codes.OK, duration: 1500 * time.Millisecond, wantLog: true, wantLvl: slog.LevelWarn, wantSlow: true},
		{name: "canceled", opts: optsAccess, code: codes.Canceled, duration: time.Second},
		{name: "permission denied", opts: optsQuiet, code: codes.PermissionDenied, duration: time.Millisecond, wantLog: true, wantLvl: slog.LevelWarn},
		{name: "internal", opts: optsQuiet, code: codes.Internal, duration: time.Millisecond, wantLog: true, wantLvl: slog.LevelError},
		{name: "slow disabled", opts: Options{Access: false, SlowMs: 0}, code: codes.OK, duration: 5 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyGRPC(tc.opts, tc.code, tc.duration)
			if got.Log != tc.wantLog || got.Level != tc.wantLvl || got.Slow != tc.wantSlow {
				t.Fatalf("got %#v want log=%v lvl=%v slow=%v", got, tc.wantLog, tc.wantLvl, tc.wantSlow)
			}
		})
	}
}

func TestClassifyHTTP(t *testing.T) {
	t.Parallel()
	opts := Options{Access: true, SlowMs: 1000}
	if got := ClassifyHTTP(opts, 200, 10*time.Millisecond); !got.Log || got.Level != slog.LevelInfo {
		t.Fatalf("200 access = %#v", got)
	}
	if got := ClassifyHTTP(Options{}, 200, 10*time.Millisecond); got.Log {
		t.Fatalf("200 quiet should not log: %#v", got)
	}
	if got := ClassifyHTTP(opts, 404, time.Millisecond); !got.Log || got.Level != slog.LevelWarn {
		t.Fatalf("404 = %#v", got)
	}
	if got := ClassifyHTTP(opts, 500, time.Millisecond); !got.Log || got.Level != slog.LevelError {
		t.Fatalf("500 = %#v", got)
	}
	if got := ClassifyHTTP(Options{SlowMs: 100}, 200, 200*time.Millisecond); !got.Slow || got.Level != slog.LevelWarn {
		t.Fatalf("slow 200 = %#v", got)
	}
}

func TestShouldSkipHTTP(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"/health":              true,
		"/healthz":             true,
		"/readyz":              true,
		"/web/assets/chunk.js": true,
		"/web/sandbox/records": false,
		"/favicon.ico":         true,
		"/web/app.js":          true,
		"/api/v1/thing":        false,
		"/web/assets/x.js?v=1": true,
		"":                     true,
	}
	for path, want := range cases {
		if got := ShouldSkipHTTP(path); got != want {
			t.Fatalf("ShouldSkipHTTP(%q)=%v want %v", path, got, want)
		}
	}
}
