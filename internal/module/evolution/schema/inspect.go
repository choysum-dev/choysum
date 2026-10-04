// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"database/sql"
	"fmt"
	"sort"
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
			primaryKey := false
			if pk, ok := idx.PrimaryKey(); ok {
				primaryKey = pk
			}
			live.Indexes[table] = append(live.Indexes[table], LiveIndex{
				Name:       name,
				Columns:    cols,
				Unique:     unique,
				PrimaryKey: primaryKey,
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
	if db != nil && db.Dialector != nil {
		switch strings.ToLower(strings.TrimSpace(db.Dialector.Name())) {
		case "sqlite":
			// GORM's sqlite GetIndexes scans index_info.name into string and fails when
			// SQLite returns NULL (expression / rowid index columns). Use a NULL-safe path.
			return sqliteGetIndexes(db, table)
		case "postgres", "postgresql":
			// GORM's postgres GetIndexes excludes constraint-backed indexes
			// (`AND con.oid IS NULL`), so UNIQUE CONSTRAINT / PK indexes such as
			// uni_auth_user_username never appear and Unique:true fields plan as
			// guarded add_index on every upgrade of a non-empty table.
			return postgresGetIndexes(db, table)
		}
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
		// Prefer the unique flag from pragma_index_list; also treat origin "u"
		// (UNIQUE constraint) and "pk" as unique so a failed/zero bool scan cannot
		// hide sqlite_autoindex_* constraints and spuriously plan guarded add_index.
		origin := strings.ToLower(strings.TrimSpace(row.Origin))
		unique := row.Unique || origin == "u" || origin == "pk"
		out = append(out, &gormmigrator.Index{
			TableName:       table,
			NameValue:       name,
			ColumnList:      cols,
			PrimaryKeyValue: sql.NullBool{Bool: origin == "pk", Valid: true},
			UniqueValue:     sql.NullBool{Bool: unique, Valid: true},
		})
	}
	return out, nil
}

// Overridable Postgres index scanner (tests inject failures and synthetic rows).
var postgresIndexScan = func(db *gorm.DB, table string, dest any) error {
	// Include constraint-backed UNIQUE/PK indexes that GORM's GetIndexes omits.
	// Column order follows indkey so composite uniques stay comparable by position.
	const sql = `
SELECT
	ci.relname AS index_name,
	i.indisunique AS is_unique,
	i.indisprimary AS is_primary,
	a.attname AS column_name,
	k.ord AS column_ord
FROM pg_index i
	JOIN pg_class ct ON ct.oid = i.indrelid
	JOIN pg_class ci ON ci.oid = i.indexrelid
	JOIN pg_namespace n ON n.oid = ct.relnamespace
	JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, ord) ON true
	JOIN pg_attribute a ON a.attrelid = ct.oid AND a.attnum = k.attnum
WHERE ct.relkind = 'r'
	AND ct.relname = ?
	AND n.nspname = current_schema()
	AND a.attnum > 0
	AND NOT a.attisdropped
ORDER BY ci.relname, k.ord`
	return db.Raw(sql, table).Scan(dest).Error
}

type postgresIndexRow struct {
	IndexName  string `gorm:"column:index_name"`
	IsUnique   bool   `gorm:"column:is_unique"`
	IsPrimary  bool   `gorm:"column:is_primary"`
	ColumnName string `gorm:"column:column_name"`
	ColumnOrd  int    `gorm:"column:column_ord"`
}

func postgresGetIndexes(db *gorm.DB, table string) ([]gorm.Index, error) {
	table = strings.TrimSpace(table)
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if table == "" {
		return nil, nil
	}
	var rows []postgresIndexRow
	if err := postgresIndexScan(db, table, &rows); err != nil {
		return nil, err
	}
	type agg struct {
		unique  bool
		primary bool
		cols    []string
	}
	order := make([]string, 0)
	byName := map[string]*agg{}
	for _, row := range rows {
		name := strings.TrimSpace(row.IndexName)
		if name == "" {
			continue
		}
		a := byName[name]
		if a == nil {
			a = &agg{unique: row.IsUnique, primary: row.IsPrimary}
			byName[name] = a
			order = append(order, name)
		} else {
			a.unique = a.unique || row.IsUnique
			a.primary = a.primary || row.IsPrimary
		}
		col := strings.TrimSpace(row.ColumnName)
		if col != "" {
			a.cols = append(a.cols, col)
		}
	}
	out := make([]gorm.Index, 0, len(order))
	for _, name := range order {
		a := byName[name]
		out = append(out, &gormmigrator.Index{
			TableName:       table,
			NameValue:       name,
			ColumnList:      append([]string(nil), a.cols...),
			PrimaryKeyValue: sql.NullBool{Bool: a.primary, Valid: true},
			UniqueValue:     sql.NullBool{Bool: a.unique || a.primary, Valid: true},
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
		// SQLite reports PRIMARY KEY columns as nullable even when they are not
		// optional in practice; treat PK as NOT NULL so Desired Id (etc.) does not
		// emit a spurious guarded tighten.
		if pk, pkOK := ct.PrimaryKey(); pkOK && pk {
			nullableCopy = false
		}
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
	names := make([]string, 0, len(desired.Tables)+len(desired.JoinTables))
	seen := map[string]struct{}{}
	tables := make([]string, 0, len(desired.Tables))
	for table := range desired.Tables {
		if trimmed := strings.TrimSpace(table); trimmed != "" {
			tables = append(tables, trimmed)
		}
	}
	sort.Strings(tables)
	for _, table := range tables {
		key := strings.ToLower(table)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, table)
	}
	joinNames := make([]string, 0, len(desired.JoinTables))
	for _, jt := range desired.JoinTables {
		if trimmed := strings.TrimSpace(jt.Table); trimmed != "" {
			joinNames = append(joinNames, trimmed)
		}
	}
	sort.Strings(joinNames)
	for _, table := range joinNames {
		key := strings.ToLower(table)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, table)
	}
	return names
}
