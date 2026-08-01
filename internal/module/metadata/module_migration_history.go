// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metadata

import (
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
)

type ModuleMigrationHistory struct {
	meta.BaseModel
	ModuleName string    `gorm:"column:module_name;size:255;index:idx_meta_module_migration_history,unique"`
	Version    string    `gorm:"column:version;size:64;index:idx_meta_module_migration_history,unique"`
	Phase      string    `gorm:"column:phase;size:16;index:idx_meta_module_migration_history,unique"`
	Script     string    `gorm:"column:script;size:255;index:idx_meta_module_migration_history,unique"`
	Checksum   string    `gorm:"column:checksum;size:128"`
	Status     string    `gorm:"column:status;size:16;index"`
	StartedAt  time.Time `gorm:"column:started_at;index"`
	FinishedAt time.Time `gorm:"column:finished_at;index"`
	Error      string    `gorm:"column:error;type:text"`
	TraceId    string    `gorm:"column:trace_id;size:64;index"`
	JobId      string    `gorm:"column:job_id;size:64;index"`
}

func (ModuleMigrationHistory) TableName() string {
	return "meta_module_migration_history"
}
