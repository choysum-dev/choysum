// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/po"
)

func TestPORequiresAuth(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPORequiresApplication(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "application is required") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPOExportAttachment(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			if offset > 0 {
				return &searchTermsResult{Lang: lang, Items: nil, Total: 1}, nil
			}
			return &searchTermsResult{
				Lang: lang,
				Total: 1,
				Items: []termItem{{
					Application: "auth",
					Module:      "auth",
					Scope:       "web/pages/Login@title",
					Src:         "Sign in",
					Value:       "登录",
					Kind:        "literal",
					Source:      "po",
					Status:      "translated",
				}},
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	cd := rr.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "auth-zh_CN.po") {
		t.Fatalf("Content-Disposition=%q", cd)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `msgctxt "web/pages/Login@title"`) {
		t.Fatalf("missing msgctxt: %s", body)
	}
	if !strings.Contains(body, `msgid "Sign in"`) || !strings.Contains(body, `msgstr "登录"`) {
		t.Fatalf("missing msgid/msgstr: %s", body)
	}
	entries, err := po.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("entries=%d", len(entries))
	}
}

func TestPOExportSetsTruncatedHeader(t *testing.T) {
	oldMax := poExportMaxItems
	poExportMaxItems = 2
	t.Cleanup(func() { poExportMaxItems = oldMax })

	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			items := []termItem{
				{Application: "auth", Module: "auth", Scope: "a@1", Src: "One", Value: "一", Kind: "literal", Status: "translated"},
				{Application: "auth", Module: "auth", Scope: "a@2", Src: "Two", Value: "二", Kind: "literal", Status: "translated"},
				{Application: "auth", Module: "auth", Scope: "a@3", Src: "Three", Value: "三", Kind: "literal", Status: "translated"},
			}
			if offset >= len(items) {
				return &searchTermsResult{Lang: lang, Total: int64(len(items))}, nil
			}
			end := offset + limit
			if end > len(items) {
				end = len(items)
			}
			return &searchTermsResult{
				Lang:  lang,
				Total: int64(len(items)),
				Items: items[offset:end],
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-Choysum-PO-Truncated"); got != "1" {
		t.Fatalf("X-Choysum-PO-Truncated=%q, want 1", got)
	}
}

func TestBuildPOEntriesMarksFuzzy(t *testing.T) {
	entries := buildPOEntries("zh_CN", []termItem{{
		Scope:  "a@b",
		Src:    "Hello",
		Value:  "你好",
		Status: "fuzzy",
		Module: "auth",
	}})
	var found bool
	for _, e := range entries {
		if e.Msgid == "Hello" {
			found = true
			if len(e.Flags) == 0 || e.Flags[0] != "fuzzy" {
				t.Fatalf("flags=%v", e.Flags)
			}
		}
	}
	if !found {
		t.Fatal("entry missing")
	}
}
