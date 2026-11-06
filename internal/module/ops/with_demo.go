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
type metaAppSetting struct {
	pkgmeta.BaseModel `gorm:"embedded"`
	Key               string `gorm:"column:key;type:varchar(255);not null;uniqueIndex"`
	Value             string `gorm:"column:value;type:text;not null"`
}

func (metaAppSetting) TableName() string { return "meta_app_setting" }

func (o *Ops) ensureAppSettingTable(ctx context.Context) error {
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	return sess.WithContext(ctx).AutoMigrate(&metaAppSetting{})
}

// ResolveWithDemo reads meta_app_setting modules.with_demo; missing/invalid => false.
func (o *Ops) ResolveWithDemo(ctx context.Context) bool {
	if o == nil {
		return false
	}
	if err := o.ensureAppSettingTable(ctx); err != nil {
		return false
	}
	sess, err := o.session(ctx)
	if err != nil {
		return false
	}
	var row metaAppSetting
	if err := sess.WithContext(ctx).Where("key = ?", withDemoSettingKey).Take(&row).Error; err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(row.Value), "true")
}

// SetWithDemo upserts modules.with_demo. Only true is written (latch does not auto-clear).
func (o *Ops) SetWithDemo(ctx context.Context, enabled bool) error {
	if !enabled {
		return errors.New("ops: SetWithDemo only supports enabling (true)")
	}
	if err := o.ensureAppSettingTable(ctx); err != nil {
		return err
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	var existing metaAppSetting
	err = sess.WithContext(ctx).Where("key = ?", withDemoSettingKey).Take(&existing).Error
	if err == nil {
		return sess.WithContext(ctx).Model(&existing).Update("value", "true").Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return sess.WithContext(ctx).Create(&metaAppSetting{
		Key:   withDemoSettingKey,
		Value: "true",
	}).Error
}
