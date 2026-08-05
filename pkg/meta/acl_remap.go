// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Test hooks (production defaults). Override in *_test.go to force error paths.
var (
	aclLoadLiveModels = func(db *gorm.DB) ([]Model, error) {
		var live []Model
		err := db.Select("id", "application", "name", "module_id", "updated_at").Find(&live).Error
		return live, err
	}
	aclLoadAllModels = func(db *gorm.DB) ([]Model, error) {
		var all []Model
		err := db.Unscoped().Select("id", "application", "name").Find(&all).Error
		return all, err
	}
	aclExec = func(db *gorm.DB, sql string, values ...interface{}) error {
		return db.Exec(sql, values...).Error
	}
	aclHasTable = func(db *gorm.DB, name string) bool {
		return db.Migrator().HasTable(name)
	}
	aclLoadFieldRules = func(db *gorm.DB, dest interface{}) error {
		return db.Table("auth_role_field_rule").
			Select("id", "meta_model_id", "meta_field_id").
			Where("meta_field_id IS NOT NULL AND meta_field_id <> ''").
			Find(dest).Error
	}
	aclTakeFieldUnscoped = func(db *gorm.DB, dest interface{}, fieldID string) error {
		return db.Unscoped().Select("id", "name", "model_id").Where("id = ?", fieldID).Take(dest).Error
	}
	aclTakeFieldByModelName = func(db *gorm.DB, dest interface{}, modelID, name string) error {
		return db.Where("model_id = ? AND name = ?", modelID, name).Take(dest).Error
	}
	aclLoadMethodAccess = func(db *gorm.DB, dest interface{}) error {
		return db.Table("auth_role_method_access").
			Select("id", "meta_service_id").
			Where("meta_service_id IS NOT NULL AND meta_service_id <> ''").
			Find(dest).Error
	}
	aclTakeServiceUnscoped = func(db *gorm.DB, dest interface{}, svcID string) error {
		return db.Unscoped().Select("id", "name", "model_id").Where("id = ?", svcID).Take(dest).Error
	}
	aclTakeServiceLive = func(db *gorm.DB, dest interface{}, svcID string) error {
		return db.Where("id = ?", svcID).Take(dest).Error
	}
	aclTakeModelUnscoped = func(db *gorm.DB, dest interface{}, modelID string) error {
		return db.Unscoped().Select("id", "application", "name").Where("id = ?", modelID).Take(dest).Error
	}
	aclLookupEffective = LookupEffectiveModel
	aclTakeServiceByModelName = func(db *gorm.DB, dest interface{}, modelID, name string) error {
		return db.Where("model_id = ? AND name = ?", modelID, name).Take(dest).Error
	}
)

// remapACLToEffectiveModelIDs rewrites auth ACL meta_model_id (and related field/service
// FKs) from historical/shell meta_model ids onto the live effective id per (application, name).
//
// Safe to run repeatedly. Missing auth_* tables are skipped (fresh DB before auth install).
// Intended for one-shot upgrades after dual-store migrate; wipe+reseed also works.
func remapACLToEffectiveModelIDs(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	live, err := aclLoadLiveModels(db)
	if err != nil {
		return fmt.Errorf("load live meta_model: %w", err)
	}
	effectiveByKey := map[string]Model{}
	for _, row := range live {
		key := strings.TrimSpace(row.Application) + "\x00" + strings.TrimSpace(row.Name)
		if key == "\x00" || !row.Id.Valid {
			continue
		}
		if existing, ok := effectiveByKey[key]; ok {
			picked := pickEffectiveAmong([]Model{existing, row})
			effectiveByKey[key] = picked
			continue
		}
		effectiveByKey[key] = row
	}
	if len(effectiveByKey) == 0 {
		return nil
	}

	effectiveIDSet := map[string]struct{}{}
	for _, m := range effectiveByKey {
		effectiveIDSet[m.Id.String] = struct{}{}
	}

	allIncludingDeleted, err := aclLoadAllModels(db)
	if err != nil {
		return fmt.Errorf("load meta_model including deleted: %w", err)
	}
	oldToEffective := map[string]string{}
	for _, row := range allIncludingDeleted {
		if !row.Id.Valid {
			continue
		}
		key := strings.TrimSpace(row.Application) + "\x00" + strings.TrimSpace(row.Name)
		eff, ok := effectiveByKey[key]
		if !ok {
			continue
		}
		if row.Id.String == eff.Id.String {
			continue
		}
		oldToEffective[row.Id.String] = eff.Id.String
	}

	return db.Transaction(func(tx *gorm.DB) error {
		tables := []string{
			"auth_role_method_access",
			"auth_role_record_rule",
			"auth_role_field_rule",
		}
		for _, table := range tables {
			if !aclHasTable(tx, table) {
				continue
			}
			for oldID, newID := range oldToEffective {
				if err := aclExec(tx,
					fmt.Sprintf("UPDATE %s SET meta_model_id = ? WHERE meta_model_id = ?", table),
					newID, oldID,
				); err != nil {
					return fmt.Errorf("remap %s meta_model_id %s→%s: %w", table, oldID, newID, err)
				}
			}
		}

		// Always remap field FKs: meta_model_id may already be effective while
		// meta_field_id still points at a replaced subtree after recompute.
		if err := remapFieldRuleFieldIDs(tx); err != nil {
			return err
		}
		return remapOrphanServices(tx, effectiveIDSet)
	})
}

