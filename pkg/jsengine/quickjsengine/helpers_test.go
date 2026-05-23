// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

func newTestQuickjsEngine(t *testing.T, options ...jsengine.JsEngineOption) *QuickjsEngine {
	t.Helper()

	engine, err := newEngine(options...)
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	t.Cleanup(func() {
		_ = engine.Close()
	})
	return engine
}

func evalString(t *testing.T, engine *QuickjsEngine, expr string) string {
	t.Helper()

	value := engine.Ctx.Eval(expr)
	defer value.Free()
	if value.IsException() {
		t.Fatalf("Eval(%q): %v", expr, engine.Ctx.Exception())
	}
	return value.String()
}

func evalBool(t *testing.T, engine *QuickjsEngine, expr string) bool {
	t.Helper()

	value := engine.Ctx.Eval(expr)
	defer value.Free()
	if value.IsException() {
		t.Fatalf("Eval(%q): %v", expr, engine.Ctx.Exception())
	}
	return value.Bool()
}

func evalInt64(t *testing.T, engine *QuickjsEngine, expr string) int64 {
	t.Helper()

	value := engine.Ctx.Eval(expr)
	defer value.Free()
	if value.IsException() {
		t.Fatalf("Eval(%q): %v", expr, engine.Ctx.Exception())
	}
	return value.Int64()
}
