// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEnableNetworkNilPage(t *testing.T) {
	if err := EnableNetwork(nil); err == nil || !strings.Contains(err.Error(), "nil page") {
		t.Fatalf("expected nil page error, got %v", err)
	}
}

func TestHeaderValue(t *testing.T) {
	if got := headerValue(nil, "content-type"); got != "" {
		t.Fatalf("nil headers: %q", got)
	}
	h := map[string]interface{}{
		"Content-Type": "application/json",
		"X-Other":      42,
	}
	if got := headerValue(h, "content-type"); got != "application/json" {
		t.Fatalf("got %q", got)
	}
	if got := headerValue(h, "x-other"); got != "42" {
		t.Fatalf("got %q", got)
	}
	if got := headerValue(h, "missing"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestWaitForResponseNilPage(t *testing.T) {
	var p *Page
	if _, err := p.WaitForResponse(ResponseMatch{}, time.Millisecond); err == nil {
		t.Fatal("expected nil page error")
	}
}

func TestWaitForResponseMatchAndTimeout(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="go">go</button>
<script>
document.getElementById('go').onclick = () => fetch('/api/ping', {method:'POST', headers:{'content-type':'application/json'}, body:'{}'});
</script></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if err := page.Goto(srv.URL+"/", "load"); err != nil {
		t.Fatalf("Goto: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, waitErr := page.WaitForResponse(ResponseMatch{
			URLIncludes:       "/api/ping",
			Method:            "POST",
			ContentTypePrefix: "application/json",
		}, 10*time.Second)
		done <- waitErr
	}()
	time.Sleep(100 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("WaitForResponse: %v", err)
	}

	_, err = page.WaitForResponse(ResponseMatch{URLIncludes: "/never"}, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got %v", err)
	}

	// Default timeout path (timeout <= 0 uses 30s) — cancel via short match that won't fire,
	// but keep duration tiny by passing positive short timeout above; cover <=0 branch separately
	// with a very short race: arm then navigate away quickly is hard — call with 0 and cancel page.
}

func TestWaitForResponseCancelDuringBodyFetch(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/slow":
			w.Header().Set("Content-Type", "application/json")
			// Flush headers/body then hang briefly so listener can arm GetResponseBody.
			_, _ = w.Write([]byte(`{"ok":true}`))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(200 * time.Millisecond)
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="go">go</button>
<script>
document.getElementById('go').onclick = () => fetch('/api/slow');
</script></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if err := page.Goto(srv.URL+"/", "load"); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, waitErr := page.WaitForResponse(ResponseMatch{URLIncludes: "/api/slow"}, 3*time.Second)
		done <- waitErr
	}()
	time.Sleep(50 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatal(err)
	}
	// Cancel listenerCtx via page/session teardown mid body fetch.
	time.Sleep(30 * time.Millisecond)
	page.Close()
	session.Close()
	<-done
}

func TestWaitForResponseResponseWithoutPriorRequest(t *testing.T) {
	// Matching a navigation document response (no RequestWillBeSent pending entry
	// for some events) exercises the pending-miss fallback construction.
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body>hi</body></html>`))
	}))
	defer srv.Close()

	done := make(chan error, 1)
	go func() {
		_, waitErr := page.WaitForResponse(ResponseMatch{
			URLIncludes: srv.URL,
		}, 5*time.Second)
		done <- waitErr
	}()
	time.Sleep(50 * time.Millisecond)
	if err := page.Goto(srv.URL+"/", "load"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("WaitForResponse nav: %v", err)
	}
}

func TestWaitForResponseDefaultTimeoutBranch(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	// Close page context immediately after starting wait so we hit ctx.Done or timeout quickly.
	go func() {
		time.Sleep(50 * time.Millisecond)
		page.Close()
		session.Close()
	}()
	_, err = page.WaitForResponse(ResponseMatch{URLIncludes: "/nope"}, 0)
	if err == nil {
		t.Fatal("expected error from canceled/timeout wait")
	}
}

func TestWaitForResponseFilterMismatches(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/get":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("ok"))
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="go">go</button>
<script>
document.getElementById('go').onclick = () => fetch('/api/get');
</script></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if err := page.Goto(srv.URL+"/", "load"); err != nil {
		t.Fatal(err)
	}

	// Method mismatch: response is GET, wait wants POST → timeout (covers matches false).
	done := make(chan error, 1)
	go func() {
		_, waitErr := page.WaitForResponse(ResponseMatch{
			URLIncludes: "/api/get",
			Method:      "POST",
		}, 200*time.Millisecond)
		done <- waitErr
	}()
	time.Sleep(30 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected method-mismatch timeout, got %v", err)
	}

	// Content-Type prefix mismatch.
	done2 := make(chan error, 1)
	go func() {
		_, waitErr := page.WaitForResponse(ResponseMatch{
			URLIncludes:       "/api/get",
			ContentTypePrefix: "application/json",
		}, 200*time.Millisecond)
		done2 <- waitErr
	}()
	time.Sleep(30 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatal(err)
	}
	if err := <-done2; err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected ct-mismatch timeout, got %v", err)
	}
}
