// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm/clause"
)

const withDemoSettingKey = "modules.with_demo"

// metaAppSetting is the Go-side row for meta_app_setting (C2/TS AppSetting shape).
type metaAppSetting struct {
	pkgmeta.BaseModel `gorm:"embedded"`
	Key               string `gorm:"column:key;type:varchar(255);not null;uniqueIndex"`
	Value             string `gorm:"column:value;type:text;not null"`
}

func (metaAppSetting) TableName() string { return "meta_app_setting" }

// ResolveWithDemo reads meta_app_setting modules.with_demo; missing/invalid => false.
// Does not AutoMigrate: latch writes (SetWithDemo) create the table; read misses stay false.
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
func (o *Ops) SetWithDemo(ctx context.Context, enabled bool) error {
	if !enabled {
		return errors.New("ops: SetWithDemo only supports enabling (true)")
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	if err := sess.WithContext(ctx).AutoMigrate(&metaAppSetting{}); err != nil {
		return err
	}
	return sess.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&metaAppSetting{
		Key:   withDemoSettingKey,
		Value: "true",
	}).Error
}
