// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package pagehost binds chromedp page operations into QuickJS globals for e2e.
package pagehost

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// hostHTTPClient is used by host.fetch; tests may override.
// Bounded so a hung endpoint cannot leave an e2e promise pending forever.
var hostHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ctxSchedule is Context.Schedule; tests may override to simulate a blocked job queue.
var ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
	return ctx.Schedule(job)
}

// jsonMarshal is encoding/json.Marshal; tests may override to force marshal failures.
var jsonMarshal = json.Marshal

// scheduleSelectTimeout / scheduleSelectTimeout2 bound schedule waits (overridable in tests).
var (
	scheduleSelectTimeout  = 500 * time.Millisecond
	scheduleSelectTimeout2 = 1500 * time.Millisecond
)

// Host owns the active page for one QuickJS e2e engine.
type Host struct {
	mu      sync.Mutex
	session *cdp.Session
	page    *cdp.Page
	pending sync.WaitGroup
	closed  atomic.Bool
}

// Install registers __choysum_e2e_runtime__ and __choysum_e2e_host__ on engine.
// runtimeJSON is the raw runtime.json object (or empty object "{}").
// The returned Host can Drain before engine.Close to avoid QuickJS abort from late Schedule.
func Install(engine jsengine.JsEngine, session *cdp.Session, runtimeJSON string) (*Host, error) {
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok || qjs == nil || qjs.Ctx == nil {
		return nil, fmt.Errorf("pagehost: engine must be *quickjsengine.QuickjsEngine")
	}
	runtimeJSON = stringsTrimJSON(runtimeJSON)
	if runtimeJSON == "" {
		runtimeJSON = "{}"
	}

	host := &Host{session: session}
	ctx := qjs.Ctx
	globals := ctx.Globals()

	runtimeVal := ctx.ParseJSON(runtimeJSON)
	if runtimeVal.IsException() {
		return nil, fmt.Errorf("pagehost: parse runtime json: %v", ctx.Exception())
	}
	globals.Set("__choysum_e2e_runtime__", runtimeVal)

	hostObj := ctx.Object()
	hostObj.Set("newPage", ctx.NewFunction(host.bindNewPage()))
	hostObj.Set("closePage", ctx.NewFunction(host.bindClosePage()))
	hostObj.Set("goto", ctx.NewFunction(host.bindGoto()))
	hostObj.Set("reload", ctx.NewFunction(host.bindReload()))
	hostObj.Set("click", ctx.NewFunction(host.bindClick()))
	hostObj.Set("fill", ctx.NewFunction(host.bindFill()))
	hostObj.Set("evaluate", ctx.NewFunction(host.bindEvaluate()))
	hostObj.Set("waitForFunction", ctx.NewFunction(host.bindWaitForFunction()))
	hostObj.Set("waitForResponse", ctx.NewFunction(host.bindWaitForResponse()))
	hostObj.Set("delay", ctx.NewFunction(host.bindDelay()))
	hostObj.Set("isVisible", ctx.NewFunction(host.bindIsVisible()))
	hostObj.Set("isEnabled", ctx.NewFunction(host.bindIsEnabled()))
	hostObj.Set("count", ctx.NewFunction(host.bindCount()))
	hostObj.Set("url", ctx.NewFunction(host.bindURL()))
	hostObj.Set("screenshot", ctx.NewFunction(host.bindScreenshot()))
	hostObj.Set("readTextFile", ctx.NewFunction(host.bindReadTextFile()))
	hostObj.Set("fetch", ctx.NewFunction(host.bindFetch()))
	hostObj.Set("clearOriginStorage", ctx.NewFunction(host.bindClearOriginStorage()))
	hostObj.Set("enableFetch", ctx.NewFunction(host.bindEnableFetch()))
	hostObj.Set("disableFetch", ctx.NewFunction(host.bindDisableFetch()))
	hostObj.Set("waitPausedRequest", ctx.NewFunction(host.bindWaitPausedRequest()))
	hostObj.Set("fulfillRequest", ctx.NewFunction(host.bindFulfillRequest()))
	hostObj.Set("continueRequest", ctx.NewFunction(host.bindContinueRequest()))
	globals.Set("__choysum_e2e_host__", hostObj)
	return host, nil
}

