// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"

	"github.com/choysum-dev/choysum/pkg/scope"
)

const savedFilterTable = "web_saved_filter"

// ensureSavedFilterCreateUidRename migrates the SavedFilter owner column from the
// model-local CreateUid (create_uid) to BaseModel CreatedUid (created_uid).
//
// Order relative to AutoMigrate:
//   - rename when only the old column exists (preserves SF11 creator data)
//   - if both exist (partial upgrade), copy then drop the old column
func ensureSavedFilterCreateUidRename(runtimeScope scope.Scope) error {
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	db := runtimeScope.Session().DB
	if db == nil {
		return nil
	}
	migrator := db.Migrator()
	if !migrator.HasTable(savedFilterTable) {
		return nil
	}

	hasOld := migrator.HasColumn(savedFilterTable, "create_uid")
	hasNew := migrator.HasColumn(savedFilterTable, "created_uid")
	if !hasOld {
		return nil
	}

	table := quoteIdent(db.Dialector.Name(), savedFilterTable)
	oldCol := quoteIdent(db.Dialector.Name(), "create_uid")
	newCol := quoteIdent(db.Dialector.Name(), "created_uid")

	if !hasNew {
		sql := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, oldCol, newCol)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("rename %s.create_uid -> created_uid: %w", savedFilterTable, err)
		}
		return nil
	}

	// Both columns present: backfill then drop the legacy column.
	backfill := fmt.Sprintf(
		`UPDATE %s SET %s = %s WHERE (%s IS NULL OR %s = '') AND %s IS NOT NULL AND %s != ''`,
		table, newCol, oldCol, newCol, newCol, oldCol, oldCol,
	)
	if err := db.Exec(backfill).Error; err != nil {
		return fmt.Errorf("backfill %s.created_uid from create_uid: %w", savedFilterTable, err)
	}
	drop := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, oldCol)
	if err := db.Exec(drop).Error; err != nil {
		return fmt.Errorf("drop %s.create_uid: %w", savedFilterTable, err)
	}
	return nil
}

func quoteIdent(dialect, name string) string {
	switch dialect {
	case "mysql", "mariadb":
		return "`" + name + "`"
	default:
		return `"` + name + `"`
	}
}
