// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// HelperOptions configure script DDL helpers.
type HelperOptions struct {
	DB      *gorm.DB
	Dialect string
	Intents IntentBag
	// AllowedNames optionally lists snapshot / known object names that may be dropped
	// even without a Choysum stable prefix (table → lower(name)).
	AllowedNames map[string]map[string]struct{}
}

func (o HelperOptions) validate() error {
	if o.DB == nil {
		return fmt.Errorf("db is nil")
	}
	return nil
}

func (o HelperOptions) allowName(table, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "idx_") ||
		strings.HasPrefix(lower, "chk_") ||
		strings.HasPrefix(lower, "ck_") ||
		strings.HasPrefix(lower, "fk_") {
		return true
	}
	if o.AllowedNames == nil {
		return false
	}
	set := o.AllowedNames[strings.ToLower(strings.TrimSpace(table))]
	_, ok := set[lower]
	return ok
}

// RenameColumn renames a physical column and registers an Intent.
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
	sql := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
		quoteIdent(opts.Dialect, table), quoteIdent(opts.Dialect, from), quoteIdent(opts.Dialect, to))
	if err := opts.DB.Exec(sql).Error; err != nil {
		return fmt.Errorf("rename column %s.%s → %s: %w", table, from, to, err)
	}
	if opts.Intents != nil {
		opts.Intents.Add(Intent{Kind: IntentRenameColumn, Table: table, Name: to, FromName: from})
	}
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
	if err := opts.DB.Exec(sql).Error; err != nil {
		return fmt.Errorf("drop column %s.%s: %w", table, column, err)
	}
	if opts.Intents != nil {
		opts.Intents.Add(Intent{Kind: IntentDropColumn, Table: table, Name: column})
	}
	return nil
}

// DropIndex drops an index after prefix/snapshot name validation.
func DropIndex(opts HelperOptions, table, name string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	name = strings.TrimSpace(name)
	if table == "" || name == "" {
		return fmt.Errorf("dropIndex requires table and name")
	}
	if !opts.allowName(table, name) {
		return fmt.Errorf("dropIndex rejects non-Choysum name %q (want idx_/snapshot-known)", name)
	}
	type stub struct{}
	if err := opts.DB.Table(table).Migrator().DropIndex(&stub{}, name); err != nil {
		sql := fmt.Sprintf("DROP INDEX %s", quoteIdent(opts.Dialect, name))
		if strings.EqualFold(strings.TrimSpace(opts.Dialect), "sqlite") {
			sql = fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteIdent(opts.Dialect, name))
		}
		if execErr := opts.DB.Exec(sql).Error; execErr != nil {
			return fmt.Errorf("drop index %s on %s: %v (raw: %w)", name, table, err, execErr)
		}
	}
	if opts.Intents != nil {
		opts.Intents.Add(Intent{Kind: IntentDropIndex, Table: table, Name: name})
	}
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
	if !opts.allowName(table, name) {
		return fmt.Errorf("dropCheck rejects non-Choysum name %q (want chk_/ck_/snapshot-known)", name)
	}
	if err := dropCheckConstraintBestEffort(opts.DB, opts.Dialect, table, name); err != nil {
		return fmt.Errorf("drop check %s on %s: %w", name, table, err)
	}
	if opts.Intents != nil {
		opts.Intents.Add(Intent{Kind: IntentDropCheck, Table: table, Name: name})
	}
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
	if !opts.allowName(table, name) {
		return fmt.Errorf("dropForeignKey rejects non-Choysum name %q (want fk_/snapshot-known)", name)
	}
	sql := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", quoteIdent(opts.Dialect, table), quoteIdent(opts.Dialect, name))
	switch strings.ToLower(strings.TrimSpace(opts.Dialect)) {
	case "mysql":
		sql = fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", quoteIdent(opts.Dialect, table), quoteIdent(opts.Dialect, name))
	case "sqlite":
		return fmt.Errorf("dropForeignKey is not supported on sqlite")
	}
	if err := opts.DB.Exec(sql).Error; err != nil {
		return fmt.Errorf("drop foreign key %s on %s: %w", name, table, err)
	}
	if opts.Intents != nil {
		opts.Intents.Add(Intent{Kind: IntentDropForeignKey, Table: table, Name: name})
	}
	return nil
}

func quoteIdent(dialect, name string) string {
	name = strings.ReplaceAll(name, `"`, `""`)
	name = strings.ReplaceAll(name, "`", "``")
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql":
		return "`" + name + "`"
	case "sqlserver":
		return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
	default:
		return `"` + name + `"`
	}
}
