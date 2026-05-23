// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csrfmiddleware

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubScope struct {
	cfg    *config.Config
	logger *slog.Logger
}

func (e *stubScope) Run(func(scope.Scope) error) error { return nil }
func (e *stubScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *stubScope) Session() *scope.Session                     { return nil }
func (e *stubScope) WithContext(ctx context.Context) scope.Scope { return e }
func (e *stubScope) Context() context.Context                    { return context.Background() }
func (e *stubScope) Logger() *slog.Logger                        { return e.logger }
func (e *stubScope) Config() *config.Config                      { return e.cfg }

func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}
func newTestScope(cfg *config.Config) scope.Scope {
	return &stubScope{cfg: cfg, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func testConfig(sameSite string) *config.Config {
	security := config.NewDefaultSecurityConfig()
	security.CSRF.SameSite = sameSite
	return &config.Config{Server: &config.ServerConfig{Security: security}}
}

func TestNewCSRFHandlerAndHelpers(t *testing.T) {
	checks := []struct {
		sameSite string
		want     http.SameSite
	}{
		{sameSite: "lax", want: http.SameSiteLaxMode},
		{sameSite: "none", want: http.SameSiteNoneMode},
		{sameSite: "strict", want: http.SameSiteStrictMode},
		{sameSite: "unknown", want: http.SameSiteStrictMode},
	}
	for _, check := range checks {
		h := NewCSRFHandler(newTestScope(testConfig(check.sameSite)))
		if h.sameSite != check.want {
			t.Fatalf("sameSite %q => %v, want %v", check.sameSite, h.sameSite, check.want)
		}
	}

	h := NewCSRFHandler(newTestScope(testConfig("strict")))
	tokenA, err := h.generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	tokenB, err := h.generateToken()
	if err != nil {
		t.Fatalf("generateToken second: %v", err)
	}
	if len(tokenA) != 64 || len(tokenB) != 64 || tokenA == tokenB {
		t.Fatalf("unexpected generated tokens: %q %q", tokenA, tokenB)
	}

	for method, want := range map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true, "POST": false, "DELETE": false} {
		if got := isSafeMethod(method); got != want {
			t.Fatalf("isSafeMethod(%q) = %v, want %v", method, got, want)
		}
	}
}

func TestCSRFHandlerSetsCookieAndBypassesSafeOrExcludedRequests(t *testing.T) {
	h := NewCSRFHandler(newTestScope(testConfig("lax")))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/web", nil)
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != h.cookieName || cookies[0].Value == "" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}

	excluded := httptest.NewRecorder()
	excludedReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(excluded, excludedReq)
	if excluded.Code != http.StatusNoContent {
		t.Fatalf("excluded status = %d, want %d", excluded.Code, http.StatusNoContent)
	}
}

func TestCSRFHandlerValidatesHeaderAndFormTokens(t *testing.T) {
	h := NewCSRFHandler(newTestScope(testConfig("strict")))
	token := strings.Repeat("a", 64)

	headerReq := httptest.NewRequest(http.MethodPost, "/submit", nil)
	headerReq.AddCookie(&http.Cookie{Name: h.cookieName, Value: token})
	headerReq.Header.Set(h.headerName, token)
	headerRec := httptest.NewRecorder()
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})).ServeHTTP(headerRec, headerReq)
	if headerRec.Code != http.StatusCreated {
		t.Fatalf("header token status = %d, want %d", headerRec.Code, http.StatusCreated)
	}

	form := url.Values{}
	form.Set("csrf_token", token)
	formReq := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	formReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	formReq.AddCookie(&http.Cookie{Name: h.cookieName, Value: token})
	formRec := httptest.NewRecorder()
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(formRec, formReq)
	if formRec.Code != http.StatusOK {
		t.Fatalf("form token status = %d, want %d", formRec.Code, http.StatusOK)
	}
}

func TestCSRFHandlerErrorResponses(t *testing.T) {
	h := NewCSRFHandler(newTestScope(testConfig("strict")))

	missingCookieReq := httptest.NewRequest(http.MethodPost, "/submit", nil)
	missingCookieReq.Header.Set("Accept", "application/json")
	missingCookieRec := httptest.NewRecorder()
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when cookie is missing")
	})).ServeHTTP(missingCookieRec, missingCookieReq)
	if missingCookieRec.Code != http.StatusForbidden || missingCookieRec.Header().Get("X-CSRF-Error") != "true" {
		t.Fatalf("unexpected missing cookie response: status=%d headers=%v", missingCookieRec.Code, missingCookieRec.Header())
	}
	if !strings.Contains(missingCookieRec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected JSON content type, got %q", missingCookieRec.Header().Get("Content-Type"))
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(missingCookieRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if payload["error"] != "csrf_error" {
		t.Fatalf("unexpected JSON payload: %#v", payload)
	}

	missingHeaderReq := httptest.NewRequest(http.MethodPost, "/submit", nil)
	missingHeaderReq.AddCookie(&http.Cookie{Name: h.cookieName, Value: "token"})
	missingHeaderRec := httptest.NewRecorder()
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when token is missing")
	})).ServeHTTP(missingHeaderRec, missingHeaderReq)
	if !strings.Contains(missingHeaderRec.Body.String(), "missing CSRF token") {
		t.Fatalf("expected HTML error body to mention missing token, got %q", missingHeaderRec.Body.String())
	}

	invalidReq := httptest.NewRequest(http.MethodDelete, "/submit", nil)
	invalidReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	invalidReq.AddCookie(&http.Cookie{Name: h.cookieName, Value: "cookie-token"})
	invalidReq.Header.Set(h.headerName, "other-token")
	invalidRec := httptest.NewRecorder()
	h.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when token is invalid")
	})).ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusForbidden {
		t.Fatalf("invalid token status = %d, want %d", invalidRec.Code, http.StatusForbidden)
	}
	if !strings.Contains(invalidRec.Body.String(), "csrf_error") {
		t.Fatalf("expected JSON error body, got %q", invalidRec.Body.String())
	}
}
