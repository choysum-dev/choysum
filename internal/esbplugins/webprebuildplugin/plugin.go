// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webprebuildplugin

import (
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/esbplugins/webplugin"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
)

// WebPrebuildPlugin precompiles Vue files before the main web build runs.
type WebPrebuildPlugin struct {
	*webplugin.WebPlugin
}

func (p *WebPrebuildPlugin) prebuildVuePlugin() api.Plugin {
	return api.Plugin{
		Name: "choysum-web-vue-prebuild",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.vue$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				p.Mu.Lock()
				defer p.Mu.Unlock()
				source, err := p.ReadNormalizedTextFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}

				pathAlias, err := parser.ParseTsconfigPathAlias(build.InitialOptions)
				if err != nil {
					return api.OnLoadResult{}, err
				}

				// Resolve the path alias for the Vue file
				source = vueplugin.ResolveVueStylePath(source, args.Path, pathAlias)

				parserResult, err := p.Parser.Parse(pathAlias, args.Path, source)
				if err != nil {
					return api.OnLoadResult{}, err
				}

				if parserResult != nil {
					p.Wg.Add(1)
					go func() {
						defer p.Wg.Done()
						p.ParserResultChan <- parserResult
					}()
				}

				rawScriptContents := ""
				if parserResult.RawScriptNode != nil {
					rawScriptContents = htmlquery.InnerText(parserResult.RawScriptNode)
				}
				if parserResult.RawScriptSetupNode != nil {
					rawScriptContents += "\n" + htmlquery.InnerText(parserResult.RawScriptSetupNode)
				}

				loader := api.LoaderJS
				if parserResult.RawScriptNode != nil {
					for _, attr := range parserResult.RawScriptNode.Attr {
						if attr.Key == "lang" && attr.Val == "ts" {
							loader = api.LoaderTS
							break
						}
					}
				}
				if parserResult.RawScriptSetupNode != nil {
					for _, attr := range parserResult.RawScriptSetupNode.Attr {
						if attr.Key == "lang" && attr.Val == "ts" {
							loader = api.LoaderTS
							break
						}
					}
				}

				// Add default export if not exists in script content to avoid esbuild error
				if !strings.Contains(rawScriptContents, "export default") {
					rawScriptContents += "\nexport default {}"
				}

				return api.OnLoadResult{
					Contents: &rawScriptContents,
					Loader:   loader,
				}, nil

			})

		},
	}
}

// DefinePlugins returns the prebuild and TypeScript plugins for the current runtime state.
func (p *WebPrebuildPlugin) DefinePlugins(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	for _, opt := range options {
		if opt != nil {
			opt(p)
		}
	}
	p.BindRuntimeState(runtimeScope, module)

	return []api.Plugin{
		p.prebuildVuePlugin(),
		p.TsPlugin(),
	}
}

// NewWebPrebuildPlugin creates a web prebuild plugin for a module entry point.
func NewWebPrebuildPlugin(runtimeScope scope.Scope, module *meta.IrModule, entryPoint string, opts ...func(*WebPrebuildPlugin)) *WebPrebuildPlugin {
	esbPlugin := webplugin.NewWebPlugin(runtimeScope, module, entryPoint)
	wp, ok := esbPlugin.(*webplugin.WebPlugin)
	if !ok {
		runtimeScope.Logger().Error("web prebuild plugin initialization failed", "reason", "web_plugin_assertion_failed")
		return nil
	}

	p := &WebPrebuildPlugin{
		WebPlugin: wp,
	}

	for _, opt := range opts {
		opt(p)
	}
	p.BindRuntimeState(runtimeScope, module)

	return p
}

// WithParser overrides the parser used by WebPrebuildPlugin.
func WithParser(parser parser.Parser) func(*WebPrebuildPlugin) {
	return func(p *WebPrebuildPlugin) {
		p.UseParser(parser)
	}
}
