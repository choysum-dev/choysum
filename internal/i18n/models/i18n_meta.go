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

// PO export still dials {app}.I18n/SearchTerms with the caller's token, so
// terminology.editor must keep SearchTerms allow until export switches to
// TranslationTerm Search.
const terminologyEditorPOExportMethod = "SearchTerms"

// EnsureI18nMeta registers declaration-layer I18n + Service methods via the meta
// declaration facade, flushes the effective projection, and seeds Terminology
// Editor ACL rows against TranslationTerm Search/Browse/Update plus I18n
// SearchTerms (PO download). Never binds GetTranslations / UpdateTerm.
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

func loadEffectiveI18nSearchTermsServiceID(db *gorm.DB, application string) (string, error) {
	var model meta.Model
	if err := db.Where("name = ? AND application = ?", i18nModelName, application).Take(&model).Error; err != nil {
		return "", err
	}
	var svc meta.Service
	if err := db.Where("model_id = ? AND name = ?", model.Id.String, terminologyEditorPOExportMethod).Take(&svc).Error; err != nil {
		return "", fmt.Errorf("lookup effective Service %s: %w", terminologyEditorPOExportMethod, err)
	}
	return svc.Id.String, nil
}

func seedRoleMethodAllow(db *gorm.DB, roleID, serviceID, methodName string) error {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return nil
	}
	var count int64
	if err := db.Table(authRoleMethodAccessTable).
		Where("role_id = ? AND meta_service_id = ? AND deleted_at IS NULL", roleID, serviceID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("lookup RoleMethodAccess: %w", err)
	}
	if count > 0 {
		return nil
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
	return nil
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
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// TranslationTerm not registered for this app yet (Ensure-only timing).
	} else {
		for _, methodName := range terminologyEditorServiceMethods {
			if err := seedRoleMethodAllow(db, roleID, serviceIDs[methodName], methodName); err != nil {
				return err
			}
		}
	}

	searchTermsID, err := loadEffectiveI18nSearchTermsServiceID(db, application)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return seedRoleMethodAllow(db, roleID, searchTermsID, terminologyEditorPOExportMethod)
}
