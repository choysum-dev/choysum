// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

// Profile identifies which Reader track executes a run.
type Profile string

const (
	ProfileRecord      Profile = "record"
	ProfileTerminology Profile = "terminology"
	ProfileUnspecified Profile = ""
)

// Valid reports whether p is an approved export profile (EX1: no initdata).
func (p Profile) Valid() bool {
	switch p {
	case ProfileRecord, ProfileTerminology:
		return true
	default:
		return false
	}
}
