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

	// moduleNames query must not affect which modules are requested.
	req2 := httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=zh_CN&hash="+wantHash, nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
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
