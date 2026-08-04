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
	terminologyEditorRoleCode = "terminology.editor"
	authRoleTable             = "auth_role"
	authRoleMethodAccessTable = "auth_role_method_access"
)

var i18nServiceMethods = []string{
	"GetTranslations",
	"SearchTerms",
	"UpdateTerm",
}

// EnsureI18nMeta registers declaration-layer I18n + Service methods on meta_raw_*,
// recomputes the effective projection, and seeds Terminology Editor ACL rows against
// effective service ids (serviceRef / CheckMethodAccess). Does not register TranslationTerm.
func EnsureI18nMeta(runtimeScope scope.Scope, application string, moduleID sql.NullString) error {
	application = strings.TrimSpace(application)
	if application == "" || application == coreApplication {
		return nil
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	db := runtimeScope.Session().DB
	if !db.Migrator().HasTable((&meta.RawModel{}).TableName()) ||
		!db.Migrator().HasTable((&meta.RawService{}).TableName()) ||
		!db.Migrator().HasTable((&meta.Model{}).TableName()) ||
		!db.Migrator().HasTable((&meta.Service{}).TableName()) {
		return nil
	}

	raw, err := ensureI18nRawModel(db, application, moduleID)
	if err != nil {
		return err
	}
	if err := ensureI18nRawServices(db, raw); err != nil {
		return err
	}
	if err := meta.RecomputeEffective(db, application, i18nModelName); err != nil {
		return fmt.Errorf("recompute I18n effective: %w", err)
	}
	serviceIDs, err := loadEffectiveI18nServiceIDs(db, application)
	if err != nil {
		return err
	}
	return ensureTerminologyEditorAllows(db, serviceIDs)
}

func ensureI18nRawModel(db *gorm.DB, application string, moduleID sql.NullString) (*meta.RawModel, error) {
	path := fmt.Sprintf("go://i18n/%s", application)
	var raw meta.RawModel
	err := db.Where("name = ? AND application = ?", i18nModelName, application).
		Order("created_at DESC, id DESC").
		Take(&raw).Error
	if err == nil {
		if moduleID.Valid && (!raw.ModuleId.Valid || raw.ModuleId.String == "") {
			raw.ModuleId = moduleID
			if saveErr := db.Save(&raw).Error; saveErr != nil {
				return nil, fmt.Errorf("update I18n raw Model module: %w", saveErr)
			}
		}
		return &raw, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup I18n raw Model: %w", err)
	}

	raw = meta.RawModel{
		Name:        i18nModelName,
		Path:        path,
		Application: application,
		ClassName:   i18nModelName,
		Abstract:    true,
		Readonly:    true,
		ModuleId:    moduleID,
	}
	falseVal := false
	raw.AutoMigrate = &falseVal
	if err := db.Create(&raw).Error; err != nil {
		return nil, fmt.Errorf("create I18n raw Model: %w", err)
	}
	return &raw, nil
}

func ensureI18nRawServices(db *gorm.DB, raw *meta.RawModel) error {
	if raw == nil || !raw.Id.Valid {
		return fmt.Errorf("I18n raw Model is nil")
	}
	for _, methodName := range i18nServiceMethods {
		var svc meta.RawService
		err := db.Where("model_id = ? AND name = ?", raw.Id.String, methodName).Take(&svc).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("lookup raw Service %s: %w", methodName, err)
		}
		svc = meta.RawService{
			Name:                  methodName,
			OriginModelPath:       raw.Path,
			AccessibilityModifier: "public",
			IsStatic:              true,
			ModelId:               raw.Id,
		}
		if err := db.Create(&svc).Error; err != nil {
			return fmt.Errorf("create raw Service %s: %w", methodName, err)
		}
	}
	return nil
}

func loadEffectiveI18nServiceIDs(db *gorm.DB, application string) (map[string]string, error) {
	var model meta.Model
	if err := db.Where("name = ? AND application = ?", i18nModelName, application).Take(&model).Error; err != nil {
		return nil, fmt.Errorf("lookup I18n effective Model: %w", err)
	}
	out := make(map[string]string, len(i18nServiceMethods))
	for _, methodName := range i18nServiceMethods {
		var svc meta.Service
		err := db.Where("model_id = ? AND name = ?", model.Id.String, methodName).Take(&svc).Error
		if err != nil {
			return nil, fmt.Errorf("lookup effective Service %s: %w", methodName, err)
		}
		out[methodName] = svc.Id.String
	}
	return out, nil
}

func ensureTerminologyEditorAllows(db *gorm.DB, serviceIDs map[string]string) error {
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

	for _, methodName := range []string{"SearchTerms", "UpdateTerm"} {
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
