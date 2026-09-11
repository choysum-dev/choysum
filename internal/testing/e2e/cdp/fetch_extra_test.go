// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
)

func TestFetchEnableIdempotentAndDisableSettlesWait(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	if err := page.EnableFetch(); err != nil {
		t.Fatal(err)
	}
	if err := page.EnableFetch(); err != nil {
		t.Fatalf("second EnableFetch: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := page.WaitPaused(5 * time.Second)
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if err := page.DisableFetch(); err != nil {
		t.Fatalf("DisableFetch: %v", err)
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "fetch disabled") {
			t.Fatalf("WaitPaused after disable: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitPaused did not settle after DisableFetch")
	}

	// Idempotent disable when already off.
	if err := page.DisableFetch(); err != nil {
		t.Fatalf("second DisableFetch: %v", err)
	}
}

func TestFetchFulfillHeadersAndEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "live")
	}))
	defer srv.Close()

	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	if err := page.EnableFetch(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = page.DisableFetch() }()

	navCh := make(chan error, 1)
	go func() {
		navCh <- page.Goto(srv.URL+"/hdr", "domcontentloaded")
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-navCh:
			if err != nil {
				t.Fatalf("navigate: %v", err)
			}
			raw, err := page.Evaluate(`document.body ? document.body.innerText : ''`)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(raw, "mocked-hdr") {
				t.Fatalf("body=%q", raw)
			}
			return
		default:
		}
		paused, err := page.WaitPaused(500 * time.Millisecond)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				continue
			}
			t.Fatal(err)
		}
		if strings.Contains(paused.URL, "/hdr") {
			if err := page.Fulfill(paused.ID, FulfillOptions{
				Status:      0, // default 200
				ContentType: "text/plain",
				Body:        []byte("mocked-hdr"),
				Headers:     map[string]string{"X-Test": "1", "Content-Type": "ignored", "": "skip"},
			}); err != nil {
				t.Fatalf("Fulfill: %v", err)
			}
			continue
		}
		if err := page.Continue(paused.ID); err != nil {
			t.Fatalf("Continue: %v", err)
		}
	}
	t.Fatal("timed out")
}

func TestFetchFulfillEmptyIDAndDoubleDecide(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	if err := page.EnableFetch(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = page.DisableFetch() }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	navDone := make(chan struct{})
	go func() {
		_ = page.Goto(srv.URL+"/x", "domcontentloaded")
		close(navDone)
	}()

	var paused *PausedRequest
	for i := 0; i < 40; i++ {
		p, err := page.WaitPaused(500 * time.Millisecond)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				select {
				case <-navDone:
					t.Fatal("nav finished without pause")
				default:
					continue
				}
			}
			t.Fatal(err)
		}
		if strings.Contains(p.URL, "/x") {
			paused = p
			break
		}
		_ = page.Continue(p.ID)
	}
	if paused == nil {
		t.Fatal("no paused document request")
	}
	if err := page.Fulfill("", FulfillOptions{}); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty fulfill: %v", err)
	}
	if err := page.Fulfill(paused.ID, FulfillOptions{Body: []byte("a")}); err != nil {
		t.Fatal(err)
	}
	if err := page.Fulfill(paused.ID, FulfillOptions{Body: []byte("b")}); err == nil || !strings.Contains(err.Error(), "unknown") {
		// id removed from byID after first fulfill → unknown (not already decided)
		if err == nil || (!strings.Contains(err.Error(), "unknown") && !strings.Contains(err.Error(), "already decided")) {
			t.Fatalf("second fulfill: %v", err)
		}
	}
	<-navDone
}

func TestFetchContinueDoubleDecide(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	if err := page.EnableFetch(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = page.DisableFetch() }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	navDone := make(chan struct{})
	go func() {
		_ = page.Goto(srv.URL+"/c", "domcontentloaded")
		close(navDone)
	}()

	var paused *PausedRequest
	for i := 0; i < 40; i++ {
		p, err := page.WaitPaused(500 * time.Millisecond)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				continue
			}
			t.Fatal(err)
		}
		if strings.Contains(p.URL, "/c") {
			paused = p
			break
		}
		_ = page.Continue(p.ID)
	}
	if paused == nil {
		t.Fatal("no paused request")
	}
	if err := page.Continue(paused.ID); err != nil {
		t.Fatal(err)
	}
	if err := page.Continue(paused.ID); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("second continue: %v", err)
	}
	<-navDone
}

func TestContinueFetchAsyncAndQueueFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	continueFetchAsync(ctx, fetch.RequestID("x"))

	st := &fetchState{
		paused:  make(chan *PausedRequest, 1),
		byID:    map[string]fetch.RequestID{},
		decided: map[string]struct{}{},
	}
	st.cancel = func() {}
	st.paused <- &PausedRequest{ID: "fetch-0", URL: "http://x/0", Method: "GET"}

	p := &Page{ctx: context.Background(), fetch: st}
	listenerCtx, stop := context.WithCancel(context.Background())
	defer stop()

	// Queue-full path via onRequestPaused.
	p.onRequestPaused(st, listenerCtx, &fetch.EventRequestPaused{
		RequestID: "r1",
		Request:   &network.Request{URL: "http://x/1", Method: "post"},
	})
	if len(st.byID) != 0 {
		// fetch-0 was only in channel, not byID; new id should have been deleted on full queue.
		for id := range st.byID {
			t.Fatalf("unexpected byID entry %q", id)
		}
	}
}

func TestFetchOnRequestPausedBranches(t *testing.T) {
	st := &fetchState{
		paused:  make(chan *PausedRequest, 4),
		byID:    map[string]fetch.RequestID{},
		decided: map[string]struct{}{},
	}
	listenerCtx, cancel := context.WithCancel(context.Background())
	st.cancel = cancel
	st.listenCtx = listenerCtx
	p := &Page{ctx: context.Background(), fetch: st}

	p.onRequestPaused(st, listenerCtx, nil)
	p.onRequestPaused(st, listenerCtx, &fetch.EventRequestPaused{})
	p.onRequestPaused(st, listenerCtx, &fetch.EventRequestPaused{
		RequestID:          "resp",
		Request:            &network.Request{URL: "http://x/", Method: "GET"},
		ResponseStatusCode: 200,
	})
	p.onRequestPaused(st, listenerCtx, &fetch.EventRequestPaused{
		RequestID:           "err",
		Request:             &network.Request{URL: "http://x/e", Method: "GET"},
		ResponseErrorReason: network.ErrorReasonFailed,
	})

	// Happy path enqueue.
	p.onRequestPaused(st, listenerCtx, &fetch.EventRequestPaused{
		RequestID: "ok",
		Request:   &network.Request{URL: "http://x/ok", Method: "get"},
	})
	select {
	case paused := <-st.paused:
		if paused.Method != "GET" || paused.URL != "http://x/ok" {
			t.Fatalf("%+v", paused)
		}
	default:
		t.Fatal("expected paused enqueue")
	}

	// Inactive after disable.
	st.mu.Lock()
	st.cancel = nil
	st.mu.Unlock()
	p.onRequestPaused(st, listenerCtx, &fetch.EventRequestPaused{
		RequestID: "inactive",
		Request:   &network.Request{URL: "http://x/i", Method: "GET"},
	})

	// listenerCtx done while trying to enqueue (buffer empty, but Done selected).
	st2 := &fetchState{
		paused:  make(chan *PausedRequest), // unbuffered
		byID:    map[string]fetch.RequestID{},
		decided: map[string]struct{}{},
	}
	doneCtx, doneCancel := context.WithCancel(context.Background())
	doneCancel()
	st2.cancel = func() {}
	p2 := &Page{ctx: context.Background(), fetch: st2}
	p2.onRequestPaused(st2, doneCtx, &fetch.EventRequestPaused{
		RequestID: "canceled",
		Request:   &network.Request{URL: "http://x/c", Method: "GET"},
	})
	if len(st2.byID) != 0 {
		t.Fatalf("canceled enqueue should drop byID: %v", st2.byID)
	}
}

