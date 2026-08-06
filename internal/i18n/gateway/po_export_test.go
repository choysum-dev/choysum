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

func TestPORequiresModule(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "module is required") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPORejectsModuleOutsideApplication(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth", "core"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			t.Fatal("search should not run for mismatched module")
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth&module=web", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "module does not belong to application") {
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
				Lang:  lang,
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
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth&module=auth", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth&module=auth", nil)
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

func TestPORequiresLangAndValidFormat(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)

	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?application=auth&module=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "lang is required") {
		t.Fatalf("missing lang: status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh/CN&application=auth&module=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "invalid lang format") {
		t.Fatalf("invalid lang: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPOModulesByAppAndUnknownApplication(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return nil, context.Canceled
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth&module=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("modulesByApp error: status=%d body=%s", rr.Code, rr.Body.String())
	}

	h.listModules = func() (map[string][]string, error) {
		return map[string][]string{"auth": {"auth"}}, nil
	}
	req = httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=web&module=web", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "unknown application") {
		t.Fatalf("unknown app: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPOCollectAllTermsErrorsAndEmpty(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			return nil, context.Canceled
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(poPath, h.servePO)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth&module=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("expected search error status, got %d", rr.Code)
	}

	h.search = func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
		return nil, nil
	}
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("nil search result: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCollectAllTermsMaxItemsZeroTruncates(t *testing.T) {
	oldMax := poExportMaxItems
	poExportMaxItems = 0
	t.Cleanup(func() { poExportMaxItems = oldMax })

	h := &handler{
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			t.Fatal("search must not run when max items is 0")
			return nil, nil
		},
	}
	items, truncated, err := h.collectAllTerms(context.Background(), "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || len(items) != 0 || !truncated {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

type errWriter struct {
	header http.Header
	code   int
}

func (w *errWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}
func (w *errWriter) Write([]byte) (int, error) { return 0, context.Canceled }
func (w *errWriter) WriteHeader(statusCode int)  { w.code = statusCode }

func TestPOWriteErrorIsLogged(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			return &searchTermsResult{
				Lang:  lang,
				Total: 1,
				Items: []termItem{{Scope: "a@b", Src: "Hi", Value: "你好", Kind: "literal"}},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/po?lang=zh_CN&application=auth&module=auth", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := &errWriter{}
	h.servePO(w, req)
	if w.code != http.StatusOK {
		t.Fatalf("WriteHeader code=%d", w.code)
	}
}
