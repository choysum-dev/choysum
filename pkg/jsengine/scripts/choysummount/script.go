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

// VuePackageVersion is the pinned Vue runtime used by the FE unit host (matches bootstrap).
const VuePackageVersion = "3.5.38"

// VueBareImportPins returns exact esmresolver pins for vue and its @vue/*
// runtime packages. Nested peers (reka-ui, @floating-ui/vue, …) must share this
// single instance; otherwise renderSlot hits a null currentRenderingInstance
// (TypeError reading 'ce').
func VueBareImportPins() map[string]string {
	v := VuePackageVersion
	return map[string]string{
		"vue":               v,
		"@vue/shared":       v,
		"@vue/reactivity":   v,
		"@vue/runtime-core": v,
		"@vue/runtime-dom":  v,
	}
}
