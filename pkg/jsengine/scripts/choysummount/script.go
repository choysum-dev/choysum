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

// VueBareImportPins returns pins for [VuePackageVersion] (generated fallback).
func VueBareImportPins() map[string]string {
	return VueBareImportPinsFor(VuePackageVersion)
}

// IsHostVueRuntimePackage reports packages owned by the single host Vue
// instance (see VueBareImportPinsFor). Other @vue/* names (test-utils,
// server-renderer, …) are not host-owned and may keep module exact pins.
func IsHostVueRuntimePackage(name string) bool {
	switch strings.TrimSpace(name) {
	case "vue", "@vue/shared", "@vue/reactivity", "@vue/runtime-core", "@vue/runtime-dom":
		return true
	default:
		return false
	}
}

// VueBareImportPinsFor returns exact esmresolver pins for vue and its @vue/*
// runtime packages at version. Nested peers (reka-ui, @floating-ui/vue, …)
// must share this single instance; otherwise renderSlot hits a null
// currentRenderingInstance (TypeError reading 'ce').
func VueBareImportPinsFor(version string) map[string]string {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	lower := strings.ToLower(v)
	if v == "" || lower == "*" || lower == "latest" || lower == "next" ||
		strings.ContainsAny(v, "^~*<>=| ") {
		v = VuePackageVersion
	}
	return map[string]string{
		"vue":               v,
		"@vue/shared":       v,
		"@vue/reactivity":   v,
		"@vue/runtime-core": v,
		"@vue/runtime-dom":  v,
	}
}
