// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// HelperOptions configure script DDL helpers.
// Intents is required so helpers only run during upgrade script execution
// (Runner attaches the bag to the exec context).
type HelperOptions struct {
	DB      *gorm.DB
	Dialect string
	Intents IntentBag
	// AllowedNames lists extra object names (table → lower(name)) that may be
	// dropped without a Choysum stable prefix.
	AllowedNames map[string]map[string]struct{}
}

func (o HelperOptions) validate() error {
	if o.DB == nil {
		return fmt.Errorf("db is nil")
	}
	if o.Intents == nil {
		return fmt.Errorf("schema helpers require an upgrade IntentBag")
	}
	return nil
}

func (o HelperOptions) allowName(table, name string, prefixes ...string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	for _, prefix := range prefixes {
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix != "" && strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	set := o.AllowedNames[strings.ToLower(strings.TrimSpace(table))]
	_, ok := set[lower]
	return ok
}

// RenameColumn renames a physical column via GORM and registers an Intent.
func RenameColumn(opts HelperOptions, table, from, to string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if table == "" || from == "" || to == "" {
		return fmt.Errorf("renameColumn requires table, from, and to")
	}
	if strings.EqualFold(from, to) {
		return fmt.Errorf("renameColumn from and to must differ")
	}
	col := columnSpecForLiveRename(opts.DB, table, from, to)
	if err := renameColumn(opts.DB, table, from, col, opts.Dialect); err != nil {
		return err
	}
	opts.Intents.Add(Intent{Kind: IntentRenameColumn, Table: table, Name: to, FromName: from})
	return nil
}

// DropColumn drops a physical column and registers an Intent.
func DropColumn(opts HelperOptions, table, column string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	column = strings.TrimSpace(column)
	if table == "" || column == "" {
		return fmt.Errorf("dropColumn requires table and column")
	}
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quoteIdent(opts.Dialect, table), quoteIdent(opts.Dialect, column))
	if err := helperExec(opts.DB, sql); err != nil {
		return fmt.Errorf("drop column %s.%s: %w", table, column, err)
	}
	opts.Intents.Add(Intent{Kind: IntentDropColumn, Table: table, Name: column})
	return nil
}

// DropIndex drops an index after prefix/allow-list validation.
func DropIndex(opts HelperOptions, table, name string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	name = strings.TrimSpace(name)
	if table == "" || name == "" {
		return fmt.Errorf("dropIndex requires table and name")
	}
	if !opts.allowName(table, name, "idx_") {
		return fmt.Errorf("dropIndex rejects non-Choysum name %q (want idx_/allowed)", name)
	}
	belongs, err := indexBelongsToTable(opts.DB, table, name)
	if err != nil {
		return fmt.Errorf("drop index %s on %s: %w", name, table, err)
	}
	if !belongs {
		return fmt.Errorf("drop index %s on %s: index does not belong to table", name, table)
	}
	sql := dropIndexSQL(opts.Dialect, table, name)
	if err := helperExec(opts.DB, sql); err != nil {
		return fmt.Errorf("drop index %s on %s: %w", name, table, err)
	}
	opts.Intents.Add(Intent{Kind: IntentDropIndex, Table: table, Name: name})
	return nil
}

// DropCheck drops a CHECK constraint after name validation.
func DropCheck(opts HelperOptions, table, name string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	name = strings.TrimSpace(name)
	if table == "" || name == "" {
		return fmt.Errorf("dropCheck requires table and name")
	}
	if strings.EqualFold(strings.TrimSpace(opts.Dialect), "sqlite") {
		return fmt.Errorf("dropCheck is not supported on sqlite")
	}
	if !opts.allowName(table, name, "chk_", "ck_") {
		return fmt.Errorf("dropCheck rejects non-Choysum name %q (want chk_/ck_/allowed)", name)
	}
	if err := dropCheckConstraintBestEffort(opts.DB, opts.Dialect, table, name); err != nil {
		return fmt.Errorf("drop check %s on %s: %w", name, table, err)
	}
	opts.Intents.Add(Intent{Kind: IntentDropCheck, Table: table, Name: name})
	return nil
}

// DropForeignKey drops a foreign key after name validation.
func DropForeignKey(opts HelperOptions, table, name string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	name = strings.TrimSpace(name)
	if table == "" || name == "" {
		return fmt.Errorf("dropForeignKey requires table and name")
	}
	if !opts.allowName(table, name, "fk_") {
		return fmt.Errorf("dropForeignKey rejects non-Choysum name %q (want fk_/allowed)", name)
	}
	sql, err := dropForeignKeySQL(opts.Dialect, table, name)
	if err != nil {
		return err
	}
	if err := helperExec(opts.DB, sql); err != nil {
		return fmt.Errorf("drop foreign key %s on %s: %w", name, table, err)
	}
	opts.Intents.Add(Intent{Kind: IntentDropForeignKey, Table: table, Name: name})
	return nil
}

// helperExec is overridable in tests for dialect paths that need a live server.
var helperExec = func(db *gorm.DB, sql string) error {
	return db.Exec(sql).Error
}

func dropIndexSQL(dialect, table, name string) string {
	qTable, qName := quoteIdent(dialect, table), quoteIdent(dialect, name)
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql", "mariadb", "sqlserver":
		return fmt.Sprintf("DROP INDEX %s ON %s", qName, qTable)
	case "sqlite":
		return fmt.Sprintf("DROP INDEX IF EXISTS %s", qName)
	default:
		return fmt.Sprintf("DROP INDEX IF EXISTS %s", qName)
	}
}

func dropForeignKeySQL(dialect, table, name string) (string, error) {
	qTable, qName := quoteIdent(dialect, table), quoteIdent(dialect, name)
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql", "mariadb":
		return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", qTable, qName), nil
	case "sqlite":
		return "", fmt.Errorf("dropForeignKey is not supported on sqlite")
	default:
		return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", qTable, qName), nil
	}
}

func quoteIdent(dialect, name string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql", "mariadb":
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	case "sqlserver":
		return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
	default:
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
}

// columnSpecForLiveRename builds a RenameColumn target that preserves the live
// column's physical type (needed when the dialect rewrites the column definition).
func columnSpecForLiveRename(db *gorm.DB, table, from, to string) ColumnSpec {
	col := ColumnSpec{Name: to, FieldName: exportIdent(to), PhysicalType: "text"}
	if db == nil {
		return col
	}
	types, err := getColumnTypes(db, table)
	if err != nil {
		return col
	}
	for _, ct := range types {
		lc, ok := liveColumnFromColumnType(ct)
		if !ok || !strings.EqualFold(lc.Name, from) {
			continue
		}
		if phys := normalizeDBType(lc.DatabaseTypeName); phys != "" && getDefaultValue(phys) != nil {
			col.PhysicalType = phys
		}
		if lc.Length != nil && *lc.Length > 0 {
			size := int(*lc.Length)
			col.Size = &size
		}
		if lc.Nullable != nil {
			col.NotNull = !*lc.Nullable
		}
		if lc.Default != nil {
			col.Default = lc.Default
		}
		return col
	}
	return col
}

func indexBelongsToTable(db *gorm.DB, table, name string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	indexes, err := getIndexes(db, table)
	if err != nil {
		return false, err
	}
	for _, idx := range indexes {
		if idx == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(idx.Name()), name) {
			return true, nil
		}
	}
	return false, nil
}
