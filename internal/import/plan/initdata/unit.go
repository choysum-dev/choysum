// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata

import "github.com/choysum-dev/choysum/internal/import/plan"

// Unit applies one initdata file batch for a module.
type Unit struct {
	Index       int
	ModuleName  string
	ModulePath  string
	Application string
	Files       []string
}

// UnitIndex implements plan.Unit.
func (u Unit) UnitIndex() int {
	if u.Index > 0 {
		return u.Index
	}
	return 1
}

var _ plan.Unit = Unit{}