// Drain marks the host closed and waits for in-flight async host ops (delay / waitForResponse).
// Bounded so a goroutine blocked in ctx.Schedule (full job queue) cannot hang the runner forever.
func (h *Host) Drain() {
	if h == nil {
		return
	}
	h.closed.Store(true)
	done := make(chan struct{})
	go func() {
		h.pending.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func (h *Host) schedule(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
	if h == nil || ctx == nil || job == nil || h.closed.Load() {
		return false
	}
	// Schedule can block when the QuickJS job queue is full. Race with Drain by
	// not waiting unbounded once the host is marked closed.
	done := make(chan bool, 1)
	go func() {
		done <- ctxSchedule(ctx, job)
	}()
	select {
	case ok := <-done:
		return ok
	case <-time.After(scheduleSelectTimeout):
		if h.closed.Load() {
			return false
		}
		select {
		case ok := <-done:
			return ok
		case <-time.After(scheduleSelectTimeout2):
			return false
		}
	}
}

func stringsTrimJSON(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	return s
}

func (h *Host) activePage() (*cdp.Page, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.page == nil {
		return nil, fmt.Errorf("pagehost: no active page (call newPage first)")
	}
	return h.page, nil
}

func (h *Host) bindNewPage() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			h.mu.Lock()
			defer h.mu.Unlock()
			if h.session == nil {
				reject(ctx.Error(fmt.Errorf("pagehost: nil session")))
				return
			}
			if h.page != nil {
				h.page.Close()
				h.page = nil
			}
			p, err := h.session.NewPage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			h.page = p
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindClosePage() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			h.mu.Lock()
			defer h.mu.Unlock()
			if h.page != nil {
				h.page.Close()
				h.page = nil
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindGoto() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			url := ""
			waitUntil := "load"
			if len(args) > 0 && args[0] != nil {
				url = args[0].String()
			}
			if len(args) > 1 && args[1] != nil && !args[1].IsUndefined() && !args[1].IsNull() {
				waitUntil = args[1].String()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			// Must not block the QuickJS thread: Fetch route handlers run via Schedule
			// while navigation waits for continued/fulfilled requests.
			h.pending.Add(1)
			go func() {
				defer h.pending.Done()
				navErr := p.Goto(url, waitUntil)
				if h.closed.Load() {
					return
				}
				_ = h.schedule(ctx, func(inner *quickjs.Context) {
					if h.closed.Load() {
						return
					}
					if navErr != nil {
						reject(inner.Error(navErr))
						return
					}
					resolve(inner.Undefined())
				})
			}()
		})
	}
}

func (h *Host) bindReload() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			waitUntil := "load"
			if len(args) > 0 && args[0] != nil && !args[0].IsUndefined() && !args[0].IsNull() {
				waitUntil = args[0].String()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			h.pending.Add(1)
			go func() {
				defer h.pending.Done()
				reloadErr := p.Reload(waitUntil)
				if h.closed.Load() {
					return
				}
				_ = h.schedule(ctx, func(inner *quickjs.Context) {
					if h.closed.Load() {
						return
					}
					if reloadErr != nil {
						reject(inner.Error(reloadErr))
						return
					}
					resolve(inner.Undefined())
				})
			}()
		})
	}
}

func (h *Host) bindClearOriginStorage() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			originURL := argString(args, 0)
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.ClearOriginStorage(originURL); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindClick() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			sel := ""
			if len(args) > 0 && args[0] != nil {
				sel = args[0].String()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.Click(sel); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindFill() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			sel, text := "", ""
			if len(args) > 0 && args[0] != nil {
				sel = args[0].String()
			}
			if len(args) > 1 && args[1] != nil {
				text = args[1].String()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.Fill(sel, text); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindEvaluate() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			src := ""
			if len(args) > 0 && args[0] != nil {
				src = args[0].String()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			out, err := p.Evaluate(src)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.String(out))
		})
	}
}

func (h *Host) bindWaitForFunction() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			src := ""
			timeoutMs := int64(30000)
			if len(args) > 0 && args[0] != nil {
				src = args[0].String()
			}
			if len(args) > 1 && args[1] != nil && args[1].IsNumber() {
				timeoutMs = args[1].Int64()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.WaitForFunction(src, time.Duration(timeoutMs)*time.Millisecond); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindWaitForResponse() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			// Must not block the QuickJS thread: login arms waitForResponse then
			// clicks submit; a sync WaitForResponse would starve the click.
			matchJSON := "{}"
			timeoutMs := int64(30000)
			if len(args) > 0 && args[0] != nil {
				matchJSON = args[0].String()
			}
			if len(args) > 1 && args[1] != nil && args[1].IsNumber() {
				timeoutMs = args[1].Int64()
			}
			var match struct {
				URLIncludes       string `json:"urlIncludes"`
				Method            string `json:"method"`
				ContentTypePrefix string `json:"contentTypePrefix"`
			}
			if err := json.Unmarshal([]byte(matchJSON), &match); err != nil {
				reject(ctx.Error(fmt.Errorf("pagehost: waitForResponse match: %w", err)))
				return
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			rm := cdp.ResponseMatch{
				URLIncludes:       match.URLIncludes,
				Method:            match.Method,
				ContentTypePrefix: match.ContentTypePrefix,
			}
			timeout := time.Duration(timeoutMs) * time.Millisecond
			h.pending.Add(1)
			go func() {
				defer h.pending.Done()
				res, waitErr := p.WaitForResponse(rm, timeout)
				if h.closed.Load() {
					return
				}
				scheduled := h.schedule(ctx, func(inner *quickjs.Context) {
					if h.closed.Load() {
						return
					}
					if waitErr != nil {
						reject(inner.Error(waitErr))
						return
					}
					payload := map[string]any{
						"status":     res.Status,
						"statusText": http.StatusText(int(res.Status)),
						"headers":    res.Headers,
						"bodyBase64": base64.StdEncoding.EncodeToString(res.Body),
						"url":        res.URL,
					}
					raw, marshalErr := jsonMarshal(payload)
					if marshalErr != nil {
						reject(inner.Error(marshalErr))
						return
					}
					resolve(inner.String(string(raw)))
				})
				if !scheduled && waitErr != nil {
					// Context already shut down; drop the result.
					_ = waitErr
				}
			}()
		})
	}
}

