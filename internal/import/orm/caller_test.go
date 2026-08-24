// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package orm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/orm"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestServiceName(t *testing.T) {
	t.Parallel()
	got, err := orm.ServiceName("base.Country", "Create")
	if err != nil || got != "base.Country.Create" {
		t.Fatalf("ServiceName = %q %v", got, err)
	}
	if _, err := orm.ServiceName("Country", "Create"); err == nil {
		t.Fatal("expected error for short model name")
	}
	if _, err := orm.ServiceName("", "Create"); err == nil {
		t.Fatal("expected error for empty model")
	}
	if _, err := orm.ServiceName("base.Country", "  "); err == nil {
		t.Fatal("expected error for empty method")
	}
}

func TestMergeImportContext_ReservedImportFile(t *testing.T) {
	t.Parallel()
	got := orm.MergeImportContext(map[string]any{
		"lang":        "zh_CN",
		"import_file": false,
		"":            "skip",
		"  ":          "skip",
	})
	if got["import_file"] != true {
		t.Fatalf("import_file = %#v, want true", got["import_file"])
	}
	if got["lang"] != "zh_CN" {
		t.Fatalf("lang = %#v", got["lang"])
	}
	if _, ok := got[""]; ok {
		t.Fatal("empty key should be skipped")
	}
}

func TestCallerContextRoundTrip(t *testing.T) {
	t.Parallel()
	caller := stubCaller{}
	ctx := orm.ContextWithCaller(nil, caller)
	got, ok := orm.CallerFromContext(ctx)
	if !ok || got != caller {
		t.Fatalf("CallerFromContext = %#v %v", got, ok)
	}
	if _, ok := orm.CallerFromContext(nil); ok {
		t.Fatal("expected false for nil context")
	}
	if _, ok := orm.CallerFromContext(context.Background()); ok {
		t.Fatal("expected false when caller missing")
	}
}

func TestNewRequestID(t *testing.T) {
	t.Parallel()
	id := orm.NewRequestID()
	if !strings.HasPrefix(id, "import-orm-") || len(id) <= len("import-orm-") {
		t.Fatalf("NewRequestID = %q", id)
	}
}

func TestRegisterAndOrmCall(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	if err := orm.Register(engine); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := orm.Register(nil); err == nil {
		t.Fatal("expected Register(nil) error")
	}

	stub := engine.Ctx.Eval(`
		globalThis.$choysum = globalThis.$choysum || {};
		globalThis.$choysum.__rpc__ = async (req) => ({
			id: req.id,
			result: { Id: "created-1", Code: "X1", service: req.service, import_file: req.context && req.context.import_file },
			context: {}
		});
		true
	`)
	defer stub.Free()
	if stub.IsException() {
		t.Fatalf("stub __rpc__: %v", engine.Ctx.Exception())
	}

	promise := engine.Ctx.Eval(`$choysum.orm.call({model:"base.Country",method:"Create",args:[{Code:"X1"}],context:{import_file:false}})`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("orm.call: %v", engine.Ctx.Exception())
	}
	result := promise.Await()
	defer result.Free()
	if result.IsException() || result.IsError() {
		t.Fatalf("orm.call result: %v", result.ToError())
	}
	raw := result.JSONStringify()
	if !strings.Contains(raw, "created-1") || !strings.Contains(raw, `"import_file":true`) {
		t.Fatalf("result = %s", raw)
	}
}

func TestRegister_CreatesChoysumWhenMissing(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	cleared := engine.Ctx.Eval(`delete globalThis.$choysum; true`)
	defer cleared.Free()
	if err := orm.Register(engine); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got := engine.Ctx.Eval(`typeof $choysum.orm.call === "function"`)
	defer got.Free()
	if !got.ToBool() {
		t.Fatal("expected $choysum.orm.call")
	}
}

