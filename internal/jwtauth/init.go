// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	_ "github.com/choysum-dev/choysum/internal/jwtauth/revocation/drivers/database"
	_ "github.com/choysum-dev/choysum/internal/jwtauth/revocation/drivers/memory"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Register the JWT authenticator factory.
func init() {
	auth.Register("jwt", createJWTAuthenticator)
}

// createJWTAuthenticator builds a JWT authenticator from the runtime scope.
func createJWTAuthenticator(runtimeScope scope.Scope) (auth.Authenticator, error) {
	return NewJWTAuthenticator(runtimeScope)
}
