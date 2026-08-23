// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

import "testing"

func TestAllowsCallerProfile_defaultProfile(t *testing.T) {
	if allowsCallerProfile(Profile("bogus"), CallerUser) {
		t.Fatal("unknown profile should deny all callers")
	}
}
