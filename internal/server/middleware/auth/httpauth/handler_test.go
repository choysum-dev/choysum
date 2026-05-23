// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package httpauth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type httpauthFakeScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (f *httpauthFakeScope) Run(fn func(scope.Scope) error) error { return fn(f) }
func (f *httpauthFakeScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(f)
}
func (f *httpauthFakeScope) Session() *scope.Session { return nil }
func (f *httpauthFakeScope) WithContext(ctx context.Context) scope.Scope {
	return &httpauthFakeScope{ctx: ctx, cfg: f.cfg, logger: f.logger}
}
func (f *httpauthFakeScope) Context() context.Context { return f.ctx }
func (f *httpauthFakeScope) Logger() *slog.Logger     { return f.logger }
func (f *httpauthFakeScope) Config() *config.Config   { return f.cfg }

func (f *httpauthFakeScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(f.Config())
}

type fakeIdentity struct {
	userID   string
	tokenID  string
	metadata map[string]interface{}
	valid    bool
}

func (f fakeIdentity) GetUserID() string                   { return f.userID }
func (f fakeIdentity) GetTokenID() string                  { return f.tokenID }
func (f fakeIdentity) GetMetadata() map[string]interface{} { return f.metadata }
func (f fakeIdentity) IsValid() bool                       { return f.valid }

type fakeAuthenticator struct {
	validateFn func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error)
}

func (f fakeAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	return f.validateFn(ctx, token, tokenType, checkRevoked)
}
func (f fakeAuthenticator) CreateTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, errors.New("not implemented")
}
func (f fakeAuthenticator) RefreshTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, errors.New("not implemented")
}
func (f fakeAuthenticator) RevokeToken(context.Context, string, string) error {
	return errors.New("not implemented")
}
func (f fakeAuthenticator) RevokeAllUserTokens(context.Context, string, string, string) (int, error) {
	return 0, errors.New("not implemented")
}
func (f fakeAuthenticator) Close() error { return nil }

func newHTTPAuthScope() *httpauthFakeScope {
	cfg := &config.Config{
		Auth: config.NewDefaultAuthConfig(),
		Log:  config.NewDefaultLogConfig(),
	}
	return &httpauthFakeScope{
		ctx:    context.WithValue(context.Background(), struct{}{}, "httpauth"),
		cfg:    cfg,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestExtractorsRespectConfiguredSources(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/resource?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-token"})

	if token, err := (&HeaderTokenExtractor{}).Extract(req); err != nil || token != "header-token" {
		t.Fatalf("header token = %q err=%v, want header-token", token, err)
	}
	if token, err := (&CookieTokenExtractor{}).Extract(req); err != nil || token != "cookie-token" {
		t.Fatalf("cookie token = %q err=%v, want cookie-token", token, err)
	}
	if token, err := (&QueryTokenExtractor{}).Extract(req); err != nil || token != "query-token" {
		t.Fatalf("query token = %q err=%v, want query-token", token, err)
	}
}

func TestExtractorsHandleCustomNamesAndMissingValues(t *testing.T) {
	t.Run("header extractor supports custom header and missing header errors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		extractor := &HeaderTokenExtractor{AuthHeader: "X-Auth-Token"}

		if _, err := extractor.Extract(req); err == nil || !strings.Contains(err.Error(), "X-Auth-Token") {
			t.Fatalf("expected missing custom header error, got %v", err)
		}

		req.Header.Set("X-Auth-Token", "Bearer custom-header-token")
		if token, err := extractor.Extract(req); err != nil || token != "custom-header-token" {
			t.Fatalf("custom header token = %q err=%v, want custom-header-token", token, err)
		}
	})

	t.Run("query extractor supports custom param and missing param errors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		extractor := &QueryTokenExtractor{ParamName: "access_token"}

		if _, err := extractor.Extract(req); err == nil || !strings.Contains(err.Error(), "access_token") {
			t.Fatalf("expected missing custom query error, got %v", err)
		}

		req = httptest.NewRequest(http.MethodGet, "/resource?access_token=custom-query-token", nil)
		if token, err := extractor.Extract(req); err != nil || token != "custom-query-token" {
			t.Fatalf("custom query token = %q err=%v, want custom-query-token", token, err)
		}
	})
}

