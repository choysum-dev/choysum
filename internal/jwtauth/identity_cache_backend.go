// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func newIdentityCache(runtimeScope scope.Scope) (*identityCache, error) {
	jwtCfg := runtimeOptionsFromScope(runtimeScope).authJWT
	if jwtCfg == nil || jwtCfg.IdentityCache == nil {
		return nil, autherrors.NewAuthError(autherrors.ErrMissingConfiguration, "JWT identity cache config is missing")
	}

	backend := strings.TrimSpace(jwtCfg.IdentityCache.Backend)
	if backend == "" {
		backend = "memory"
	}

	store, err := auth.NewJWTIdentityCacheStore(backend, runtimeScope)
	if err != nil {
		return nil, err
	}
	return &identityCache{store: store}, nil
}
