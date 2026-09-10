// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionPageNetworkIntegration(t *testing.T) {
	sess := startTestSession(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<input id="u" placeholder="username" autocomplete="username" />
<input id="p" placeholder="password" autocomplete="current-password" />
<button id="go" type="submit">Go</button>
<script>
document.getElementById('go').onclick = () => {
  fetch('/auth.User/Login', {method:'POST', headers:{'content-type':'application/grpc-web+proto'}, body:'x'})
    // Delay navigation so WaitForResponse can fetch the body before the document is dropped.
    .then(() => { setTimeout(() => { location.href = '/done'; }, 300); });
};
</script></body></html>`))
		case "/auth.User/Login":
			w.Header().Set("Content-Type", "application/grpc-web+proto")
			w.WriteHeader(200)
			_, _ = w.Write([]byte{0x80, 0, 0, 0, 11, 'g', 'r', 'p', 'c', '-', 's', 't', 'a', 't', 'u', 's', ':', '0'})
		case "/done":
			_, _ = w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if sess.ExecPath() == "" {
		t.Fatal("empty ExecPath")
	}
	if sess.Context() == nil {
		t.Fatal("nil context")
	}

	page, err := sess.NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	defer page.Close()

	if err := page.Goto(srv.URL+"/", "domcontentloaded"); err != nil {
		t.Fatalf("Goto: %v", err)
	}
	if err := page.Fill("#u", "e2e-admin"); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if err := page.Fill("#p", "e2e-admin"); err != nil {
		t.Fatalf("Fill pass: %v", err)
	}
	out, err := page.Evaluate(`document.getElementById('u').value`)
	if err != nil || !strings.Contains(out, "e2e-admin") {
		t.Fatalf("Evaluate got %q err=%v", out, err)
	}
	promOut, err := page.Evaluate(`Promise.resolve(1+1)`)
	if err != nil || promOut != "2" {
		t.Fatalf("promise Evaluate got %q err=%v", promOut, err)
	}
	if err := page.WaitForFunction(`document.getElementById('go')`, time.Second); err != nil {
		t.Fatalf("WaitForFunction: %v", err)
	}
	n, err := page.QueryCount("input")
	if err != nil || n < 2 {
		t.Fatalf("QueryCount=%d err=%v", n, err)
	}
	ok, err := page.IsVisible("#go")
	if err != nil || !ok {
		t.Fatalf("IsVisible: %v %v", ok, err)
	}
	ok, err = page.IsEnabled("#go")
	if err != nil || !ok {
		t.Fatalf("IsEnabled: %v %v", ok, err)
	}

	waitDone := make(chan error, 1)
	go func() {
		_, err := page.WaitForResponse(ResponseMatch{
			URLIncludes: "/auth.User/Login",
			Method:      "POST",
		}, 15*time.Second)
		waitDone <- err
	}()
	time.Sleep(100 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if err := <-waitDone; err != nil {
		t.Fatalf("WaitForResponse: %v", err)
	}
	if err := page.WaitForFunction(`location.pathname === '/done'`, 5*time.Second); err != nil {
		t.Fatalf("nav: %v", err)
	}
	u, err := page.URL()
	if err != nil || !strings.Contains(u, "/done") {
		t.Fatalf("URL=%q err=%v", u, err)
	}
	shot := filepath.Join(t.TempDir(), "shot.png")
	if err := page.Screenshot(shot); err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	st, err := os.Stat(shot)
	if err != nil || st.Size() < 8 {
		t.Fatalf("screenshot missing/empty: %v", err)
	}
}
