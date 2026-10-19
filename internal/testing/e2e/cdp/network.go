// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ResponseMatch selects network responses for WaitForResponse.
type ResponseMatch struct {
	URLIncludes       string
	Method            string
	ContentTypePrefix string
}

// MatchedResponse is a completed network response.
type MatchedResponse struct {
	Status  int64
	Headers map[string]string
	Body    []byte
	URL     string
}

type pendingReq struct {
	method string
	url    string
	ct     string
}

type pendingResp struct {
	status  int64
	headers map[string]string
	url     string
	method  string
	ct      string
}

// fetchResponseBody loads a CDP response body; tests may override.
var fetchResponseBody = func(ctx context.Context, reqID network.RequestID) ([]byte, error) {
	res, err := chromedp.Call(ctx, network.GetResponseBody, network.GetResponseBodyParams{RequestID: reqID})
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

// EnableNetwork turns on CDP Network domain for the page.
func EnableNetwork(p *Page) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	_, err := chromedp.Call(p.ctx, network.Enable, network.EnableParams{})
	return err
}

// WaitForResponse waits until a response matching m arrives (or timeout).
// Subscribes via Events on the calling goroutine before select, so callers can
// arm this concurrently with the action that triggers the request.
func (p *Page) WaitForResponse(m ResponseMatch, timeout time.Duration) (*MatchedResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("cdp: nil page")
	}
	return p.WaitForResponseContext(p.ctx, m, timeout)
}

// WaitForResponseContext is WaitForResponse with an extra cancel context
// (page context is still honored).
func (p *Page) WaitForResponseContext(ctx context.Context, m ResponseMatch, timeout time.Duration) (*MatchedResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("cdp: nil page")
	}
	if ctx == nil {
		ctx = p.ctx
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	methodWant := strings.ToUpper(strings.TrimSpace(m.Method))
	urlPart := strings.TrimSpace(m.URLIncludes)
	ctPrefix := strings.ToLower(strings.TrimSpace(m.ContentTypePrefix))

	var mu sync.Mutex
	pending := map[network.RequestID]pendingReq{}
	matchedResp := map[network.RequestID]pendingResp{}
	finishing := map[network.RequestID]bool{}
	resultCh := make(chan *MatchedResponse, 1)

	matches := func(method, url, ct string) bool {
		if urlPart != "" && !strings.Contains(url, urlPart) {
			return false
		}
		if methodWant != "" && method != "" && method != methodWant {
			return false
		}
		if ctPrefix != "" && ct != "" && !strings.HasPrefix(ct, ctPrefix) {
			return false
		}
		return true
	}

	// Permanent: resource already dropped (e.g. navigated away). Transient
	// "No data found…" must keep retrying until the body is ready.
	bodyGone := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "No resource with given identifier")
	}

	listenerCtx, stopListener := context.WithCancel(p.ctx)
	defer stopListener()

	// finishedEarly records LoadingFinished that arrived before ResponseReceived
	// populated matchedResp. Three Events iterators are not causally ordered, so
	// tryFinish must tolerate that race (ListenTarget is gone in chromedp v0.21).
	finishedEarly := map[network.RequestID]struct{}{}

	tryFinish := func(reqID network.RequestID) {
		mu.Lock()
		resp, ok := matchedResp[reqID]
		if !ok {
			finishedEarly[reqID] = struct{}{}
			mu.Unlock()
			return
		}
		if finishing[reqID] {
			mu.Unlock()
			return
		}
		finishing[reqID] = true
		mu.Unlock()

		go finishMatchedResponse(listenerCtx, &mu, matchedResp, pending, finishing, resultCh, reqID, resp, bodyGone, fetchResponseBody)
	}

	// Events subscribes on return, before ranging — construct on this goroutine
	// before select so concurrent triggers cannot race the subscription.
	reqEvents := chromedp.Events(listenerCtx, network.RequestWillBeSent)
	respEvents := chromedp.Events(listenerCtx, network.ResponseReceived)
	loadEvents := chromedp.Events(listenerCtx, network.LoadingFinished)

	go func() {
		for ev, err := range reqEvents {
			if err != nil {
				return
			}
			ingestRequestWillBeSent(ev, &mu, pending)
		}
	}()
	go func() {
		for ev, err := range respEvents {
			if err != nil {
				return
			}
			ingestResponseReceived(ev, &mu, pending, matchedResp, finishedEarly, methodWant, matches, tryFinish)
		}
	}()
	go func() {
		for ev, err := range loadEvents {
			if err != nil {
				return
			}
			tryFinish(ev.RequestID)
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case res := <-resultCh:
		return res, nil
	case <-timer.C:
		return nil, fmt.Errorf("cdp: waitForResponse timeout after %s (urlIncludes=%q method=%q)", timeout, urlPart, methodWant)
	}
}