func TestHandlerBypassesWhenAuthenticatorNilOrPathExcluded(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	NewAuthHandler(runtimeScope, nil).Handler(next).ServeHTTP(rec, req)
	if called != 1 || rec.Code != http.StatusNoContent {
		t.Fatalf("expected nil authenticator to bypass, called=%d code=%d", called, rec.Code)
	}

	called = 0
	rec = httptest.NewRecorder()
	h := NewAuthHandler(runtimeScope, fakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
		return nil, errors.New("should not be called")
	}}, WithExcludePaths("/health"), WithPathRegexExclude(`^/metrics/\d+$`))
	h.Handler(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics/42", nil))
	if called != 1 || rec.Code != http.StatusNoContent {
		t.Fatalf("expected excluded path to bypass, called=%d code=%d", called, rec.Code)
	}
}

func TestHandlerInjectsIdentityAndTokenIntoContext(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	var gotCtx context.Context
	authn := fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		if token != "header-token" || tokenType != auth.AccessToken || !checkRevoked {
			t.Fatalf("unexpected validate args token=%q type=%q revoked=%v", token, tokenType, checkRevoked)
		}
		return fakeIdentity{userID: "u1", tokenID: "tok1", valid: true}, nil
	}}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtx = r.Context()
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	rec := httptest.NewRecorder()

	NewAuthHandler(runtimeScope, authn).Handler(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want 204", rec.Code)
	}
	if id := auth.IdentityFromContext(gotCtx); id == nil || id.GetUserID() != "u1" {
		t.Fatalf("identity in ctx = %#v, want user u1", id)
	}
	if token, ok := auth.AccessTokenFromContext(gotCtx); !ok || token != "header-token" {
		t.Fatalf("access token in ctx = %q ok=%v, want header-token", token, ok)
	}
}

func TestHandlerUsesCustomErrorHandlerForMissingToken(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	var captured error
	h := NewAuthHandler(
		runtimeScope,
		fakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
			return nil, errors.New("unexpected validate")
		}},
		WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			captured = err
			w.WriteHeader(http.StatusTeapot)
		}),
	)

	rec := httptest.NewRecorder()
	h.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/secure", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status code = %d, want 418", rec.Code)
	}
	if !autherrors.IsAuthError(captured, autherrors.ErrMissingToken) {
		t.Fatalf("expected missing token auth error, got %v", captured)
	}
}

func TestAuthHandlerFromConfigBuildsConfiguredExtractorsAndTextResponse(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	runtimeScope.cfg.Auth.HttpAuth = &config.HttpAuthConfig{
		Enabled:         true,
		ExcludedPaths:   []string{"/open"},
		ExcludedRegex:   []string{`^/assets/`},
		TokenExtractors: []string{"cookie", "query"},
		ResponseFormat:  "text",
		CookieName:      "custom_cookie",
		QueryParamName:  "access_token",
	}
	validateCalled := 0
	mw := AuthHandlerFromConfig(runtimeScope, fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		validateCalled++
		if token != "cookie-token" {
			t.Fatalf("token = %q, want cookie-token", token)
		}
		return nil, autherrors.NewAuthError(autherrors.ErrInvalidAccessToken, "invalid token")
	}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure?access_token=query-token", nil)
	req.AddCookie(&http.Cookie{Name: "custom_cookie", Value: "cookie-token"})
	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rec, req)

	if validateCalled != 1 {
		t.Fatalf("validate called = %d, want 1", validateCalled)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, autherrors.ErrInvalidAccessToken.String()) || !strings.Contains(body, "invalid token") {
		t.Fatalf("unexpected body: %q", body)
	}

	rec = httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/open", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("excluded path code = %d, want 204", rec.Code)
	}
}

