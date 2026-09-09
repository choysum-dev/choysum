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

		go func() {
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
				fetchErr := chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
					b, err := network.GetResponseBody(reqID).Do(ctx)
					if err != nil {
						return err
					}
					body = b
					return nil
				}))
				// "No data found…" is often transient before the body is ready;
				// keep retrying until success or the waiter is canceled.
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
		}()
	}

	chromedp.ListenTarget(listenerCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			if e.Request == nil {
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
		case *network.EventResponseReceived:
			if e.Response == nil {
				return
			}
			mu.Lock()
			req, ok := pending[e.RequestID]
			if !ok {
				req = pendingReq{
					url:    e.Response.URL,
					method: methodWant,
					ct:     strings.ToLower(headerValue(e.Response.Headers, "content-type")),
				}
			}
			url := e.Response.URL
			if url == "" {
				url = req.url
			}
			// Prefer response Content-Type; fall back to request only when absent.
			ct := strings.ToLower(headerValue(e.Response.Headers, "content-type"))
			if ct == "" {
				ct = req.ct
			}
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
