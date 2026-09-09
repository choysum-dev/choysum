// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import "testing"

func TestIDEToolingTypePackages(t *testing.T) {
	pkgs := IDEToolingTypePackages()
	seen := map[string]bool{}
	for _, pkg := range pkgs {
		if pkg == "" {
			t.Fatal("unexpected empty tooling package name")
		}
		if seen[pkg] {
			t.Fatalf("duplicate tooling package %q", pkg)
		}
		seen[pkg] = true
	}
}
