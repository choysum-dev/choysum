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
	if got.Source != "" || got.Status != "missing" {
		t.Fatalf("nil source / empty-value status = %#v", got)
	}
}

func TestParseSearchResultAndToInt64(t *testing.T) {
	result := parseSearchResult("auth", "zh_CN", 10, 2, 42, map[string]any{
		"result": []any{
			map[string]any{"Module": "auth", "Scope": "a", "Src": "Hi", "Value": "你好"},
			"skip-me",
			map[string]any{"module": "auth", "scope": "b", "src": "Bye", "kind": "custom"},
		},
	})
	if result.Lang != "zh_CN" || result.Limit != 10 || result.Offset != 2 {
		t.Fatalf("pagination = %#v", result)
	}
	if result.Total != 42 {
		t.Fatalf("total = %d, want Count result 42", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("items = %#v", result.Items)
	}
	if result.Items[0].Kind != "literal" || result.Items[1].Kind != "custom" {
		t.Fatalf("kinds = %#v", result.Items)
	}

	cond := buildTermSearchCondition("zh_CN", []string{"auth", "web"}, "Hi")
	and, ok := cond["And"].([]any)
	if !ok || len(and) < 3 {
		t.Fatalf("condition = %#v", cond)
	}

	if toInt64(nil) != 0 || toInt64("") != 0 || toInt64("<nil>") != 0 {
		t.Fatal("expected zero for empty inputs")
	}
	if toInt64(7) != 7 || toInt64(int64(8)) != 8 || toInt64("9") != 9 || toInt64(float64(10)) != 10 {
		t.Fatal("expected numeric coercion")
	}
}

func TestFetchAppSearchTermsRequireApplication(t *testing.T) {
	if _, err := fetchAppSearchTerms(context.Background(), nil, "tok", "  ", "zh_CN", nil, "", 10, 0); err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("fetchAppSearchTerms empty app: %v", err)
	}
}

func TestWriteTermsRPCError(t *testing.T) {
	cases := []struct {
		err     error
		code    int
		wantSub string
	}{
		{status.Error(codes.PermissionDenied, "no"), http.StatusForbidden, `"error":"permission denied"`},
		{status.Error(codes.Unauthenticated, "auth"), http.StatusUnauthorized, `"error":"authentication is required"`},
		{status.Error(codes.InvalidArgument, "bad"), http.StatusBadRequest, "InvalidArgument"},
		{status.Error(codes.Unavailable, "down"), http.StatusBadGateway, "Unavailable"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeTermsRPCError(rr, tc.err)
		if rr.Code != tc.code {
			t.Fatalf("code=%v status=%d body=%s", status.Code(tc.err), rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), tc.wantSub) {
			t.Fatalf("body=%s, want substring %q", rr.Body.String(), tc.wantSub)
		}
	}
}

func TestSearchAppUsesFetchWhenHookNil(t *testing.T) {
	h := &handler{}
	_, err := h.searchApp(context.Background(), "tok", "  ", "zh_CN", nil, "", 10, 0)
	if err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("expected fetchAppSearchTerms empty-app error, got %v", err)
	}
}
