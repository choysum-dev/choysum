// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

// Profile identifies which Writer track executes a run.
type Profile string

const (
	ProfileInitdata    Profile = "initdata"
	ProfileTerminology Profile = "terminology"
	ProfileRecord      Profile = "record"
	ProfileUnspecified Profile = ""
)

// Valid reports whether p is a known profile constant.
func (p Profile) Valid() bool {
	switch p {
	case ProfileInitdata, ProfileTerminology, ProfileRecord:
		return true
	default:
		return false
	}
}
