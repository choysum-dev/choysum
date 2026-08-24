// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata_test

import (
	"testing"

	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
)

func TestUnit_UnitIndex(t *testing.T) {
	if got := (initdataplan.Unit{Index: 3}).UnitIndex(); got != 3 {
		t.Fatalf("UnitIndex() = %d, want 3", got)
	}
	if got := (initdataplan.Unit{}).UnitIndex(); got != 1 {
		t.Fatalf("UnitIndex() default = %d, want 1", got)
	}
}
