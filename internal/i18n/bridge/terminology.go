// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"strings"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// LookupFunc is the sync terminology lookup used by $choysum.i18n.t.
type LookupFunc func(module, lang, scope, src, kind string) (value string, ok bool)

// WithTerminology registers sync $choysum.i18n.t(module, lang, scope, src[, kind]).
// Miss returns empty string; TS _t is responsible for falling back to src.
func WithTerminology(reg *store.Registry) jsengine.JsEngineOption {
	return WithTerminologyLookup(reg.Lookup)
}

// WithTerminologyLookup registers $choysum.i18n.t with a custom lookup (tests).
func WithTerminologyLookup(lookup LookupFunc) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() || choysumObj.IsNull() {
			choysumObj = jse.Ctx.Object()
		}

		i18nObj := jse.Ctx.Object()
		i18nObj.Set("t", jse.Ctx.Function(terminologyLookupFunc(lookup)))
		choysumObj.Set("i18n", i18nObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

func terminologyLookupFunc(lookup LookupFunc) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if lookup == nil || len(args) < 4 {
			return ctx.String("")
		}
		module := strings.TrimSpace(args[0].String())
		lang := strings.TrimSpace(args[1].String())
		scopeKey := strings.TrimSpace(args[2].String())
		src := args[3].String()
		kind := models.KindLiteral
		if len(args) >= 5 && !args[4].IsUndefined() && !args[4].IsNull() {
			if k := strings.TrimSpace(args[4].String()); k != "" {
				kind = k
			}
		}
		val, ok := lookup(module, lang, scopeKey, src, kind)
		if !ok {
			return ctx.String("")
		}
		return ctx.String(val)
	}
}
