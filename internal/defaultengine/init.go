// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultengine

import (
	"github.com/choysum-dev/choysum/internal/defaultengine/quickjsruntime"
	i18nbridge "github.com/choysum-dev/choysum/internal/i18n/bridge"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsbridge"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumrpc"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/vuesfc"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func defaultQuickjsReplaceableRuntimePlugins() []jsengine.RuntimePlugin {
	return []jsengine.RuntimePlugin{
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginConsole, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsengine.WithConsole(runtimeScope.Logger())}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginXid, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsengine.WithXid()}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginDB, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsbridge.WithDb(runtimeOptionsFromScope(runtimeScope).dbDialect, runtimeScope.Logger())}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginAuth, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsbridge.WithAuth(authenticator)}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginCrypto, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsengine.WithCrypto()}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginHtml, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsengine.WithHtml()}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginFS, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsruntime.WithCompilerFs()}
		}),
		jsengine.NewRuntimePluginWithProvider(quickjsengine.RuntimePluginI18n, func(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{i18nbridge.WithTerminologyProvider(scopeProvider)}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginGRPC, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsbridge.WithGrpc(runtimeScope)}
		}),
		jsengine.NewRuntimePluginWithProvider(quickjsengine.RuntimePluginBus, func(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsbridge.WithBusProvider(scopeProvider)}
		}),
		jsengine.NewRuntimePluginWithProvider(quickjsengine.RuntimePluginDocumentStorage, func(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsruntime.WithDocumentStorageProvider(scopeProvider)}
		}),
		jsengine.NewRuntimePluginWithProvider(quickjsengine.RuntimePluginModuleManagement, func(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsruntime.WithModuleManagementProvider(scopeProvider)}
		}),
	}
}

func defaultQuickjsDefaultOnlyRuntimePlugins() []jsengine.RuntimePlugin {
	return []jsengine.RuntimePlugin{
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginScriptVueSFC, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsengine.WithScript(&jsengine.JsScript{
				FileName: "scripts/vuesfc/dist/index.js",
				Content:  vuesfc.VueSfcScript,
			})}
		}),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginScriptChoysumRPC, func(runtimeScope scope.Scope, authenticator auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{quickjsengine.WithScript(&jsengine.JsScript{
				FileName: "scripts/choysumrpc/choysumrpc.js",
				Content:  choysumrpc.ChoysumRpcScript,
			})}
		}),
	}
}

func defaultQuickjsRuntimePlugins() []jsengine.RuntimePlugin {
	plugins := append([]jsengine.RuntimePlugin{}, defaultQuickjsReplaceableRuntimePlugins()...)
	plugins = append(plugins, defaultQuickjsDefaultOnlyRuntimePlugins()...)
	return plugins
}

// QuickjsRuntimePlugins keeps the mixed override input shape for callers that
// still pass one list containing both slot replacements and appended plugins.
// Deprecated: use QuickjsRuntimePluginAssembly to pass replacements and extras
// as separate inputs.
func QuickjsRuntimePlugins(overrides ...jsengine.RuntimePlugin) []jsengine.RuntimePlugin {
	return quickjsengine.MergeRuntimePluginOverrides(defaultQuickjsRuntimePlugins(), overrides...)
}

// QuickjsRuntimePluginAssembly returns the built-in QuickJS runtime plugin list
// from an explicit assembly spec containing replacements and appended extras.
func QuickjsRuntimePluginAssembly(assembly jsengine.RuntimePluginAssembly) []jsengine.RuntimePlugin {
	return quickjsengine.ApplyRuntimePluginAssembly(defaultQuickjsRuntimePlugins(), assembly)
}

// QuickjsFactoryWithRuntimePluginOverrides keeps the mixed override input shape
// for callers that still pass one list containing both replacements and extras.
// Deprecated: use QuickjsFactoryWithRuntimePluginAssembly to pass replacements
// and extras as separate inputs.
func QuickjsFactoryWithRuntimePluginOverridesProvider(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator, overrides []jsengine.RuntimePlugin, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return quickjsengine.NewFactoryWithRuntimePluginsProvider(scopeProvider, authenticator, QuickjsRuntimePlugins(overrides...), options...)
}

// QuickjsFactoryWithRuntimePluginOverrides adapts a runtime scope to the mixed plugin override input shape.
func QuickjsFactoryWithRuntimePluginOverrides(runtimeScope scope.Scope, authenticator auth.Authenticator, overrides []jsengine.RuntimePlugin, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return QuickjsFactoryWithRuntimePluginOverridesProvider(jsengine.StaticScopeProvider(runtimeScope), authenticator, overrides, options...)
}

// QuickjsFactoryWithRuntimePluginAssemblyProvider builds a QuickJS factory from the
// built-in plugin assembly plus an explicit replacement and extra plugin spec.
func QuickjsFactoryWithRuntimePluginAssemblyProvider(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator, assembly jsengine.RuntimePluginAssembly, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return quickjsengine.NewFactoryWithRuntimePluginsProvider(scopeProvider, authenticator, QuickjsRuntimePluginAssembly(assembly), options...)
}

// QuickjsFactoryWithRuntimePluginAssembly adapts a runtime scope to the explicit plugin assembly input shape.
func QuickjsFactoryWithRuntimePluginAssembly(runtimeScope scope.Scope, authenticator auth.Authenticator, assembly jsengine.RuntimePluginAssembly, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return QuickjsFactoryWithRuntimePluginAssemblyProvider(jsengine.StaticScopeProvider(runtimeScope), authenticator, assembly, options...)
}

// QuickjsFactoryProvider builds the default QuickJS engine factory for a scope provider.
func QuickjsFactoryProvider(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return QuickjsFactoryWithRuntimePluginAssemblyProvider(scopeProvider, authenticator, jsengine.RuntimePluginAssembly{}, options...)
}

// QuickjsFactory builds the default QuickJS engine factory for a fixed runtime scope.
func QuickjsFactory(runtimeScope scope.Scope, authenticator auth.Authenticator, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return QuickjsFactoryProvider(jsengine.StaticScopeProvider(runtimeScope), authenticator, options...)
}

func init() {
	jsengine.Register("quickjs", QuickjsFactoryProvider)
}
