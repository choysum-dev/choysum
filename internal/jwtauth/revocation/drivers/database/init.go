// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package database

import (
	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
)

func init() {
	// Register the database store driver.
	revocation.RegisterDriver("database", NewDatabaseStore)
}
