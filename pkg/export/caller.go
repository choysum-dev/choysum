// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

// Caller identifies who initiated an export run.
type Caller string

const (
	CallerUser        Caller = "user"
	CallerCLI         Caller = "cli"
	CallerE2E         Caller = "e2e"
	CallerUnspecified Caller = ""

	// CallerHTTP maps thin HTTP gateway entry points onto CallerUser.
	CallerHTTP = CallerUser
)

// Valid reports whether c is a known export caller constant.
func (c Caller) Valid() bool {
	switch c {
	case CallerUser, CallerCLI, CallerE2E:
		return true
	default:
		return false
	}
}
