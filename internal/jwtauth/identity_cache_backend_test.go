// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/cache"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestIdentityCacheStoreDriverRegistrationAndLookup(t *testing.T) {
	factoryCalled := false
	driverName := "fake-" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-")
	auth.RegisterJWTIdentityCacheStoreDriver(driverName, func(scope.Scope) (cache.Cache, error) {
		factoryCalled = true
		return nil, errors.New("boom")
	})

	runtimeScope := newFakeScope()
	runtimeScope.cfg.Auth.JWT.IdentityCache.Backend = driverName
	if _, err := newIdentityCache(runtimeScope); err == nil || err.Error() != "boom" {
		t.Fatalf("expected fake driver error, got %v", err)
	}
	if !factoryCalled {
		t.Fatal("expected fake driver factory to be called")
	}
}

func TestNewIdentityCacheRejectsMissingDriver(t *testing.T) {
	runtimeScope := newFakeScope()
	runtimeScope.cfg.Auth.JWT.IdentityCache.Backend = "missing-" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-")

	_, err := newIdentityCache(runtimeScope)
	if err == nil || !autherrors.IsAuthError(err, autherrors.ErrDriverNotRegistered) {
		t.Fatalf("expected missing driver auth error, got %v", err)
	}
}
