// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type fakeJsEngine struct{}

func (fakeJsEngine) Load([]*jsengine.JsScript) error { return nil }
func (fakeJsEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (fakeJsEngine) Close() error { return nil }

func TestWithSchemaDDL_RecreatesNonObjectChoysum(t *testing.T) {
	engine := newTestQuickjsEngine(t)
	globals := engine.Ctx.Globals()
	globals.Set("$choysum", engine.Ctx.String("not-an-object"))
	if err := WithSchemaDDL("sqlite")(engine); err != nil {
		t.Fatal(err)
	}
	v := engine.Ctx.Eval(`typeof $choysum.schema.renameColumn`)
	defer v.Free()
	if v.IsException() || v.String() != "function" {
		t.Fatalf("schema helpers missing: %v %s", engine.Ctx.Exception(), v.String())
	}
}

func TestWithSchemaDDL_RejectsNonQuickjsEngine(t *testing.T) {
	err := WithSchemaDDL("sqlite")(fakeJsEngine{})
	if err == nil || !strings.Contains(err.Error(), "QuickjsEngine") {
		t.Fatalf("got %v", err)
	}
}

func TestWithSchemaDDL_Helpers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{Db: &config.DbConfig{
		Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "schema-bridge.db"),
		MaxIdleConns: 2, MaxOpenConns: 4, ConnMaxLifetime: 30,
	}}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), logger)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE bridge_tbl (old_code text, keep text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_bridge_tbl_keep ON bridge_tbl (keep)`).Error; err != nil {
		t.Fatal(err)
	}

	engine := newTestQuickjsEngine(t, WithSchemaDDL("sqlite"))
	bag := schema.NewMemoryIntentBag()

	// Missing IntentBag / session
	if v := engine.Ctx.Eval(`$choysum.schema.dropColumn('bridge_tbl','old_code')`); !v.IsException() {
		t.Fatal("expected missing intent/session error")
	} else {
		_ = engine.Ctx.Exception()
		v.Free()
	}

	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		execCtx := schema.ContextWithIntentBag(tx.Context(), bag)
		restore := engine.SwapExecContext(execCtx)
		defer restore()

		call := func(expr string) {
			t.Helper()
			v := engine.Ctx.Eval(expr)
			defer v.Free()
			if v.IsException() {
				t.Fatalf("%s: %v", expr, engine.Ctx.Exception())
			}
		}
		fail := func(expr string) {
			t.Helper()
			v := engine.Ctx.Eval(expr)
			defer v.Free()
			if !v.IsException() {
				t.Fatalf("expected error for %s", expr)
			}
			_ = engine.Ctx.Exception()
		}

		fail(`$choysum.schema.renameColumn('bridge_tbl')`)
		fail(`$choysum.schema.dropColumn('bridge_tbl')`)
		fail(`$choysum.schema.dropIndex('bridge_tbl')`)
		fail(`$choysum.schema.dropCheck('bridge_tbl')`)
		fail(`$choysum.schema.dropForeignKey('bridge_tbl')`)
		fail(`$choysum.schema.dropColumn('bridge_tbl', null)`)
		fail(`$choysum.schema.renameColumn('bridge_tbl', 1, 'new_code')`)

		call(`$choysum.schema.renameColumn('bridge_tbl','old_code','new_code')`)
		call(`$choysum.schema.dropIndex('bridge_tbl','idx_bridge_tbl_keep')`)
		call(`$choysum.schema.dropColumn('bridge_tbl','keep')`)
		if err := txScope.Session().Exec(`CREATE TABLE bridge_idx (code text)`).Error; err != nil {
			return err
		}
		if err := txScope.Session().Exec(`CREATE INDEX idx_bridge_idx_code ON bridge_idx (code)`).Error; err != nil {
			return err
		}
		call(`$choysum.schema.dropIndex('bridge_idx','idx_bridge_idx_code')`)
		fail(`$choysum.schema.dropCheck('bridge_idx','chk_bridge_idx_code')`)
		fail(`$choysum.schema.dropForeignKey('bridge_idx','fk_bridge')`)

		// Session present but IntentBag missing
		restore2 := engine.SwapExecContext(tx.Context())
		fail(`$choysum.schema.dropColumn('bridge_idx','code')`)
		restore2()

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bag.List()) < 3 {
		t.Fatalf("intents = %#v", bag.List())
	}
}

func TestSchemaHelperOpts_NilEngine(t *testing.T) {
	_, err := schemaHelperOpts(nil, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "engine is nil") {
		t.Fatalf("got %v", err)
	}
}

func TestRequireStringArgs_NilElem(t *testing.T) {
	err := requireStringArgs([]*quickjs.Value{nil}, 1, "dropColumn(table, column)")
	if err == nil || !strings.Contains(err.Error(), "must be a string") {
		t.Fatalf("got %v", err)
	}
}

func TestSchemaHelperOpts_UnknownMethod(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{Db: &config.DbConfig{
		Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "schema-unknown.db"),
		MaxIdleConns: 1, MaxOpenConns: 2, ConnMaxLifetime: 30,
	}}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), logger)
	engine := newTestQuickjsEngine(t, WithSchemaDDL("sqlite"))
	bag := schema.NewMemoryIntentBag()
	_ = runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		restore := engine.SwapExecContext(schema.ContextWithIntentBag(tx.Context(), bag))
		defer restore()
		fn := schemaSyncFactory(engine, "sqlite", "nope")
		v := fn(engine.Ctx, nil, nil)
		if v == nil || !v.IsException() {
			t.Fatal("expected unknown helper error")
		}
		_ = engine.Ctx.Exception()
		v.Free()
		return nil
	})
}
