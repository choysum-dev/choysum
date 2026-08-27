// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	exportbridge "github.com/choysum-dev/choysum/internal/export/bridge"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestWithExportProvider_ExposesRun(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(exportbridge.WithExportProvider(jsengine.StaticScopeProvider(nil)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	got := engine.Ctx.Eval(`typeof $choysum.export.run === "function"`)
	defer got.Free()
	if got.IsException() {
		t.Fatalf("Eval: %v", engine.Ctx.Exception())
	}
	if !got.ToBool() {
		t.Fatal("expected $choysum.export.run to be a function")
	}

	promise := engine.Ctx.Eval(`$choysum.export.run({profile:"record",caller:"user",model:"base.Country",format:"csv"})`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("run call: %v", engine.Ctx.Exception())
	}
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected error when scope unavailable")
	}
}

func TestWithExportProvider_CreatesChoysumNamespace(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		cleared := jse.Ctx.Eval(`delete globalThis.$choysum; true`)
		defer cleared.Free()
		return exportbridge.WithExportProvider(jsengine.StaticScopeProvider(nil))(jsEngine)
	})()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	got := engine.Ctx.Eval(`typeof $choysum.export.run === "function"`)
	defer got.Free()
	if !got.ToBool() {
		t.Fatal("expected export.run after creating $choysum")
	}
}

func TestExportRun_ArgValidation(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(exportbridge.WithExportProvider(jsengine.StaticScopeProvider(nil)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	promise := engine.Ctx.Eval(`$choysum.export.run()`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected missing arg error")
	}

	promise2 := engine.Ctx.Eval(`$choysum.export.run("not-json-object")`)
	defer promise2.Free()
	result2 := promise2.Await()
	defer result2.Free()
	if !result2.IsError() && !result2.IsException() {
		t.Fatal("expected error for non-JSON export.run argument")
	}

	promise3 := engine.Ctx.Eval(`$choysum.export.run(null)`)
	defer promise3.Free()
	result3 := promise3.Await()
	defer result3.Free()
	if !result3.IsError() && !result3.IsException() {
		t.Fatal("expected error for null export.run argument")
	}
}

func TestExportRun_WithScope(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "export-bridge.db")},
		ModulesPath: filepath.Join(root, "modules"),
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	engineIface, err := quickjsengine.NewFactory(exportbridge.WithExportProvider(jsengine.StaticScopeProvider(runtimeScope)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	execCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	restore := engine.SwapExecContext(execCtx)
	defer restore()

	promise := engine.Ctx.Eval(`$choysum.export.run({profile:"record",caller:"user",model:"base.Country",format:"csv",async:true})`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected export validation error without seeded model")
	}
}

func TestRun(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "export-run.db")},
		ModulesPath: filepath.Join(root, "modules"),
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	_, err := exportbridge.Run(context.Background(), runtimeScope, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
		Async:   true,
	})
	if err == nil {
		t.Fatal("expected validation error for async spec via bridge")
	}
}
