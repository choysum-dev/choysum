// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import "github.com/choysum-dev/choysum/internal/import/plan"

// Unit applies one PO file for a module language.
type Unit struct {
	Index       int
	Application string
	ModuleName  string
	ModulePath  string
	Lang        string
	PoPath      string
}

// UnitIndex implements plan.Unit.
func (u Unit) UnitIndex() int {
	if u.Index > 0 {
		return u.Index
	}
	return 1
}

var _ plan.Unit = Unit{}
