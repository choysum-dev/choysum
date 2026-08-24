// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package orm

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestInvokeRPC_ErrorObject(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	stub := engine.Ctx.Eval(`
		globalThis.$choysum = { __rpc__: async () => new Error("rpc-error-obj") };
		true
	`)
	defer stub.Free()
	_, err = invokeRPC(engine.Ctx, &jsengine.JsRequest{Id: "1", Service: "base.Country.Create"})
	if err == nil || !strings.Contains(err.Error(), "rpc-error-obj") && !strings.Contains(err.Error(), "Error") {
		t.Fatalf("err=%v", err)
	}
}

func TestInvokeRPC_EvalException(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	// Remove $choysum so Eval of $choysum.__rpc__ throws / is not a function
	clear := engine.Ctx.Eval(`globalThis.$choysum = undefined; true`)
	defer clear.Free()
	_, err = invokeRPC(engine.Ctx, &jsengine.JsRequest{Id: "1", Service: "base.Country.Create"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvokeRPC_MarshalAndUnmarshalErrors(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	stub := engine.Ctx.Eval(`globalThis.$choysum = { __rpc__: async () => 123 }; true`)
	defer stub.Free()
	_, err = invokeRPC(engine.Ctx, &jsengine.JsRequest{Id: "1", Service: "base.Country.Create", Args: []any{make(chan int)}})
	if err == nil {
		t.Fatal("expected marshal error")
	}

	stub2 := engine.Ctx.Eval(`globalThis.$choysum.__rpc__ = async () => 123; true`)
	defer stub2.Free()
	_, err = invokeRPC(engine.Ctx, &jsengine.JsRequest{Id: "1", Service: "base.Country.Create", Args: []any{}})
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}
