// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

// ModuleCatalog is one row per module name for the product module directory.
// Rewritten after origin sync and after install/upgrade/uninstall/load-demo.
type ModuleCatalog struct {
	pkgmeta.BaseModel `gorm:"embedded"`

	ModuleName             string         `gorm:"column:module_name;type:varchar(255);not null;uniqueIndex"`
	Summary                string         `gorm:"column:summary;type:text"`
	Description            string         `gorm:"column:description;type:text"`
	Category               string         `gorm:"column:category;type:varchar(255);index"`
	Application            string         `gorm:"column:application;type:varchar(255)"`
	Author                 string         `gorm:"column:author;type:varchar(255)"`
	License                string         `gorm:"column:license;type:text"`
	LocalVersion           sql.NullString `gorm:"column:local_version;type:varchar(255)"`
	RegistryVersion        sql.NullString `gorm:"column:registry_version;type:varchar(255)"`
	Installed              bool           `gorm:"column:installed;not null;default:false;index"`
	InstalledVersion       sql.NullString `gorm:"column:installed_version;type:varchar(255)"`
	UpdateAvailable        bool           `gorm:"column:update_available;not null;default:false;index"`
	RegistryNewerThanLocal bool           `gorm:"column:registry_newer_than_local;not null;default:false"`
	Available              bool           `gorm:"column:available;not null;default:true;index"`
	UnavailableReason      sql.NullString `gorm:"column:unavailable_reason;type:text"`
	LocalPath              sql.NullString `gorm:"column:local_path;type:varchar(512)"`
	DependsJson            datatypes.JSON `gorm:"column:depends_json"`
	DependentsJson         datatypes.JSON `gorm:"column:dependents_json"`
	HasDemo                bool           `gorm:"column:has_demo;not null;default:false"`
	DemoLoaded             bool           `gorm:"column:demo_loaded;not null;default:false"`
	LastSyncAt             *time.Time     `gorm:"column:last_sync_at"`
	ActiveOperationId      sql.NullString `gorm:"column:active_operation_id;type:char(20);index"`
}

func (ModuleCatalog) TableName() string {
	return "meta_module_catalog"
}
