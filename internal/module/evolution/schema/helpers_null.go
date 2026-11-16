// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// SetNotNull enforces NOT NULL on an existing column after data has no NULLs,
// and registers IntentTightenNull so ValidatePlan accepts guarded null tighten.
func SetNotNull(opts HelperOptions, table, column string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	column = strings.TrimSpace(column)
	if table == "" || column == "" {
		return fmt.Errorf("setNotNull requires table and column")
	}
	has, err := helperHasColumn(opts.DB, table, column)
	if err != nil {
		return fmt.Errorf("setNotNull: %w", err)
	}
	if !has {
		return fmt.Errorf("setNotNull: column %s.%s not found", table, column)
	}

	qTable := quoteIdent(opts.Dialect, table)
	qCol := quoteIdent(opts.Dialect, column)
	var nulls int64
	if err := opts.DB.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NULL", qTable, qCol)).Scan(&nulls).Error; err != nil {
		return fmt.Errorf("setNotNull count nulls %s.%s: %w", table, column, err)
	}
	if nulls > 0 {
		return fmt.Errorf("setNotNull: %s.%s still has %d NULL row(s)", table, column, nulls)
	}

	if err := applySetNotNull(opts, table, column); err != nil {
		return err
	}
	opts.Intents.Add(Intent{
		Kind:  IntentTightenNull,
		Table: table,
		Name:  intentPhysicalName(column),
	})
	return nil
}

func applySetNotNull(opts HelperOptions, table, column string) error {
	dialect := strings.ToLower(strings.TrimSpace(opts.Dialect))
	qTable := quoteIdent(opts.Dialect, table)
	qCol := quoteIdent(opts.Dialect, column)
	switch dialect {
	case "sqlite":
		return sqliteRewriteTable(opts.DB, table, sqliteTableRewrite{
			ForceNotNull: map[string]struct{}{strings.ToLower(column): {}},
		})
	case "postgres", "postgresql":
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", qTable, qCol)
		if err := helperExec(opts.DB, sql); err != nil {
			return fmt.Errorf("setNotNull postgres: %w", err)
		}
		return nil
	case "mysql", "mariadb":
		typeName, err := liveColumnTypeName(opts.DB, table, column)
		if err != nil {
			return err
		}
		sql := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NOT NULL", qTable, qCol, typeName)
		if err := helperExec(opts.DB, sql); err != nil {
			return fmt.Errorf("setNotNull mysql: %w", err)
		}
		return nil
	case "sqlserver":
		typeName, err := liveColumnTypeName(opts.DB, table, column)
		if err != nil {
			return err
		}
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s NOT NULL", qTable, qCol, typeName)
		if err := helperExec(opts.DB, sql); err != nil {
			return fmt.Errorf("setNotNull sqlserver: %w", err)
		}
		return nil
	default:
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", qTable, qCol)
		if err := helperExec(opts.DB, sql); err != nil {
			return fmt.Errorf("setNotNull %s: %w", dialect, err)
		}
		return nil
	}
}

func liveColumnTypeName(db *gorm.DB, table, column string) (string, error) {
	types, err := getColumnTypes(db, table)
	if err != nil {
		return "", fmt.Errorf("inspect %s.%s: %w", table, column, err)
	}
	for _, ct := range types {
		if ct == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(ct.Name()), column) {
			continue
		}
		typeName := strings.TrimSpace(ct.DatabaseTypeName())
		if typeName == "" {
			return "", fmt.Errorf("empty type for %s.%s", table, column)
		}
		typeName = decorateLiveColumnType(typeName, ct)
		if def, ok := ct.DefaultValue(); ok {
			def = strings.TrimSpace(def)
			if def != "" {
				typeName += " DEFAULT " + def
			}
		}
		return typeName, nil
	}
	return "", fmt.Errorf("column %s.%s not found", table, column)
}

func decorateLiveColumnType(typeName string, ct gorm.ColumnType) string {
	lower := strings.ToLower(strings.TrimSpace(typeName))
	if strings.Contains(lower, "char") || strings.Contains(lower, "binary") {
		if length, ok := ct.Length(); ok && length > 0 {
			return fmt.Sprintf("%s(%d)", typeName, length)
		}
	}
	if lower == "decimal" || lower == "numeric" {
		if precision, scale, ok := ct.DecimalSize(); ok {
			return fmt.Sprintf("%s(%d,%d)", typeName, precision, scale)
		}
	}
	return typeName
}
