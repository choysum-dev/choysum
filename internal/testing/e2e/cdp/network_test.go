// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
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

func TestCoalesceResponseMeta(t *testing.T) {
	url, ct, method := coalesceResponseMeta("", "", pendingReq{url: "http://x/a", method: "POST", ct: "application/json"}, true, "GET")
	if url != "http://x/a" || ct != "application/json" || method != "POST" {
		t.Fatalf("pending fallback: %q %q %q", url, ct, method)
	}
	url, ct, method = coalesceResponseMeta("http://x/b", "text/plain", pendingReq{}, false, "PUT")
	if url != "http://x/b" || ct != "text/plain" || method != "PUT" {
		t.Fatalf("no-pending: %q %q %q", url, ct, method)
	}
	url, ct, method = coalesceResponseMeta("http://x/c", "TEXT/HTML", pendingReq{method: "GET"}, true, "POST")
	if url != "http://x/c" || ct != "text/html" || method != "GET" {
		t.Fatalf("resp wins: %q %q %q", url, ct, method)
	}
	url, ct, method = coalesceResponseMeta("http://x/d", "application/json", pendingReq{}, true, "PATCH")
	if method != "PATCH" {
		t.Fatalf("empty pending method → methodWant: %q", method)
	}
}

func TestNetworkEventNilGuards(t *testing.T) {
	if !networkRequestNil(nil) || !networkRequestNil(&network.EventRequestWillBeSent{}) {
		t.Fatal("expected nil request guard")
	}
	if !networkResponseNil(nil) || !networkResponseNil(&network.EventResponseReceived{}) {
		t.Fatal("expected nil response guard")
	}

	var mu sync.Mutex
	pending := map[network.RequestID]pendingReq{}
	matched := map[network.RequestID]pendingResp{}
	// Nil events return without mutating maps (covers ingest early returns).
	ingestRequestWillBeSent(nil, &mu, pending)
	ingestRequestWillBeSent(&network.EventRequestWillBeSent{}, &mu, pending)
	ingestResponseReceived(nil, &mu, pending, matched, "GET", func(string, string, string) bool { return true }, func(network.RequestID) {})
	ingestResponseReceived(&network.EventResponseReceived{}, &mu, pending, matched, "GET", func(string, string, string) bool { return true }, func(network.RequestID) {})
	if len(pending) != 0 || len(matched) != 0 {
		t.Fatalf("nil ingest mutated maps: pending=%v matched=%v", pending, matched)
	}

	id := network.RequestID("req-1")
	ingestRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: id,
		Request: &network.Request{
			Method:  "POST",
			URL:     "http://x/a",
			Headers: network.Headers{"Content-Type": "application/json"},
		},
	}, &mu, pending)
	if pending[id].method != "POST" || pending[id].url != "http://x/a" || pending[id].ct != "application/json" {
		t.Fatalf("pending=%v", pending[id])
	}
}

