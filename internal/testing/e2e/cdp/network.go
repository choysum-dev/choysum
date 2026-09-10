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
	var body []byte
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		b, err := network.GetResponseBody(reqID).Do(ctx)
		if err != nil {
			return err
		}
		body = b
		return nil
	}))
	return body, err
}

// EnableNetwork turns on CDP Network domain for the page.
func EnableNetwork(p *Page) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	return chromedp.Run(p.ctx, network.Enable())
}

// WaitForResponse waits until a response matching m arrives (or timeout).
// Registers the CDP listener before returning to the select, so callers can
// arm this concurrently with the action that triggers the request.
func (p *Page) WaitForResponse(m ResponseMatch, timeout time.Duration) (*MatchedResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("cdp: nil page")
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

	tryFinish := func(reqID network.RequestID) {
		mu.Lock()
		resp, ok := matchedResp[reqID]
		if !ok || finishing[reqID] {
			mu.Unlock()
			return
		}
		finishing[reqID] = true
		mu.Unlock()

		go finishMatchedResponse(listenerCtx, &mu, matchedResp, pending, finishing, resultCh, reqID, resp, bodyGone, fetchResponseBody)
	}

	chromedp.ListenTarget(listenerCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			ingestRequestWillBeSent(e, &mu, pending)
		case *network.EventResponseReceived:
			ingestResponseReceived(e, &mu, pending, matchedResp, methodWant, matches, tryFinish)
		case *network.EventLoadingFinished:
			tryFinish(e.RequestID)
		}
	})

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case res := <-resultCh:
		return res, nil
	case <-timer.C:
		return nil, fmt.Errorf("cdp: waitForResponse timeout after %s (urlIncludes=%q method=%q)", timeout, urlPart, methodWant)
	}
}

func ingestRequestWillBeSent(e *network.EventRequestWillBeSent, mu *sync.Mutex, pending map[network.RequestID]pendingReq) {
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
	e *network.EventResponseReceived,
	mu *sync.Mutex,
	pending map[network.RequestID]pendingReq,
	matchedResp map[network.RequestID]pendingResp,
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
	mu.Unlock()
	// Prefer LoadingFinished for body availability; also arm a fallback in case
	// that event is skipped for some response types.
	go func() {
		time.Sleep(50 * time.Millisecond)
		tryFinish(reqID)
	}()
}

// finishMatchedResponse fetches a matched response body and publishes once.
// bodyGone deletes matched state and returns without publishing.
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
			// Resource already dropped (e.g. navigated away). Do not publish
			// a nil-body match; leave the waiter to time out or retry via a
			// later LoadingFinished if another finish arm still holds state.
			mu.Lock()
			delete(matchedResp, reqID)
			delete(pending, reqID)
			mu.Unlock()
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

func networkRequestNil(e *network.EventRequestWillBeSent) bool {
	return e == nil || e.Request == nil
}

func networkResponseNil(e *network.EventResponseReceived) bool {
	return e == nil || e.Response == nil
}

func headerValue(headers map[string]interface{}, name string) string {
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
