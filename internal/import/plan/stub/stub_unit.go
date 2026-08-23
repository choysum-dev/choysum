// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub

import "github.com/choysum-dev/choysum/internal/import/plan"

// Unit is a skeleton plan item for PR-import-1 tests.
type Unit struct {
	Index int
	Fail  bool
}

// UnitIndex implements plan.Unit.
func (u Unit) UnitIndex() int {
	return u.Index
}

var _ plan.Unit = Unit{}
