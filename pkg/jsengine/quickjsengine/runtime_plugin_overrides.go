// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

func filterReplaceableRuntimePlugins(plugins []jsengine.RuntimePlugin) []jsengine.RuntimePlugin {
	filtered := make([]jsengine.RuntimePlugin, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin == nil {
			continue
		}
		name := strings.TrimSpace(plugin.Name())
		if name == "" || !IsReplaceableRuntimePlugin(name) {
			continue
		}
		filtered = append(filtered, plugin)
	}
	return filtered
}

// NormalizeRuntimePluginAssembly filters replacement slots through the
// published QuickJS replacement policy while leaving appended extras unchanged.
// Use this helper when callers need a policy-filtered assembly spec before it
// is applied to a concrete base plugin list.
func NormalizeRuntimePluginAssembly(assembly jsengine.RuntimePluginAssembly) jsengine.RuntimePluginAssembly {
	return jsengine.RuntimePluginAssembly{
		Replacements: filterReplaceableRuntimePlugins(assembly.Replacements),
		Extras:       append([]jsengine.RuntimePlugin(nil), assembly.Extras...),
	}
}

// ApplyRuntimePluginAssembly applies a QuickJS runtime plugin assembly spec to
// the provided base plugin list. This is the preferred package-boundary helper
// once an assembly spec has been constructed.
func ApplyRuntimePluginAssembly(base []jsengine.RuntimePlugin, assembly jsengine.RuntimePluginAssembly) []jsengine.RuntimePlugin {
	return NormalizeRuntimePluginAssembly(assembly).Apply(base)
}

func partitionRuntimePluginOverrides(overrides []jsengine.RuntimePlugin) ([]jsengine.RuntimePlugin, []jsengine.RuntimePlugin) {
	replacements := make([]jsengine.RuntimePlugin, 0, len(overrides))
	extras := make([]jsengine.RuntimePlugin, 0, len(overrides))
	for _, plugin := range overrides {
		if plugin == nil {
			continue
		}
		name := strings.TrimSpace(plugin.Name())
		if name != "" && IsReplaceableRuntimePlugin(name) {
			replacements = append(replacements, plugin)
			continue
		}
		extras = append(extras, plugin)
	}
	return replacements, extras
}

// AssembleRuntimePlugins keeps default slot replacements and appended plugins in
// separate inputs so callers can describe their intent explicitly.
// Deprecated: use ApplyRuntimePluginAssembly with jsengine.RuntimePluginAssembly
// to keep the assembly boundary explicit.
func AssembleRuntimePlugins(base []jsengine.RuntimePlugin, replacements []jsengine.RuntimePlugin, extras []jsengine.RuntimePlugin) []jsengine.RuntimePlugin {
	return ApplyRuntimePluginAssembly(base, jsengine.RuntimePluginAssembly{
		Replacements: replacements,
		Extras:       extras,
	})
}

// MergeRuntimePluginOverrides applies QuickJS runtime plugin overrides against a
// base plugin set. Only published replaceable plugin names are replaced in place.
// Plugins targeting default-only or unknown names are appended after the base
// plugin list so they cannot silently take over a reserved default slot.
// Deprecated: use ApplyRuntimePluginAssembly with
// jsengine.RuntimePluginAssembly to keep replacement intent separate from
// appended plugin intent.
func MergeRuntimePluginOverrides(base []jsengine.RuntimePlugin, overrides ...jsengine.RuntimePlugin) []jsengine.RuntimePlugin {
	replacements, extras := partitionRuntimePluginOverrides(overrides)
	return ApplyRuntimePluginAssembly(base, jsengine.RuntimePluginAssembly{
		Replacements: replacements,
		Extras:       extras,
	})
}
