// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import "testing"

func TestApplyBuildOptionsNilAndSkipWebShell(t *testing.T) {
	WithSkipWebShell(true)(nil)

	out := applyBuildOptions(nil)
	if out.SkipWebShell {
		t.Fatal("expected zero options")
	}
	out = applyBuildOptions([]BuildOption{nil, WithSkipWebShell(true)})
	if !out.SkipWebShell {
		t.Fatal("expected SkipWebShell")
	}
}
