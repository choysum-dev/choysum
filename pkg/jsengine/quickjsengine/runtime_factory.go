// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// NewFactoryWithRuntimePluginsProvider builds a QuickJS factory from explicit
// runtime plugins plus a provider that can resolve runtime-bound scopes.
func NewFactoryWithRuntimePluginsProvider(scopeProvider jsengine.ScopeProvider, authenticator auth.Authenticator, plugins []jsengine.RuntimePlugin, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	allOptions := append([]jsengine.JsEngineOption{}, options...)
	allOptions = append(allOptions, jsengine.BuildRuntimePluginOptionsWithProvider(scopeProvider, authenticator, plugins...)...)
	return NewFactory(allOptions...)
}

// NewFactoryWithRuntimePlugins builds a QuickJS factory from explicit runtime plugins
// plus any direct engine options passed by the caller.
func NewFactoryWithRuntimePlugins(runtimeScope scope.Scope, authenticator auth.Authenticator, plugins []jsengine.RuntimePlugin, options ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
	return NewFactoryWithRuntimePluginsProvider(jsengine.StaticScopeProvider(runtimeScope), authenticator, plugins, options...)
}
