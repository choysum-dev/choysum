// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pagehost

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
)

func (h *Host) bindEnableFetch() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.EnableFetch(); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindDisableFetch() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.DisableFetch(); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindWaitPausedRequest() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			timeoutMs := int64(30000)
			if len(args) > 0 && args[0] != nil && args[0].IsNumber() {
				timeoutMs = args[0].Int64()
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			timeout := time.Duration(timeoutMs) * time.Millisecond
			h.pending.Add(1)
			go func() {
				defer h.pending.Done()
				paused, waitErr := p.WaitPaused(timeout)
				if h.closed.Load() {
					return
				}
				_ = h.schedule(ctx, func(inner *quickjs.Context) {
					if h.closed.Load() {
						return
					}
					if waitErr != nil {
						// Timeout → null so the JS route pump can keep polling.
						if strings.Contains(waitErr.Error(), "timeout") {
							resolve(inner.Null())
							return
						}
						reject(inner.Error(waitErr))
						return
					}
					b, err := jsonMarshal(paused)
					if err != nil {
						reject(inner.Error(err))
						return
					}
					resolve(inner.String(string(b)))
				})
			}()
		})
	}
}

func (h *Host) bindFulfillRequest() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			id := argString(args, 0)
			optsJSON := "{}"
			if len(args) > 1 && args[1] != nil {
				optsJSON = args[1].String()
			}
			var opts struct {
				Status      int               `json:"status"`
				ContentType string            `json:"contentType"`
				Body        string            `json:"body"`
				BodyBase64  string            `json:"bodyBase64"`
				Headers     map[string]string `json:"headers"`
			}
			if err := json.Unmarshal([]byte(optsJSON), &opts); err != nil {
				reject(ctx.Error(fmt.Errorf("pagehost: fulfillRequest opts: %w", err)))
				return
			}
			body := []byte(opts.Body)
			if opts.BodyBase64 != "" {
				decoded, err := base64.StdEncoding.DecodeString(opts.BodyBase64)
				if err != nil {
					reject(ctx.Error(fmt.Errorf("pagehost: fulfillRequest bodyBase64: %w", err)))
					return
				}
				body = decoded
			}
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.Fulfill(id, cdp.FulfillOptions{
				Status:      opts.Status,
				ContentType: opts.ContentType,
				Body:        body,
				Headers:     opts.Headers,
			}); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}

func (h *Host) bindContinueRequest() func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			id := argString(args, 0)
			p, err := h.activePage()
			if err != nil {
				reject(ctx.Error(err))
				return
			}
			if err := p.Continue(id); err != nil {
				reject(ctx.Error(err))
				return
			}
			resolve(ctx.Undefined())
		})
	}
}