func TestAuthHandlerFromConfigReturnsPassthroughWhenDisabled(t *testing.T) {
	t.Run("passes through when auth is globally disabled", func(t *testing.T) {
		runtimeScope := newHTTPAuthScope()
		runtimeScope.cfg.Auth.Enabled = false
		nextCalled := 0
		rec := httptest.NewRecorder()
		AuthHandlerFromConfig(runtimeScope, fakeAuthenticator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled++
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/any", nil))
		if nextCalled != 1 || rec.Code != http.StatusNoContent {
			t.Fatalf("expected passthrough when disabled, called=%d code=%d", nextCalled, rec.Code)
		}
	})

	t.Run("passes through when authenticator or http auth config is missing", func(t *testing.T) {
		runtimeScope := newHTTPAuthScope()
		runtimeScope.cfg.Auth.HttpAuth = nil

		for _, tc := range []struct {
			name          string
			authenticator auth.Authenticator
		}{
			{name: "nil authenticator", authenticator: nil},
			{name: "nil http auth config", authenticator: fakeAuthenticator{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				nextCalled := 0
				rec := httptest.NewRecorder()
				AuthHandlerFromConfig(runtimeScope, tc.authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled++
					w.WriteHeader(http.StatusNoContent)
				})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/any", nil))
				if nextCalled != 1 || rec.Code != http.StatusNoContent {
					t.Fatalf("expected passthrough, called=%d code=%d", nextCalled, rec.Code)
				}
			})
		}
	})
}

func TestAuthHandlerFromConfigFallsBackToDefaultExtractorsForUnknownNames(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	runtimeScope.cfg.Auth.HttpAuth = &config.HttpAuthConfig{
		Enabled:         true,
		TokenExtractors: []string{"unknown", "ignored"},
	}
	validated := 0
	mw := AuthHandlerFromConfig(runtimeScope, fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		validated++
		if token != "fallback-header-token" {
			t.Fatalf("token = %q, want fallback-header-token", token)
		}
		return fakeIdentity{userID: "fallback", tokenID: "tok", valid: true}, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer fallback-header-token")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if validated != 1 {
		t.Fatalf("validate called = %d, want 1", validated)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want 204", rec.Code)
	}
}

func TestAuthHandlerFromConfigBuildsHeaderExtractor(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	runtimeScope.cfg.Auth.HttpAuth = &config.HttpAuthConfig{
		Enabled:         true,
		TokenExtractors: []string{"header"},
	}
	validated := 0
	mw := AuthHandlerFromConfig(runtimeScope, fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		validated++
		if token != "header-only-token" {
			t.Fatalf("token = %q, want header-only-token", token)
		}
		return fakeIdentity{userID: "header-user", tokenID: "header-token-id", valid: true}, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer header-only-token")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if validated != 1 {
		t.Fatalf("validate called = %d, want 1", validated)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want 204", rec.Code)
	}
}

func TestAuthHandlerFuncWrapsNewHandler(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	called := 0
	mw := AuthHandlerFunc(runtimeScope, fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		if token != "wrapped-header" {
			t.Fatalf("token = %q, want wrapped-header", token)
		}
		return fakeIdentity{userID: "wrapped-user", tokenID: "tok", valid: true}, nil
	}})
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer wrapped-header")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		id := auth.IdentityFromContext(r.Context())
		if id == nil || id.GetUserID() != "wrapped-user" {
			t.Fatalf("identity = %#v, want wrapped-user", id)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)
	if called != 1 || rec.Code != http.StatusNoContent {
		t.Fatalf("wrapped handler called=%d code=%d", called, rec.Code)
	}
}

func TestDefaultErrorHandlerReturnsJSONErrorResponse(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	h := NewAuthHandler(runtimeScope, fakeAuthenticator{}, WithResponseFormat(middleware.JSONResponseFormat))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	h.defaultErrorHandler(rec, req, autherrors.NewAuthError(autherrors.ErrPermissionDenied, "access denied"))

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	var payload middleware.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json error response: %v", err)
	}
	if payload.Code != autherrors.ErrPermissionDenied.String() || payload.RequestPath != "/secure" || payload.Message != "access denied" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestDefaultExtractorsOrderPrefersHeaderThenCookieThenQuery(t *testing.T) {
	runtimeScope := newHTTPAuthScope()
	seen := make(chan string, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _ := auth.AccessTokenFromContext(r.Context())
		seen <- token
		w.WriteHeader(http.StatusNoContent)
	})
	h := NewAuthHandler(runtimeScope, fakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		return fakeIdentity{userID: "ordered", tokenID: "1", valid: true}, nil
	}})
	req := httptest.NewRequest(http.MethodGet, "/secure?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-token"})
	rec := httptest.NewRecorder()
	h.Handler(next).ServeHTTP(rec, req)
	select {
	case token := <-seen:
		if token != "header-token" {
			t.Fatalf("selected token = %q, want header-token", token)
		}
	case <-time.After(time.Second):
		t.Fatal("handler did not reach next")
	}
}
