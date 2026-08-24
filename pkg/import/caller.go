// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

// Caller identifies who initiated an import run.
type Caller string

const (
	CallerLifecycle   Caller = "lifecycle"
	CallerUser        Caller = "user"
	CallerE2E         Caller = "e2e"
	CallerCLI         Caller = "cli"
	CallerUnspecified Caller = ""
)

// Valid reports whether c is a known caller constant.
func (c Caller) Valid() bool {
	switch c {
	case CallerLifecycle, CallerUser, CallerE2E, CallerCLI:
		return true
	default:
		return false
	}
}
