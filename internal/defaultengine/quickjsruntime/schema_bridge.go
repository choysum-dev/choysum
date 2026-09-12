// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// WithSchemaDDL installs $choysum.schema.* helpers for upgrade migration scripts.
func WithSchemaDDL(dialect string) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		engine, ok := jsEngine.(*quickjsengine.QuickjsEngine)
		if !ok {
			return fmt.Errorf("WithSchemaDDL requires *quickjsengine.QuickjsEngine, got %T", jsEngine)
		}
		globalsObj := engine.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = engine.Ctx.Object()
		}

		schemaObj := engine.Ctx.Object()
		schemaObj.Set("renameColumn", engine.Ctx.NewFunction(schemaSyncFactory(engine, dialect, "renameColumn")))
		schemaObj.Set("dropColumn", engine.Ctx.NewFunction(schemaSyncFactory(engine, dialect, "dropColumn")))
		schemaObj.Set("dropIndex", engine.Ctx.NewFunction(schemaSyncFactory(engine, dialect, "dropIndex")))
		schemaObj.Set("dropCheck", engine.Ctx.NewFunction(schemaSyncFactory(engine, dialect, "dropCheck")))
		schemaObj.Set("dropForeignKey", engine.Ctx.NewFunction(schemaSyncFactory(engine, dialect, "dropForeignKey")))

		choysumObj.Set("schema", schemaObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

func schemaSyncFactory(engine *quickjsengine.QuickjsEngine, dialect, method string) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		opts, err := schemaHelperOpts(engine, dialect)
		if err != nil {
			return ctx.ThrowError(err)
		}
		switch method {
		case "renameColumn":
			if len(args) < 3 {
				return ctx.ThrowError(fmt.Errorf("renameColumn(table, from, to) requires 3 args"))
			}
			err = schema.RenameColumn(opts, args[0].String(), args[1].String(), args[2].String())
		case "dropColumn":
			if len(args) < 2 {
				return ctx.ThrowError(fmt.Errorf("dropColumn(table, column) requires 2 args"))
			}
			err = schema.DropColumn(opts, args[0].String(), args[1].String())
		case "dropIndex":
			if len(args) < 2 {
				return ctx.ThrowError(fmt.Errorf("dropIndex(table, name) requires 2 args"))
			}
			err = schema.DropIndex(opts, args[0].String(), args[1].String())
		case "dropCheck":
			if len(args) < 2 {
				return ctx.ThrowError(fmt.Errorf("dropCheck(table, name) requires 2 args"))
			}
			err = schema.DropCheck(opts, args[0].String(), args[1].String())
		case "dropForeignKey":
			if len(args) < 2 {
				return ctx.ThrowError(fmt.Errorf("dropForeignKey(table, name) requires 2 args"))
			}
			err = schema.DropForeignKey(opts, args[0].String(), args[1].String())
		default:
			err = fmt.Errorf("unknown schema helper %s", method)
		}
		if err != nil {
			return ctx.ThrowError(err)
		}
		return ctx.Undefined()
	}
}

func schemaHelperOpts(engine *quickjsengine.QuickjsEngine, dialect string) (schema.HelperOptions, error) {
	execCtx := engine.ExecContext()
	session, ok := scope.SessionFromContext(execCtx)
	if !ok || session == nil || session.DB == nil {
		return schema.HelperOptions{}, fmt.Errorf("$choysum.schema: no db session on exec context")
	}
	bag := schema.IntentBagFromContext(execCtx)
	if bag == nil {
		return schema.HelperOptions{}, fmt.Errorf("$choysum.schema helpers require an upgrade IntentBag")
	}
	return schema.HelperOptions{DB: session.DB, Dialect: dialect, Intents: bag}, nil
}
