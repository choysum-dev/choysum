// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package caller_test

import (
	"context"
	"strings"
	"testing"

	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestServiceName(t *testing.T) {
	t.Parallel()
	got, err := importcaller.ServiceName("base.Country", "Create")
	if err != nil || got != "base.Country.Create" {
		t.Fatalf("ServiceName = %q %v", got, err)
	}
	if _, err := importcaller.ServiceName("Country", "Create"); err == nil {
		t.Fatal("expected error for short model name")
	}
	if _, err := importcaller.ServiceName("", "Create"); err == nil {
		t.Fatal("expected error for empty model")
	}
	if _, err := importcaller.ServiceName("base.Country", "  "); err == nil {
		t.Fatal("expected error for empty method")
	}
}

func TestMergeImportContext_ReservedImportFile(t *testing.T) {
	t.Parallel()
	got := importcaller.MergeImportContext(map[string]any{
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
	ctx := importcaller.ContextWithCaller(nil, caller)
	got, ok := importcaller.CallerFromContext(ctx)
	if !ok || got != caller {
		t.Fatalf("CallerFromContext = %#v %v", got, ok)
	}
	if _, ok := importcaller.CallerFromContext(nil); ok {
		t.Fatal("expected false for nil context")
	}
	if _, ok := importcaller.CallerFromContext(context.Background()); ok {
		t.Fatal("expected false when caller missing")
	}
}

func TestNewRequestID(t *testing.T) {
	t.Parallel()
	id := importcaller.NewRequestID()
	if !strings.HasPrefix(id, "import-caller-") || len(id) <= len("import-caller-") {
		t.Fatalf("NewRequestID = %q", id)
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

	engCaller := importcaller.EngineCaller{Engine: engine}
	restoreOuter := engine.SwapExecContext(context.WithValue(context.Background(), markerKey{}, "outer"))
	defer restoreOuter()

	got, err := engCaller.Call(marker, importcaller.CallRequest{
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
	if _, err := (importcaller.EngineCaller{}).Call(context.Background(), importcaller.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected nil engine error")
	}

	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	engCaller := importcaller.EngineCaller{Engine: engine}
	if _, err := engCaller.Call(context.Background(), importcaller.CallRequest{Model: "Country", Method: "Create"}); err == nil {
		t.Fatal("expected ServiceName error")
	}

	clear := engine.Ctx.Eval(`globalThis.$choysum = {}; true`)
	defer clear.Free()
	if _, err := engCaller.Call(context.Background(), importcaller.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected missing __rpc__ error")
	}

	errStub := engine.Ctx.Eval(`
		globalThis.$choysum.__rpc__ = async () => { throw new Error("boom"); };
		true
	`)
	defer errStub.Free()
	if _, err := engCaller.Call(context.Background(), importcaller.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected boom error")
	}
}

func TestExecutorCaller(t *testing.T) {
	stub := &stubJsEngine{resp: &jsengine.JsResponse{Result: map[string]any{"Id": "1"}}}
	execCaller := importcaller.ExecutorCaller{Engine: stub}
	got, err := execCaller.Call(context.WithValue(context.Background(), markerKey{}, "x"), importcaller.CallRequest{
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

	if _, err := (importcaller.ExecutorCaller{}).Call(context.Background(), importcaller.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected nil engine error")
	}
	if _, err := (importcaller.ExecutorCaller{Engine: stub}).Call(context.Background(), importcaller.CallRequest{Model: "Country", Method: "Create"}); err == nil {
		t.Fatal("expected ServiceName error")
	}
	stub.err = context.Canceled
	if _, err := (importcaller.ExecutorCaller{Engine: stub}).Call(context.Background(), importcaller.CallRequest{Model: "base.Country", Method: "Create"}); err == nil {
		t.Fatal("expected execute error")
	}
	stub.err = nil
	stub.resp = nil
	got, err = (importcaller.ExecutorCaller{Engine: stub}).Call(context.Background(), importcaller.CallRequest{Model: "base.Country", Method: "Create"})
	if err != nil || got != nil {
		t.Fatalf("nil resp: got %#v err %v", got, err)
	}
}

type markerKey struct{}

type stubCaller struct{}

func (stubCaller) Call(context.Context, importcaller.CallRequest) (any, error) {
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
