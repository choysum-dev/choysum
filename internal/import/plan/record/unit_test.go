// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"testing"

	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
)

func TestUnit_UnitIndex(t *testing.T) {
	if (recordplan.Unit{}).UnitIndex() != 1 {
		t.Fatal("expected default unit index 1")
	}
	if (recordplan.Unit{Index: 4}).UnitIndex() != 4 {
		t.Fatal("expected explicit unit index")
	}
}
