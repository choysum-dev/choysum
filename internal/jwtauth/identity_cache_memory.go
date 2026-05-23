// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"github.com/choysum-dev/choysum/internal/defaultcache"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/cache"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	auth.RegisterJWTIdentityCacheStoreDriver("memory", newMemoryIdentityCacheStore)
}

func newMemoryIdentityCacheStore(runtimeScope scope.Scope) (cache.Cache, error) {
	jwtCfg := runtimeOptionsFromScope(runtimeScope).authJWT
	if jwtCfg == nil || jwtCfg.IdentityCache == nil {
		return nil, autherrors.NewAuthError(autherrors.ErrMissingConfiguration, "JWT identity cache config is missing")
	}

	return defaultcache.NewInMemory(defaultcache.Options{MaxEntries: jwtCfg.IdentityCache.Size})
}
