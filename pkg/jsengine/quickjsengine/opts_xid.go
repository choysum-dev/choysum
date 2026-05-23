// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/rs/xid"
)

func newXidFunction(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	xid := xid.New().String()
	return ctx.String(xid)
}

func WithXid() jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		xidObj := jse.Ctx.Object()
		xidObj.Set("New", jse.Ctx.Function(newXidFunction))
		choysumObj.Set("xid", xidObj)

		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}
