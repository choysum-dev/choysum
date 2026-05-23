// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"time"
)

// IsExpired reports whether a token has expired.
func IsExpired(expiresAt time.Time) bool {
	return expiresAt.Before(time.Now())
}

// IsValidTokenID reports whether a token ID is valid.
func IsValidTokenID(tokenID string) bool {
	return tokenID != ""
}

// IsValidUserID reports whether a user ID is valid.
func IsValidUserID(userID string) bool {
	return userID != ""
}
