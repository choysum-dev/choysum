// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

// IDEToolingTypePackages lists workspace packages that tests may import via bare
// specifiers but are not declared in module package.json product dependencies.
//
// After FE unit hard-cut, Vitest/VTU are gone; keep this empty unless a new
// IDE-only bare specifier needs tsconfig paths.
func IDEToolingTypePackages() []string {
	return nil
}
