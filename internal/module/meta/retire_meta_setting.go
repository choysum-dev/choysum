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
// Safe when the table is already absent.
func RetireMetaSettingTable(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	migrator := db.Migrator()
	if !migrator.HasTable(retiredMetaSettingTable) {
		return nil
	}
	if err := backfillSetupCompletedAtFromLegacyInit(db); err != nil {
		return xfmt.Errorf("backfill setup.completed_at from legacy meta_setting: %w", err)
	}
	if err := migrator.DropTable(retiredMetaSettingTable); err != nil {
		return xfmt.Errorf("drop retired meta_setting: %w", err)
	}
	return nil
}

func backfillSetupCompletedAtFromLegacyInit(db *gorm.DB) error {
	if !db.Migrator().HasTable(authAppSettingTable) {
		// Auth not installed yet — nothing to preserve for SetupComplete.
		return nil
	}
	var existing struct {
		Value string `gorm:"column:value"`
	}
	err := db.Table(authAppSettingTable).
		Select("value").
		Where(map[string]any{"key": authSetupCompletedAtKey}).
		Take(&existing).Error
	if err == nil && strings.TrimSpace(existing.Value) != "" {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var legacy struct {
		Value string `gorm:"column:value"`
	}
	err = db.Table(retiredMetaSettingTable).
		Select("value").
		Where(map[string]any{"key": legacySystemInitDoneKey}).
		Take(&legacy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	v := strings.ToLower(strings.TrimSpace(legacy.Value))
	if v != "true" && v != "1" && v != "yes" {
		return nil
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
	return db.Table(authAppSettingTable).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{"value": iso, "updated_at": now}),
	}).Create(&authAppSettingRow{
		Id:        xid.New().String(),
		Key:       authSetupCompletedAtKey,
		Value:     iso,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error
}
