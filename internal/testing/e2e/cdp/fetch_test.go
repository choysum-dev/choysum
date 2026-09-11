// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchNilPageGuards(t *testing.T) {
	t.Parallel()
	if err := (*Page)(nil).EnableFetch(); err == nil || !strings.Contains(err.Error(), "nil page") {
		t.Fatalf("EnableFetch: %v", err)
	}
	if err := (*Page)(nil).DisableFetch(); err == nil || !strings.Contains(err.Error(), "nil page") {
		t.Fatalf("DisableFetch: %v", err)
	}
	if _, err := (*Page)(nil).WaitPaused(time.Millisecond); err == nil || !strings.Contains(err.Error(), "nil page") {
		t.Fatalf("WaitPaused: %v", err)
	}
	if err := (*Page)(nil).Fulfill("x", FulfillOptions{}); err == nil || !strings.Contains(err.Error(), "nil page") {
		t.Fatalf("Fulfill: %v", err)
	}
	if err := (*Page)(nil).Continue("x"); err == nil || !strings.Contains(err.Error(), "nil page") {
		t.Fatalf("Continue: %v", err)
	}
}

func TestFetchWaitPausedRequiresEnable(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	if _, err := page.WaitPaused(10 * time.Millisecond); err == nil || !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("expected not enabled, got %v", err)
	}
	if err := page.Fulfill("missing", FulfillOptions{}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown id, got %v", err)
	}
	if err := page.Continue(""); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty id, got %v", err)
	}
}

func TestFetchFulfillAndContinue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = io.WriteString(w, "live-ok")
		case "/mock-me":
			_, _ = io.WriteString(w, "should-not-see")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	if err := page.EnableFetch(); err != nil {
		t.Fatalf("EnableFetch: %v", err)
	}
	defer func() { _ = page.DisableFetch() }()

	type navResult struct {
		body string
		err  error
	}

	runInterceptedNav := func(url string, onPaused func(*PausedRequest) error) (string, error) {
		navCh := make(chan navResult, 1)
		go func() {
			err := page.Goto(url, "domcontentloaded")
			if err != nil {
				navCh <- navResult{err: err}
				return
			}
			raw, err := page.Evaluate(`document.body ? document.body.innerText : ''`)
			navCh <- navResult{body: raw, err: err}
		}()

		deadline := time.Now().Add(20 * time.Second)
		for {
			select {
			case res := <-navCh:
				return res.body, res.err
			default:
			}
			if time.Now().After(deadline) {
				return "", errTimeout("navigate")
			}
			paused, waitErr := page.WaitPaused(500 * time.Millisecond)
			if waitErr != nil {
				if strings.Contains(waitErr.Error(), "timeout") {
					continue
				}
				return "", waitErr
			}
			if err := onPaused(paused); err != nil {
				return "", err
			}
		}
	}

	var sawMock atomic.Bool
	body, err := runInterceptedNav(srv.URL+"/mock-me", func(paused *PausedRequest) error {
		if strings.Contains(paused.URL, "/mock-me") {
			sawMock.Store(true)
			return page.Fulfill(paused.ID, FulfillOptions{
				Status:      200,
				ContentType: "text/plain",
				Body:        []byte("mocked-outage"),
			})
		}
		return page.Continue(paused.ID)
	})
	if err != nil {
		t.Fatalf("mock navigate: %v", err)
	}
	if !sawMock.Load() {
		t.Fatal("expected /mock-me to be paused")
	}
	if !strings.Contains(body, "mocked-outage") {
		t.Fatalf("body=%q, want mocked-outage", body)
	}

	body, err = runInterceptedNav(srv.URL+"/ok", func(paused *PausedRequest) error {
		return page.Continue(paused.ID)
	})
	if err != nil {
		t.Fatalf("live navigate: %v", err)
	}
	if !strings.Contains(body, "live-ok") {
		t.Fatalf("body=%q, want live-ok", body)
	}

	// FailRequest must use the real CDP path (network error, not an HTTP response).
	var sawAbort atomic.Bool
	_, err = runInterceptedNav(srv.URL+"/mock-me", func(paused *PausedRequest) error {
		if strings.Contains(paused.URL, "/mock-me") {
			sawAbort.Store(true)
			return page.Fail(paused.ID)
		}
		return page.Continue(paused.ID)
	})
	if !sawAbort.Load() {
		t.Fatal("expected /mock-me to be paused for Fail")
	}
	if err == nil {
		t.Fatal("expected navigation error after FailRequest")
	}
}

func errTimeout(op string) error {
	return &timeoutError{op: op}
}

type timeoutError struct{ op string }

func (e *timeoutError) Error() string { return "cdp test: " + e.op + " timeout" }
