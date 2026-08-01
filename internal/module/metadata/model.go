// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metadata

// Entities returns module admin/ops metadata models.
func Entities() []any {
	return []any{
		&ModuleIndex{},
		&ModelData{},
		&Setting{},
		&ModuleManagementLog{},
		&ModuleMigrationHistory{},
	}
}