func TestFinishMatchedResponseBranches(t *testing.T) {
	bodyGone := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "No resource with given identifier")
	}
	id := network.RequestID("r1")
	resp := pendingResp{status: 200, headers: map[string]string{"content-type": "application/json"}, url: "http://x/a"}

	t.Run("retryCancelInSelect", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {url: "http://x/a"}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		n := 0
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			n++
			if n == 1 {
				// Transient error → enter retry select; cancel before After fires.
				go func() {
					time.Sleep(5 * time.Millisecond)
					cancel()
				}()
				return nil, errors.New("No data found for resource with given identifier")
			}
			return []byte(`{}`), nil
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		select {
		case <-resultCh:
			t.Fatal("should not publish after cancel")
		default:
		}
	})

	t.Run("matchedRespGoneAfterFetch", func(t *testing.T) {
		ctx := context.Background()
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			mu.Lock()
			delete(matched, id) // race: cleared while fetch in flight (e.g. bodyGone peer)
			mu.Unlock()
			return []byte(`{"ok":1}`), nil
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		select {
		case <-resultCh:
			t.Fatal("should not publish when matchedResp cleared")
		default:
		}
	})

	t.Run("resultChDefault", func(t *testing.T) {
		ctx := context.Background()
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		resultCh <- &MatchedResponse{Status: 201} // already full
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			return []byte(`{"ok":1}`), nil
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		got := <-resultCh
		if got.Status != 201 {
			t.Fatalf("default branch should leave prior value, got %+v", got)
		}
	})

	t.Run("bodyGonePublishesEmptyBody", func(t *testing.T) {
		ctx := context.Background()
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			return nil, errors.New("No resource with given identifier found")
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		if _, ok := matched[id]; ok {
			t.Fatal("bodyGone should delete matchedResp")
		}
		select {
		case got := <-resultCh:
			if got.Status != 200 || got.Body != nil {
				t.Fatalf("expected empty-body publish, got %+v", got)
			}
		default:
			t.Fatal("bodyGone must publish status/headers with empty body")
		}
	})

	t.Run("retryThenOK", func(t *testing.T) {
		ctx := context.Background()
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		n := 0
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			n++
			if n == 1 {
				return nil, errors.New("No data found for resource with given identifier")
			}
			return []byte(`{"ok":1}`), nil
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		got := <-resultCh
		if string(got.Body) != `{"ok":1}` || n != 2 {
			t.Fatalf("retryThenOK: n=%d got=%+v", n, got)
		}
	})

	t.Run("alreadyCanceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			t.Fatal("fetch should not run when ctx already canceled")
			return nil, nil
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		select {
		case <-resultCh:
			t.Fatal("should not publish")
		default:
		}
	})

	t.Run("successPublish", func(t *testing.T) {
		ctx := context.Background()
		var mu sync.Mutex
		matched := map[network.RequestID]pendingResp{id: resp}
		pending := map[network.RequestID]pendingReq{id: {}}
		finishing := map[network.RequestID]bool{id: true}
		resultCh := make(chan *MatchedResponse, 1)
		fetch := func(context.Context, network.RequestID) ([]byte, error) {
			return []byte(`{"ok":1}`), nil
		}
		finishMatchedResponse(ctx, &mu, matched, pending, finishing, resultCh, id, resp, bodyGone, fetch)
		got := <-resultCh
		if got.Status != 200 || string(got.Body) != `{"ok":1}` {
			t.Fatalf("got %+v", got)
		}
	})
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

func TestWaitForResponseBodyRetryThenOK(t *testing.T) {
	// Large delayed response encourages GetResponseBody "No data found…" retries
	// (time.After branch) before the body is available.
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	payload := strings.Repeat("x", 64*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/big":
			w.Header().Set("Content-Type", "application/json")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(80 * time.Millisecond)
			_, _ = w.Write([]byte(`{"data":"` + payload + `"}`))
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="go">go</button>
<script>
document.getElementById('go').onclick = () => fetch('/api/big');
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
		_, waitErr := page.WaitForResponse(ResponseMatch{URLIncludes: "/api/big"}, 10*time.Second)
		done <- waitErr
	}()
	time.Sleep(50 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("WaitForResponse: %v", err)
	}
}

func TestWaitForResponseDualMatchResultChDefault(t *testing.T) {
	// Two matching responses: first fills resultCh; second tryFinish hits default.
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/a", "/api/b":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="go">go</button>
<script>
document.getElementById('go').onclick = () => {
  fetch('/api/a');
  fetch('/api/b');
};
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
		_, waitErr := page.WaitForResponse(ResponseMatch{URLIncludes: "/api/"}, 5*time.Second)
		done <- waitErr
	}()
	time.Sleep(50 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("WaitForResponse: %v", err)
	}
	time.Sleep(250 * time.Millisecond)
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

	// URLIncludes mismatch: response URL does not contain the filter.
	done3 := make(chan error, 1)
	go func() {
		_, waitErr := page.WaitForResponse(ResponseMatch{
			URLIncludes: "/never-this-path",
		}, 200*time.Millisecond)
		done3 <- waitErr
	}()
	time.Sleep(30 * time.Millisecond)
	if err := page.Click("#go"); err != nil {
		t.Fatal(err)
	}
	if err := <-done3; err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected url-mismatch timeout, got %v", err)
	}
}

func TestFetchResponseBodyGetBodyError(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	if err := EnableNetwork(page); err != nil {
		t.Fatal(err)
	}
	_, err = fetchResponseBody(page.ctx, network.RequestID("missing-request-id"))
	if err == nil {
		t.Fatal("expected GetResponseBody error")
	}
}
