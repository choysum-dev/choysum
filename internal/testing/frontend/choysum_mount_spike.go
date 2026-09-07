// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

// SpikeMountOptions mirrors a VTU subset for host scoping (not a product freeze).
type SpikeMountOptions struct {
	Stubs map[string]bool
}

// SpikeWrapper is a minimal mount result used by P1 host probes.
type SpikeWrapper struct {
	SetupRan      bool
	StubsAccepted map[string]bool
}

// setupComp is the Go-side stand-in for a compiled SFC options object with setup().
// Real product mounts run in QuickJS (see TestVueSFCCoverageSpike_P0); this probe
// only documents that host mount must invoke setup when present.
type setupComp interface {
	Setup(props any, ctx any) any
}

// PlannedVTUSubsetAPIs lists mount-host surface to implement in PR-unit-vue-host.
// flushPromises / find / trigger are not implemented in this Go spike.
var PlannedVTUSubsetAPIs = []string{
	"mount",
	"shallowMount",
	"stubs",
	"flushPromises",
	"wrapper.find",
	"wrapper.trigger",
}

// SpikeMount runs component.Setup when present (createApp().mount equivalent for coverage).
// Stubs are recorded for API surface inventory only in this spike.
func SpikeMount(comp any, opts SpikeMountOptions) SpikeWrapper {
	w := SpikeWrapper{}
	if len(opts.Stubs) > 0 {
		w.StubsAccepted = make(map[string]bool, len(opts.Stubs))
		for k, v := range opts.Stubs {
			w.StubsAccepted[k] = v
		}
	}
	setup, ok := comp.(setupComp)
	if !ok || setup == nil {
		return w
	}
	setup.Setup(nil, nil)
	w.SetupRan = true
	return w
}

// SpikeShallowMount is an alias documenting the shallowMount subset target.
func SpikeShallowMount(comp any, opts SpikeMountOptions) SpikeWrapper {
	return SpikeMount(comp, opts)
}
