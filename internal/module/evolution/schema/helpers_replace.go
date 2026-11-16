// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ObsoleteColumnSuffix is appended to a live column name during ReplaceColumnPre.
const ObsoleteColumnSuffix = "__obsolete"

// ObsoleteColumnName returns the bypass column name for a ReplaceColumn flow.
func ObsoleteColumnName(column string) string {
	column = strings.TrimSpace(column)
	if column == "" {
		return ""
	}
	return column + ObsoleteColumnSuffix
}

// ReplaceColumnOptions configure ReplaceColumnPre.
type ReplaceColumnOptions struct {
	Table      string
	FromColumn string // live physical name (snake)
	ToColumn   string // default = FromColumn; may differ for rename + type change
}

// ReplaceColumnPre renames FromColumn to ObsoleteColumnName(FromColumn) so Migrate
// can Auto-add ToColumn with the desired physical type. Registers Intent via RenameColumn.
func ReplaceColumnPre(opts HelperOptions, o ReplaceColumnOptions) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table := strings.TrimSpace(o.Table)
	from := strings.TrimSpace(o.FromColumn)
	to := strings.TrimSpace(o.ToColumn)
	if to == "" {
		to = from
	}
	if table == "" || from == "" {
		return fmt.Errorf("replaceColumnPre requires table and fromColumn")
	}
	obsolete := ObsoleteColumnName(from)

	hasFrom, err := helperHasColumn(opts.DB, table, from)
	if err != nil {
		return fmt.Errorf("replaceColumnPre %s.%s: %w", table, from, err)
	}
	hasObsolete, err := helperHasColumn(opts.DB, table, obsolete)
	if err != nil {
		return fmt.Errorf("replaceColumnPre %s.%s: %w", table, obsolete, err)
	}
	hasTo, err := helperHasColumn(opts.DB, table, to)
	if err != nil {
		return fmt.Errorf("replaceColumnPre %s.%s: %w", table, to, err)
	}

	if hasObsolete {
		if hasFrom {
			return fmt.Errorf("replaceColumnPre: obsolete column %s.%s already exists", table, obsolete)
		}
		// Pre already applied (live column renamed; Migrate may still need to add ToColumn).
		return nil
	}
	if !hasFrom {
		if hasTo {
			// From was renamed away and ToColumn already exists (rename+type-change path after Migrate).
			return nil
		}
		return fmt.Errorf("replaceColumnPre: column %s.%s not found", table, from)
	}
	// hasFrom && !hasObsolete: either a fresh ReplaceColumn or a completed cycle
	// (FromColumn == ToColumn). Helpers cannot tell those apart without Desired;
	// callers must not re-run Pre after DropObsolete when names coincide.
	return RenameColumn(opts, table, from, obsolete)
}

// ReplaceColumnCopyCast copies values from an obsolete column into the new column
// using a small set of dialect-safe kind mappings.
func ReplaceColumnCopyCast(opts HelperOptions, table, fromObsolete, toColumn, kind string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	fromObsolete = strings.TrimSpace(fromObsolete)
	toColumn = strings.TrimSpace(toColumn)
	kind = strings.ToLower(strings.TrimSpace(kind))
	if table == "" || fromObsolete == "" || toColumn == "" || kind == "" {
		return fmt.Errorf("replaceColumnCopyCast requires table, fromObsolete, toColumn, and kind")
	}
	hasFrom, err := helperHasColumn(opts.DB, table, fromObsolete)
	if err != nil {
		return fmt.Errorf("replaceColumnCopyCast: %w", err)
	}
	hasTo, err := helperHasColumn(opts.DB, table, toColumn)
	if err != nil {
		return fmt.Errorf("replaceColumnCopyCast: %w", err)
	}
	if !hasFrom || !hasTo {
		return fmt.Errorf("replaceColumnCopyCast: both %s.%s and %s.%s must exist", table, fromObsolete, table, toColumn)
	}

	qTable := quoteIdent(opts.Dialect, table)
	qFrom := quoteIdent(opts.Dialect, fromObsolete)
	qTo := quoteIdent(opts.Dialect, toColumn)
	expr, err := copyCastExpr(opts.Dialect, qFrom, kind)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("UPDATE %s SET %s = %s", qTable, qTo, expr)
	if err := helperExec(opts.DB, sql); err != nil {
		return fmt.Errorf("replaceColumnCopyCast %s: %w", kind, err)
	}
	return nil
}

// ReplaceColumnDropObsolete drops ObsoleteColumnName(fromColumn). Missing column is a no-op.
func ReplaceColumnDropObsolete(opts HelperOptions, table, fromColumn string) error {
	if err := opts.validate(); err != nil {
		return err
	}
	table = strings.TrimSpace(table)
	fromColumn = strings.TrimSpace(fromColumn)
	if table == "" || fromColumn == "" {
		return fmt.Errorf("replaceColumnDropObsolete requires table and fromColumn")
	}
	obsolete := ObsoleteColumnName(fromColumn)
	has, err := helperHasColumn(opts.DB, table, obsolete)
	if err != nil {
		return fmt.Errorf("replaceColumnDropObsolete: %w", err)
	}
	if !has {
		return nil
	}
	return DropColumn(opts, table, obsolete)
}

func helperHasColumn(db *gorm.DB, table, column string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	types, err := getColumnTypes(db, table)
	if err != nil {
		return false, err
	}
	for _, ct := range types {
		if ct == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(ct.Name()), column) {
			return true, nil
		}
	}
	return false, nil
}

func copyCastExpr(dialect, quotedFrom, kind string) (string, error) {
	switch kind {
	case "int_to_varchar", "as_text":
		switch strings.ToLower(strings.TrimSpace(dialect)) {
		case "mysql", "mariadb":
			return fmt.Sprintf("CAST(%s AS CHAR)", quotedFrom), nil
		case "sqlserver":
			return fmt.Sprintf("CAST(%s AS NVARCHAR(MAX))", quotedFrom), nil
		default:
			// sqlite / postgres
			return fmt.Sprintf("CAST(%s AS TEXT)", quotedFrom), nil
		}
	default:
		return "", fmt.Errorf("replaceColumnCopyCast: unknown kind %q (want int_to_varchar or as_text)", kind)
	}
}
