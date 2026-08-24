// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package orm

import (
	"context"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

// EngineCaller invokes Model methods on the current QuickJS engine via $choysum.__rpc__.
// Safe to use from inside $choysum.import.run (same Ctx + Await; does not re-enter engine.Execute).
type EngineCaller struct {
	Engine *quickjsengine.QuickjsEngine
}

// Call implements Caller.
func (c EngineCaller) Call(ctx context.Context, req CallRequest) (any, error) {
	if c.Engine == nil || c.Engine.Ctx == nil {
		return nil, fmt.Errorf("orm: quickjs engine is required")
	}
	// Propagate the runner transaction context so $choysum.db / ORM see the same tx
	// as RecordWriter (Nested/RequiresNew), not only the outer Execute ExecContext.
	if ctx != nil {
		restore := c.Engine.SwapExecContext(ctx)
		defer restore()
	}
	service, err := ServiceName(req.Model, req.Method)
	if err != nil {
		return nil, err
	}
	jsReq := &jsengine.JsRequest{
		Id:      NewRequestID(),
		Service: service,
		Args:    req.Args,
		Context: MergeImportContext(req.Context),
	}
	return invokeRPC(c.Engine.Ctx, jsReq)
}

func invokeRPC(qctx *quickjs.Context, req *jsengine.JsRequest) (any, error) {
	fn := qctx.Eval("$choysum.__rpc__")
	defer fn.Free()
	if fn.IsException() {
		return nil, fmt.Errorf("orm: evaluate __rpc__: %w", qctx.Exception())
	}
	if !fn.IsFunction() {
		return nil, fmt.Errorf("orm: $choysum.__rpc__ is not a function")
	}
	jsReq, err := qctx.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("orm: marshal request: %w", err)
	}
	defer jsReq.Free()

	jsResp := fn.Execute(qctx.Null(), jsReq).Await()
	defer jsResp.Free()
	if jsResp.IsException() {
		return nil, fmt.Errorf("orm: call %s: %w", req.Service, qctx.Exception())
	}
	if jsResp.IsError() {
		return nil, fmt.Errorf("orm: call %s: %v", req.Service, jsResp.ToError())
	}

	var res jsengine.JsResponse
	if err := qctx.Unmarshal(jsResp, &res); err != nil {
		return nil, fmt.Errorf("orm: unmarshal response: %w", err)
	}
	return res.Result, nil
}

// ExecutorCaller invokes Model methods through a jsengine.JsEngine (Hub / CLI).
type ExecutorCaller struct {
	Engine jsengine.JsEngine
}

// Call implements Caller.
func (c ExecutorCaller) Call(ctx context.Context, req CallRequest) (any, error) {
	if c.Engine == nil {
		return nil, fmt.Errorf("orm: js engine is required")
	}
	service, err := ServiceName(req.Model, req.Method)
	if err != nil {
		return nil, err
	}
	jsReq := &jsengine.JsRequest{
		Id:      NewRequestID(),
		Service: service,
		Args:    req.Args,
		Context: MergeImportContext(req.Context),
	}
	resp, err := c.Engine.Execute(ctx, jsReq)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.Result, nil
}