func TestFetchEnableDisableErrorBranches(t *testing.T) {
	oldEnable := runFetchEnable
	oldDisable := runFetchDisable
	defer func() {
		runFetchEnable = oldEnable
		runFetchDisable = oldDisable
	}()

	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	runFetchEnable = func(ctx context.Context) error {
		return fmt.Errorf("enable boom")
	}
	if err := page.EnableFetch(); err == nil || !strings.Contains(err.Error(), "enable boom") {
		t.Fatalf("EnableFetch: %v", err)
	}
	if page.fetchOrInit().cancel != nil {
		t.Fatal("failed enable should clear cancel")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p2 := &Page{ctx: ctx, fetch: &fetchState{
		cancel:    func() {},
		listenCtx: ctx,
		byID:      map[string]fetch.RequestID{"a": "r"},
		decided:   map[string]struct{}{},
		paused:    make(chan *PausedRequest, 1),
	}}
	runFetchDisable = func(ctx context.Context) error {
		return fmt.Errorf("disable boom")
	}
	if err := p2.DisableFetch(); err != nil {
		t.Fatalf("canceled ctx should swallow disable error: %v", err)
	}

	p3 := &Page{ctx: context.Background(), fetch: &fetchState{
		cancel:    func() {},
		listenCtx: context.Background(),
		byID:      map[string]fetch.RequestID{},
		decided:   map[string]struct{}{},
		paused:    make(chan *PausedRequest, 1),
	}}
	if err := p3.DisableFetch(); err == nil || !strings.Contains(err.Error(), "disable boom") {
		t.Fatalf("DisableFetch error: %v", err)
	}

	st := &fetchState{
		byID:    map[string]fetch.RequestID{"id1": "req"},
		decided: map[string]struct{}{"id1": {}},
		paused:  make(chan *PausedRequest, 1),
	}
	p4 := &Page{ctx: context.Background(), fetch: st}
	if err := p4.Fulfill("id1", FulfillOptions{}); err == nil || !strings.Contains(err.Error(), "already decided") {
		t.Fatalf("Fulfill already decided: %v", err)
	}
	if err := p4.Continue("id1"); err == nil || !strings.Contains(err.Error(), "already decided") {
		t.Fatalf("Continue already decided: %v", err)
	}
	if err := p4.Fulfill(" ", FulfillOptions{}); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("Fulfill whitespace id: %v", err)
	}
}

func TestFetchWaitPausedChannelClosed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *PausedRequest)
	close(ch)
	p := &Page{ctx: ctx, fetch: &fetchState{
		cancel:    cancel,
		listenCtx: ctx,
		paused:    ch,
		byID:      map[string]fetch.RequestID{},
		decided:   map[string]struct{}{},
	}}
	_, err := p.WaitPaused(time.Second)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("WaitPaused closed: %v", err)
	}
}

func TestFetchWaitPausedNilPayload(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *PausedRequest, 1)
	ch <- nil
	p := &Page{ctx: ctx, fetch: &fetchState{
		cancel:    cancel,
		listenCtx: ctx,
		paused:    ch,
		byID:      map[string]fetch.RequestID{},
		decided:   map[string]struct{}{},
	}}
	_, err := p.WaitPaused(time.Second)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("nil paused: %v", err)
	}
	// cancel set but listenCtx nil → not enabled
	p2 := &Page{ctx: ctx, fetch: &fetchState{
		cancel:    cancel,
		listenCtx: nil,
		paused:    make(chan *PausedRequest),
	}}
	if _, err := p2.WaitPaused(time.Millisecond); err == nil || !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("nil listenCtx: %v", err)
	}
}

func TestFetchWaitPausedDefaultTimeoutBranch(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	listenCtx, listenCancel := context.WithCancel(context.Background())
	defer listenCancel()
	p := &Page{ctx: ctx, fetch: &fetchState{
		cancel:    listenCancel,
		listenCtx: listenCtx,
		paused:    make(chan *PausedRequest),
		byID:      map[string]fetch.RequestID{},
		decided:   map[string]struct{}{},
	}}
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	// timeout <= 0 selects the default 30s branch, then page ctx wins.
	_, err := p.WaitPaused(0)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestFetchWaitPausedPageContextDone(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	listenCtx, listenCancel := context.WithCancel(context.Background())
	defer listenCancel()
	p := &Page{ctx: ctx, fetch: &fetchState{
		cancel:    listenCancel,
		listenCtx: listenCtx,
		paused:    make(chan *PausedRequest),
		byID:      map[string]fetch.RequestID{},
		decided:   map[string]struct{}{},
	}}
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := p.WaitPaused(2 * time.Second)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestFetchFulfillContinueRunErrors(t *testing.T) {
	t.Parallel()
	oldF := runFulfillRequest
	oldC := runContinueRequest
	defer func() {
		runFulfillRequest = oldF
		runContinueRequest = oldC
	}()
	runFulfillRequest = func(ctx context.Context, params *fetch.FulfillRequestParams) error {
		return fmt.Errorf("fulfill boom")
	}
	runContinueRequest = func(ctx context.Context, reqID fetch.RequestID) error {
		return fmt.Errorf("continue boom")
	}
	st := &fetchState{
		byID:    map[string]fetch.RequestID{"a": "r1", "b": "r2"},
		decided: map[string]struct{}{},
		paused:  make(chan *PausedRequest, 1),
	}
	p := &Page{ctx: context.Background(), fetch: st}
	if err := p.Fulfill("a", FulfillOptions{Body: nil}); err == nil || !strings.Contains(err.Error(), "fulfill boom") {
		t.Fatalf("Fulfill: %v", err)
	}
	if err := p.Continue("b"); err == nil || !strings.Contains(err.Error(), "continue boom") {
		t.Fatalf("Continue: %v", err)
	}
}
