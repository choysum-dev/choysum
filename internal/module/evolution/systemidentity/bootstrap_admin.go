// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package systemidentity

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

// BootstrapAdmin is the seeded auth.user_admin principal (User.Id + company).
type BootstrapAdmin struct {
	UserID    string
	CompanyID string
}

// ResolveBootstrapAdmin returns auth.user_admin (+ company) when auth is installed.
// Missing auth / seeds yield ok=false (no inject). Operational DB errors are
// returned so callers do not fall back to a fake Username principal.
func ResolveBootstrapAdmin(ctx context.Context, runtimeScope scope.Scope) (BootstrapAdmin, bool, error) {
	var out BootstrapAdmin
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
	transactor := txRoot.Transactor()
	if transactor == nil {
		// Log-only / fixture scopes may omit a transactor; treat as no inject.
		return out, false, nil
	}
	err := transactor.Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
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
			return xfmt.Errorf("load auth module for bootstrap admin identity: %w", err)
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
			return xfmt.Errorf("load auth.user_admin mapping for bootstrap admin identity: %w", err)
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
			return xfmt.Errorf("lookup auth.User for bootstrap admin identity: %w", lookupErr)
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
			return xfmt.Errorf("load auth.user_admin row for bootstrap admin identity: %w", qErr)
		}

		companyID := strings.TrimSpace(row.CompanyID)
		if companyID == "" {
			var companyData modmeta.ModelData
			if err := session.Select("res_id").Where("module = ? AND name = ?", "base", "company_main").Take(&companyData).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return xfmt.Errorf("load base.company_main mapping for bootstrap admin identity: %w", err)
			}
			companyID = strings.TrimSpace(companyData.ResID)
		}
		if companyID == "" {
			return nil
		}

		out = BootstrapAdmin{UserID: userID, CompanyID: companyID}
		return nil
	})
	if err != nil {
		return BootstrapAdmin{}, false, err
	}
	return out, out.UserID != "" && out.CompanyID != "", nil
}

// ErrBootstrapAdminUnavailable is returned when auth is Installed but
// auth.user_admin cannot be resolved (no Username literal "admin" fallback).
var ErrBootstrapAdminUnavailable = errors.New("bootstrap admin identity unavailable (auth.user_admin)")

// ResolvePrincipalUserID returns a session identity UserID when present, otherwise
// ResolveBootstrapAdmin. Never returns the Username literal "admin".
//
// When auth is not installed (or the scope cannot see it), empty ids are returned
// without error so auth-free CLI/fixtures can still run lifecycle hooks. When auth
// is Installed but the seed is missing, ErrBootstrapAdminUnavailable is returned.
func ResolvePrincipalUserID(ctx context.Context, runtimeScope scope.Scope, sessionUserID string) (userID string, companyID string, err error) {
	sessionUserID = strings.TrimSpace(sessionUserID)
	if sessionUserID != "" {
		return sessionUserID, "", nil
	}
	admin, ok, resolveErr := ResolveBootstrapAdmin(ctx, runtimeScope)
	if resolveErr != nil {
		return "", "", resolveErr
	}
	if ok {
		return admin.UserID, admin.CompanyID, nil
	}
	installed, checkErr := authModuleInstalled(ctx, runtimeScope)
	if checkErr != nil {
		return "", "", checkErr
	}
	if installed {
		return "", "", ErrBootstrapAdminUnavailable
	}
	return "", "", nil
}

// authModuleInstalled reports whether the auth module row exists with Installed status.
func authModuleInstalled(ctx context.Context, runtimeScope scope.Scope) (bool, error) {
	if runtimeScope == nil {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	txRoot := runtimeScope.WithContext(ctx)
	if txRoot == nil {
		txRoot = runtimeScope
	}
	transactor := txRoot.Transactor()
	if transactor == nil {
		return false, nil
	}
	var installed bool
	err := transactor.Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
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
			return xfmt.Errorf("load auth module status for principal identity: %w", err)
		}
		installed = authMod.Status == meta.Installed
		return nil
	})
	return installed, err
}

// CompanyContextPayload builds the JS ctx map for active/enabled company scope.
func CompanyContextPayload(companyID string) map[string]any {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return map[string]any{}
	}
	return map[string]any{
		"activeCompanyId":   companyID,
		"enabledCompanyIds": []string{companyID},
	}
}
