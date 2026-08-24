// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"context"
	_ "embed"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
)

// Engine represents a QuickJS engine instance with its runtime, context, and options.
type QuickjsEngine struct {
	Runtime *quickjs.Runtime          // QuickJS runtime instance
	Ctx     *quickjs.Context          // QuickJS context instance
	Option  *EngineOption             // Engine configuration options
	opts    []jsengine.JsEngineOption // Store original options for reloading

	// Current Go execution context for this engine execution.
	// Used by Go-side bridges and by the runtime interrupt handler (timeout/cancel).
	// atomic.Value requires a single concrete type; store a stable wrapper.
	execCtx atomic.Value // stores *execContextHolder
}

type execContextHolder struct {
	ctx context.Context
}

func (e *QuickjsEngine) setExecContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	e.execCtx.Store(&execContextHolder{ctx: ctx})
}

func (e *QuickjsEngine) getExecContext() context.Context {
	if v := e.execCtx.Load(); v != nil {
		if holder, ok := v.(*execContextHolder); ok && holder != nil && holder.ctx != nil {
			return holder.ctx
		}
	}
	return context.Background()
}

func (e *QuickjsEngine) ExecContext() context.Context {
	return e.getExecContext()
}

// SwapExecContext temporarily replaces the Go execution context used by bridges
// ($choysum.db, ORM, …). The returned restore function puts the previous context back.
func (e *QuickjsEngine) SwapExecContext(ctx context.Context) (restore func()) {
	prev := e.getExecContext()
	e.setExecContext(ctx)
	return func() { e.setExecContext(prev) }
}

func (e *QuickjsEngine) installInterruptHandler() {
	// Use a single interrupt handler to enforce both cancellation and per-exec deadlines.
	// This avoids Runtime.SetExecuteTimeout(), which would override/disable the interrupt handler.
	e.Runtime.SetInterruptHandler(func() int {
		execCtx := e.getExecContext()
		if execCtx == nil {
			return 0
		}
		select {
		case <-execCtx.Done():
			return 1
		default:
		}
		if deadline, ok := execCtx.Deadline(); ok {
			if time.Now().After(deadline) {
				return 1
			}
		}
		return 0
	})
}

// Load Load the provided scripts in the engine context.
// Each script is evaluated in order. If any script fails, an error is returned.
func (e *QuickjsEngine) Load(scripts []*jsengine.JsScript) error {
	for _, script := range scripts {
		ret := e.Ctx.Eval(script.Content, quickjs.EvalFileName(script.FileName), quickjs.EvalAwait(true))
		if ret == nil {
			return fmt.Errorf("failed to execute init script %s: eval returned nil", script.FileName)
		}
		if ret.IsException() {
			ret.Free()
			return fmt.Errorf("failed to execute init script %s: %w", script.FileName, e.Ctx.Exception())
		}
		ret.Free()
	}
	return nil
}

// Execute runs a JavaScript request using the embedded RPC script as the entry point.
// The request is marshaled to JS, passed to the RPC function, and the response is unmarshaled back to Go.
func (e *QuickjsEngine) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// If the caller didn't provide a deadline, apply the engine default timeout (if configured).
	// This deadline is enforced by the runtime interrupt handler.
	execCtx := ctx
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok && e.Option != nil && e.Option.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(e.Option.Timeout)*time.Second)
	}
	if cancel != nil {
		defer cancel()
	}

	// Make Go ctx available to Go-side bridges during this execution.
	e.setExecContext(execCtx)
	defer e.setExecContext(context.Background())
	defer e.Runtime.RunGC()

	// Evaluate the RPC script to get the function
	fn := e.Ctx.Eval("$choysum.__rpc__")
	defer fn.Free()
	if fn.IsException() {
		return nil, fmt.Errorf("failed to evaluate RPC script: %w", e.Ctx.Exception())
	}

	// Marshal the request to a JS value
	jsReq, err := e.Ctx.Marshal(req)
	defer jsReq.Free()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call the RPC function with the marshaled request
	jsResp := fn.Execute(e.Ctx.Null(), jsReq).Await()
	defer jsResp.Free()
	if jsResp.IsException() {
		// If we were interrupted due to cancellation/deadline, prefer returning ctx.Err().
		if err := execCtx.Err(); err != nil {
			_ = e.Ctx.Exception() // clear exception in QuickJS so runtime can be closed cleanly
			return nil, err
		}
		// Some interruptions may happen slightly before ctx.Err() is observable.
		if dl, ok := execCtx.Deadline(); ok && time.Now().After(dl) {
			_ = e.Ctx.Exception()
			return nil, context.DeadlineExceeded
		}
		return nil, fmt.Errorf("failed to call function: %w", e.Ctx.Exception())
	}

	// Unmarshal the JS response back to Go
	res := &jsengine.JsResponse{}
	if err := e.Ctx.Unmarshal(jsResp, res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return res, nil
}

// Close releases all resources associated with the engine, including context and runtime.
func (e *QuickjsEngine) Close() error {
	if e.Ctx != nil {
		e.Ctx.Close()
		e.Ctx = nil
	}
	if e.Runtime != nil {
		e.Runtime.RunGC()
		e.Runtime.Close()
		e.Runtime = nil
	}
	return nil
}

// newEngine creates a new QuickJS engine instance with the given options.
// It initializes the runtime, context, and applies all provided engine options.
func newEngine(options ...jsengine.JsEngineOption) (*QuickjsEngine, error) {
	// Create QuickJS runtime
	rt := quickjs.NewRuntime()

	// Create QuickJS context
	ctx := rt.NewContext()

	allOptions := append([]jsengine.JsEngineOption{WithTextEncodingPolyfillJS()}, options...)

	// Create engine instance with default options
	engine := &QuickjsEngine{
		Runtime: rt,
		Ctx:     ctx,
		Option: &EngineOption{
			MemoryLimit:        0,     // Default memory limit (no limit)
			GCThreshold:        -1,    // Default GC threshold. -1 means no threshold
			Timeout:            0,     // Default timeout (no timeout)
			MaxStackSize:       0,     // Default max stack size
			CanBlock:           false, // Blocking not allowed by default
			EnableModuleImport: false, // Module import disabled by default
			Strip:              1,     // Default strip behavior
		},
		opts: allOptions, // Store for Reload
	}
	engine.execCtx.Store(&execContextHolder{ctx: context.Background()})
	engine.installInterruptHandler()

	// Apply additional engine options
	for _, option := range allOptions {
		if err := option(engine); err != nil {
			engine.Close()
			return nil, err
		}
	}

	return engine, nil
}

// NewFactory returns a JsEngineFactory that creates QuickJS engines with the given options.
func NewFactory(options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return func() (jsengine.JsEngine, error) {
		return newEngine(options...)
	}
}
