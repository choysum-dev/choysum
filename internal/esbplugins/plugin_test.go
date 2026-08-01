// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esbplugins

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
)

func TestWithEntryPointImports(t *testing.T) {
	t.Run("clones input and applies to setter", func(t *testing.T) {
		plugin := &fakePlugin{}
		imports := []string{"./boot", "./routes"}

		opt := WithEntryPointImports(imports)
		imports[0] = "./changed"
		opt(plugin)

		if len(plugin.entryPointImports) != 2 || plugin.entryPointImports[0] != "./boot" || plugin.entryPointImports[1] != "./routes" {
			t.Fatalf("unexpected entry point imports: %#v", plugin.entryPointImports)
		}
	})

	t.Run("ignores nil plugin and unsupported setter", func(t *testing.T) {
		WithEntryPointImports([]string{"./boot"})(nil)
		WithEntryPointImports([]string{"./boot"})(fakePluginWithoutSetters{})
	})
}

func TestWithIndexHtmlOutFile(t *testing.T) {
	t.Run("applies out file to setter", func(t *testing.T) {
		plugin := &fakePlugin{}

		WithIndexHtmlOutFile("staging/index.html")(plugin)

		if plugin.indexHtmlOutFile != "staging/index.html" {
			t.Fatalf("unexpected index html out file: %q", plugin.indexHtmlOutFile)
		}
	})

	t.Run("ignores nil plugin and unsupported setter", func(t *testing.T) {
		WithIndexHtmlOutFile("staging/index.html")(nil)
		WithIndexHtmlOutFile("staging/index.html")(fakePluginWithoutSetters{})
	})
}

type fakePlugin struct {
	entryPointImports []string
	indexHtmlOutFile  string
}

func (p *fakePlugin) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, _ ...EsbPluginOptions) []api.Plugin {
	return nil
}

func (p *fakePlugin) GetParserResults() ([]*parser.ParserResult, error) {
	return nil, nil
}

func (p *fakePlugin) SetParserResults(_ []*parser.ParserResult) error {
	return nil
}

func (p *fakePlugin) SetEntryPointImports(imports []string) {
	p.entryPointImports = imports
}

func (p *fakePlugin) SetIndexHtmlOutFile(outFile string) {
	p.indexHtmlOutFile = outFile
}

type fakePluginWithoutSetters struct{}

func (fakePluginWithoutSetters) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, _ ...EsbPluginOptions) []api.Plugin {
	return nil
}

func (fakePluginWithoutSetters) GetParserResults() ([]*parser.ParserResult, error) {
	return nil, nil
}

func (fakePluginWithoutSetters) SetParserResults(_ []*parser.ParserResult) error {
	return nil
}
