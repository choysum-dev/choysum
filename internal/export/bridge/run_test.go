// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"testing"

	exportbridge "github.com/choysum-dev/choysum/internal/export/bridge"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
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
