// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esbplugins

import (
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
)

// EsbPluginOptions configures a plugin that implements EsbPlugin.
type EsbPluginOptions func(plugin EsbPlugin)

// EsbPlugin defines the shared lifecycle for Choysum esbuild plugins.
type EsbPlugin interface {
	DefinePlugins(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.Module, options ...EsbPluginOptions) []api.Plugin
	GetParserResults() ([]*parser.ParserResult, error)
	SetParserResults(parserResults []*parser.ParserResult) error
}

type entryPointImportsSetter interface {
	SetEntryPointImports(imports []string)
}

type indexHtmlOutFileSetter interface {
	SetIndexHtmlOutFile(outFile string)
}

// WithEntryPointImports injects imports into plugins that support entry-point imports.
func WithEntryPointImports(imports []string) EsbPluginOptions {
	cloned := append([]string(nil), imports...)
	return func(plugin EsbPlugin) {
		if plugin == nil {
			return
		}
		if setter, ok := plugin.(entryPointImportsSetter); ok {
			setter.SetEntryPointImports(cloned)
		}
	}
}

// WithIndexHtmlOutFile overrides the generated index.html output path when supported.
func WithIndexHtmlOutFile(outFile string) EsbPluginOptions {
	return func(plugin EsbPlugin) {
		if plugin == nil {
			return
		}
		if setter, ok := plugin.(indexHtmlOutFileSetter); ok {
			setter.SetIndexHtmlOutFile(outFile)
		}
	}
}