func ingestRequestWillBeSent(e network.EventRequestWillBeSent, mu *sync.Mutex, pending map[network.RequestID]pendingReq) {
	if networkRequestNil(e) {
		return
	}
	ct := strings.ToLower(headerValue(e.Request.Headers, "content-type"))
	mu.Lock()
	pending[e.RequestID] = pendingReq{
		method: strings.ToUpper(e.Request.Method),
		url:    e.Request.URL,
		ct:     ct,
	}
	mu.Unlock()
}

func ingestResponseReceived(
	e network.EventResponseReceived,
	mu *sync.Mutex,
	pending map[network.RequestID]pendingReq,
	matchedResp map[network.RequestID]pendingResp,
	finishedEarly map[network.RequestID]struct{},
	methodWant string,
	matches func(method, url, ct string) bool,
	tryFinish func(network.RequestID),
) {
	if networkResponseNil(e) {
		return
	}
	mu.Lock()
	req, ok := pending[e.RequestID]
	url, ct, method := coalesceResponseMeta(e.Response.URL, headerValue(e.Response.Headers, "content-type"), req, ok, methodWant)
	req.method = method
	req.url = url
	req.ct = ct
	if !matches(req.method, url, ct) && !matches(req.method, req.url, ct) {
		mu.Unlock()
		return
	}
	headers := map[string]string{}
	if e.Response.Headers != nil {
		for k, v := range e.Response.Headers {
			headers[strings.ToLower(k)] = fmt.Sprint(v)
		}
	}
	matchedResp[e.RequestID] = pendingResp{
		status:  e.Response.Status,
		headers: headers,
		url:     url,
		method:  req.method,
		ct:      ct,
	}
	reqID := e.RequestID
	early := false
	if finishedEarly != nil {
		_, early = finishedEarly[reqID]
		delete(finishedEarly, reqID)
	}
	mu.Unlock()
	if early {
		tryFinish(reqID)
		return
	}
	// Prefer LoadingFinished for body availability; also arm a fallback in case
	// that event is skipped for some response types.
	go func() {
		time.Sleep(50 * time.Millisecond)
		tryFinish(reqID)
	}()
}
// finishMatchedResponse fetches a matched response body and publishes once.
// When the body is already gone (navigated away), it still publishes status/headers
// with a nil body so WaitForResponse does not hang until timeout.
func finishMatchedResponse(
	listenerCtx context.Context,
	mu *sync.Mutex,
	matchedResp map[network.RequestID]pendingResp,
	pending map[network.RequestID]pendingReq,
	finishing map[network.RequestID]bool,
	resultCh chan *MatchedResponse,
	reqID network.RequestID,
	resp pendingResp,
	bodyGone func(error) bool,
	fetch func(context.Context, network.RequestID) ([]byte, error),
) {
	defer func() {
		mu.Lock()
		delete(finishing, reqID)
		mu.Unlock()
	}()

	var body []byte
	for {
		if listenerCtx.Err() != nil {
			return
		}
		var fetchErr error
		body, fetchErr = fetch(listenerCtx, reqID)
		if bodyGone(fetchErr) {
			// Resource already dropped (e.g. navigated away). Still publish status
			// + headers with an empty body so waiters are not stuck until timeout;
			// callers that need the body can fail fast on empty content.
			mu.Lock()
			delete(matchedResp, reqID)
			delete(pending, reqID)
			mu.Unlock()
			select {
			case resultCh <- &MatchedResponse{Status: resp.status, Headers: resp.headers, Body: nil, URL: resp.url}:
			default:
			}
			return
		}
		if fetchErr == nil {
			break
		}
		select {
		case <-listenerCtx.Done():
			return
		case <-time.After(25 * time.Millisecond):
		}
	}

	mu.Lock()
	if _, still := matchedResp[reqID]; !still {
		mu.Unlock()
		return
	}
	delete(matchedResp, reqID)
	delete(pending, reqID)
	mu.Unlock()

	select {
	case resultCh <- &MatchedResponse{Status: resp.status, Headers: resp.headers, Body: body, URL: resp.url}:
	default:
	}
}

// coalesceResponseMeta fills URL/method/content-type when CDP omits fields or
// ResponseReceived arrives without a prior RequestWillBeSent.
func coalesceResponseMeta(respURL, respCT string, pending pendingReq, hadPending bool, methodWant string) (url, ct, method string) {
	if !hadPending {
		pending = pendingReq{
			url:    respURL,
			method: methodWant,
			ct:     strings.ToLower(respCT),
		}
	}
	url = respURL
	if url == "" {
		url = pending.url
	}
	ct = strings.ToLower(respCT)
	if ct == "" {
		ct = pending.ct
	}
	method = pending.method
	if method == "" {
		method = methodWant
	}
	return url, ct, method
}

func networkRequestNil(e network.EventRequestWillBeSent) bool {
	return e.Request == nil
}

func networkResponseNil(e network.EventResponseReceived) bool {
	return e.Response == nil
}

func headerValue(headers network.Headers, name string) string {
	if headers == nil {
		return ""
	}
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return fmt.Sprint(v)
		}
	}
	return ""
}
