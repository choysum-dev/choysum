// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	gormmigrator "gorm.io/gorm/migrator"
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
	getIndexes           = defaultGetIndexes
	probeTableNonEmptyFn = probeTableNonEmpty
)

func defaultGetIndexes(db *gorm.DB, table string) ([]gorm.Index, error) {
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "sqlite" {
		// GORM's sqlite GetIndexes scans index_info.name into string and fails when
		// SQLite returns NULL (expression / rowid index columns). Use a NULL-safe path.
		return sqliteGetIndexes(db, table)
	}
	return db.Migrator().GetIndexes(table)
}

// Overridable SQLite pragma scanners (tests inject failures and synthetic rows).
var (
	sqliteIndexListScan = func(db *gorm.DB, table string, dest any) error {
		return db.Raw(`SELECT name, "unique" AS "unique", origin FROM pragma_index_list(?)`, table).Scan(dest).Error
	}
	sqliteIndexInfoScan = func(db *gorm.DB, indexName string, dest any) error {
		return db.Raw(`SELECT name FROM pragma_index_info(?)`, indexName).Scan(dest).Error
	}
)

type sqliteIndexListRow struct {
	Name   sql.NullString `gorm:"column:name"`
	Unique bool           `gorm:"column:unique"`
	Origin string         `gorm:"column:origin"`
}

func sqliteGetIndexes(db *gorm.DB, table string) ([]gorm.Index, error) {
	var list []sqliteIndexListRow
	if err := sqliteIndexListScan(db, table, &list); err != nil {
		return nil, err
	}
	out := make([]gorm.Index, 0, len(list))
	for _, row := range list {
		name := strings.TrimSpace(row.Name.String)
		if !row.Name.Valid || name == "" {
			continue
		}
		if row.Origin == "u" {
			// Skip indexes created by UNIQUE constraints (matches GORM sqlite Migrator).
			continue
		}
		var colRows []sql.NullString
		if err := sqliteIndexInfoScan(db, name, &colRows); err != nil {
			return nil, err
		}
		cols := make([]string, 0, len(colRows))
		for _, col := range colRows {
			if !col.Valid {
				continue
			}
			c := strings.TrimSpace(col.String)
			if c != "" {
				cols = append(cols, c)
			}
		}
		out = append(out, &gormmigrator.Index{
			TableName:       table,
			NameValue:       name,
			ColumnList:      cols,
			PrimaryKeyValue: sql.NullBool{Bool: row.Origin == "pk", Valid: true},
			UniqueValue:     sql.NullBool{Bool: row.Unique, Valid: true},
		})
	}
	return out, nil
}

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
