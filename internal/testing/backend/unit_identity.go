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
func resolveUnitTestDefaultIdentity(ctx context.Context, runtimeScope scope.Scope) (unitTestDefaultIdentity, bool) {
	var out unitTestDefaultIdentity
	if runtimeScope == nil {
		return out, false
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
			return err
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
			return err
		}
		userID := strings.TrimSpace(userData.ResID)
		if userID == "" {
			return nil
		}

		companyID := ""
		userModel, lookupErr := modmeta.LookupEffectiveModel(session.DB, "auth", "User")
		if lookupErr == nil && userModel != nil && strings.TrimSpace(userModel.ModelTable) != "" {
			var row struct {
				CompanyID string `gorm:"column:company_id"`
			}
			qErr := session.Table(userModel.ModelTable).Select("company_id").Where("id = ?", userID).Take(&row).Error
			if qErr == nil {
				companyID = strings.TrimSpace(row.CompanyID)
			}
		}
		if companyID == "" {
			var companyData modmeta.ModelData
			if err := session.Select("res_id").Where("module = ? AND name = ?", "base", "company_main").Take(&companyData).Error; err == nil {
				companyID = strings.TrimSpace(companyData.ResID)
			}
		}
		if companyID == "" {
			return nil
		}

		out = unitTestDefaultIdentity{UserID: userID, CompanyID: companyID}
		return nil
	})
	if err != nil {
		return unitTestDefaultIdentity{}, false
	}
	return out, out.UserID != "" && out.CompanyID != ""
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
