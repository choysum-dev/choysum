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
		Indexes:  make(map[string][]LiveIndex),
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
		live.Indexes[table] = nil

		columnTypes, err := getColumnTypes(db, table)
		if err != nil {
			return LiveSchema{}, fmt.Errorf("column types for %s: %w", table, err)
		}
		for _, ct := range columnTypes {
			if lc, ok := liveColumnFromColumnType(ct); ok {
				live.Columns[table][strings.ToLower(lc.Name)] = lc
			}
		}

		indexes, err := getIndexes(db, table)
		if err != nil {
			return LiveSchema{}, fmt.Errorf("indexes for %s: %w", table, err)
		}
		for _, idx := range indexes {
			if idx == nil {
				continue
			}
			name := strings.TrimSpace(idx.Name())
			if name == "" {
				continue
			}
			cols := make([]string, 0, len(idx.Columns()))
			for _, col := range idx.Columns() {
				col = strings.TrimSpace(col)
				if col != "" {
					cols = append(cols, col)
				}
			}
			unique := false
			if u, ok := idx.Unique(); ok {
				unique = u
			}
			live.Indexes[table] = append(live.Indexes[table], LiveIndex{
				Name:    name,
				Columns: cols,
				Unique:  unique,
			})
		}

		count, err := probeTableNonEmptyFn(db, table)
		if err != nil {
			return LiveSchema{}, fmt.Errorf("probe rows for %s: %w", table, err)
		}
		live.RowCount[table] = count
	}
	return live, nil
}

// Overridable migrator helpers (tests replace these for failure paths).
var (
	getColumnTypes = func(db *gorm.DB, table string) ([]gorm.ColumnType, error) {
		return db.Migrator().ColumnTypes(table)
	}
	getIndexes = func(db *gorm.DB, table string) ([]gorm.Index, error) {
		return db.Migrator().GetIndexes(table)
	}
	probeTableNonEmptyFn = probeTableNonEmpty
)

func liveColumnFromColumnType(ct gorm.ColumnType) (LiveColumn, bool) {
	if ct == nil {
		return LiveColumn{}, false
	}
	name := strings.TrimSpace(ct.Name())
	if name == "" {
		return LiveColumn{}, false
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
	if def, ok := ct.DefaultValue(); ok {
		def = strings.TrimSpace(def)
		if def != "" {
			defCopy := def
			lc.Default = &defCopy
		}
	}
	return lc, true
}

// probeTableNonEmpty returns 1 if the table has at least one row, otherwise 0.
func probeTableNonEmpty(db *gorm.DB, table string) (int64, error) {
	var probe int
	tx := db.Table(table).Select("1").Limit(1).Scan(&probe)
	if tx.Error != nil {
		return 0, tx.Error
	}
	if tx.RowsAffected > 0 {
		return 1, nil
	}
	return 0, nil
}

func desiredTableNames(desired DesiredSchema) []string {
	names := make([]string, 0, len(desired.Tables))
	for table := range desired.Tables {
		names = append(names, table)
	}
	return names
}
