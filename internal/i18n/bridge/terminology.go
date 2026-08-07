// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/buke/quickjs-go"
	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// LookupFunc is the sync terminology lookup used by $choysum.i18n.t.
type LookupFunc func(module, lang, scope, src, kind string) (value string, ok bool)

// WithTerminology registers sync $choysum.i18n.t against a fixed registry (lookup only).
// Prefer WithTerminologyProvider when invalidate/import are needed.
func WithTerminology(reg *store.Registry) jsengine.JsEngineOption {
	if reg == nil {
		return WithTerminologyLookup(nil)
	}
	return WithTerminologyLookup(reg.Lookup)
}

// WithTerminologyProvider registers $choysum.i18n.t, invalidateModule, and upsertPackagedTerms.
// The process-shared Registry is captured once at install (StaticScopeProvider wraps a new
// scope per ResolveScope call; re-calling RegistryFor would reset the cache). DB writes for
// upsert resolve the request scope via provider.
//
// Note: quickjs-go Value.Set consumes the property value (JS_SetProperty without
// Dup). Do not Free() values after Set — that double-frees and crashes.
func WithTerminologyProvider(scopeProvider jsengine.ScopeProvider) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		base := jsengine.ResolveScope(scopeProvider, context.Background())
		var reg *store.Registry
		if base != nil {
			reg = store.RegistryFor(base)
		}
		return installI18nObject(jse, scopeProvider, reg, nil)
	}
}

// WithTerminologyLookup registers $choysum.i18n.t with a custom lookup (tests).
func WithTerminologyLookup(lookup LookupFunc) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		return installI18nObject(jse, nil, nil, lookup)
	}
}

func installI18nObject(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, reg *store.Registry, lookup LookupFunc) error {
	globalsObj := jse.Ctx.Globals()

	choysumObj := globalsObj.Get("$choysum")
	if !choysumObj.IsObject() {
		// Get() returns an owned handle; free before replacing non-objects
		// (undefined/null/primitives). Do not Free() Globals() itself.
		choysumObj.Free()
		choysumObj = jse.Ctx.Object()
	}

	i18nObj := jse.Ctx.Object()
	if lookup != nil {
		i18nObj.Set("t", jse.Ctx.Function(terminologyLookupFunc(lookup)))
	} else if reg != nil {
		i18nObj.Set("t", jse.Ctx.Function(terminologyLookupFunc(reg.Lookup)))
	} else {
		i18nObj.Set("t", jse.Ctx.Function(terminologyLookupFunc(nil)))
	}
	if reg != nil {
		i18nObj.Set("invalidateModule", jse.Ctx.Function(invalidateModuleFunc(reg)))
	}
	if scopeProvider != nil && reg != nil {
		i18nObj.Set("upsertPackagedTerms", jse.Ctx.NewFunction(upsertPackagedTermsAsyncFactory(jse, scopeProvider, reg)))
	}
	choysumObj.Set("i18n", i18nObj)
	globalsObj.Set("$choysum", choysumObj)
	return nil
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

func invalidateModuleFunc(reg *store.Registry) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if reg == nil || len(args) < 2 {
			return ctx.Bool(false)
		}
		if args[0] == nil || args[0].IsUndefined() || args[0].IsNull() ||
			args[1] == nil || args[1].IsUndefined() || args[1].IsNull() {
			return ctx.Bool(false)
		}
		application := strings.TrimSpace(args[0].String())
		module := strings.TrimSpace(args[1].String())
		if application == "" || application == "core" || module == "" {
			return ctx.Bool(false)
		}
		ts, ok := reg.ExistingStore(application)
		if !ok {
			return ctx.Bool(false)
		}
		ts.InvalidateModule(module)
		return ctx.Bool(true)
	}
}

func upsertPackagedTermsAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, reg *store.Registry) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performUpsertPackagedTerms(ctx, jse, scopeProvider, reg, args)
			// NewError values are IsError; never resolve ThrowError's JS_EXCEPTION sentinel.
			if ret.IsError() {
				defer ret.Free()
				reject(ret)
				return
			}
			defer ret.Free()
			resolve(ret)
		})
	}
}

func performUpsertPackagedTerms(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, reg *store.Registry, args []*quickjs.Value) *quickjs.Value {
	if len(args) < 4 {
		return ctx.NewError(fmt.Errorf("upsertPackagedTerms requires application, module, lang, poText"))
	}
	if args[0] == nil || args[0].IsUndefined() || args[0].IsNull() ||
		args[1] == nil || args[1].IsUndefined() || args[1].IsNull() ||
		args[2] == nil || args[2].IsUndefined() || args[2].IsNull() {
		return ctx.NewError(fmt.Errorf("upsertPackagedTerms: application, module, and lang are required"))
	}
	application := strings.TrimSpace(args[0].String())
	module := strings.TrimSpace(args[1].String())
	lang := strings.TrimSpace(args[2].String())
	poText, err := poTextBytes(args[3])
	if err != nil {
		return ctx.NewError(err)
	}
	if application == "" || application == "core" || module == "" || lang == "" {
		return ctx.NewError(fmt.Errorf("upsertPackagedTerms: application, module, and lang are required"))
	}

	execCtx := jse.ExecContext()
	if execCtx == nil {
		execCtx = context.Background()
	}
	rs := jsengine.ResolveScope(scopeProvider, execCtx)
	if rs == nil || rs.Session() == nil {
		return ctx.NewError(fmt.Errorf("upsertPackagedTerms: missing runtime session"))
	}
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, application, module, lang, poText)
	if err != nil {
		return ctx.NewError(err)
	}
	if stats == nil {
		stats = &i18nimport.ImportStats{Lang: lang}
	}
	payload := map[string]any{
		"upserted":        stats.Upserted,
		"skippedOverride": stats.SkippedOverride,
		"rejectedNoCtxt":  stats.RejectedNoCtxt,
		"skippedObsolete": stats.SkippedObsolete,
		"purgedRetired":   stats.PurgedRetired,
		"lang":            stats.Lang,
	}
	val, marshalErr := ctx.Marshal(payload)
	if marshalErr != nil {
		return ctx.NewError(marshalErr)
	}
	return val
}

func poTextBytes(v *quickjs.Value) ([]byte, error) {
	if v == nil || v.IsUndefined() || v.IsNull() {
		return nil, fmt.Errorf("poText is required")
	}
	if v.IsString() {
		return []byte(v.String()), nil
	}
	if v.IsUint8Array() || v.IsUint8ClampedArray() {
		return v.ToUint8Array()
	}
	if v.IsByteArray() {
		return v.ToByteArray(uint(v.ByteLen()))
	}
	return nil, fmt.Errorf("poText must be a string or Uint8Array")
}
