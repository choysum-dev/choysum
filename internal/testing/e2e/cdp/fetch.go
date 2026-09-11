// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/chromedp"
)

// PausedRequest is a Fetch-domain request paused at the Request stage.
type PausedRequest struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Method string `json:"method"`
}

// FulfillOptions mocks an HTTP response for a paused request.
type FulfillOptions struct {
	Status      int
	ContentType string
	Body        []byte
	Headers     map[string]string
}

type fetchState struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	listenCtx context.Context
	paused    chan *PausedRequest
	byID      map[string]fetch.RequestID
	decided   map[string]struct{}
	seq       atomic.Uint64
}

func (p *Page) fetchOrInit() *fetchState {
	p.fetchMu.Lock()
	defer p.fetchMu.Unlock()
	if p.fetch == nil {
		p.fetch = &fetchState{
			paused:  make(chan *PausedRequest, 64),
			byID:    map[string]fetch.RequestID{},
			decided: map[string]struct{}{},
		}
	}
	return p.fetch
}

// continueFetchAsync continues a paused request off the ListenTarget callback.
func continueFetchAsync(ctx context.Context, reqID fetch.RequestID) {
	go func() {
		_ = chromedp.Run(ctx, fetch.ContinueRequest(reqID))
	}()
}

// runFetchEnable enables CDP Fetch; tests may override to force errors.
var runFetchEnable = func(ctx context.Context) error {
	return chromedp.Run(ctx, fetch.Enable().WithPatterns([]*fetch.RequestPattern{{
		URLPattern:   "*",
		RequestStage: fetch.RequestStageRequest,
	}}))
}

// runFetchDisable disables CDP Fetch; tests may override to force errors.
var runFetchDisable = func(ctx context.Context) error {
	return chromedp.Run(ctx, fetch.Disable())
}

// runFulfillRequest fulfills a paused request; tests may override to force errors.
var runFulfillRequest = func(ctx context.Context, params *fetch.FulfillRequestParams) error {
	return chromedp.Run(ctx, params)
}

// runContinueRequest continues a paused request; tests may override to force errors.
var runContinueRequest = func(ctx context.Context, reqID fetch.RequestID) error {
	return chromedp.Run(ctx, fetch.ContinueRequest(reqID))
}

// EnableFetch turns on CDP Fetch interception for all URLs at the Request stage.
// Callers must WaitPaused + Fulfill/Continue each paused request or the page stalls.
func (p *Page) EnableFetch() error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	st := p.fetchOrInit()
	st.mu.Lock()
	if st.cancel != nil {
		st.mu.Unlock()
		return nil
	}
	listenerCtx, cancel := context.WithCancel(p.ctx)
	st.cancel = cancel
	st.listenCtx = listenerCtx
	st.paused = make(chan *PausedRequest, 64)
	st.byID = map[string]fetch.RequestID{}
	st.decided = map[string]struct{}{}
	st.mu.Unlock()

	chromedp.ListenTarget(listenerCtx, func(ev any) {
		p.onRequestPaused(st, listenerCtx, ev)
	})

	err := runFetchEnable(p.ctx)
	if err != nil {
		st.mu.Lock()
		if st.cancel != nil {
			st.cancel()
			st.cancel = nil
			st.listenCtx = nil
		}
		st.mu.Unlock()
		return fmt.Errorf("cdp: fetch enable: %w", err)
	}
	return nil
}

func (p *Page) onRequestPaused(st *fetchState, listenerCtx context.Context, ev any) {
	e, ok := ev.(*fetch.EventRequestPaused)
	if !ok || e == nil || e.Request == nil {
		return
	}
	// Response-stage pauses must be continued; we only mock at Request stage.
	if e.ResponseStatusCode != 0 || e.ResponseErrorReason != "" {
		continueFetchAsync(p.ctx, e.RequestID)
		return
	}
	id := fmt.Sprintf("fetch-%d", st.seq.Add(1))
	st.mu.Lock()
	if st.cancel == nil {
		st.mu.Unlock()
		continueFetchAsync(p.ctx, e.RequestID)
		return
	}
	st.byID[id] = e.RequestID
	st.mu.Unlock()
	paused := &PausedRequest{
		ID:     id,
		URL:    e.Request.URL,
		Method: strings.ToUpper(strings.TrimSpace(e.Request.Method)),
	}
	select {
	case st.paused <- paused:
	case <-listenerCtx.Done():
		st.mu.Lock()
		delete(st.byID, id)
		st.mu.Unlock()
		continueFetchAsync(p.ctx, e.RequestID)
	default:
		// Queue full: fail open so the page cannot hang forever.
		st.mu.Lock()
		delete(st.byID, id)
		st.mu.Unlock()
		continueFetchAsync(p.ctx, e.RequestID)
	}
}

