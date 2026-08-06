// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTermsRoutesRemoved(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)

	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodPost} {
		req := httptest.NewRequest(method, "/web/i18n/terms?lang=zh_CN&application=auth", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s /web/i18n/terms status=%d, want 404", method, rr.Code)
		}
	}
}

func TestTranslationsAndPOStillRegistered(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)

	// Method not allowed (route exists) vs 404 (route gone).
	tr := httptest.NewRecorder()
	mux.ServeHTTP(tr, httptest.NewRequest(http.MethodPost, "/web/i18n/translations", nil))
	if tr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("translations POST status=%d, want 405", tr.Code)
	}

	pr := httptest.NewRecorder()
	mux.ServeHTTP(pr, httptest.NewRequest(http.MethodPost, "/web/i18n/po", nil))
	if pr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("po POST status=%d, want 405", pr.Code)
	}
}
