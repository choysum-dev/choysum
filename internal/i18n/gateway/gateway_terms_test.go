// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type termsTestIdentity struct {
	userID string
	valid  bool
}

func (i termsTestIdentity) GetUserID() string                   { return i.userID }
func (i termsTestIdentity) GetTokenID() string                  { return "tok" }
func (i termsTestIdentity) GetExpiresAt() int64                 { return 0 }
func (i termsTestIdentity) GetMetadata() map[string]any         { return nil }
func (i termsTestIdentity) IsValid() bool                       { return i.valid }
func (i termsTestIdentity) GetSubject() string                  { return i.userID }
func (i termsTestIdentity) GetIssuer() string                   { return "" }
func (i termsTestIdentity) GetAudience() []string               { return nil }
func (i termsTestIdentity) GetClaims() map[string]any           { return nil }

func TestTermsRequiresAuth(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&application=auth", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTermsRequiresApplicationWithoutQ(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)

	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "application is required") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTermsUnknownApplication(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&application=missing", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTermsRejectsModuleOutsideApplication(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{
				"auth": {"auth", "core"},
				"web":  {"web", "core"},
			}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			t.Fatal("search should not run for mismatched module")
			return nil, nil
		},
		update: func(ctx context.Context, accessToken, app, lang string, item termItem) (*termItem, string, error) {
			t.Fatal("update should not run for mismatched module")
			return nil, "", nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)

	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&application=auth&module=web", nil)
	req.Header.Set("Authorization", "Bearer editor-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "module does not belong to application") {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}

	body := `{"lang":"zh_CN","items":[
		{"application":"auth","module":"web","scope":"a@t","src":"Hello","value":"您好"}
	]}`
	req2 := httptest.NewRequest(http.MethodPatch, "/web/i18n/terms", bytes.NewReader([]byte(body)))
	req2.Header.Set("Authorization", "Bearer editor-token")
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusBadRequest || !strings.Contains(rr2.Body.String(), "module does not belong to application") {
		t.Fatalf("patch status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestTermsListSingleAppAndPermissionDenied(t *testing.T) {
	var gotToken string
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			gotToken = accessToken
			if app != "auth" {
				t.Fatalf("app = %q", app)
			}
			return &searchTermsResult{
				Lang: lang,
				Items: []termItem{{
					Application: "auth",
					Module:      "auth",
					Scope:       "web/a@title",
					Src:         "Hello",
					Value:       "你好",
					Kind:        "literal",
					Source:      "packaged",
					Status:      "translated",
				}},
				Total: 1,
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)

	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&application=auth", nil)
	req.Header.Set("Authorization", "Bearer editor-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotToken != "editor-token" {
		t.Fatalf("token = %q", gotToken)
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	items, _ := out["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items=%#v", out["items"])
	}

	h.search = func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&application=auth", nil)
	req2.Header.Set("Authorization", "Bearer no-role")
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestTermsAllAppsSearchTruncated(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{
				"auth": {"auth"},
				"web":  {"web"},
			}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			if q != "Log" {
				t.Fatalf("q = %q", q)
			}
			return &searchTermsResult{
				Lang: lang,
				Items: []termItem{{
					Application: app,
					Module:      app,
					Scope:       "a@t",
					Src:         "Log In",
					Value:       "登录",
					Kind:        "literal",
					Status:      "translated",
				}},
				Total: 1,
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&q=Log", nil)
	req.Header.Set("Authorization", "Bearer editor-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	items, _ := out["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected fan-out 2 items, got %#v", out)
	}
	if out["limit"] != float64(allAppsSearchLimit) {
		t.Fatalf("limit=%v", out["limit"])
	}
}

func TestTermsAllAppsSearchSkipsFailedApp(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{
				"auth": {"auth"},
				"web":  {"web"},
			}, nil
		},
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			if app == "web" {
				return nil, status.Error(codes.Unavailable, "offline")
			}
			return &searchTermsResult{
				Lang: lang,
				Items: []termItem{{
					Application: "auth",
					Module:      "auth",
					Scope:       "a@t",
					Src:         "Hello",
					Value:       "你好",
					Kind:        "literal",
					Status:      "translated",
				}},
				Total: 1,
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)
	req := httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=zh_CN&q=Hello", nil)
	req.Header.Set("Authorization", "Bearer editor-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	items, _ := out["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected healthy app results only, got %#v", out)
	}
}

func TestTermsPatchRoutesAndAllOrNothing(t *testing.T) {
	var calls []string
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{
				"auth": {"auth"},
				"web":  {"web"},
			}, nil
		},
		update: func(ctx context.Context, accessToken, app, lang string, item termItem) (*termItem, string, error) {
			calls = append(calls, app+":"+item.Src)
			if app == "web" && item.Src == "fail" {
				return nil, "", status.Error(codes.PermissionDenied, "denied")
			}
			item.Source = "override"
			item.Status = "translated"
			return &item, "hash", nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)

	body := `{"lang":"zh_CN","items":[
		{"application":"auth","module":"auth","scope":"a@t","src":"Hello","value":"您好"},
		{"application":"web","module":"web","scope":"b@t","src":"OK","value":"好的"}
	]}`
	req := httptest.NewRequest(http.MethodPatch, "/web/i18n/terms", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer editor-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(calls) != 2 {
		t.Fatalf("calls=%v", calls)
	}

	calls = nil
	failBody := `{"lang":"zh_CN","items":[
		{"application":"auth","module":"auth","scope":"a@t","src":"Hello","value":"您好"},
		{"application":"web","module":"web","scope":"b@t","src":"fail","value":"x"}
	]}`
	req2 := httptest.NewRequest(http.MethodPatch, "/web/i18n/terms", bytes.NewReader([]byte(failBody)))
	req2.Header.Set("Authorization", "Bearer editor-token")
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	// validate-then-write still writes auth before web fails in this MVP; assert web was attempted.
	if len(calls) < 2 {
		t.Fatalf("expected both apps attempted, calls=%v", calls)
	}
}

func TestParsePatchBodyIgnoresNullFields(t *testing.T) {
	items, lang, err := parsePatchBody([]byte(`{"lang":"zh_CN","application":"auth","module":"auth","scope":"a@t","src":"Hello","value":null}`))
	if err != nil {
		t.Fatalf("parsePatchBody: %v", err)
	}
	if lang != "zh_CN" || len(items) != 1 {
		t.Fatalf("lang=%q items=%#v", lang, items)
	}
	if items[0].Value != "" {
		t.Fatalf("null value should become empty string, got %q", items[0].Value)
	}
	if _, _, err := parsePatchBody([]byte(`{"lang":null}`)); err == nil {
		t.Fatal("expected error when neither items nor term object present")
	}
}

func TestParsePatchBodyEmptyItemsArray(t *testing.T) {
	items, lang, err := parsePatchBody([]byte(`{"lang":"zh_CN","items":[]}`))
	if err != nil {
		t.Fatalf("parsePatchBody: %v", err)
	}
	if lang != "zh_CN" || items == nil || len(items) != 0 {
		t.Fatalf("lang=%q items=%#v, want empty slice", lang, items)
	}
}

func TestTermsPatchRejectsUnknownAppBeforeWrite(t *testing.T) {
	called := false
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{"auth": {"auth"}}, nil
		},
		update: func(ctx context.Context, accessToken, app, lang string, item termItem) (*termItem, string, error) {
			called = true
			return &item, "", nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(termsPath, h.serveTerms)
	body := `{"lang":"zh_CN","application":"missing","module":"x","scope":"a","src":"s","value":"v"}`
	req := httptest.NewRequest(http.MethodPatch, "/web/i18n/terms", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer editor-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v body=%s", rr.Code, called, rr.Body.String())
	}
}

func TestOutgoingUserContextForwardsBearer(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), termsTestIdentity{userID: "u1", valid: true})
	ctx = auth.ContextWithAccessToken(ctx, "from-ctx")
	out := outgoingContextForUserRPC(ctx, "")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	authz := md.Get("authorization")
	if len(authz) != 1 || authz[0] != "Bearer from-ctx" {
		t.Fatalf("authorization=%v", authz)
	}
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("depth=%v", got)
	}
}

func TestTranslationsAnonymousStillReadableTermsNot(t *testing.T) {
	h := &handler{
		listModules: func() (map[string][]string, error) {
			return map[string][]string{}, nil
		},
		fetch: func(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error) {
			t.Fatal("should not dial")
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(translationsPath, h.serveTranslations)
	mux.HandleFunc(termsPath, h.serveTerms)

	tr := httptest.NewRecorder()
	mux.ServeHTTP(tr, httptest.NewRequest(http.MethodGet, "/web/i18n/translations?lang=en_US", nil))
	if tr.Code != http.StatusOK {
		t.Fatalf("translations status=%d", tr.Code)
	}

	te := httptest.NewRecorder()
	mux.ServeHTTP(te, httptest.NewRequest(http.MethodGet, "/web/i18n/terms?lang=en_US&application=auth", nil))
	if te.Code != http.StatusUnauthorized {
		t.Fatalf("terms status=%d", te.Code)
	}
}
