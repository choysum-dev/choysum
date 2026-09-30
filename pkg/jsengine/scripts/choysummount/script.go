// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysummount

import (
	_ "embed"
	"strings"
)

//go:embed dom.js
var MinimalDOMScript string

//go:embed choysummount.js
var ChoysumMountScript string

//go:generate go run gen_vue_version.go

// hostVueRuntimePackages must share the single embedded host Vue instance.
// IsHostVueRuntimePackage and VueBareImportPinsFor both derive from this list.
var hostVueRuntimePackages = []string{
	"vue",
	"@vue/shared",
	"@vue/reactivity",
	"@vue/runtime-core",
	"@vue/runtime-dom",
}

// VueBareImportPins returns pins for [VuePackageVersion] (generated fallback).
func VueBareImportPins() map[string]string {
	return VueBareImportPinsFor(VuePackageVersion)
}

// IsHostVueRuntimePackage reports packages owned by the single host Vue
// instance. Other @vue/* names (test-utils, server-renderer, …) are not
// host-owned and may keep module exact pins.
func IsHostVueRuntimePackage(name string) bool {
	n := strings.TrimSpace(name)
	for _, pkg := range hostVueRuntimePackages {
		if n == pkg {
			return true
		}
	}
	return false
}

// VueBareImportPinsFor returns exact esmresolver pins for vue and its @vue/*
// runtime packages at version. Nested peers (reka-ui, @floating-ui/vue, …)
// must share this single instance; otherwise renderSlot hits a null
// currentRenderingInstance (TypeError reading 'ce').
func VueBareImportPinsFor(version string) map[string]string {
	v := exactVuePin(version)
	pins := make(map[string]string, len(hostVueRuntimePackages))
	for _, pkg := range hostVueRuntimePackages {
		pins[pkg] = v
	}
	return pins
}

// exactVuePin returns version when it is an exact major.minor.patch pin
// (optional prerelease/build suffix allowed) and VuePackageVersion otherwise.
// Floating forms such as "3" or "3.x" would let esm.sh resolve a second Vue
// copy and reintroduce the multi-instance failure this prevents.
func exactVuePin(version string) string {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	lower := strings.ToLower(v)
	if v == "" || lower == "*" || lower == "latest" || lower == "next" ||
		strings.ContainsAny(v, "^~*<>=| ") {
		return VuePackageVersion
	}
	core := v
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	if len(parts) < 3 {
		return VuePackageVersion
	}
	for _, p := range parts {
		if p == "" {
			return VuePackageVersion
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return VuePackageVersion
			}
		}
	}
	return v
}
