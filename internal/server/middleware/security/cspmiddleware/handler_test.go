// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cspmiddleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func newTestScope(cfg *config.Config, output io.Writer) scope.Scope {
	return &stubScope{cfg: cfg, logger: slog.New(slog.NewTextHandler(output, nil))}
}

func TestNewCSPHandlerUsesDefaultsWhenSecurityConfigMissing(t *testing.T) {
	var logBuf bytes.Buffer
	cfg := &config.Config{Server: &config.ServerConfig{Environment: "default"}}
	handler := NewCSPHandler(newTestScope(cfg, &logBuf))

	if handler.cspConfig == nil || handler.hstsConfig == nil {
		t.Fatal("expected default security configs to be initialized")
	}
	if cfg.Server.Security != nil {
		t.Fatal("expected input security config to remain unchanged")
	}
	if !strings.Contains(logBuf.String(), "security config missing; using defaults") {
		t.Fatalf("expected warning about missing security config, got %q", logBuf.String())
	}
}

func TestNewCSPHandlerUsesDefaultSubConfigsWhenMissing(t *testing.T) {
	var logBuf bytes.Buffer
	cfg := &config.Config{Server: &config.ServerConfig{
		Environment: "default",
		Security:    &config.SecurityConfig{},
	}}
	handler := NewCSPHandler(newTestScope(cfg, &logBuf))

	if handler.cspConfig == nil || handler.hstsConfig == nil {
		t.Fatal("expected default CSP and HSTS configs to be initialized")
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "csp config missing; using defaults") || !strings.Contains(logs, "hsts config missing; using defaults") {
		t.Fatalf("expected warnings for missing CSP and HSTS config, got %q", logs)
	}
}

func TestBuildCSPHeaderAndDisabledHandler(t *testing.T) {
	cfg := &config.Config{Server: &config.ServerConfig{Environment: "default", Security: config.NewDefaultSecurityConfig()}}
	handler := NewCSPHandler(newTestScope(cfg, io.Discard))

	header := handler.buildCSPHeader(config.CSPDirectives{
		DefaultSrc:     []string{"'self'"},
		ScriptSrc:      []string{"'self'", "'unsafe-inline'"},
		FrameAncestors: []string{"'none'"},
		BaseURI:        []string{"'self'"},
	}, true)
	for _, want := range []string{"default-src 'self'", "script-src 'self' 'unsafe-inline'", "frame-ancestors 'none'", "base-uri 'self'", "upgrade-insecure-requests", "block-all-mixed-content"} {
		if !strings.Contains(header, want) {
			t.Fatalf("buildCSPHeader missing %q in %q", want, header)
		}
	}

	handler.cspConfig.Enabled = false
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/web", nil)
	handler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(recorder, request)
	if recorder.Header().Get("Content-Security-Policy") != "" {
		t.Fatalf("expected CSP header to be absent when disabled, got %q", recorder.Header().Get("Content-Security-Policy"))
	}
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
}

func TestBuildCSPHeaderIncludesAllDirectivesWithoutHTTPSUpgrade(t *testing.T) {
	handler := NewCSPHandler(newTestScope(&config.Config{Server: &config.ServerConfig{Security: config.NewDefaultSecurityConfig()}}, io.Discard))
	header := handler.buildCSPHeader(config.CSPDirectives{
		DefaultSrc:     []string{"'self'"},
		ScriptSrc:      []string{"'self'", "https://cdn.example.com"},
		StyleSrc:       []string{"'self'", "'unsafe-inline'"},
		ImgSrc:         []string{"'self'", "data:"},
		ConnectSrc:     []string{"'self'", "https://api.example.com"},
		FontSrc:        []string{"'self'"},
		ObjectSrc:      []string{"'none'"},
		MediaSrc:       []string{"https://media.example.com"},
		FrameSrc:       []string{"https://frame.example.com"},
		WorkerSrc:      []string{"blob:"},
		FrameAncestors: []string{"'none'"},
		FormAction:     []string{"'self'"},
		BaseURI:        []string{"'self'"},
		ChildSrc:       []string{"'self'"},
		ManifestSrc:    []string{"'self'"},
	}, false)

	for _, want := range []string{
		"default-src 'self'",
		"script-src 'self' https://cdn.example.com",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"connect-src 'self' https://api.example.com",
		"font-src 'self'",
		"object-src 'none'",
		"media-src https://media.example.com",
		"frame-src https://frame.example.com",
		"worker-src blob:",
		"frame-ancestors 'none'",
		"form-action 'self'",
		"base-uri 'self'",
		"child-src 'self'",
		"manifest-src 'self'",
	} {
		if !strings.Contains(header, want) {
			t.Fatalf("buildCSPHeader missing %q in %q", want, header)
		}
	}
	if strings.Contains(header, "upgrade-insecure-requests") || strings.Contains(header, "block-all-mixed-content") {
		t.Fatalf("did not expect HTTPS-only directives in %q", header)
	}
}

