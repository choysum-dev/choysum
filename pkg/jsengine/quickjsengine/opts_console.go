// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"context"
	"log/slog"
	"strings"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
)

const jsConsoleMessage = "js console message"

func logConsoleMessage(logger *slog.Logger, level slog.Level, consoleLevel string, consoleText string, extra ...any) {
	if logger == nil {
		logger = slog.Default()
	}
	attrs := []any{
		"emitter", "js_console",
		"console_level", consoleLevel,
		"passthrough", true,
		"console_text", consoleText,
	}
	attrs = append(attrs, extra...)
	logger.Log(context.Background(), level, jsConsoleMessage, attrs...)
}

func consoleFunc(logger *slog.Logger, fnType string) (Fn func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value) {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		fn := ctx.Eval(`(obj) => {
				return JSON.stringify(obj, (key, value) => {
					if(value instanceof Error) {
						const serialized = {};
						for (const prop of Object.getOwnPropertyNames(value)) {
							serialized[prop] = value[prop];
						}
						if(serialized.name == null || serialized.name === '') {
							serialized.name = value.name;
						}
						if(serialized.message == null || serialized.message === '') {
							serialized.message = value.message;
						}
						if(serialized.stack == null || serialized.stack === '') {
							serialized.stack = value.stack;
						}
						return serialized;
					}
					if(value instanceof Map) {
						return {
						  dataType: 'Map',
						  value: Array.from(value.entries()), // or with spread: value: [...value]
						};
					  } else {
						return value;
					  }
				});
			}`)
		defer fn.Free()

		logs := make([]string, 0)
		for _, arg := range args {
			var log string
			if arg.IsObject() {
				jsonStr := ctx.Invoke(fn, ctx.Null(), arg)
				defer jsonStr.Free()
				log = jsonStr.String()
			} else {
				log = arg.String()
			}
			logs = append(logs, log)
		}
		joined := strings.Join(logs, " ")

		switch fnType {
		case "trace":
			logConsoleMessage(logger, slog.LevelDebug, fnType, joined)
		case "debug":
			logConsoleMessage(logger, slog.LevelDebug, fnType, joined)
		case "info":
			logConsoleMessage(logger, slog.LevelInfo, fnType, joined)
		case "log":
			logConsoleMessage(logger, slog.LevelInfo, fnType, joined)
		case "warn":
			logConsoleMessage(logger, slog.LevelWarn, fnType, joined)
		case "error":
			if len(args) > 0 && args[len(args)-1].IsError() {
				err := args[len(args)-1].Error()
				logConsoleMessage(logger, slog.LevelError, fnType, joined, "error", err)
			} else {
				logConsoleMessage(logger, slog.LevelError, fnType, joined)
			}
		}
		return ctx.Null()
	}
}

func WithConsole(logger *slog.Logger) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*QuickjsEngine)
		ctx := jse.Ctx
		globalsObj := ctx.Globals()
		consoleObj := globalsObj.Get("console")
		if consoleObj.IsUndefined() {
			consoleObj = ctx.Object()
		}

		consoleObj.Set("trace", ctx.Function(consoleFunc(logger, "trace")))
		consoleObj.Set("debug", ctx.Function(consoleFunc(logger, "debug")))
		consoleObj.Set("info", ctx.Function(consoleFunc(logger, "info")))
		consoleObj.Set("log", ctx.Function(consoleFunc(logger, "log")))
		consoleObj.Set("warn", ctx.Function(consoleFunc(logger, "warn")))
		consoleObj.Set("error", ctx.Function(consoleFunc(logger, "error")))
		globalsObj.Set("console", consoleObj)
		return nil
	}
}
