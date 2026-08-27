// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/defaultscope"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestMarshalExportReport_Error(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	ret := marshalExportReport(engine.Ctx, make(chan int))
	defer ret.Free()
	if !ret.IsError() {
		t.Fatal("expected marshal error")
	}
}

func TestWithExportProvider_NilEngine(t *testing.T) {
	err := WithExportProvider(nil)(&quickjsengine.QuickjsEngine{})
	if err == nil {
		t.Fatal("expected error for nil Ctx")
	}
}

func TestDecodeExportSpec_Branches(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	if _, err := decodeExportSpec(nil); err == nil {
		t.Fatal("nil")
	}
	null := engine.Ctx.Null()
	defer null.Free()
	if _, err := decodeExportSpec(null); err == nil {
		t.Fatal("null")
	}
	str := engine.Ctx.String(`{"profile":"record","caller":"user","model":"base.Country","format":"csv","mode":"template"}`)
	defer str.Free()
	spec, err := decodeExportSpec(str)
	if err != nil || spec.Model != "base.Country" {
		t.Fatalf("%#v %v", spec, err)
	}
}

func TestResolveExportScope_ProviderFallback(t *testing.T) {
	cfg := &config.Config{
		Db: &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "scope.db")},
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	got := resolveExportScope(jsengine.StaticScopeProvider(runtimeScope), context.Background())
	if got == nil {
		t.Fatal("expected scope from provider")
	}
}

func TestPerformExportRun_SuccessPath(t *testing.T) {
	cfg := &config.Config{
		Db: &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "ok.db")},
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	engineIface, err := quickjsengine.NewFactory(WithExportProvider(jsengine.StaticScopeProvider(runtimeScope)))()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	execCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	restore := engine.SwapExecContext(execCtx)
	defer restore()

	arg := engine.Ctx.ParseJSON(`{"profile":"record","caller":"user","model":"base.Country","format":"csv","mode":"template"}`)
	defer arg.Free()
	ret := performExportRun(engine.Ctx, engine, jsengine.StaticScopeProvider(runtimeScope), []*quickjs.Value{arg})
	defer ret.Free()
	if ret.IsError() {
		t.Fatalf("happy path failed: %s", ret.String())
	}
}

func TestPerformExportRun_HappyPromiseResolve(t *testing.T) {
	cfg := &config.Config{
		Db: &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "promise.db")},
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	engineIface, err := quickjsengine.NewFactory(WithExportProvider(jsengine.StaticScopeProvider(runtimeScope)))()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	execCtx := importcaller.ContextWithCaller(
		scope.ContextWithScope(context.Background(), runtimeScope),
		&templateExportCaller{},
	)
	restore := engine.SwapExecContext(execCtx)
	defer restore()

	promise := engine.Ctx.Eval(`$choysum.export.run('{"profile":"record","caller":"user","model":"base.Country","format":"csv","mode":"template"}')`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if result.IsException() || result.IsError() {
		t.Fatalf("promise resolve failed: %s", result.String())
	}
}

type templateExportCaller struct{}

func (templateExportCaller) Call(context.Context, importcaller.CallRequest) (any, error) {
	return nil, nil
}
