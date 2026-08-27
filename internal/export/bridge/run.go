// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/export/runner"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// WithExportProvider registers $choysum.export.run.
func WithExportProvider(scopeProvider jsengine.ScopeProvider) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		if jse == nil || jse.Ctx == nil {
			return fmt.Errorf("export: engine is required")
		}
		globalsObj := jse.Ctx.Globals()
		choysumObj := globalsObj.Get("$choysum")
		if !choysumObj.IsObject() {
			choysumObj.Free()
			choysumObj = jse.Ctx.Object()
		}
		exportObj := jse.Ctx.Object()
		exportObj.Set("run", jse.Ctx.NewFunction(runExportAsyncFactory(jse, scopeProvider)))
		choysumObj.Set("export", exportObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

func runExportAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performExportRun(ctx, jse, scopeProvider, args)
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

func performExportRun(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, args []*quickjs.Value) *quickjs.Value {
	if len(args) != 1 {
		return ctx.NewError(fmt.Errorf("export.run requires one spec argument"))
	}
	spec, err := decodeExportSpec(args[0])
	if err != nil {
		return ctx.NewError(fmt.Errorf("decode export spec: %w", err))
	}
	execCtx := jse.ExecContext()
	runtimeScope := resolveExportScope(scopeProvider, execCtx)
	if runtimeScope == nil {
		return ctx.NewError(fmt.Errorf("export.run: scope unavailable"))
	}
	report, err := runner.Run(execCtx, runtimeScope, spec)
	if err != nil {
		return ctx.NewError(err)
	}
	return marshalExportReport(ctx, report)
}

func marshalExportReport(ctx *quickjs.Context, report any) *quickjs.Value {
	val, err := ctx.Marshal(report)
	if err != nil {
		return ctx.NewError(fmt.Errorf("marshal export report: %w", err))
	}
	return val
}

func resolveExportScope(scopeProvider jsengine.ScopeProvider, execCtx context.Context) scope.Scope {
	if rs, ok := scope.ScopeFromContext(execCtx); ok && rs != nil {
		return rs
	}
	return jsengine.ResolveScope(scopeProvider, execCtx)
}

// Run executes export runner with an explicit scope (non-JS callers).
func Run(ctx context.Context, runtimeScope scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
	return runner.Run(ctx, runtimeScope, spec)
}

func decodeExportSpec(arg *quickjs.Value) (exportpkg.Spec, error) {
	var spec exportpkg.Spec
	if arg == nil || arg.IsUndefined() || arg.IsNull() {
		return spec, fmt.Errorf("spec is required")
	}
	raw := arg.JSONStringify()
	if arg.IsString() {
		raw = arg.String()
	}
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return spec, err
	}
	return spec, nil
}
