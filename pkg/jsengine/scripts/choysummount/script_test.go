// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysummount

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

func TestIsHostVueRuntimePackage(t *testing.T) {
	for _, name := range []string{"vue", "@vue/shared", "@vue/reactivity", "@vue/runtime-core", "@vue/runtime-dom", "  vue  "} {
		if !IsHostVueRuntimePackage(name) {
			t.Fatalf("%q should be host-owned", name)
		}
	}
	for _, name := range []string{"", "@vue/test-utils", "@vue/server-renderer", "reka-ui", "vue-router"} {
		if IsHostVueRuntimePackage(name) {
			t.Fatalf("%q must not be host-owned", name)
		}
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
	custom := VueBareImportPinsFor("3.9.9")
	if custom["vue"] != "3.9.9" || custom["@vue/runtime-core"] != "3.9.9" {
		t.Fatalf("VueBareImportPinsFor = %#v", custom)
	}
	if VueBareImportPinsFor("v3.9.9")["vue"] != "3.9.9" {
		t.Fatal("leading v must be stripped")
	}
	for _, bad := range []string{"", " ", "^3.5.38", "latest", "next", "*", "  ^1.0.0  "} {
		if VueBareImportPinsFor(bad)["vue"] != VuePackageVersion {
			t.Fatalf("non-exact %q must fall back to VuePackageVersion", bad)
		}
	}
}

func TestVuePackageVersionMatchesWebKit(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// pkg/jsengine/scripts/choysummount → repo root
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
	pkgPath := filepath.Join(repoRoot, "modules", "web", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("read %s: %v (fallback const is only valid when kit package.json is present)", pkgPath, err)
	}
	var pkg struct {
		Dependencies     map[string]string `json:"dependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	depVue := strings.TrimPrefix(strings.TrimSpace(pkg.Dependencies["vue"]), "v")
	peerVue := strings.TrimPrefix(strings.TrimSpace(pkg.PeerDependencies["vue"]), "v")
	if depVue != "" && peerVue != "" && depVue != peerVue {
		t.Fatalf("modules/web declares conflicting vue pins: dependencies=%q peerDependencies=%q", depVue, peerVue)
	}
	ver := depVue
	if ver == "" {
		ver = peerVue
	}
	if ver != VuePackageVersion {
		t.Fatalf("modules/web exact vue = %q, choysummount.VuePackageVersion = %q; keep them identical (kit package.json is SSOT)", ver, VuePackageVersion)
	}
	if strings.ContainsAny(ver, "^~*<>=| ") {
		t.Fatalf("modules/web vue must be an exact pin for host SSOT, got %q", ver)
	}
}
