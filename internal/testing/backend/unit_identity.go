// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"

	"github.com/choysum-dev/choysum/internal/module/evolution/systemidentity"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// unitTestDefaultIdentity is the bootstrap admin principal injected into
// backend unit-test JsRequest.Context when the auth module is installed.
type unitTestDefaultIdentity struct {
	UserID    string
	CompanyID string
}

// resolveUnitTestDefaultIdentity returns auth.user_admin (+ company) when auth
// is installed in the test DB. Missing auth / seeds yield ok=false (no inject).
// Operational DB errors are returned so callers do not run anonymously by accident.
func resolveUnitTestDefaultIdentity(ctx context.Context, runtimeScope scope.Scope) (unitTestDefaultIdentity, bool, error) {
	admin, ok, err := systemidentity.ResolveBootstrapAdmin(ctx, runtimeScope)
	if err != nil || !ok {
		return unitTestDefaultIdentity{}, ok, err
	}
	return unitTestDefaultIdentity{UserID: admin.UserID, CompanyID: admin.CompanyID}, true, nil
}

func unitTestJsRequestContext(identity unitTestDefaultIdentity) map[string]interface{} {
	return map[string]interface{}{
		"identity": map[string]interface{}{
			"userId": identity.UserID,
		},
		"ctx": map[string]interface{}{
			"activeCompanyId":   identity.CompanyID,
			"enabledCompanyIds": []string{identity.CompanyID},
		},
		"req": map[string]interface{}{
			"depth": 0,
		},
	}
}
