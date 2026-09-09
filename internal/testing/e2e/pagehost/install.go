// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package pagehost binds chromedp page operations into QuickJS globals for e2e.
package pagehost

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// Host owns the active page for one QuickJS e2e engine.
type Host struct {
	mu      sync.Mutex
	session *cdp.Session
	page    *cdp.Page
}

// Install registers __choysum_e2e_runtime__ and __choysum_e2e_host__ on engine.
// runtimeJSON is the raw runtime.json object (or empty object "{}").
func Install(engine jsengine.JsEngine, session *cdp.Session, runtimeJSON string) error {
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok || qjs == nil || qjs.Ctx == nil {
		return fmt.Errorf("pagehost: engine must be *quickjsengine.QuickjsEngine")
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
		return fmt.Errorf("pagehost: parse runtime json: %v", ctx.Exception())
	}
	globals.Set("__choysum_e2e_runtime__", runtimeVal)

	hostObj := ctx.Object()
	hostObj.Set("newPage", ctx.NewFunction(host.bindNewPage()))
	hostObj.Set("closePage", ctx.NewFunction(host.bindClosePage()))
	hostObj.Set("goto", ctx.NewFunction(host.bindGoto()))
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
	globals.Set("__choysum_e2e_host__", hostObj)
	return nil
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
			if err := p.Goto(url, waitUntil); err != nil {
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
			go func() {
				res, waitErr := p.WaitForResponse(rm, timeout)
				ctx.Schedule(func(inner *quickjs.Context) {
					if waitErr != nil {
						reject(inner.Error(waitErr))
						return
					}
					payload := map[string]any{
						"status":     res.Status,
						"headers":    res.Headers,
						"bodyBase64": base64.StdEncoding.EncodeToString(res.Body),
						"url":        res.URL,
					}
					raw, marshalErr := json.Marshal(payload)
					if marshalErr != nil {
						reject(inner.Error(marshalErr))
						return
					}
					resolve(inner.String(string(raw)))
				})
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
			go func() {
				if ms > 0 {
					time.Sleep(time.Duration(ms) * time.Millisecond)
				}
				ctx.Schedule(func(inner *quickjs.Context) {
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

func argString(args []*quickjs.Value, i int) string {
	if len(args) <= i || args[i] == nil {
		return ""
	}
	return args[i].String()
}
