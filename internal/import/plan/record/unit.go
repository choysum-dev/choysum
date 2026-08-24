// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import "github.com/choysum-dev/choysum/internal/import/plan"

// Unit is one CSV data row for record import.
type Unit struct {
	Index      int
	RowNumber  int
	Model      string
	ExternalID string
	Values     map[string]string
}

// UnitIndex implements plan.Unit.
func (u Unit) UnitIndex() int {
	if u.Index > 0 {
		return u.Index
	}
	return 1
}

var _ plan.Unit = Unit{}
