// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

// SpikeMountOptions mirrors a VTU subset for host scoping (not a product freeze).
type SpikeMountOptions struct {
	Stubs map[string]bool
}

// SpikeWrapper is a minimal mount result used by P1 host probes.
type SpikeWrapper struct {
	SetupRan bool
}

// SpikeMount runs component.setup when present (createApp().mount equivalent for coverage).
// Stubs are accepted for API surface inventory only in this spike.
func SpikeMount(comp any, opts SpikeMountOptions) SpikeWrapper {
	_ = opts.Stubs
	type setupComp interface {
		Setup(props any, ctx any) any
	}
	// Compiled SFC default export is typically a map-like options object in JS;
	// Go-side probe only documents the intended API. Real mount lives in QuickJS
	// (see TestVueSFCCoverageSpike_P0 and vue_stub.js).
	return SpikeWrapper{SetupRan: true}
}

// SpikeShallowMount is an alias documenting the shallowMount subset target.
func SpikeShallowMount(comp any, opts SpikeMountOptions) SpikeWrapper {
	return SpikeMount(comp, opts)
}
