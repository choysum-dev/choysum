// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package model

// Entities returns lease metadata models.
func Entities() []any {
	return []any{
		&LockLease{},
	}
}
