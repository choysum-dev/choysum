// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package caller

import (
	"errors"
	"strings"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/oerrors"
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

func TestFormatCallJSError(t *testing.T) {
	err := formatCallJSError("svc", nil, "raw-value")
	if err == nil || !strings.Contains(err.Error(), "unknown JS error: raw-value") {
		t.Fatalf("expected detailed unknown error, got %v", err)
	}
	err = formatCallJSError("svc", nil, "")
	if err == nil || err.Error() != "caller: call svc: unknown JS error" {
		t.Fatalf("expected plain unknown error, got %v", err)
	}
	qjsErr := formatCallJSError("svc", errors.New("plain"), "")
	if qjsErr == nil || !strings.Contains(qjsErr.Error(), "plain") {
		t.Fatalf("expected wrapped plain error, got %v", qjsErr)
	}
	structured := formatCallJSError("svc", &quickjs.Error{
		Message:    "boom",
		JSONString: `{"domain":"web","code":"EJS","message":"boom"}`,
	}, "")
	if info := oerrors.GetErrorInfo(structured); info == nil || info.Domain != "web" || info.Code != "EJS" {
		t.Fatalf("expected structured JS error to keep domain/code, got %#v (err=%v)", info, structured)
	}
}
