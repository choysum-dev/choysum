// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

// ModuleIndex stores module availability and manifest snapshots.
type ModuleIndex struct {
	meta.BaseModel `gorm:"embedded"`

	ModuleName       string         `gorm:"column:module_name;type:varchar(255);not null;uniqueIndex:idx_meta_module_index_origin,priority:1"`
	OriginType       string         `gorm:"column:origin_type;type:varchar(32);not null;uniqueIndex:idx_meta_module_index_origin,priority:2"`
	OriginRef        string         `gorm:"column:origin_ref;type:varchar(255);not null;uniqueIndex:idx_meta_module_index_origin,priority:3"`
	Available        bool           `gorm:"column:available;not null"`
	Version          sql.NullString `gorm:"column:version;type:varchar(255)"`
	ManifestJson     datatypes.JSON `gorm:"column:manifest_json"`
	LocalPath        sql.NullString `gorm:"column:local_path;type:varchar(512)"`
	LastSyncAt       *time.Time     `gorm:"column:last_sync_at"`
	LastBatchSyncAt  *time.Time     `gorm:"column:last_batch_sync_at"`
	SyncRevision     sql.NullString `gorm:"column:sync_revision;type:varchar(255)"`
	LastErrorMessage sql.NullString `gorm:"column:last_error_message;type:text"`
}

func (ModuleIndex) TableName() string {
	return "meta_module_index"
}
