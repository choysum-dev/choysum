// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	importbridge "github.com/choysum-dev/choysum/internal/import/bridge"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestWithImportProvider_ExposesRun(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(importbridge.WithImportProvider(jsengine.StaticScopeProvider(nil)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	got := engine.Ctx.Eval(`typeof $choysum.import.run === "function" && typeof $choysum.orm.call === "function"`)
	defer got.Free()
	if got.IsException() {
		t.Fatalf("Eval: %v", engine.Ctx.Exception())
	}
	if !got.ToBool() {
		t.Fatal("expected $choysum.import.run and $choysum.orm.call to be functions")
	}

	promise := engine.Ctx.Eval(`$choysum.import.run({profile:"record",caller:"user",policy:"atomic",model:"base.Country",source:{format:"csv",path:"missing.csv"}})`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("run call: %v", engine.Ctx.Exception())
	}
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected error when scope unavailable / missing file")
	}
}

func TestWithImportProvider_CreatesChoysumNamespace(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		cleared := jse.Ctx.Eval(`delete globalThis.$choysum; true`)
		defer cleared.Free()
		return importbridge.WithImportProvider(jsengine.StaticScopeProvider(nil))(jsEngine)
	})()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	got := engine.Ctx.Eval(`typeof $choysum.import.run === "function"`)
	defer got.Free()
	if !got.ToBool() {
		t.Fatal("expected import.run after creating $choysum")
	}
}

func TestImportRun_WithScopeMissingFile(t *testing.T) {
	root := t.TempDir()
	modulesPath := filepath.Join(root, "modules")
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "bridge.db")},
		ModulesPath: modulesPath,
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	engineIface, err := quickjsengine.NewFactory(importbridge.WithImportProvider(jsengine.StaticScopeProvider(runtimeScope)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	execCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	restore := engine.SwapExecContext(execCtx)
	defer restore()

	promise := engine.Ctx.Eval(`$choysum.import.run({profile:"record",caller:"user",policy:"atomic",model:"base.Country",source:{format:"csv",path:"modules/missing.csv"}})`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected missing file error")
	}
}

func TestImportRun_ArgValidation(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(importbridge.WithImportProvider(jsengine.StaticScopeProvider(nil)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	promise := engine.Ctx.Eval(`$choysum.import.run()`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected missing arg error")
	}

	promise2 := engine.Ctx.Eval(`$choysum.import.run("not-json-object")`)
	defer promise2.Free()
	result2 := promise2.Await()
	defer result2.Free()
	// string that is not valid JSON object for Spec may fail decode
	_ = result2
}

func TestRun_RejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	modulesPath := filepath.Join(root, "modules")
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "path.db")},
		ModulesPath: modulesPath,
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	_, err := importbridge.Run(context.Background(), runtimeScope, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: "../secret.csv"},
	})
	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes modules root") {
		t.Fatalf("error = %v, want escapes modules root", err)
	}
}

func TestRun_RejectsAbsoluteSourcePath(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "abs.db")},
		ModulesPath: filepath.Join(root, "modules"),
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	_, err := importbridge.Run(context.Background(), runtimeScope, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: filepath.Join(root, "modules", "x.csv")},
	})
	if err == nil {
		t.Fatal("expected absolute path rejection")
	}
	if !strings.Contains(err.Error(), "must be relative") {
		t.Fatalf("error = %v, want must be relative", err)
	}
}

func TestRun_NullBytePath(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "null.db")},
		ModulesPath: filepath.Join(root, "modules"),
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	_, err := importbridge.Run(context.Background(), runtimeScope, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: "x\x00y.csv"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error=%v", err)
	}
}
