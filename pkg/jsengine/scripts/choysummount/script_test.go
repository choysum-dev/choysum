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
}

func TestExactVuePin(t *testing.T) {
	fb := VuePackageVersion
	cases := []struct {
		in, want string
	}{
		{"3.9.9", "3.9.9"},
		{"v3.9.9", "3.9.9"},
		{" 3.9.9 ", "3.9.9"},
		{"3.9.9-beta.1", "3.9.9-beta.1"},
		{"3.9.9+build.1", "3.9.9+build.1"},
		{"3.9.9-beta.1+exp.sha", "3.9.9-beta.1+exp.sha"},
		{"", fb},
		{" ", fb},
		{"*", fb},
		{"latest", fb},
		{"next", fb},
		{"^3.5.38", fb},
		{"~3.5.38", fb},
		{"3", fb},
		{"3.5", fb},
		{"3.x", fb},
		{"3.5.38.4", fb},     // surplus core component
		{"3.5.38-", fb},      // empty prerelease
		{"3.5.38+", fb},      // empty build
		{"3.5.38-.", fb},     // empty prerelease id
		{"3.5.38+.", fb},     // empty build id
		{"3.5.38-beta!", fb}, // invalid id char
		{"3..38", fb},
		{"3.5.", fb},
		{".5.38", fb},
		{"3.5.x", fb},
	}
	for _, tc := range cases {
		if got := exactVuePin(tc.in); got != tc.want {
			t.Fatalf("exactVuePin(%q) = %q want %q", tc.in, got, tc.want)
		}
		if got := VueBareImportPinsFor(tc.in)["vue"]; got != tc.want {
			t.Fatalf("VueBareImportPinsFor(%q)[vue] = %q want %q", tc.in, got, tc.want)
		}
		trimmed := strings.TrimPrefix(strings.TrimSpace(tc.in), "v")
		wantExact := trimmed != "" && exactVuePin(tc.in) == trimmed
		if IsExactVuePin(tc.in) != wantExact {
			t.Fatalf("IsExactVuePin(%q) = %v want %v", tc.in, IsExactVuePin(tc.in), wantExact)
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
		t.Fatalf("read %s: %v", pkgPath, err)
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
		t.Fatalf("VuePackageVersion=%q out of date vs modules/web vue=%q; run: go generate ./pkg/jsengine/scripts/choysummount/...", VuePackageVersion, ver)
	}
	if !IsExactVuePin(ver) {
		t.Fatalf("modules/web vue must be an exact major.minor.patch pin for host SSOT, got %q", ver)
	}

	bootPath := filepath.Join(repoRoot, "internal", "bootstrap", "web", "package.json")
	bootData, err := os.ReadFile(bootPath)
	if err != nil {
		t.Fatalf("read bootstrap: %v", err)
	}
	var boot struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(bootData, &boot); err != nil {
		t.Fatal(err)
	}
	bootVue := strings.TrimPrefix(strings.TrimSpace(boot.Dependencies["vue"]), "v")
	if bootVue != VuePackageVersion {
		t.Fatalf("bootstrap web vue=%q out of date; run: go generate ./pkg/jsengine/scripts/choysummount/...", bootVue)
	}
	if !IsExactVuePin(bootVue) {
		t.Fatalf("bootstrap web vue must be an exact major.minor.patch pin, got %q", bootVue)
	}
}
