// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import "testing"

type spikeSetupProbe struct {
	calls int
}

func (p *spikeSetupProbe) Setup(props any, ctx any) any {
	_ = props
	_ = ctx
	p.calls++
	return nil
}

// TestVueHostSubsetSpike_P1 documents the intended VTU subset surface for migration.
// Full QuickJS VTU (flushPromises/find/trigger) lands in PR-unit-vue-host; this spike
// only proves Go-side mount must invoke setup and accept stubs.
func TestVueHostSubsetSpike_P1(t *testing.T) {
	if w := SpikeMount(nil, SpikeMountOptions{}); w.SetupRan {
		t.Fatal("nil component must not report SetupRan")
	}
	var typedNil *spikeSetupProbe
	if w := SpikeMount(typedNil, SpikeMountOptions{}); w.SetupRan {
		t.Fatal("typed nil pointer must not report SetupRan")
	}

	probe := &spikeSetupProbe{}
	w := SpikeMount(probe, SpikeMountOptions{
		Stubs: map[string]bool{
			"el-button": true,
			"el-select": true,
		},
	})
	if !w.SetupRan {
		t.Fatal("expected SpikeMount to run Setup")
	}
	if probe.calls != 1 {
		t.Fatalf("Setup calls = %d, want 1", probe.calls)
	}
	if !w.StubsAccepted["el-button"] || !w.StubsAccepted["el-select"] {
		t.Fatalf("stubs not accepted: %#v", w.StubsAccepted)
	}

	w2 := SpikeShallowMount(probe, SpikeMountOptions{})
	if !w2.SetupRan || probe.calls != 2 {
		t.Fatalf("SpikeShallowMount SetupRan=%v calls=%d", w2.SetupRan, probe.calls)
	}

	if len(PlannedVTUSubsetAPIs) < 6 {
		t.Fatalf("PlannedVTUSubsetAPIs too short: %v", PlannedVTUSubsetAPIs)
	}
	for _, api := range PlannedVTUSubsetAPIs {
		if api == "" {
			t.Fatal("empty planned API name")
		}
	}
}
