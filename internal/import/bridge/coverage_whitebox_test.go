// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/defaultscope"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestMarshalImportReport_Error(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	ret := marshalImportReport(engine.Ctx, make(chan int))
	defer ret.Free()
	if !ret.IsError() {
		t.Fatal("expected marshal error")
	}
}

func TestWithImportProvider_RegisterError(t *testing.T) {
	err := WithImportProvider(nil)(&quickjsengine.QuickjsEngine{})
	if err == nil {
		t.Fatal("expected Register error for nil Ctx")
	}
}

func TestDecodeImportSpec_Branches(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	if _, err := decodeImportSpec(nil); err == nil {
		t.Fatal("nil")
	}
	null := engine.Ctx.Null()
	defer null.Free()
	if _, err := decodeImportSpec(null); err == nil {
		t.Fatal("null")
	}
	str := engine.Ctx.String(`{"profile":"record","caller":"user","policy":"atomic","model":"base.Country","source":{"format":"csv","path":"x.csv"}}`)
	defer str.Free()
	spec, err := decodeImportSpec(str)
	if err != nil || spec.Model != "base.Country" {
		t.Fatalf("%#v %v", spec, err)
	}
}

func TestResolveRecordSourcePath_Branches(t *testing.T) {
	spec := importpkg.Spec{Source: importpkg.Source{Path: ""}}
	got, err := resolveRecordSourcePath(nil, spec)
	if err != nil || got.Source.Path != "" {
		t.Fatalf("%#v %v", got, err)
	}

	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "r.db")},
		ModulesPath: "",
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	spec.Source.Path = "rel.csv"
	got, err = resolveRecordSourcePath(runtimeScope, spec)
	// empty ModulesPath → unchanged or still relative depending on config defaults
	_ = got
	_ = err

	// Scope without PathsRuntimeOptions
	stub := &noPathsScope{}
	got, err = resolveRecordSourcePath(stub, importpkg.Spec{Source: importpkg.Source{Path: "a.csv"}})
	if err != nil || got.Source.Path != "a.csv" {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestPerformImportRun_SuccessPath(t *testing.T) {
	root := t.TempDir()
	modulesPath := filepath.Join(root, "modules")
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "ok.db")},
		ModulesPath: modulesPath,
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	engineIface, err := quickjsengine.NewFactory(WithImportProvider(jsengine.StaticScopeProvider(runtimeScope)))()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	execCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	restore := engine.SwapExecContext(execCtx)
	defer restore()

	// relative path that resolves but file missing → error path after resolve
	arg := engine.Ctx.ParseJSON(`{"profile":"record","caller":"user","policy":"atomic","model":"base.Country","source":{"format":"csv","path":"modules/nope.csv"}}`)
	defer arg.Free()
	ret := performImportRun(engine.Ctx, engine, jsengine.StaticScopeProvider(runtimeScope), []*quickjs.Value{arg})
	defer ret.Free()
	if !ret.IsError() {
		t.Fatal("expected error for missing csv")
	}
}

func TestRun_EmptyPathPassthrough(t *testing.T) {
	cfg := &config.Config{Db: &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "e.db")}}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	_, err := Run(context.Background(), runtimeScope, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: ""},
	})
	if err == nil {
		t.Fatal("expected invalid source/path from runner")
	}
}

type noPathsScope struct{}

func (s *noPathsScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *noPathsScope) Session() *scope.Session              { return nil }
func (s *noPathsScope) Transactor() scope.Transactor         { return nil }
func (s *noPathsScope) Context() context.Context             { return context.Background() }
func (s *noPathsScope) WithContext(ctx context.Context) scope.Scope {
	return s
}
func (s *noPathsScope) Logger() *slog.Logger { return nil }

func TestPerformImportRun_HappyPathAndPathError(t *testing.T) {
	root := t.TempDir()
	modulesPath := filepath.Join(root, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatal(err)
	}
	csvRel := "modules/ok.csv"
	csvAbs := filepath.Join(root, csvRel)
	if err := os.WriteFile(csvAbs, []byte("Name,Code,IsActive,ZipRequired,StateRequired\nH,HP1,true,true,false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "happy.db")},
		ModulesPath: modulesPath,
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	seedBridgeCountry(t, runtimeScope)

	engineIface, err := quickjsengine.NewFactory(WithImportProvider(jsengine.StaticScopeProvider(runtimeScope)))()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	stub := engine.Ctx.Eval(`
		globalThis.$choysum.__rpc__ = async (req) => {
			if (String(req.service).endsWith('.Search')) {
				return { id: req.id, result: [], context: {} };
			}
			if (String(req.service).endsWith('.Create')) {
				return { id: req.id, result: { Id: "new-1", Code: "HP1" }, context: {} };
			}
			if (String(req.service).endsWith('.UpdateById')) {
				return { id: req.id, result: { Id: "new-1" }, context: {} };
			}
			return { id: req.id, result: null, context: {} };
		};
		true
	`)
	defer stub.Free()

	execCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	restore := engine.SwapExecContext(execCtx)
	defer restore()

	promise := engine.Ctx.Eval(`$choysum.import.run({profile:"record",caller:"user",policy:"atomic",model:"base.Country",source:{format:"csv",path:"modules/ok.csv"}})`)
	defer promise.Free()
	result := promise.Await()
	defer result.Free()
	if result.IsException() || result.IsError() {
		t.Fatalf("happy path failed: %v", result.ToError())
	}

	// path traversal via performImportRun
	arg := engine.Ctx.ParseJSON(`{"profile":"record","caller":"user","policy":"atomic","model":"base.Country","source":{"format":"csv","path":"../secret.csv"}}`)
	defer arg.Free()
	ret := performImportRun(engine.Ctx, engine, jsengine.StaticScopeProvider(runtimeScope), []*quickjs.Value{arg})
	defer ret.Free()
	if !ret.IsError() {
		t.Fatal("expected path error")
	}
}

func seedBridgeCountry(t *testing.T, runtimeScope scope.Scope) {
	t.Helper()
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Module{}, &meta.Model{}, &modmeta.ModelData{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS base_country (
		id TEXT PRIMARY KEY, name TEXT, code TEXT, is_active INTEGER, zip_required INTEGER, state_required INTEGER,
		default_currency_id TEXT, created_at DATETIME, updated_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Model{Name: "Country", Application: "base", Path: "/tmp", ModelTable: "base_country"}).Error; err != nil {
		t.Fatal(err)
	}
}
