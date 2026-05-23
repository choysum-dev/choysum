// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"os"
	"path/filepath"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func compilerFileExistsFunc(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	file := args[0].String()
	if _, err := os.Stat(file); err != nil {
		return ctx.Bool(false)
	}
	return ctx.Bool(true)
}

func compilerReadFileFunc(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	file := args[0].String()
	data, err := os.ReadFile(file)
	if err != nil {
		return ctx.ThrowError(err)
	}
	return ctx.String(string(data))
}

func compilerRealpathFunc(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	file := args[0].String()
	resolved, err := filepath.EvalSymlinks(file)
	if err != nil {
		return ctx.ThrowError(err)
	}
	realpath, _ := filepath.Abs(resolved)
	return ctx.String(realpath)
}

// WithCompilerFs exposes filesystem helpers used by the QuickJS compiler bridge.
func WithCompilerFs() jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()
		compilerFsObj := jse.Ctx.Object()
		compilerFsObj.Set("fileExists", jse.Ctx.Function(compilerFileExistsFunc))
		compilerFsObj.Set("readFile", jse.Ctx.Function(compilerReadFileFunc))
		compilerFsObj.Set("realpath", jse.Ctx.Function(compilerRealpathFunc))
		globalsObj.Set("compilerFs", compilerFsObj)
		return nil
	}
}
