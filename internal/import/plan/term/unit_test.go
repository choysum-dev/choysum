// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import "testing"

func TestUnitIndex(t *testing.T) {
	if (Unit{Index: 3}.UnitIndex()) != 3 {
		t.Fatal("Index=3")
	}
	if (Unit{}.UnitIndex()) != 1 {
		t.Fatal("default index")
	}
}
