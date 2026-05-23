// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

// Identity describes an authenticated user identity.
type Identity interface {
	GetUserID() string
	GetTokenID() string
	GetMetadata() map[string]interface{}
	IsValid() bool
}
