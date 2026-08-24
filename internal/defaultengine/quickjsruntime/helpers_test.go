// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func newTestQuickjsEngine(t *testing.T, options ...jsengine.JsEngineOption) *quickjsengine.QuickjsEngine {
	t.Helper()

	factory := quickjsengine.NewFactory(options...)
	engine, err := factory()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	quickEngine, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		_ = engine.Close()
		t.Fatalf("expected *quickjsengine.QuickjsEngine, got %T", engine)
	}
	t.Cleanup(func() {
		_ = engine.Close()
	})
	return quickEngine
}

func evalString(t *testing.T, engine *quickjsengine.QuickjsEngine, expr string) string {
	t.Helper()

	value := engine.Ctx.Eval(expr)
	defer value.Free()
	if value.IsException() {
		t.Fatalf("Eval(%q): %v", expr, engine.Ctx.Exception())
	}
	return value.String()
}

func evalBool(t *testing.T, engine *quickjsengine.QuickjsEngine, expr string) bool {
	t.Helper()

	value := engine.Ctx.Eval(expr)
	defer value.Free()
	if value.IsException() {
		t.Fatalf("Eval(%q): %v", expr, engine.Ctx.Exception())
	}
	return value.Bool()
}
