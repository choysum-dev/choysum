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
