// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

// IDEToolingTypePackages lists workspace test-runner packages that frontend
// tests import via bare specifiers (for example `from 'vitest'`) but are not
// declared in module package.json product dependencies.
//
// Keep this list focused on packages that need tsconfig paths for IDE module
// resolution. Runtime preflight for `go run . test` may require a broader set
// (vite, sass-embedded, coverage plugins, …) and lives separately in the
// frontend test harness.
func IDEToolingTypePackages() []string {
	return []string{
		"vitest",
		"@vue/test-utils",
	}
}
