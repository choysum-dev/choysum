// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import "testing"

// TestVueHostSubsetSpike_P1 documents the intended VTU subset surface for migration.
// Full QuickJS VTU compatibility is out of scope for this spike (see vue_cov_spike_migration.md).
func TestVueHostSubsetSpike_P1(t *testing.T) {
	w := SpikeMount(nil, SpikeMountOptions{
		Stubs: map[string]bool{
			"el-button": true,
			"el-select": true,
		},
	})
	if !w.SetupRan {
		t.Fatal("expected SpikeMount to report setup path")
	}
	w2 := SpikeShallowMount(nil, SpikeMountOptions{})
	if !w2.SetupRan {
		t.Fatal("expected SpikeShallowMount")
	}

	// Inventory of APIs to implement in PR-unit-vue-host (from main VTU corpus):
	needed := []string{
		"mount",
		"shallowMount",
		"stubs",
		"flushPromises",
		"wrapper.find",
		"wrapper.trigger",
	}
	for _, api := range needed {
		if api == "" {
			t.Fatal("empty api")
		}
	}
}