func TestCSPHandlerSetsHeadersAndSkipsExcludedPaths(t *testing.T) {
	cfg := &config.Config{Server: &config.ServerConfig{
		Environment: "production",
		EnabledTLS:  true,
		Security: &config.SecurityConfig{
			CSP: &config.CSPConfig{
				Enabled:       true,
				ReportOnly:    true,
				ReportURI:     "/csp-report",
				ExcludedPaths: []string{"/health"},
				Production: config.CSPDirectives{
					DefaultSrc: []string{"'self'"},
					ScriptSrc:  []string{"'self'"},
					StyleSrc:   []string{"'self'"},
				},
				Development: config.CSPDirectives{DefaultSrc: []string{"'self'", "http:"}},
			},
			HSTS: &config.HSTSConfig{Enabled: true, MaxAge: 3600, IncludeSubdomains: true, Preload: true},
		},
	}}
	handler := NewCSPHandler(newTestScope(cfg, io.Discard))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/web", nil)
	handler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Security-Policy") != "" {
		t.Fatalf("did not expect enforcing CSP header in report-only mode")
	}
	reportOnly := recorder.Header().Get("Content-Security-Policy-Report-Only")
	for _, want := range []string{"default-src 'self'", "script-src 'self'", "style-src 'self'", "report-uri /csp-report", "upgrade-insecure-requests", "block-all-mixed-content"} {
		if !strings.Contains(reportOnly, want) {
			t.Fatalf("report-only header missing %q in %q", want, reportOnly)
		}
	}
	if recorder.Header().Get("Strict-Transport-Security") != "max-age=3600; includeSubDomains; preload" {
		t.Fatalf("unexpected HSTS header: %q", recorder.Header().Get("Strict-Transport-Security"))
	}
	if recorder.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("unexpected referrer policy: %q", recorder.Header().Get("Referrer-Policy"))
	}
	for key, want := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "X-XSS-Protection": "1; mode=block"} {
		if recorder.Header().Get(key) != want {
			t.Fatalf("%s = %q, want %q", key, recorder.Header().Get(key), want)
		}
	}

	excluded := httptest.NewRecorder()
	excludedReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(excluded, excludedReq)
	if excluded.Header().Get("Content-Security-Policy-Report-Only") != "" {
		t.Fatalf("expected excluded path to bypass CSP headers, got %q", excluded.Header().Get("Content-Security-Policy-Report-Only"))
	}
}

func TestCSPHandlerUsesDevelopmentHeadersWithoutTLS(t *testing.T) {
	cfg := &config.Config{Server: &config.ServerConfig{
		Environment: "development",
		EnabledTLS:  false,
		Security: &config.SecurityConfig{
			CSP: &config.CSPConfig{
				Enabled:       true,
				ExcludedPaths: []string{"/health"},
				Development: config.CSPDirectives{
					DefaultSrc: []string{"'self'", "http:"},
					ConnectSrc: []string{"'self'", "ws:"},
				},
				Production: config.CSPDirectives{
					DefaultSrc: []string{"'none'"},
				},
			},
			HSTS: &config.HSTSConfig{Enabled: true, MaxAge: 3600},
		},
	}}
	handler := NewCSPHandler(newTestScope(cfg, io.Discard))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	handler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(recorder, request)

	cspHeader := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(cspHeader, "default-src 'self' http:") || !strings.Contains(cspHeader, "connect-src 'self' ws:") {
		t.Fatalf("unexpected development CSP header: %q", cspHeader)
	}
	if recorder.Header().Get("Content-Security-Policy-Report-Only") != "" {
		t.Fatalf("did not expect report-only header, got %q", recorder.Header().Get("Content-Security-Policy-Report-Only"))
	}
	if got := recorder.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("did not expect HSTS header without TLS, got %q", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "origin-when-cross-origin" {
		t.Fatalf("unexpected referrer policy: %q", got)
	}
}
