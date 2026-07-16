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
	i18nModelName              = "I18n"
	terminologyEditorRoleCode  = "terminology.editor"
	authRoleTable              = "auth_role"
	authRoleMethodAccessTable  = "auth_role_method_access"
)

var i18nServiceMethods = []string{
	"GetTranslations",
	"SearchTerms",
	"UpdateTerm",
}

// EnsureI18nIrMeta registers meta.IrModel "I18n" + IrService methods for ACL
// (serviceRef / CheckMethodAccess). Does not register TranslationTerm.
// When the Terminology Editor role exists, seeds precise allow rows for
// SearchTerms and UpdateTerm (idempotent).
func EnsureI18nIrMeta(runtimeScope scope.Scope, application string, moduleID sql.NullString) error {
	application = strings.TrimSpace(application)
	if application == "" || application == coreApplication {
		return nil
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	db := runtimeScope.Session().DB
	if !db.Migrator().HasTable((&meta.IrModel{}).TableName()) || !db.Migrator().HasTable((&meta.IrService{}).TableName()) {
		return nil
	}

	model, err := ensureI18nModel(db, application, moduleID)
	if err != nil {
		return err
	}
	serviceIDs, err := ensureI18nServices(db, model)
	if err != nil {
		return err
	}
	return ensureTerminologyEditorAllows(db, serviceIDs)
}

func ensureI18nModel(db *gorm.DB, application string, moduleID sql.NullString) (*meta.IrModel, error) {
	var model meta.IrModel
	err := db.Where("name = ? AND application = ?", i18nModelName, application).Take(&model).Error
	if err == nil {
		if moduleID.Valid && (!model.ModuleId.Valid || model.ModuleId.String == "") {
			model.ModuleId = moduleID
			if saveErr := db.Save(&model).Error; saveErr != nil {
				return nil, fmt.Errorf("update I18n IrModel module: %w", saveErr)
			}
		}
		return &model, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup I18n IrModel: %w", err)
	}

	model = meta.IrModel{
		Name:        i18nModelName,
		Path:        fmt.Sprintf("go://i18n/%s", application),
		Application: application,
		ClassName:   i18nModelName,
		Abstract:    true,
		Readonly:    true,
		ModuleId:    moduleID,
	}
	falseVal := false
	model.AutoMigrate = &falseVal
	if err := db.Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create I18n IrModel: %w", err)
	}
	return &model, nil
}

func ensureI18nServices(db *gorm.DB, model *meta.IrModel) (map[string]string, error) {
	out := make(map[string]string, len(i18nServiceMethods))
	for _, methodName := range i18nServiceMethods {
		var svc meta.IrService
		err := db.Where("model_id = ? AND name = ?", model.Id.String, methodName).Take(&svc).Error
		if err == nil {
			out[methodName] = svc.Id.String
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("lookup IrService %s: %w", methodName, err)
		}
		svc = meta.IrService{
			Name:                 methodName,
			OriginModelPath:      model.Path,
			AccessibilityModifier: "public",
			IsStatic:             true,
			ModelId:              model.Id,
		}
		if err := db.Create(&svc).Error; err != nil {
			return nil, fmt.Errorf("create IrService %s: %w", methodName, err)
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
			Where("role_id = ? AND ir_service_id = ? AND deleted_at IS NULL", roleID, serviceID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("lookup RoleMethodAccess: %w", err)
		}
		if count > 0 {
			continue
		}
		now := time.Now().UTC()
		row := map[string]any{
			"id":                xid.New().String(),
			"role_id":           roleID,
			"ir_application_id": nil,
			"ir_model_id":       nil,
			"ir_service_id":     serviceID,
			"mode":              "allow",
			"source":            "manual",
			"created_at":        now,
			"updated_at":        now,
		}
		if err := db.Table(authRoleMethodAccessTable).Create(row).Error; err != nil {
			return fmt.Errorf("seed RoleMethodAccess for %s: %w", methodName, err)
		}
	}
	return nil
}
