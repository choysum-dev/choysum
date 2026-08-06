// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

const (
	i18nModelName             = "I18n"
	translationTermModelName  = "TranslationTerm"
	terminologyEditorRoleCode = "terminology.editor"
	authRoleTable             = "auth_role"
	authRoleMethodAccessTable = "auth_role_method_access"
)

var i18nServiceMethods = []string{
	"GetTranslations",
	"SearchTerms",
	"UpdateTerm",
}

// TranslationTerm methods bound to terminology.editor (never GetTranslations).
// Choysum ORM uses Browse (not Read) for single-record fetch.
var terminologyEditorServiceMethods = []string{
	"Search",
	"Browse",
	"Update",
}

// EnsureI18nMeta registers declaration-layer I18n + Service methods via the meta
// declaration facade, flushes the effective projection, and seeds Terminology
// Editor ACL rows against TranslationTerm Search/Browse/Update (not I18n methods).
func EnsureI18nMeta(runtimeScope scope.Scope, application string, moduleID sql.NullString) error {
	application = strings.TrimSpace(application)
	if application == "" || application == coreApplication {
		return nil
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	db := runtimeScope.Session().DB
	if !meta.HasDeclarationCatalog(db) || !meta.HasEffectiveCatalog(db) {
		return nil
	}

	path := fmt.Sprintf("go://i18n/%s", application)
	if err := meta.EnsureAbstractModel(db, meta.AbstractModelSpec{
		Name:         i18nModelName,
		Path:         path,
		Application:  application,
		ModuleID:     moduleID,
		ServiceNames: i18nServiceMethods,
	}); err != nil {
		return err
	}
	// Flush before any effective reads (install boundary for I18n).
	if err := meta.FlushEffective(db, []meta.LogicalKey{{Application: application, Name: i18nModelName}}); err != nil {
		return fmt.Errorf("flush I18n effective: %w", err)
	}
	return ensureTerminologyEditorAllows(db, application)
}

func loadEffectiveTranslationTermServiceIDs(db *gorm.DB, application string) (map[string]string, error) {
	var model meta.Model
	if err := db.Where("name = ? AND application = ?", translationTermModelName, application).Take(&model).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(terminologyEditorServiceMethods))
	for _, methodName := range terminologyEditorServiceMethods {
		var svc meta.Service
		err := db.Where("model_id = ? AND name = ?", model.Id.String, methodName).Take(&svc).Error
		if err != nil {
			return nil, fmt.Errorf("lookup effective Service %s: %w", methodName, err)
		}
		out[methodName] = svc.Id.String
	}
	return out, nil
}

func ensureTerminologyEditorAllows(db *gorm.DB, application string) error {
	if !db.Migrator().HasTable(authRoleTable) || !db.Migrator().HasTable(authRoleMethodAccessTable) {
		return nil
	}
	var roleID string
	if err := db.Table(authRoleTable).
		Select("id").
		Where("code = ? AND deleted_at IS NULL", terminologyEditorRoleCode).
		Limit(1).
		Scan(&roleID).Error; err != nil {
		return fmt.Errorf("lookup Terminology Editor role: %w", err)
	}
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil
	}

	serviceIDs, err := loadEffectiveTranslationTermServiceIDs(db, application)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// TranslationTerm not registered for this app yet (Ensure-only timing).
			return nil
		}
		return err
	}

	for _, methodName := range terminologyEditorServiceMethods {
		serviceID := strings.TrimSpace(serviceIDs[methodName])
		if serviceID == "" {
			continue
		}
		var count int64
		if err := db.Table(authRoleMethodAccessTable).
			Where("role_id = ? AND meta_service_id = ? AND deleted_at IS NULL", roleID, serviceID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("lookup RoleMethodAccess: %w", err)
		}
		if count > 0 {
			continue
		}
		now := time.Now().UTC()
		row := map[string]any{
			"id":                  xid.New().String(),
			"role_id":             roleID,
			"meta_application_id": nil,
			"meta_model_id":       nil,
			"meta_service_id":     serviceID,
			"mode":                "allow",
			"source":              "manual",
			"created_at":          now,
			"updated_at":          now,
		}
		if err := db.Table(authRoleMethodAccessTable).Create(row).Error; err != nil {
			return fmt.Errorf("seed RoleMethodAccess for %s: %w", methodName, err)
		}
	}
	return nil
}
