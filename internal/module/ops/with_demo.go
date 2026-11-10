// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

const withDemoSettingKey = "modules.with_demo"

// metaAppSetting is the Go-side row for meta_app_setting (C2/TS AppSetting shape).
// Schema authority for this table is the meta module TS evolve path; Go only
// creates the table when missing (bootstrap latch) and otherwise does DML only.
type metaAppSetting struct {
	pkgmeta.BaseModel `gorm:"embedded"`
	Key               string `gorm:"column:key;type:varchar(255);not null;uniqueIndex"`
	Value             string `gorm:"column:value;type:text;not null"`
}

func (metaAppSetting) TableName() string { return "meta_app_setting" }

// ensureMetaAppSettingTable creates meta_app_setting only when absent.
// Never AutoMigrate: altering an existing table (e.g. varchar→char on audit
// uids) is owned by meta schema evolve / explicit migrations.
func ensureMetaAppSettingTable(db *gorm.DB) error {
	if db == nil {
		return errors.New("ops: nil db")
	}
	m := db.Migrator()
	if m.HasTable(&metaAppSetting{}) {
		return nil
	}
	return m.CreateTable(&metaAppSetting{})
}

// ResolveWithDemo reads meta_app_setting modules.with_demo; missing/invalid => false.
// Does not create the table; read misses stay false until SetWithDemo (or meta install).
func (o *Ops) ResolveWithDemo(ctx context.Context) bool {
	if o == nil {
		return false
	}
	sess, err := o.session(ctx)
	if err != nil {
		return false
	}
	var row metaAppSetting
	// Map condition so dialects quote reserved column name `key` (MySQL KEY).
	if err := sess.WithContext(ctx).Where(map[string]any{"key": withDemoSettingKey}).Take(&row).Error; err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(row.Value), "true")
}

// SetWithDemo upserts modules.with_demo. Only true is written (latch does not auto-clear).
// Uses find-then-update/create so it works when key is indexed but not UNIQUE
// (Postgres ON CONFLICT requires a matching unique/exclusion constraint).
func (o *Ops) SetWithDemo(ctx context.Context, enabled bool) error {
	if !enabled {
		return errors.New("ops: SetWithDemo only supports enabling (true)")
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	db := sess.WithContext(ctx)
	if err := ensureMetaAppSettingTable(db); err != nil {
		return err
	}
	var row metaAppSetting
	err = db.Where(map[string]any{"key": withDemoSettingKey}).Take(&row).Error
	if err == nil {
		return db.Model(&row).Update("value", "true").Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := db.Create(&metaAppSetting{
		Key:   withDemoSettingKey,
		Value: "true",
	}).Error; err != nil {
		// Lost a create race: another writer inserted the latch; update that row.
		var existing metaAppSetting
		takeErr := db.Where(map[string]any{"key": withDemoSettingKey}).Take(&existing).Error
		if takeErr == nil {
			return db.Model(&existing).Update("value", "true").Error
		}
		return errors.Join(err, takeErr)
	}
	return nil
}
