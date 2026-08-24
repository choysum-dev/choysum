// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package orm

import (
	"encoding/json"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// Register mounts $choysum.orm.call on the engine (same slot as import.run).
func Register(jse *quickjsengine.QuickjsEngine) error {
	if jse == nil || jse.Ctx == nil {
		return fmt.Errorf("orm: engine is required")
	}
	globalsObj := jse.Ctx.Globals()
	choysumObj := globalsObj.Get("$choysum")
	if !choysumObj.IsObject() {
		choysumObj.Free()
		choysumObj = jse.Ctx.Object()
	}
	ormObj := jse.Ctx.Object()
	ormObj.Set("call", jse.Ctx.NewFunction(ormCallAsyncFactory(jse)))
	choysumObj.Set("orm", ormObj)
	globalsObj.Set("$choysum", choysumObj)
	return nil
}

func ormCallAsyncFactory(jse *quickjsengine.QuickjsEngine) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	caller := EngineCaller{Engine: jse}
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performOrmCall(ctx, caller, args)
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

func performOrmCall(ctx *quickjs.Context, caller EngineCaller, args []*quickjs.Value) *quickjs.Value {
	if len(args) != 1 {
		return ctx.NewError(fmt.Errorf("orm.call requires one request argument"))
	}
	req, err := decodeCallRequest(args[0])
	if err != nil {
		return ctx.NewError(fmt.Errorf("decode orm.call request: %w", err))
	}
	result, err := caller.Call(caller.Engine.ExecContext(), req)
	if err != nil {
		return ctx.NewError(err)
	}
	return marshalORMResult(ctx, result)
}

func decodeCallRequest(arg *quickjs.Value) (CallRequest, error) {
	var req CallRequest
	if arg == nil || arg.IsUndefined() || arg.IsNull() {
		return req, fmt.Errorf("request is required")
	}
	raw := arg.JSONStringify()
	if arg.IsString() {
		raw = arg.String()
	}
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return req, err
	}
	return req, nil
}

func marshalORMResult(ctx *quickjs.Context, result any) *quickjs.Value {
	val, err := ctx.Marshal(result)
	if err != nil {
		return ctx.NewError(fmt.Errorf("marshal orm.call result: %w", err))
	}
	return val
}

// Ensure jsengine import retained for docs/links in this file.
var _ = jsengine.JsRequest{}
