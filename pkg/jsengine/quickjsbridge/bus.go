// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type busPublishRequest struct {
	Topic   string         `json:"topic"`
	Source  string         `json:"source"`
	At      int64          `json:"at"`
	Payload map[string]any `json:"payload"`
}

// WithBusProvider installs $choysum.bus.publish against the host EventBus.
func WithBusProvider(scopeProvider jsengine.ScopeProvider) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		busObj := jse.Ctx.Object()
		busObj.Set("publish", jse.Ctx.Function(busPublishFactory(scopeProvider, jse)))
		busObj.Set("topicMessageThreadChanged", jse.Ctx.String(bus.TopicMessageThreadChanged))
		busObj.Set("topicMessageNotificationUser", jse.Ctx.String(bus.TopicMessageNotificationUser))
		busObj.Set("topicDispatchWakeup", jse.Ctx.String(bus.TopicDispatchWakeup))
		choysumObj.Set("bus", busObj)

		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

// WithBus installs $choysum.bus.publish for a fixed runtime scope.
func WithBus(runtimeScope scope.Scope) jsengine.JsEngineOption {
	return WithBusProvider(jsengine.StaticScopeProvider(runtimeScope))
}

func busPublishFactory(scopeProvider jsengine.ScopeProvider, engine *quickjsengine.QuickjsEngine) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("need 1 arg: event"))
		}

		var req busPublishRequest
		if err := ctx.Unmarshal(args[0], &req); err != nil {
			return ctx.ThrowError(fmt.Errorf("invalid bus.publish payload: %w", err))
		}
		req.Topic = strings.TrimSpace(req.Topic)
		if req.Topic == "" {
			return ctx.ThrowError(fmt.Errorf("topic is required"))
		}

		execCtx := engine.ExecContext()
		if execCtx == nil {
			execCtx = context.Background()
		}
		runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)
		events := bus.EnsureHost(runtimeScope)
		if events == nil {
			return ctx.Null()
		}

		at := time.Now().UTC()
		if req.At > 0 {
			at = time.UnixMilli(req.At).UTC()
		}
		if err := events.Publish(execCtx, bus.Event{
			Topic:   req.Topic,
			Source:  strings.TrimSpace(req.Source),
			At:      at,
			Payload: req.Payload,
		}); err != nil {
			return ctx.ThrowError(err)
		}
		return ctx.Null()
	}
}
