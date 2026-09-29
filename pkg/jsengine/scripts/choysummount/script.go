// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysummount

import (
	_ "embed"
)

//go:embed dom.js
var MinimalDOMScript string

//go:embed choysummount.js
var ChoysumMountScript string

// VuePackageVersion is the fallback Vue runtime pin when modules/web/package.json
// has no exact "vue" peer/dependency. Keep it identical to that exact pin when
// present (see TestVuePackageVersionMatchesWebKit).
const VuePackageVersion = "3.5.38"

// VueBareImportPins returns pins for [VuePackageVersion] (fallback host).
func VueBareImportPins() map[string]string {
	return VueBareImportPinsFor(VuePackageVersion)
}

// VueBareImportPinsFor returns exact esmresolver pins for vue and its @vue/*
// runtime packages at version. Nested peers (reka-ui, @floating-ui/vue, …)
// must share this single instance; otherwise renderSlot hits a null
// currentRenderingInstance (TypeError reading 'ce').
func VueBareImportPinsFor(version string) map[string]string {
	v := version
	if v == "" {
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
