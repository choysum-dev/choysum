// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

// Unit is one executable import item after adapter planning.
type Unit interface {
	UnitIndex() int
}

// Plan is the ordered unit list produced by an adapter.
type Plan struct {
	Units []Unit
}

// Len returns the number of units.
func (p Plan) Len() int {
	return len(p.Units)
}
