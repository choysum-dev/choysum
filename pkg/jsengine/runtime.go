// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsengine

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// RuntimePlugin contributes one named runtime slice into an engine factory.
// Public assembly should usually flow through RuntimePluginAssembly, with the
// function helpers kept as lower-level building blocks for callers that already
// manage plugin slices directly.
type RuntimePlugin interface {
	Name() string
	Options(runtimeScope scope.Scope, authenticator auth.Authenticator) []JsEngineOption
}

type runtimePluginWithProvider interface {
	RuntimePlugin
	OptionsWithProvider(scopeProvider ScopeProvider, authenticator auth.Authenticator) []JsEngineOption
}

// RuntimePluginAssembly keeps base slot replacements separate from appended
// plugins so API boundaries can describe assembly intent explicitly.
type RuntimePluginAssembly struct {
	// Replacements target existing base plugin slots by name.
	Replacements []RuntimePlugin
	// Extras are appended after the assembled base plugin list.
	Extras []RuntimePlugin
}

// Apply assembles the provided base plugins with explicit replacements and
// extras from the assembly spec. This is the preferred package-boundary entry
// point once a RuntimePluginAssembly has been constructed.
func (assembly RuntimePluginAssembly) Apply(base []RuntimePlugin) []RuntimePlugin {
	return AssembleRuntimePlugins(base, assembly.Replacements, assembly.Extras)
}

type runtimePlugin struct {
	name            string
	options         func(runtimeScope scope.Scope, authenticator auth.Authenticator) []JsEngineOption
	providerOptions func(scopeProvider ScopeProvider, authenticator auth.Authenticator) []JsEngineOption
}

// NewRuntimePlugin creates a named runtime plugin contribution.
func NewRuntimePlugin(name string, options func(runtimeScope scope.Scope, authenticator auth.Authenticator) []JsEngineOption) RuntimePlugin {
	return runtimePlugin{
		name:    strings.TrimSpace(name),
		options: options,
	}
}

// NewRuntimePluginWithProvider creates a named runtime plugin contribution that
// can resolve a runtime-bound scope from the current execution context.
func NewRuntimePluginWithProvider(name string, options func(scopeProvider ScopeProvider, authenticator auth.Authenticator) []JsEngineOption) RuntimePlugin {
	return runtimePlugin{
		name:            strings.TrimSpace(name),
		providerOptions: options,
	}
}

func (c runtimePlugin) Name() string {
	return c.name
}

func (c runtimePlugin) Options(runtimeScope scope.Scope, authenticator auth.Authenticator) []JsEngineOption {
	if c.providerOptions != nil {
		return c.providerOptions(StaticScopeProvider(runtimeScope), authenticator)
	}
	if c.options == nil {
		return nil
	}
	return c.options(runtimeScope, authenticator)
}

func (c runtimePlugin) OptionsWithProvider(scopeProvider ScopeProvider, authenticator auth.Authenticator) []JsEngineOption {
	if c.providerOptions != nil {
		return c.providerOptions(scopeProvider, authenticator)
	}
	return c.Options(ResolveScope(scopeProvider, nil), authenticator)
}

func runtimePluginNameIndex(plugins []RuntimePlugin) map[string]int {
	indexByName := make(map[string]int, len(plugins))
	for index, plugin := range plugins {
		if plugin == nil {
			continue
		}
		name := strings.TrimSpace(plugin.Name())
		if name == "" {
			continue
		}
		if _, exists := indexByName[name]; exists {
			continue
		}
		indexByName[name] = index
	}
	return indexByName
}

// ReplaceRuntimePlugins is a lower-level helper that keeps base order stable and
// replaces only base plugins with the same non-empty name. Unknown names are
// ignored so replacement intent stays separate from appended plugin intent.
func ReplaceRuntimePlugins(base []RuntimePlugin, replacements ...RuntimePlugin) []RuntimePlugin {
	replaced := MergeRuntimePlugins(nil, base...)
	indexByName := runtimePluginNameIndex(replaced)

	for _, plugin := range replacements {
		if plugin == nil {
			continue
		}
		name := strings.TrimSpace(plugin.Name())
		if idx, ok := indexByName[name]; ok {
			replaced[idx] = plugin
		}
	}

	return replaced
}

// AssembleRuntimePlugins is a lower-level helper that first applies explicit
// base replacements and then appends extra plugins after the assembled base
// slice. Prefer RuntimePluginAssembly.Apply at package boundaries.
func AssembleRuntimePlugins(base []RuntimePlugin, replacements []RuntimePlugin, extras []RuntimePlugin) []RuntimePlugin {
	assembled := ReplaceRuntimePlugins(base, replacements...)
	for _, plugin := range extras {
		if plugin == nil {
			continue
		}
		assembled = append(assembled, plugin)
	}
	return assembled
}

// BuildRuntimePluginOptions materializes runtime plugin contributions into the
// engine options consumed by a concrete engine factory.
func BuildRuntimePluginOptions(runtimeScope scope.Scope, authenticator auth.Authenticator, plugins ...RuntimePlugin) []JsEngineOption {
	return BuildRuntimePluginOptionsWithProvider(StaticScopeProvider(runtimeScope), authenticator, plugins...)
}

// BuildRuntimePluginOptionsWithProvider materializes runtime plugin
// contributions using a scope provider that can rebind to execution contexts
// at bridge invocation time.
func BuildRuntimePluginOptionsWithProvider(scopeProvider ScopeProvider, authenticator auth.Authenticator, plugins ...RuntimePlugin) []JsEngineOption {
	options := make([]JsEngineOption, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin == nil {
			continue
		}
		if withProvider, ok := plugin.(runtimePluginWithProvider); ok {
			options = append(options, withProvider.OptionsWithProvider(scopeProvider, authenticator)...)
			continue
		}
		options = append(options, plugin.Options(ResolveScope(scopeProvider, nil), authenticator)...)
	}
	return options
}

// MergeRuntimePlugins keeps base order stable, replaces plugins with the same
// non-empty name in place, and appends newly introduced names at the end.
// Deprecated: use RuntimePluginAssembly.Apply, or ReplaceRuntimePlugins for
// replacement-only flows, to keep replacement intent separate from appended
// plugin intent.
func MergeRuntimePlugins(base []RuntimePlugin, overrides ...RuntimePlugin) []RuntimePlugin {
	merged := make([]RuntimePlugin, 0, len(base)+len(overrides))
	indexByName := make(map[string]int, len(base)+len(overrides))

	appendPlugin := func(plugin RuntimePlugin) {
		if plugin == nil {
			return
		}
		name := strings.TrimSpace(plugin.Name())
		if name == "" {
			merged = append(merged, plugin)
			return
		}
		if idx, ok := indexByName[name]; ok {
			merged[idx] = plugin
			return
		}
		indexByName[name] = len(merged)
		merged = append(merged, plugin)
	}

	for _, plugin := range base {
		appendPlugin(plugin)
	}
	for _, plugin := range overrides {
		appendPlugin(plugin)
	}

	return merged
}
