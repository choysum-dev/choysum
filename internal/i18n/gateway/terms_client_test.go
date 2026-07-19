// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseTermItemDefaults(t *testing.T) {
	if got := parseTermItem("auth", nil); got.Application != "auth" {
		t.Fatalf("nil map Application = %q", got.Application)
	}

	got := parseTermItem("auth", map[string]any{
		"application": "<nil>",
		"module":      "auth",
		"scope":       "a@b",
		"src":         "<nil>",
		"value":       "<nil>",
		"kind":        "<nil>",
		"source":      "<nil>",
		"status":      "<nil>",
	})
	if got.Application != "auth" {
		t.Fatalf("Application = %q", got.Application)
	}
	if got.Src != "" || got.Value != "" || got.Kind != "literal" {
		t.Fatalf("defaults = %#v", got)
	}
	if got.Source != "" || got.Status != "" {
		t.Fatalf("nil source/status = %#v", got)
	}
}

func TestParseSearchTermsResultAndToInt64(t *testing.T) {
	result := parseSearchTermsResult("auth", map[string]any{
		"lang":   "zh_CN",
		"total":  float64(3),
		"limit":  int32(10),
		"offset": "2",
		"items": []any{
			map[string]any{"module": "auth", "scope": "a", "src": "Hi", "value": "你好"},
			"skip-me",
			map[string]any{"module": "auth", "scope": "b", "src": "Bye", "kind": "custom"},
		},
	})
	if result.Lang != "zh_CN" || result.Total != 3 || result.Limit != 10 || result.Offset != 2 {
		t.Fatalf("pagination = %#v", result)
	}
	if len(result.Items) != 2 {
		t.Fatalf("items = %#v", result.Items)
	}
	if result.Items[0].Kind != "literal" || result.Items[1].Kind != "custom" {
		t.Fatalf("kinds = %#v", result.Items)
	}

	if toInt64(nil) != 0 || toInt64("") != 0 || toInt64("<nil>") != 0 {
		t.Fatal("expected zero for empty inputs")
	}
	if toInt64(7) != 7 || toInt64(int64(8)) != 8 || toInt64("9") != 9 {
		t.Fatal("expected numeric coercion")
	}
}

func TestFetchAndUpdateRequireApplication(t *testing.T) {
	if _, err := fetchAppSearchTerms(context.Background(), nil, "tok", "  ", "zh_CN", nil, "", 10, 0); err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("fetchAppSearchTerms empty app: %v", err)
	}
	if _, _, err := invokeAppUpdateTerm(context.Background(), nil, "tok", "", "zh_CN", termItem{}); err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("invokeAppUpdateTerm empty app: %v", err)
	}
}

func TestWriteTermsRPCErrorAndParseQueryInt(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{status.Error(codes.PermissionDenied, "no"), http.StatusForbidden},
		{status.Error(codes.Unauthenticated, "auth"), http.StatusUnauthorized},
		{status.Error(codes.InvalidArgument, "bad"), http.StatusBadRequest},
		{status.Error(codes.Unavailable, "down"), http.StatusBadGateway},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeTermsRPCError(rr, tc.err)
		if rr.Code != tc.code {
			t.Fatalf("code=%v status=%d body=%s", status.Code(tc.err), rr.Code, rr.Body.String())
		}
	}

	if parseQueryInt("", 7) != 7 || parseQueryInt("x", 3) != 3 || parseQueryInt("12", 0) != 12 {
		t.Fatal("parseQueryInt mismatch")
	}
}

func TestTermsInfersApplicationFromModule(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth", "core"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			if app != "auth" || len(modules) != 1 || modules[0] != "auth" {
				t.Fatalf("unexpected search app=%q modules=%v", app, modules)
			}
			return &searchTermsResult{Lang: lang, Items: nil, Total: 0, Limit: limit, Offset: offset}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&module=auth", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTermsMethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)
	req := httptest.NewRequest(http.MethodPost, "/web/i18n/terms", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rr.Code)
	}
}
