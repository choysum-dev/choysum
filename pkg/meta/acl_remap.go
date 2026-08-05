// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// RemapACLToEffectiveModelIDs rewrites auth ACL meta_model_id (and related field/service
// FKs) from historical/shell meta_model ids onto the live effective id per (application, name).
//
// Safe to run repeatedly. Missing auth_* tables are skipped (fresh DB before auth install).
// Intended for one-shot upgrades after dual-store migrate; wipe+reseed also works.
func RemapACLToEffectiveModelIDs(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	var live []Model
	if err := db.Select("id", "application", "name", "module_id", "updated_at").Find(&live).Error; err != nil {
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

	// Map any historical id (including soft-deleted) → effective id by (app, name).
	var allIncludingDeleted []Model
	if err := db.Unscoped().Select("id", "application", "name").Find(&allIncludingDeleted).Error; err != nil {
		return fmt.Errorf("load meta_model including deleted: %w", err)
	}
	oldToEffective := map[string]string{}
	for _, row := range allIncludingDeleted {
		if !row.Id.Valid {
			continue
		}
		key := strings.TrimSpace(row.Application) + "\x00" + strings.TrimSpace(row.Name)
		eff, ok := effectiveByKey[key]
		if !ok || !eff.Id.Valid {
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
			if !tx.Migrator().HasTable(table) {
				continue
			}
			for oldID, newID := range oldToEffective {
				if err := tx.Exec(
					fmt.Sprintf("UPDATE %s SET meta_model_id = ? WHERE meta_model_id = ?", table),
					newID, oldID,
				).Error; err != nil {
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
	if !db.Migrator().HasTable("auth_role_field_rule") {
		return nil
	}
	type fieldRuleRow struct {
		ID          string `gorm:"column:id"`
		MetaModelID string `gorm:"column:meta_model_id"`
		MetaFieldID string `gorm:"column:meta_field_id"`
	}
	var rules []fieldRuleRow
	if err := db.Table("auth_role_field_rule").
		Select("id", "meta_model_id", "meta_field_id").
		Where("meta_field_id IS NOT NULL AND meta_field_id <> ''").
		Find(&rules).Error; err != nil {
		return fmt.Errorf("load field rules for remap: %w", err)
	}

	for _, r := range rules {
		fieldID := strings.TrimSpace(r.MetaFieldID)
		modelID := strings.TrimSpace(r.MetaModelID)
		if fieldID == "" || modelID == "" {
			continue
		}
		var oldField Field
		// Field may still exist under the new model, or only under a deleted tree.
		err := db.Unscoped().Select("id", "name", "model_id").Where("id = ?", fieldID).Take(&oldField).Error
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
		if takeErr := db.Where("model_id = ? AND name = ?", modelID, name).Take(&newField).Error; takeErr != nil {
			if errors.Is(takeErr, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup field rule %s replacement field %s/%s: %w", r.ID, modelID, name, takeErr)
		}
		if !newField.Id.Valid || newField.Id.String == fieldID {
			continue
		}
		if err := db.Exec(
			"UPDATE auth_role_field_rule SET meta_field_id = ? WHERE id = ?",
			newField.Id.String, r.ID,
		).Error; err != nil {
			return fmt.Errorf("remap field rule %s field_id: %w", r.ID, err)
		}
	}
	return nil
}

func remapOrphanServices(db *gorm.DB, effectiveIDSet map[string]struct{}) error {
	if !db.Migrator().HasTable("auth_role_method_access") {
		return nil
	}
	type maRow struct {
		ID            string `gorm:"column:id"`
		MetaServiceID string `gorm:"column:meta_service_id"`
	}
	var rows []maRow
	if err := db.Table("auth_role_method_access").
		Select("id", "meta_service_id").
		Where("meta_service_id IS NOT NULL AND meta_service_id <> ''").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load method access for service remap: %w", err)
	}
	for _, r := range rows {
		svcID := strings.TrimSpace(r.MetaServiceID)
		if svcID == "" {
			continue
		}
		var svc Service
		if err := db.Unscoped().Select("id", "name", "model_id").Where("id = ?", svcID).Take(&svc).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup method access %s service %s: %w", r.ID, svcID, err)
		}
		if svc.ModelId.Valid {
			if _, ok := effectiveIDSet[svc.ModelId.String]; ok {
				// Still under an effective model — leave alone if the service row is live
				// (EDS-2 reuses service ids). Soft-deleted under effective → fall through.
				var live Service
				if err := db.Where("id = ?", svcID).Take(&live).Error; err == nil {
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
		// Resolve effective model for the service's historical model, then find service by name.
		var hist Model
		if err := db.Unscoped().Select("id", "application", "name").Where("id = ?", svc.ModelId.String).Take(&hist).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup historical model for service %s: %w", svcID, err)
		}
		eff, err := LookupEffectiveModel(db, hist.Application, hist.Name)
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
		if err := db.Where("model_id = ? AND name = ?", eff.Id.String, name).Take(&replacement).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("lookup replacement service %s/%s: %w", eff.Id.String, name, err)
		}
		if !replacement.Id.Valid || replacement.Id.String == svcID {
			continue
		}
		if err := db.Exec(
			"UPDATE auth_role_method_access SET meta_service_id = ? WHERE id = ?",
			replacement.Id.String, r.ID,
		).Error; err != nil {
			return fmt.Errorf("remap method access %s service_id: %w", r.ID, err)
		}
	}
	return nil
}
