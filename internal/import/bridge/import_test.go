// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"testing"

	importbridge "github.com/choysum-dev/choysum/internal/import/bridge"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestWithImportProvider_ExposesRun(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(importbridge.WithImportProvider(jsengine.StaticScopeProvider(nil)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	got := engine.Ctx.Eval(`typeof $choysum.import.run`)
	defer got.Free()
	if got.IsException() {
		t.Fatalf("Eval: %v", engine.Ctx.Exception())
	}
	if got.String() != "function" {
		t.Fatalf("typeof run = %q, want function", got.String())
	}

	promise := engine.Ctx.Eval(`$choysum.import.run({profile:"record",caller:"user",policy:"atomic",model:"base.Country",source:{format:"csv",path:"missing.csv"}})`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("run call: %v", engine.Ctx.Exception())
	}
}
