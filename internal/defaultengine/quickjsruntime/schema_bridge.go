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
			if err = requireStringArgs(args, 3, "renameColumn(table, from, to)"); err != nil {
				return ctx.ThrowError(err)
			}
			err = schema.RenameColumn(opts, args[0].String(), args[1].String(), args[2].String())
		case "dropColumn":
			if err = requireStringArgs(args, 2, "dropColumn(table, column)"); err != nil {
				return ctx.ThrowError(err)
			}
			err = schema.DropColumn(opts, args[0].String(), args[1].String())
		case "dropIndex":
			if err = requireStringArgs(args, 2, "dropIndex(table, name)"); err != nil {
				return ctx.ThrowError(err)
			}
			err = schema.DropIndex(opts, args[0].String(), args[1].String())
		case "dropCheck":
			if err = requireStringArgs(args, 2, "dropCheck(table, name)"); err != nil {
				return ctx.ThrowError(err)
			}
			err = schema.DropCheck(opts, args[0].String(), args[1].String())
		case "dropForeignKey":
			if err = requireStringArgs(args, 2, "dropForeignKey(table, name)"); err != nil {
				return ctx.ThrowError(err)
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

func requireStringArgs(args []*quickjs.Value, n int, label string) error {
	if len(args) < n {
		return fmt.Errorf("%s requires %d args", label, n)
	}
	for i := 0; i < n; i++ {
		if args[i] == nil || !args[i].IsString() {
			return fmt.Errorf("%s: argument %d must be a string", label, i+1)
		}
	}
	return nil
}

func schemaHelperOpts(engine *quickjsengine.QuickjsEngine, dialect string) (schema.HelperOptions, error) {
	if engine == nil {
		return schema.HelperOptions{}, fmt.Errorf("$choysum.schema: engine is nil")
	}
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