func TestPerformOrmCall_ValidationErrors(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	if err := orm.Register(engine); err != nil {
		t.Fatalf("Register: %v", err)
	}

	promise := engine.Ctx.Eval(`$choysum.orm.call()`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if !result.IsError() && !result.IsException() {
		t.Fatal("expected error for missing args")
	}

	promise2 := engine.Ctx.Eval(`$choysum.orm.call(null)`)
	defer promise2.Free()
	result2 := promise2.Await()
	defer result2.Free()
	if !result2.IsError() && !result2.IsException() {
		t.Fatal("expected error for null request")
	}
}

func TestEngineCaller_CallPropagatesExecContext(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	marker := context.WithValue(context.Background(), markerKey{}, "tx")
	stub := engine.Ctx.Eval(`
		globalThis.$choysum = globalThis.$choysum || {};
		globalThis.$choysum.__rpc__ = async (req) => ({ id: req.id, result: { ok: true }, context: {} });
		true
	`)
	defer stub.Free()

	caller := orm.EngineCaller{Engine: engine}
	restoreOuter := engine.SwapExecContext(context.WithValue(context.Background(), markerKey{}, "outer"))
	defer restoreOuter()

	got, err := caller.Call(marker, orm.CallRequest{
		Model:  "base.Country",
		Method: "Search",
		Args:   []any{},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["ok"] != true {
		t.Fatalf("result = %#v", got)
	}
	if engine.ExecContext().Value(markerKey{}) != "outer" {
		t.Fatalf("ExecContext not restored: %#v", engine.ExecContext().Value(markerKey{}))
	}
}

func TestEngineCaller_Errors(t *testing.T) {
	if _, err := (orm.EngineCaller{}).Call(context.Background(), orm.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected nil engine error")
	}

	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	caller := orm.EngineCaller{Engine: engine}
	if _, err := caller.Call(context.Background(), orm.CallRequest{Model: "Country", Method: "Create"}); err == nil {
		t.Fatal("expected ServiceName error")
	}

	clear := engine.Ctx.Eval(`globalThis.$choysum = {}; true`)
	defer clear.Free()
	if _, err := caller.Call(context.Background(), orm.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected missing __rpc__ error")
	}

	errStub := engine.Ctx.Eval(`
		globalThis.$choysum.__rpc__ = async () => { throw new Error("boom"); };
		true
	`)
	defer errStub.Free()
	if _, err := caller.Call(context.Background(), orm.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected boom error")
	}
}

func TestExecutorCaller(t *testing.T) {
	stub := &stubJsEngine{resp: &jsengine.JsResponse{Result: map[string]any{"Id": "1"}}}
	caller := orm.ExecutorCaller{Engine: stub}
	got, err := caller.Call(context.WithValue(context.Background(), markerKey{}, "x"), orm.CallRequest{
		Model:   "base.Country",
		Method:  "Create",
		Args:    []any{map[string]any{"Code": "A"}},
		Context: map[string]any{"import_file": false},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if stub.lastCtx.Value(markerKey{}) != "x" {
		t.Fatal("Execute did not receive Call context")
	}
	if stub.lastReq.Context["import_file"] != true {
		t.Fatalf("context = %#v", stub.lastReq.Context)
	}
	if m, ok := got.(map[string]any); !ok || m["Id"] != "1" {
		t.Fatalf("result = %#v", got)
	}

	if _, err := (orm.ExecutorCaller{}).Call(context.Background(), orm.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected nil engine error")
	}
	if _, err := (orm.ExecutorCaller{Engine: stub}).Call(context.Background(), orm.CallRequest{Model: "Country", Method: "Create"}); err == nil {
		t.Fatal("expected ServiceName error")
	}
	stub.err = context.Canceled
	if _, err := (orm.ExecutorCaller{Engine: stub}).Call(context.Background(), orm.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected execute error")
	}
	stub.err = nil
	stub.resp = nil
	got, err = (orm.ExecutorCaller{Engine: stub}).Call(context.Background(), orm.CallRequest{Model: "base.Country", Method: "Create"})
	if err != nil || got != nil {
		t.Fatalf("nil resp: got %#v err %v", got, err)
	}
}

type markerKey struct{}

type stubCaller struct{}

func (stubCaller) Call(context.Context, orm.CallRequest) (any, error) {
	return nil, nil
}

type stubJsEngine struct {
	lastCtx context.Context
	lastReq *jsengine.JsRequest
	resp    *jsengine.JsResponse
	err     error
}

func (s *stubJsEngine) Load([]*jsengine.JsScript) error { return nil }
func (s *stubJsEngine) Close() error                    { return nil }
func (s *stubJsEngine) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	s.lastCtx = ctx
	s.lastReq = req
	return s.resp, s.err
}