// DisableFetch stops Fetch interception and continues any still-paused requests.
func (p *Page) DisableFetch() error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	st := p.fetchOrInit()
	st.mu.Lock()
	cancel := st.cancel
	st.cancel = nil
	st.listenCtx = nil
	ids := make([]fetch.RequestID, 0, len(st.byID))
	for id, reqID := range st.byID {
		if _, done := st.decided[id]; !done {
			ids = append(ids, reqID)
		}
	}
	st.byID = map[string]fetch.RequestID{}
	st.decided = map[string]struct{}{}
	wasEnabled := cancel != nil
	st.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	for _, reqID := range ids {
		_ = chromedp.Run(p.ctx, fetch.ContinueRequest(reqID))
	}
	if !wasEnabled {
		return nil
	}
	if err := runFetchDisable(p.ctx); err != nil {
		// Domain may already be off after target reset.
		if p.ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("cdp: fetch disable: %w", err)
	}
	return nil
}

// WaitPaused waits for the next paused request (or timeout / fetch disable / context cancel).
func (p *Page) WaitPaused(timeout time.Duration) (*PausedRequest, error) {
	if p == nil {
		return nil, fmt.Errorf("cdp: nil page")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	st := p.fetchOrInit()
	st.mu.Lock()
	ch := st.paused
	listenCtx := st.listenCtx
	cancel := st.cancel
	st.mu.Unlock()
	if cancel == nil || listenCtx == nil {
		return nil, fmt.Errorf("cdp: fetch not enabled")
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case <-listenCtx.Done():
		return nil, fmt.Errorf("cdp: fetch disabled")
	case <-timer.C:
		return nil, fmt.Errorf("cdp: wait paused request: timeout")
	case paused, ok := <-ch:
		if !ok || paused == nil {
			return nil, fmt.Errorf("cdp: fetch paused channel closed")
		}
		return paused, nil
	}
}

// Fulfill responds to a paused request with a mocked HTTP response.
func (p *Page) Fulfill(id string, opts FulfillOptions) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("cdp: empty fetch request id")
	}
	st := p.fetchOrInit()
	st.mu.Lock()
	reqID, ok := st.byID[id]
	if !ok {
		st.mu.Unlock()
		return fmt.Errorf("cdp: unknown fetch request id %q", id)
	}
	if _, done := st.decided[id]; done {
		st.mu.Unlock()
		return fmt.Errorf("cdp: fetch request %q already decided", id)
	}
	st.decided[id] = struct{}{}
	delete(st.byID, id)
	st.mu.Unlock()

	status := opts.Status
	if status == 0 {
		status = 200
	}
	headers := []*fetch.HeaderEntry{}
	ct := strings.TrimSpace(opts.ContentType)
	if ct != "" {
		headers = append(headers, &fetch.HeaderEntry{Name: "Content-Type", Value: ct})
	}
	for k, v := range opts.Headers {
		k = strings.TrimSpace(k)
		if k == "" || (strings.EqualFold(k, "Content-Type") && ct != "") {
			continue
		}
		headers = append(headers, &fetch.HeaderEntry{Name: k, Value: v})
	}
	bodyB64 := base64.StdEncoding.EncodeToString(opts.Body)
	params := fetch.FulfillRequest(reqID, int64(status)).WithResponseHeaders(headers)
	if bodyB64 != "" {
		params = params.WithBody(bodyB64)
	}
	if err := runFulfillRequest(p.ctx, params); err != nil {
		return fmt.Errorf("cdp: fulfill request: %w", err)
	}
	return nil
}

// Continue lets a paused request proceed to the network.
func (p *Page) Continue(id string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("cdp: empty fetch request id")
	}
	st := p.fetchOrInit()
	st.mu.Lock()
	reqID, ok := st.byID[id]
	if !ok {
		st.mu.Unlock()
		return fmt.Errorf("cdp: unknown fetch request id %q", id)
	}
	if _, done := st.decided[id]; done {
		st.mu.Unlock()
		return fmt.Errorf("cdp: fetch request %q already decided", id)
	}
	st.decided[id] = struct{}{}
	delete(st.byID, id)
	st.mu.Unlock()
	if err := runContinueRequest(p.ctx, reqID); err != nil {
		return fmt.Errorf("cdp: continue request: %w", err)
	}
	return nil
}
