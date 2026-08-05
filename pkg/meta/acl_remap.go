// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
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
	if len(oldToEffective) == 0 {
		return remapOrphanServices(db, effectiveIDSet)
	}

	tables := []string{
		"auth_role_method_access",
		"auth_role_record_rule",
		"auth_role_field_rule",
	}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		for oldID, newID := range oldToEffective {
			if err := db.Exec(
				fmt.Sprintf("UPDATE %s SET meta_model_id = ? WHERE meta_model_id = ?", table),
				newID, oldID,
			).Error; err != nil {
				return fmt.Errorf("remap %s meta_model_id %s→%s: %w", table, oldID, newID, err)
			}
		}
	}

	if err := remapFieldRuleFieldIDs(db, oldToEffective); err != nil {
		return err
	}
	return remapOrphanServices(db, effectiveIDSet)
}

func remapFieldRuleFieldIDs(db *gorm.DB, oldToEffective map[string]string) error {
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
			continue
		}
		name := strings.TrimSpace(oldField.Name)
		if name == "" {
			continue
		}
		var newField Field
		if takeErr := db.Where("model_id = ? AND name = ?", modelID, name).Take(&newField).Error; takeErr != nil {
			continue
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
		_ = oldToEffective // retained for call-site clarity
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
			continue
		}
		if svc.ModelId.Valid {
			if _, ok := effectiveIDSet[svc.ModelId.String]; ok {
				// Still under an effective model — leave alone (EDS-2 reuses service ids).
				var live Service
				if err := db.Where("id = ?", svcID).Take(&live).Error; err == nil {
					continue
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
			continue
		}
		eff, err := LookupEffectiveModel(db, hist.Application, hist.Name)
		if err != nil || eff == nil || !eff.Id.Valid {
			continue
		}
		var replacement Service
		if err := db.Where("model_id = ? AND name = ?", eff.Id.String, name).Take(&replacement).Error; err != nil {
			continue
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
