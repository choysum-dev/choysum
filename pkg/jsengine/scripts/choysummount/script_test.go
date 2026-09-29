// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysummount

import (
	"strings"
	"testing"
)

func TestEmbeddedScripts(t *testing.T) {
	if MinimalDOMScript == "" || !strings.Contains(MinimalDOMScript, "__choysumMinimalDOM") {
		t.Fatal("MinimalDOMScript missing")
	}
	if ChoysumMountScript == "" || !strings.Contains(ChoysumMountScript, "export function mount") {
		t.Fatal("ChoysumMountScript missing mount")
	}
	if VuePackageVersion == "" {
		t.Fatal("empty VuePackageVersion")
	}
}

func TestVueBareImportPins(t *testing.T) {
	pins := VueBareImportPins()
	if pins["vue"] != VuePackageVersion {
		t.Fatalf("vue pin = %q want %q", pins["vue"], VuePackageVersion)
	}
	for _, pkg := range []string{"@vue/shared", "@vue/reactivity", "@vue/runtime-core", "@vue/runtime-dom"} {
		if pins[pkg] != VuePackageVersion {
			t.Fatalf("%s pin = %q want %q", pkg, pins[pkg], VuePackageVersion)
		}
	}
	// Caller may mutate without affecting the next call.
	pins["vue"] = "0.0.0"
	if VueBareImportPins()["vue"] != VuePackageVersion {
		t.Fatal("VueBareImportPins must return a fresh map")
	}
}
