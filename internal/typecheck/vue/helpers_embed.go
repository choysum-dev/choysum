// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	_ "embed"
)

//go:embed helpers/template-helpers.d.ts
var templateHelpersDTS string

//go:embed helpers/props-fallback.d.ts
var propsFallbackDTS string

// HelperOverlays returns VFS overlays for language-core helper declaration files.
func HelperOverlays() map[string]string {
	return map[string]string{
		HelperTemplatePath: templateHelpersDTS,
		HelperPropsPath:    propsFallbackDTS,
	}
}