func (h *Host) bindDelay() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			// EvalAwait does not pump QuickJS os.setTimeout; delay via Go + Schedule.
			ms := int64(0)
			if len(args) > 0 && args[0] != nil && args[0].IsNumber() {
				ms = args[0].Int64()
			}
			if ms < 0 {
				ms = 0
			}
			h.pending.Add(1)
			go func() {
				defer h.pending.Done()
				if ms > 0 {
					time.Sleep(time.Duration(ms) * time.Millisecond)
				}
				if h.closed.Load() {
					return
				}
				h.schedule(ctx, func(inner *quickjs.Context) {
					if h.closed.Load() {
						return
					}
					resolve(inner.Undefined())
				})
			}()
		})
	}
}

func (h *Host) bindIsVisible() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			sel := argString(args, 0)
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			ok, err := p.IsVisible(sel)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Bool(ok))
		})
	}
}

func (h *Host) bindIsEnabled() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			sel := argString(args, 0)
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			ok, err := p.IsEnabled(sel)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Bool(ok))
		})
	}
}

func (h *Host) bindCount() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			sel := argString(args, 0)
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			n, err := p.QueryCount(sel)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Int32(int32(n)))
		})
	}
}

func (h *Host) bindURL() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			u, err := p.URL()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.String(u))
		})
	}
}

func (h *Host) bindScreenshot() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			path := argString(args, 0)
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.Screenshot(path); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindReadTextFile() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			path := argString(args, 0)
			raw, err := os.ReadFile(path)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.String(string(raw)))
		})
	}
}

func (h *Host) bindFetch() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			url := argString(args, 0)
			initJSON := "{}"
			if len(args) > 1 && args[1] != nil && !args[1].IsUndefined() && !args[1].IsNull() {
				initJSON = args[1].String()
			}
			var init struct {
				Method     string            `json:"method"`
				Headers    map[string]string `json:"headers"`
				Body       string            `json:"body"`
				BodyBase64 string            `json:"bodyBase64"`
			}
			if strings.TrimSpace(initJSON) == "" {
				initJSON = "{}"
			}
			if err := json.Unmarshal([]byte(initJSON), &init); err != nil {
				reject(ctx.Error(fmt.Errorf("pagehost: fetch init: %w", err)))
				return
			}
			method := strings.TrimSpace(init.Method)
			if method == "" {
				method = http.MethodGet
			}
			var bodyReader io.Reader
			if init.BodyBase64 != "" {
				decoded, err := base64.StdEncoding.DecodeString(init.BodyBase64)
				if err != nil {
					reject(ctx.Error(fmt.Errorf("pagehost: fetch bodyBase64: %w", err)))
					return
				}
				bodyReader = bytes.NewReader(decoded)
			} else if init.Body != "" {
				bodyReader = strings.NewReader(init.Body)
			}
			req, err := http.NewRequest(method, url, bodyReader)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			for k, v := range init.Headers {
				req.Header.Set(k, v)
			}
			resp, err := hostHTTPClient.Do(req)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			headers := map[string]string{}
			for k, vals := range resp.Header {
				if len(vals) > 0 {
					headers[k] = strings.Join(vals, ", ")
				}
			}
			requestURL := url
			if resp.Request != nil && resp.Request.URL != nil {
				requestURL = resp.Request.URL.String()
			}
			payload := map[string]any{
				"status":     resp.StatusCode,
				"statusText": http.StatusText(resp.StatusCode),
				"headers":    headers,
				"bodyBase64": base64.StdEncoding.EncodeToString(body),
				"url":        requestURL,
			}
			raw, err := jsonMarshal(payload)
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.String(string(raw)))
		})
	}
}

func argString(args []*quickjs.Value, i int) string {
	if len(args) <= i || args[i] == nil {
		return ""
	}
	if args[i].IsUndefined() || args[i].IsNull() {
		return ""
	}
	return args[i].String()
}
