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

	// Count NULLs then apply DDL. Non-SQLite dialects run both in one transaction so a
	// concurrent NULL insert cannot race the check. SQLite skips the outer transaction:
	// rewrite must disable foreign_keys outside an open txn, and the rewritten NOT NULL
	// column still rejects NULLs on INSERT.
	run := func(db *gorm.DB) error {
		txOpts := opts
		txOpts.DB = db
		qTable := quoteIdent(opts.Dialect, table)
		qCol := quoteIdent(opts.Dialect, column)
		var nulls int64
		if err := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NULL", qTable, qCol)).Scan(&nulls).Error; err != nil {
			return fmt.Errorf("setNotNull count nulls %s.%s: %w", table, column, err)
		}
		if nulls > 0 {
			return fmt.Errorf("setNotNull: %s.%s still has %d NULL row(s)", table, column, nulls)
		}
		return applySetNotNull(txOpts, table, column)
	}
	if isSQLiteDialect(opts.Dialect) {
		// NULL count is not in the same txn as the rewrite (FK pragma must toggle
		// outside an open txn). A racing NULL insert surfaces as a copy failure.
		if err := run(opts.DB); err != nil {
			return err
		}
	} else if err := opts.DB.Transaction(func(tx *gorm.DB) error { return run(tx) }); err != nil {
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
		// MODIFY COLUMN replaces the full definition; we preserve type/length/decimal,
		// DEFAULT, and AUTO_INCREMENT only (not COMMENT/COLLATE/ON UPDATE).
		typeName, err := liveColumnTypeName(opts.DB, table, column, true, dialect)
		if err != nil {
			return err
		}
		typeName = normalizeMySQLTypeName(typeName)
		ai, aiErr := liveColumnIsAutoIncrement(opts.DB, table, column)
		if aiErr != nil {
			return fmt.Errorf("setNotNull mysql auto_increment: %w", aiErr)
		}
		if ai {
			typeName += " AUTO_INCREMENT"
		}
		sql := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NOT NULL", qTable, qCol, typeName)
		if err := helperExec(opts.DB, sql); err != nil {
			return fmt.Errorf("setNotNull mysql: %w", err)
		}
		return nil
	case "sqlserver":
		// T-SQL ALTER COLUMN does not accept a DEFAULT clause in the type string.
		typeName, err := liveColumnTypeName(opts.DB, table, column, false, dialect)
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

func liveColumnTypeName(db *gorm.DB, table, column string, withDefault bool, dialect string) (string, error) {
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
		if withDefault {
			rawDef, defOK := ct.DefaultValue()
			escapeBS := false
			if isMySQLFamily(dialect) {
				escapeBS = mysqlEscapesBackslash(db)
			}
			if def := formatLiveColumnDefault(dialect, typeName, rawDef, defOK, escapeBS); def != "" {
				typeName += " DEFAULT " + def
			}
		}
		return typeName, nil
	}
	return "", fmt.Errorf("column %s.%s not found", table, column)
}

func isMySQLFamily(dialect string) bool {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql", "mariadb":
		return true
	default:
		return false
	}
}

// mysqlSessionSQLMode is overridable in tests.
var mysqlSessionSQLMode = func(db *gorm.DB) (string, error) {
	var mode string
	err := db.Raw("SELECT @@SESSION.sql_mode").Scan(&mode).Error
	return mode, err
}

// mysqlEscapesBackslash reports whether string literals need backslash escaping
// (true unless SESSION sql_mode includes NO_BACKSLASH_ESCAPES).
func mysqlEscapesBackslash(db *gorm.DB) bool {
	if db == nil {
		return true
	}
	mode, err := mysqlSessionSQLMode(db)
	if err != nil {
		return true
	}
	return !strings.Contains(strings.ToUpper(mode), "NO_BACKSLASH_ESCAPES")
}

// formatLiveColumnDefault turns driver ColumnType.DefaultValue into a DDL fragment,
// or "" to omit DEFAULT. MySQL's gorm driver strips surrounding quotes from string
// defaults, so stringish types are re-quoted here without TrimSpace (preserves
// leading/trailing spaces in literals).
func formatLiveColumnDefault(dialect, typeName, def string, ok bool, escapeBackslash bool) string {
	if !ok {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql", "mariadb":
		return formatMySQLColumnDefault(typeName, def, escapeBackslash)
	default:
		def = strings.TrimSpace(def)
		// SQLite and others typically return DDL-ready literals/expressions.
		if def == "" || strings.EqualFold(def, "NULL") {
			return ""
		}
		return def
	}
}

func formatMySQLColumnDefault(typeName, def string, escapeBackslash bool) string {
	// Stringish columns: always quote the driver value as a literal. Do not infer
	// expressions from contents (e.g. DEFAULT '(1)' must stay '(1)', not (1)).
	if isStringishSQLType(typeName) {
		return quoteMySQLStringLiteral(def, escapeBackslash)
	}
	def = strings.TrimSpace(def)
	if looksLikeSQLDefaultExpr(def) {
		return def
	}
	if def == "" || strings.EqualFold(def, "NULL") {
		return ""
	}
	return def
}

func looksLikeSQLDefaultExpr(def string) bool {
	if def == "" {
		return false
	}
	if strings.HasPrefix(def, "(") {
		return true
	}
	upper := strings.ToUpper(def)
	switch {
	case strings.HasPrefix(upper, "CURRENT_"),
		strings.HasPrefix(upper, "LOCALTIME"),
		strings.HasPrefix(upper, "LOCALTIMESTAMP"),
		strings.HasPrefix(upper, "NOW("):
		return true
	default:
		return false
	}
}

func isStringishSQLType(typeName string) bool {
	lower := strings.ToLower(typeName)
	for _, tok := range []string{"char", "text", "blob", "binary", "enum", "set", "json"} {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	return false
}

func quoteMySQLStringLiteral(s string, escapeBackslash bool) string {
	if escapeBackslash {
		s = strings.ReplaceAll(s, `\`, `\\`)
	}
	s = strings.ReplaceAll(s, `'`, `''`)
	return "'" + s + "'"
}

// appendableColumnDefault is the SQLite/passthrough helper used by table rewrite.
func appendableColumnDefault(def string, ok bool) string {
	return formatLiveColumnDefault("sqlite", "", def, ok, false)
}

// normalizeMySQLTypeName rewrites driver forms like "UNSIGNED INT" to "INT UNSIGNED".
func normalizeMySQLTypeName(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return typeName
	}
	upper := strings.ToUpper(typeName)
	const prefix = "UNSIGNED "
	if strings.HasPrefix(upper, prefix) {
		return strings.TrimSpace(typeName[len(prefix):]) + " UNSIGNED"
	}
	return typeName
}

func liveColumnIsAutoIncrement(db *gorm.DB, table, column string) (bool, error) {
	types, err := getColumnTypes(db, table)
	if err != nil {
		return false, err
	}
	for _, ct := range types {
		if ct == nil || !strings.EqualFold(strings.TrimSpace(ct.Name()), column) {
			continue
		}
		ai, ok := ct.AutoIncrement()
		return ok && ai, nil
	}
	return false, nil
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
