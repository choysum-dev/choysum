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

// exactVuePin returns version when it is an exact SemVer major.minor.patch pin
// (optional nonempty prerelease/build suffixes allowed) and VuePackageVersion
// otherwise. Floating or malformed forms (e.g. "3", "3.5.38.4", "3.5.38-")
// would let esm.sh resolve a second Vue copy.
func exactVuePin(version string) string {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	lower := strings.ToLower(v)
	if v == "" || lower == "*" || lower == "latest" || lower == "next" ||
		strings.ContainsAny(v, "^~*<>=| ") {
		return VuePackageVersion
	}

	orig := v
	build := ""
	if i := strings.IndexByte(v, '+'); i >= 0 {
		build = v[i+1:]
		v = v[:i]
		if build == "" || !semverDotIds(build) {
			return VuePackageVersion
		}
	}
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
		if pre == "" || !semverDotIds(pre) {
			return VuePackageVersion
		}
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return VuePackageVersion
	}
	for _, p := range parts {
		if p == "" || !allASCIIDigits(p) {
			return VuePackageVersion
		}
	}
	return orig
}

func semverDotIds(s string) bool {
	for _, id := range strings.Split(s, ".") {
		if id == "" {
			return false
		}
		for _, c := range id {
			ok := (c >= '0' && c <= '9') ||
				(c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				c == '-'
			if !ok {
				return false
			}
		}
	}
	return true
}

func allASCIIDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
