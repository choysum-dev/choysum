// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package memory

import (
	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
)

func init() {
	// Register the in-memory store driver.
	revocation.RegisterDriver("memory", NewMemoryStore)
}