func remapFieldRuleFieldIDs(db *gorm.DB) error {
	if !aclHasTable(db, "auth_role_field_rule") {
		return nil
	}
	type fieldRuleRow struct {
		ID          string `gorm:"column:id"`
		MetaModelID string `gorm:"column:meta_model_id"`
		MetaFieldID string `gorm:"column:meta_field_id"`
	}
	var rules []fieldRuleRow
	if err := aclLoadFieldRules(db, &rules); err != nil {
		return fmt.Errorf("load field rules for remap: %w", err)
	}

	for _, r := range rules {
		fieldID := strings.TrimSpace(r.MetaFieldID)
		modelID := strings.TrimSpace(r.MetaModelID)
		if fieldID == "" || modelID == "" {
			continue
		}
		var oldField Field
		err := aclTakeFieldUnscoped(db, &oldField, fieldID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup field rule %s field %s: %w", r.ID, fieldID, err)
		}
		name := strings.TrimSpace(oldField.Name)
		if name == "" {
			continue
		}
		var newField Field
		if takeErr := aclTakeFieldByModelName(db, &newField, modelID, name); takeErr != nil {
			if errors.Is(takeErr, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup field rule %s replacement field %s/%s: %w", r.ID, modelID, name, takeErr)
		}
		if !newField.Id.Valid || newField.Id.String == fieldID {
			continue
		}
		if err := aclExec(db,
			"UPDATE auth_role_field_rule SET meta_field_id = ? WHERE id = ?",
			newField.Id.String, r.ID,
		); err != nil {
			return fmt.Errorf("remap field rule %s field_id: %w", r.ID, err)
		}
	}
	return nil
}

func remapOrphanServices(db *gorm.DB, effectiveIDSet map[string]struct{}) error {
	if !aclHasTable(db, "auth_role_method_access") {
		return nil
	}
	type maRow struct {
		ID            string `gorm:"column:id"`
		MetaServiceID string `gorm:"column:meta_service_id"`
	}
	var rows []maRow
	if err := aclLoadMethodAccess(db, &rows); err != nil {
		return fmt.Errorf("load method access for service remap: %w", err)
	}
	for _, r := range rows {
		svcID := strings.TrimSpace(r.MetaServiceID)
		if svcID == "" {
			continue
		}
		var svc Service
		if err := aclTakeServiceUnscoped(db, &svc, svcID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup method access %s service %s: %w", r.ID, svcID, err)
		}
		if svc.ModelId.Valid {
			if _, ok := effectiveIDSet[svc.ModelId.String]; ok {
				var live Service
				if err := aclTakeServiceLive(db, &live, svcID); err == nil {
					continue
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("lookup live service %s: %w", svcID, err)
				}
			}
		}
		name := strings.TrimSpace(svc.Name)
		if name == "" || !svc.ModelId.Valid {
			continue
		}
		var hist Model
		if err := aclTakeModelUnscoped(db, &hist, svc.ModelId.String); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup historical model for service %s: %w", svcID, err)
		}
		if strings.TrimSpace(hist.Application) == "" || strings.TrimSpace(hist.Name) == "" {
			continue
		}
		eff, err := aclLookupEffective(db, hist.Application, hist.Name)
		if err != nil {
			if IsEffectiveModelNotFound(err) {
				continue
			}
			return fmt.Errorf("lookup effective model for service %s: %w", svcID, err)
		}
		if eff == nil || !eff.Id.Valid {
			continue
		}
		var replacement Service
		if err := aclTakeServiceByModelName(db, &replacement, eff.Id.String, name); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup replacement service %s/%s: %w", eff.Id.String, name, err)
		}
		if !replacement.Id.Valid || replacement.Id.String == svcID {
			continue
		}
		if err := aclExec(db,
			"UPDATE auth_role_method_access SET meta_service_id = ? WHERE id = ?",
			replacement.Id.String, r.ID,
		); err != nil {
			return fmt.Errorf("remap method access %s service_id: %w", r.ID, err)
		}
	}
	return nil
}
