// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buke/quickjs-go"
	documentgateway "github.com/choysum-dev/choysum/internal/document/gateway"
	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	"github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/internal/import/runner"
	"github.com/choysum-dev/choysum/pkg/auth"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// WithImportProvider registers $choysum.import.run.
func WithImportProvider(scopeProvider jsengine.ScopeProvider) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		if jse == nil || jse.Ctx == nil {
			return fmt.Errorf("import: engine is required")
		}
		globalsObj := jse.Ctx.Globals()
		choysumObj := globalsObj.Get("$choysum")
		if !choysumObj.IsObject() {
			choysumObj.Free()
			choysumObj = jse.Ctx.Object()
		}
		importObj := jse.Ctx.Object()
		importObj.Set("run", jse.Ctx.NewFunction(runImportAsyncFactory(jse, scopeProvider)))
		choysumObj.Set("import", importObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

func runImportAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performImportRun(ctx, jse, scopeProvider, args)
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

func performImportRun(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, args []*quickjs.Value) *quickjs.Value {
	if len(args) != 1 {
		return ctx.NewError(fmt.Errorf("import.run requires one spec argument"))
	}
	spec, err := decodeImportSpec(args[0])
	if err != nil {
		return ctx.NewError(fmt.Errorf("decode import spec: %w", err))
	}
	execCtx := jse.ExecContext()
	// Prefer the transactional scope stored on ExecContext (same as $choysum.db).
	// ResolveScope alone rebinds the factory base scope and can open a second SQLite
	// connection while BE unit tests already hold a write transaction → "database is locked".
	runtimeScope := resolveImportScope(scopeProvider, execCtx)
	if runtimeScope == nil {
		return ctx.NewError(fmt.Errorf("import.run: scope unavailable"))
	}
	spec, err = resolveRecordSourcePath(runtimeScope, spec)
	if err != nil {
		return ctx.NewError(fmt.Errorf("resolve import source path: %w", err))
	}
	runCtx := caller.ContextWithCaller(execCtx, caller.EngineCaller{Engine: jse})
	runCtx = attachImportSourceLoader(runCtx, runtimeScope, spec)
	report, err := runner.Run(runCtx, runtimeScope, spec)
	if err != nil {
		return ctx.NewError(err)
	}
	return marshalImportReport(ctx, report)
}

func marshalImportReport(ctx *quickjs.Context, report any) *quickjs.Value {
	val, err := ctx.Marshal(report)
	if err != nil {
		return ctx.NewError(fmt.Errorf("marshal import report: %w", err))
	}
	return val
}

func resolveImportScope(scopeProvider jsengine.ScopeProvider, execCtx context.Context) scope.Scope {
	if rs, ok := scope.ScopeFromContext(execCtx); ok && rs != nil {
		return rs
	}
	return jsengine.ResolveScope(scopeProvider, execCtx)
}

// Run executes import.Run with an explicit scope (non-JS callers).
func Run(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
	spec, err := resolveRecordSourcePath(runtimeScope, spec)
	if err != nil {
		return importpkg.Report{}, err
	}
	ctx = attachImportSourceLoader(ctx, runtimeScope, spec)
	return runner.Run(ctx, runtimeScope, spec)
}

func attachImportSourceLoader(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec) context.Context {
	if strings.TrimSpace(spec.Source.DocumentRef) == "" || runtimeScope == nil {
		return ctx
	}
	if csv.HasSourceBytesLoader(ctx) {
		return ctx
	}
	return csv.ContextWithSourceBytes(ctx, func(ctx context.Context, documentRef string) ([]byte, error) {
		identity := auth.IdentityFromContext(ctx)
		if identity == nil || !identity.IsValid() {
			return nil, fmt.Errorf("authentication is required")
		}
		return readImportSourceBytes(ctx, runtimeScope, documentRef, identity)
	})
}

var readImportSourceBytes = documentgateway.ReadSourceRefBytes

func decodeImportSpec(arg *quickjs.Value) (importpkg.Spec, error) {
	var spec importpkg.Spec
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

func resolveRecordSourcePath(runtimeScope scope.Scope, spec importpkg.Spec) (importpkg.Spec, error) {
	path := strings.TrimSpace(spec.Source.Path)
	if path == "" || runtimeScope == nil {
		return spec, nil
	}
	// Absolute paths must not come through the JS bridge (callers use runner.Run directly).
	if filepath.IsAbs(path) {
		return spec, importpkg.Errorf(importpkg.CodeInvalidFormat, "import source path must be relative")
	}
	if strings.Contains(path, "\x00") {
		return spec, importpkg.Errorf(importpkg.CodeInvalidFormat, "import source path is invalid")
	}
	paths, ok := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	if !ok {
		return spec, nil
	}
	modulesPath := strings.TrimSpace(paths.ModulesPath)
	if modulesPath == "" {
		return spec, nil
	}
	baseDir := filepath.Clean(filepath.Dir(modulesPath))
	resolved := filepath.Clean(filepath.Join(baseDir, path))
	rel, err := filepath.Rel(baseDir, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return spec, importpkg.Errorf(importpkg.CodeInvalidFormat, "import source path escapes modules root")
	}
	spec.Source.Path = resolved
	return spec, nil
}
