// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan/stub"
)

func TestUnit_UnitIndex(t *testing.T) {
	u := stub.Unit{Index: 4}
	if got := u.UnitIndex(); got != 4 {
		t.Fatalf("UnitIndex() = %d, want 4", got)
	}
}
