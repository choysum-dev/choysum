// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSharedSessionSequentialPagesDoNotLeakCookies(t *testing.T) {
	if !chromeSharedEnabled() {
		t.Skip("shared chrome disabled")
	}
	session := startTestSession(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set":
			http.SetCookie(w, &http.Cookie{Name: "p2", Value: "leak"})
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>set</body></html>`))
		case "/check":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<script>
document.title = document.cookie.includes('p2=leak') ? 'has' : 'clean';
</script></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	page1, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	if err := page1.Goto(srv.URL+"/set", "load"); err != nil {
		t.Fatal(err)
	}
	// Control: cookie must be visible before clearing, otherwise this test
	// cannot detect a leak across NewPage on the shared session.
	if err := page1.Goto(srv.URL+"/check", "load"); err != nil {
		t.Fatal(err)
	}
	if got, err := page1.Evaluate(`document.title`); err != nil || got != `"has"` {
		t.Fatalf("expected cookie before clearing, got %s (%v)", got, err)
	}
	if err := page1.ClearOriginStorage(srv.URL); err != nil {
		t.Fatalf("ClearOriginStorage: %v", err)
	}
	page1.Close()

	page2, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page2.Close()
	if err := page2.Goto(srv.URL+"/check", "load"); err != nil {
		t.Fatal(err)
	}
	title, err := page2.Evaluate(`document.title`)
	if err != nil {
		t.Fatal(err)
	}
	if title != `"clean"` {
		t.Fatalf("cookie leaked across NewPage on shared session: title=%s", title)
	}
}

func TestSharedSessionCloseIsNoop(t *testing.T) {
	if !chromeSharedEnabled() {
		t.Skip("shared chrome disabled")
	}
	session := startTestSession(t)
	session.Close()
	page, err := session.NewPage()
	if err != nil {
		t.Fatalf("shared Close must not kill browser: %v", err)
	}
	page.Close()
}
