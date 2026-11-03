// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"strings"
	"time"

	"github.com/rs/xid"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	retiredMetaSettingTable = "meta_setting"
	legacySystemInitDoneKey = "system.init.done"
	authAppSettingTable     = "auth_app_setting"
	authSetupCompletedAtKey = "setup.completed_at"
)

// RetireMetaSettingTable drops the retired meta_setting table after best-effort
// backfill of system.init.done into auth_app_setting.setup.completed_at.
// Safe when the table is already absent. When auth_app_setting is missing but a
// truthy legacy marker remains, the table is kept so a later call can backfill.
func RetireMetaSettingTable(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	migrator := db.Migrator()
	if !migrator.HasTable(retiredMetaSettingTable) {
		return nil
	}
	canDrop, err := backfillSetupCompletedAtFromLegacyInit(db)
	if err != nil {
		return xfmt.Errorf("backfill setup.completed_at from legacy meta_setting: %w", err)
	}
	if !canDrop {
		return nil
	}
	if err := migrator.DropTable(retiredMetaSettingTable); err != nil {
		return xfmt.Errorf("drop retired meta_setting: %w", err)
	}
	return nil
}

// backfillSetupCompletedAtFromLegacyInit copies a truthy system.init.done into
// auth_app_setting.setup.completed_at when needed. canDrop is false when the
// legacy marker must be preserved until auth_app_setting exists.
func backfillSetupCompletedAtFromLegacyInit(db *gorm.DB) (canDrop bool, err error) {
	if !db.Migrator().HasTable(authAppSettingTable) {
		truthy, legacyErr := legacySystemInitDoneTruthy(db)
		if legacyErr != nil {
			return false, legacyErr
		}
		if truthy {
			// Keep meta_setting until auth is installed and backfill can run.
			return false, nil
		}
		return true, nil
	}
	var existing struct {
		Value string `gorm:"column:value"`
	}
	err = db.Table(authAppSettingTable).
		Select("value").
		Where(map[string]any{"key": authSetupCompletedAtKey}).
		Take(&existing).Error
	if err == nil && strings.TrimSpace(existing.Value) != "" {
		return true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	// Blank row still occupies the unique key; free it before create-only claim.
	if err == nil {
		if delErr := db.Table(authAppSettingTable).
			Where(map[string]any{"key": authSetupCompletedAtKey}).
			Delete(nil).Error; delErr != nil {
			return false, delErr
		}
	}

	truthy, legacyErr := legacySystemInitDoneTruthy(db)
	if legacyErr != nil {
		return false, legacyErr
	}
	if !truthy {
		return true, nil
	}

	now := time.Now().UTC()
	iso := now.Format(time.RFC3339)
	type authAppSettingRow struct {
		Id        string    `gorm:"column:id;primaryKey"`
		Key       string    `gorm:"column:key"`
		Value     string    `gorm:"column:value"`
		CreatedAt time.Time `gorm:"column:created_at"`
		UpdatedAt time.Time `gorm:"column:updated_at"`
	}
	// DoNothing (no Columns): dialect-portable create-only claim — SQLite/Postgres
	// ON CONFLICT DO NOTHING, MySQL INSERT IGNORE — never clobber a concurrent marker.
	if err := db.Table(authAppSettingTable).Clauses(clause.OnConflict{DoNothing: true}).Create(&authAppSettingRow{
		Id:        xid.New().String(),
		Key:       authSetupCompletedAtKey,
		Value:     iso,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		return false, err
	}
	return true, nil
}

func legacySystemInitDoneTruthy(db *gorm.DB) (bool, error) {
	var legacy struct {
		Value string `gorm:"column:value"`
	}
	err := db.Table(retiredMetaSettingTable).
		Select("value").
		Where(map[string]any{"key": legacySystemInitDoneKey}).
		Take(&legacy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	v := strings.ToLower(strings.TrimSpace(legacy.Value))
	return v == "true" || v == "1" || v == "yes", nil
}
