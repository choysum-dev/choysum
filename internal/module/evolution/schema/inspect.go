// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// inspectTables inspects live tables using GORM migrator APIs.
func inspectTables(db *gorm.DB, tables []string) (LiveSchema, error) {
	live := LiveSchema{
		Tables:   make(map[string]bool),
		Columns:  make(map[string]map[string]LiveColumn),
		RowCount: make(map[string]int64),
	}
	if db == nil {
		return live, fmt.Errorf("db is nil")
	}
	mig := db.Migrator()
	for _, table := range tables {
		table = strings.TrimSpace(table)
		if table == "" {
			continue
		}
		if !mig.HasTable(table) {
			live.Tables[table] = false
			continue
		}
		live.Tables[table] = true
		live.Columns[table] = make(map[string]LiveColumn)

		columnTypes, err := mig.ColumnTypes(table)
		if err != nil {
			return LiveSchema{}, fmt.Errorf("column types for %s: %w", table, err)
		}
		for _, ct := range columnTypes {
			if ct == nil {
				continue
			}
			name := strings.TrimSpace(ct.Name())
			if name == "" {
				continue
			}
			lc := LiveColumn{
				Name:             name,
				DatabaseTypeName: strings.TrimSpace(ct.DatabaseTypeName()),
			}
			if length, ok := ct.Length(); ok {
				lengthCopy := length
				lc.Length = &lengthCopy
			}
			if nullable, ok := ct.Nullable(); ok {
				nullableCopy := nullable
				lc.Nullable = &nullableCopy
			}
			live.Columns[table][strings.ToLower(name)] = lc
		}

		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			// Best-effort: treat unknown as empty for NOT NULL policy (conservative would be non-empty).
			// Prefer failing closed on count errors.
			return LiveSchema{}, fmt.Errorf("count rows for %s: %w", table, err)
		}
		live.RowCount[table] = count
	}
	return live, nil
}

func desiredTableNames(desired DesiredSchema) []string {
	names := make([]string, 0, len(desired.Tables))
	for table := range desired.Tables {
		names = append(names, table)
	}
	return names
}
