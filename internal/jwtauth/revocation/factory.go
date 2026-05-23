// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// NewStore creates a token revocation store.
func NewStore(runtimeScope scope.Scope) (Store, error) {
	// Load configuration.
	cfg := runtimeOptionsFromScope(runtimeScope).authJWT
	if cfg == nil {
		return nil, autherrors.NewAuthError(autherrors.ErrMissingConfiguration, "missing JWT configuration")
	}

	// Determine the store type.
	storeType := cfg.RevokeStore
	if storeType == "" {
		storeType = "memory" // Default to the memory store.
	}

	// Look up the driver.
	factory, ok := GetDriver(storeType)
	if !ok {
		return nil, autherrors.NewAuthErrorf(autherrors.ErrDriverNotRegistered,
			"revocation store driver '%s' is not registered", storeType)
	}

	// Create the store.
	store, err := factory(runtimeScope)
	if err != nil {
		return nil, autherrors.WrapAuthError(err, autherrors.ErrRevocationStoreFailed,
			"failed to create revocation store")
	}

	return store, nil
}
