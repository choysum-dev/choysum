// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"errors"
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
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
	var out unitTestDefaultIdentity
	if runtimeScope == nil {
		return out, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	txRoot := runtimeScope.WithContext(ctx)
	if txRoot == nil {
		txRoot = runtimeScope
	}
	err := txRoot.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		session := txScope.Session()
		if session == nil || session.DB == nil {
			return nil
		}
		if !session.Migrator().HasTable((&meta.Module{}).TableName()) {
			return nil
		}

		var authMod meta.Module
		if err := session.Select("id", "status").Where("name = ?", "auth").Take(&authMod).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return xfmt.Errorf("load auth module for unit identity: %w", err)
		}
		if authMod.Status != meta.Installed {
			return nil
		}

		if !session.Migrator().HasTable((&modmeta.ModelData{}).TableName()) {
			return nil
		}
		var userData modmeta.ModelData
		if err := session.Select("res_id").Where("module = ? AND name = ?", "auth", "user_admin").Take(&userData).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return xfmt.Errorf("load auth.user_admin mapping for unit identity: %w", err)
		}
		userID := strings.TrimSpace(userData.ResID)
		if userID == "" {
			return nil
		}

		var userModel meta.Model
		if lookupErr := session.DB.Where("application = ? AND name = ?", "auth", "User").First(&userModel).Error; lookupErr != nil {
			if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				// Auth installed but User seed/projection missing — treat as no inject.
				return nil
			}
			return xfmt.Errorf("lookup auth.User for unit identity: %w", lookupErr)
		}
		if strings.TrimSpace(userModel.ModelTable) == "" {
			return nil
		}

		var row struct {
			CompanyID string `gorm:"column:company_id"`
		}
		qErr := session.Table(userModel.ModelTable).Select("company_id").Where("id = ?", userID).Take(&row).Error
		if qErr != nil {
			if errors.Is(qErr, gorm.ErrRecordNotFound) {
				// Mapping exists but the user row does not — fail closed (no phantom admin).
				return nil
			}
			return xfmt.Errorf("load auth.user_admin row for unit identity: %w", qErr)
		}

		companyID := strings.TrimSpace(row.CompanyID)
		if companyID == "" {
			var companyData modmeta.ModelData
			if err := session.Select("res_id").Where("module = ? AND name = ?", "base", "company_main").Take(&companyData).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return xfmt.Errorf("load base.company_main mapping for unit identity: %w", err)
			}
			companyID = strings.TrimSpace(companyData.ResID)
		}
		if companyID == "" {
			return nil
		}

		out = unitTestDefaultIdentity{UserID: userID, CompanyID: companyID}
		return nil
	})
	if err != nil {
		return unitTestDefaultIdentity{}, false, err
	}
	return out, out.UserID != "" && out.CompanyID != "", nil
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
