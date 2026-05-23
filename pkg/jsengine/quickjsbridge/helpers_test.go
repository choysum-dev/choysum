// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func newTestQuickjsEngine(t *testing.T, options ...jsengine.JsEngineOption) *quickjsengine.QuickjsEngine {
	t.Helper()

	engine, err := quickjsengine.NewFactory(options...)()
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	quickjsEngine, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatalf("expected *quickjsengine.QuickjsEngine, got %T", engine)
	}
	t.Cleanup(func() {
		_ = quickjsEngine.Close()
	})
	return quickjsEngine
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