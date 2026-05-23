// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package middleware

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type commonTestScope struct{}

func (e *commonTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *commonTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *commonTestScope) Session() *scope.Session                 { return nil }
func (e *commonTestScope) WithContext(context.Context) scope.Scope { return e }
func (e *commonTestScope) Context() context.Context                { return context.Background() }
func (e *commonTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *commonTestScope) Config() *config.Config { return &config.Config{} }

func (e *commonTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func TestPathAndMethodHelpers(t *testing.T) {
	regex := CompileRegexPatterns([]string{`^/metrics/\d+$`, `[`})
	if len(regex) != 1 {
		t.Fatalf("compiled regex len = %d, want 1", len(regex))
	}

	t.Run("root slash exclusion is exact-only", func(t *testing.T) {
		if !IsPathExcluded("/", []string{"/"}, nil) {
			t.Fatal("expected root path to be excluded by '/'")
		}
		if IsPathExcluded("/_document/uploads/up_001", []string{"/"}, nil) {
			t.Fatal("did not expect non-root path to be excluded by '/'")
		}
	})

	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{name: "exact", path: "/healthz", want: true},
		{name: "prefix", path: "/assets/app.js", want: true},
		{name: "glob", path: "/api/v1/users", want: true},
		{name: "regex", path: "/metrics/42", want: true},
		{name: "no match", path: "/private", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := IsPathExcluded(tc.path, []string{"/healthz", "/assets/", "/api/*/users"}, regex)
			if got != tc.want {
				t.Fatalf("IsPathExcluded(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name   string
		method string
		want   bool
	}{
		{name: "exact full method", method: "/svc.Auth/Login", want: true},
		{name: "normalized", method: "/svc.Auth/Register", want: true},
		{name: "suffix", method: "/svc.Auth/Refresh", want: true},
		{name: "glob", method: "/grpc.channelz.v1.Channelz/GetTopChannels", want: true},
		{name: "blank ignored", method: "/svc.Auth/Other", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := IsMethodExcluded(tc.method, []string{"/svc.Auth/Login", "svc.Auth/Register", "Refresh", "grpc.channelz.v1.Channelz/*", "   "})
			if got != tc.want {
				t.Fatalf("IsMethodExcluded(%q) = %v, want %v", tc.method, got, tc.want)
			}
		})
	}
}

func TestTokenAndConfigHelpers(t *testing.T) {
	if got := ExtractBearerToken("Bearer token-1"); got != "token-1" {
		t.Fatalf("ExtractBearerToken() = %q, want token-1", got)
	}
	if got := ExtractBearerToken("token-2"); got != "token-2" {
		t.Fatalf("ExtractBearerToken() passthrough = %q, want token-2", got)
	}

	extractors := TokenExtractorsFromConfig([]string{"header", "cookie", "query", "unknown"}, "auth_cookie", "token")
	if len(extractors) != 3 {
		t.Fatalf("extractors len = %d, want 3", len(extractors))
	}
	if got := extractors[0].(map[string]string)["type"]; got != "header" {
		t.Fatalf("header extractor type = %q, want header", got)
	}
	if got := extractors[1].(map[string]interface{})["name"]; got != "auth_cookie" {
		t.Fatalf("cookie extractor name = %#v, want auth_cookie", got)
	}
	if got := extractors[2].(map[string]interface{})["name"]; got != "token" {
		t.Fatalf("query extractor name = %#v, want token", got)
	}

	if got := GetConfigResponseFormat("text"); got != PlainTextResponseFormat {
		t.Fatalf("GetConfigResponseFormat(text) = %v, want plain text", got)
	}
	if got := GetConfigResponseFormat("json"); got != JSONResponseFormat {
		t.Fatalf("GetConfigResponseFormat(json) = %v, want json", got)
	}
}

func TestFormatHTTPErrorAndGRPCMapping(t *testing.T) {
	runtimeScope := &commonTestScope{}
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)

	t.Run("formats auth error as json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		FormatHTTPError(rec, req, autherrors.NewAuthError(autherrors.ErrPermissionDenied, "access denied"), JSONResponseFormat, runtimeScope)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status code = %d, want 401", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type = %q, want application/json", got)
		}
		var payload ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.Code != autherrors.ErrPermissionDenied.String() || payload.RequestPath != "/secure" || payload.Message != "access denied" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
	})

	t.Run("formats auth domain error as plain text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := oerrors.New(autherrors.Domain, autherrors.ErrMissingToken.String(), "missing token")
		FormatHTTPError(rec, req, err, PlainTextResponseFormat, runtimeScope)
		if got := rec.Body.String(); got != autherrors.ErrMissingToken.String()+": missing token" {
			t.Fatalf("plain text body = %q", got)
		}
	})

	t.Run("formats unknown error with default json branch", func(t *testing.T) {
		rec := httptest.NewRecorder()
		FormatHTTPError(rec, req, status.Error(codes.Internal, "boom"), ResponseFormat(99), runtimeScope)
		var payload ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal default payload: %v", err)
		}
		if payload.Code != "AUTH_FAILURE" || payload.Message != "authentication failed" {
			t.Fatalf("unexpected default payload: %#v", payload)
		}
	})

	t.Run("maps errors to grpc unauthenticated status", func(t *testing.T) {
		mapped := AuthErrorToGRPCStatus(autherrors.NewAuthError(autherrors.ErrInvalidAccessToken, "invalid token"))
		if status.Code(mapped) != codes.Unauthenticated || status.Convert(mapped).Message() != autherrors.ErrInvalidAccessToken.String()+": invalid token" {
			t.Fatalf("unexpected mapped auth error: %v", mapped)
		}

		mapped = AuthErrorToGRPCStatus(oerrors.New(autherrors.Domain, autherrors.ErrMissingToken.String(), "missing token"))
		if status.Code(mapped) != codes.Unauthenticated || status.Convert(mapped).Message() != autherrors.ErrMissingToken.String()+": missing token" {
			t.Fatalf("unexpected mapped domain error: %v", mapped)
		}

		mapped = AuthErrorToGRPCStatus(io.EOF)
		if status.Code(mapped) != codes.Unauthenticated || status.Convert(mapped).Message() != "authentication failed" {
			t.Fatalf("unexpected mapped generic error: %v", mapped)
		}
	})
}
