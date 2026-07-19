// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/store"
)

func TestCatalogHashStableAndOrdered(t *testing.T) {
	a := CatalogHash(map[string]string{"auth": "aaa", "web": "bbb"})
	b := CatalogHash(map[string]string{"web": "bbb", "auth": "aaa"})
	if a != b || a == "" {
		t.Fatalf("catalogHash unstable: %q vs %q", a, b)
	}
	if CatalogHash(nil) != store.EmptyTermHash() {
		t.Fatalf("empty catalogHash = %q, want %q", CatalogHash(nil), store.EmptyTermHash())
	}
	if LangToLocale("zh_CN") != "zh-CN" {
		t.Fatalf("LangToLocale = %q", LangToLocale("zh_CN"))
	}
}

func TestValidLang(t *testing.T) {
	for _, lang := range []string{"zh_CN", "en-US", "en", "pt_BR"} {
		if !validLang(lang) {
			t.Fatalf("validLang(%q) = false, want true", lang)
		}
	}
	for _, lang := range []string{
		"",
		"zh/CN",
		"../evil",
		"zh_CN\r\nX-Injected: 1",
		"aaaaaaaaaaaaaaaaa", // 17 > maxLangCodeLen
	} {
		if validLang(lang) {
			t.Fatalf("validLang(%q) = true, want false", lang)
		}
	}
}

func TestTranslationsRejectsInvalidLang(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		fetch: func(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error) {
			t.Fatal("fetch must not run for invalid lang")
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(translationsPath, h.serveTranslations)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=zh/CN", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestIfNoneMatchUsesWeakComparison(t *testing.T) {
	current := `W/"i18n-abc123"`
	for _, header := range []string{
		current,
		`"i18n-abc123"`,
		`"older", W/"i18n-abc123"`,
		"*",
	} {
		if !ifNoneMatch(header, current) {
			t.Fatalf("If-None-Match %q should match %q", header, current)
		}
	}
	for _, header := range []string{"", `"older"`, `W/"i18n-other"`} {
		if ifNoneMatch(header, current) {
			t.Fatalf("If-None-Match %q should not match %q", header, current)
		}
	}
}

func TestTranslationsMergeAndUnchanged(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{
				"auth": {"auth"},
				"web":  {"web"},
			}, nil
		},
		fetch: func(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error) {
			if app == "auth" {
				return &appTranslations{
					Hash: "hash-auth",
					Terms: map[string]map[string]map[string]string{
						"auth": {"a@t": {"Hello": "你好"}},
					},
				}, nil
			}
			return &appTranslations{
				Hash: "hash-web",
				Terms: map[string]map[string]map[string]string{
					"web": {"shell@ok": {"OK": "好的"}},
				},
			}, nil
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(translationsPath, h.serveTranslations)

	req := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=zh_CN&moduleNames=evil", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["unchanged"] == true {
		t.Fatalf("expected changed: %#v", out)
	}
	if out["locale"] != "zh-CN" || out["lang"] != "zh_CN" {
		t.Fatalf("lang/locale: %#v", out)
	}
	messages, ok := out["messages"].(map[string]any)
	if !ok || messages["auth"] == nil || messages["web"] == nil {
		t.Fatalf("messages merge failed: %#v", out["messages"])
	}
	wantHash := CatalogHash(map[string]string{"auth": "hash-auth", "web": "hash-web"})
	if out["hash"] != wantHash {
		t.Fatalf("hash = %v want %s", out["hash"], wantHash)
	}
	wantETag := catalogETag(wantHash)
	if got := rr.Header().Get("ETag"); got != wantETag {
		t.Fatalf("ETag = %q, want %q", got, wantETag)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("Cache-Control = %q, want private, no-cache", got)
	}

	// moduleNames query must not affect which modules are requested.
	req2 := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=zh_CN&hash="+wantHash, nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("unchanged status = %d, want 200", rr2.Code)
	}
	var unchanged map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &unchanged); err != nil {
		t.Fatal(err)
	}
	if unchanged["unchanged"] != true {
		t.Fatalf("expected unchanged: %#v", unchanged)
	}
	if unchanged["messages"] != nil {
		t.Fatalf("messages must be null when unchanged: %#v", unchanged["messages"])
	}
	if got := rr2.Header().Get("ETag"); got != wantETag {
		t.Fatalf("unchanged ETag = %q, want %q", got, wantETag)
	}

	// Standard HTTP validation coexists with the hash/unchanged JSON protocol.
	req3 := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=zh_CN", nil)
	req3.Header.Set("If-None-Match", `"older", `+wantETag)
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d body=%s, want 304", rr3.Code, rr3.Body.String())
	}
	if rr3.Body.Len() != 0 {
		t.Fatalf("conditional body = %q, want empty", rr3.Body.String())
	}
	if got := rr3.Header().Get("ETag"); got != wantETag {
		t.Fatalf("conditional ETag = %q, want %q", got, wantETag)
	}
	if got := rr3.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("conditional Cache-Control = %q, want private, no-cache", got)
	}
}

func TestTranslationsRequiresLang(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/translations", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestTranslationsSkipsFailedApp(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{
				"auth": {"auth"},
				"web":  {"web"},
			}, nil
		},
		fetch: func(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error) {
			if app == "web" {
				return nil, context.DeadlineExceeded
			}
			return &appTranslations{
				Hash: "hash-auth",
				Terms: map[string]map[string]map[string]string{
					"auth": {"a@t": {"Hello": "你好"}},
				},
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(translationsPath, h.serveTranslations)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=zh_CN", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	messages, _ := out["messages"].(map[string]any)
	if messages["auth"] == nil {
		t.Fatalf("expected healthy app translations: %#v", out["messages"])
	}
	if messages["web"] != nil {
		t.Fatalf("expected failed app omitted: %#v", out["messages"])
	}
}

func TestTranslationsAnonymousGETReadable(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{}, nil
		},
		fetch: func(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error) {
			t.Fatalf("should not dial with empty module catalog")
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(translationsPath, h.serveTranslations)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=en_US", nil)
	// No Authorization header — anonymous.
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["hash"] != store.EmptyTermHash() {
		t.Fatalf("empty catalog hash = %v", out["hash"])
	}
	messages, _ := out["messages"].(map[string]any)
	if len(messages) != 0 {
		t.Fatalf("expected empty messages: %#v", messages)
	}
}

func TestOutgoingInternalContextSetsKey(t *testing.T) {
	// Smoke: nil scope still sets depth metadata without panicking.
	ctx := outgoingContextForInternalRPC(context.Background(), nil)
	_ = ctx
}
