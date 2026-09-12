// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func normalizeCheckExpr(expr string) string {
	// Some check constraints come from TS template strings and can carry
	// outer backticks in meta (e.g. `(... )`). Most SQL dialects don't accept
	// backticks inside CHECK expressions, so strip them (and defensively strip
	// outer single-quotes).
	trimmed := strings.TrimSpace(expr)
	for {
		if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`") {
			trimmed = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "`"), "`"))
			continue
		}
		if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") {
			trimmed = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "'"), "'"))
			continue
		}
		break
	}

	// Collapse all whitespace (including newlines) into single spaces.
	collapsed := strings.Join(strings.Fields(trimmed), " ")
	collapsed = strings.TrimSpace(collapsed)
	if collapsed == "" {
		return ""
	}
	if strings.HasPrefix(collapsed, "(") {
		return collapsed
	}
	return "(" + collapsed + ")"
}

func ensureCheckConstraint(db *gorm.DB, dialect, tableName, constraintName, expr string) error {
	switch dialect {
	case "postgres":
		return ensurePostgresCheckConstraint(db, tableName, constraintName, expr)
	case "mysql":
		return ensureMySQLCheckConstraint(db, tableName, constraintName, expr)
	case "sqlserver":
		return ensureSQLServerCheckConstraint(db, tableName, constraintName, expr)
	case "sqlite":
		// SQLite can't ALTER TABLE ... ADD CONSTRAINT; only CREATE TABLE can define CHECK.
		// We still emit gorm check tags so newly created tables include the constraint.
		return nil
	default:
		return nil
	}
}

func ensurePostgresCheckConstraint(db *gorm.DB, tableName, constraintName, expr string) error {
	expr = normalizeCheckExpr(expr)
	if expr == "" {
		return nil
	}

	// Postgres doesn't support ADD CONSTRAINT IF NOT EXISTS for CHECK, so we
	// drop+add to make migrations idempotent and allow expression updates.
	if err := db.Exec(
		fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT IF EXISTS "%s"`, tableName, constraintName),
	).Error; err != nil {
		return err
	}

	return db.Exec(
		fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT "%s" CHECK %s`, tableName, constraintName, expr),
	).Error
}

func dropCheckConstraintBestEffort(db *gorm.DB, dialect, tableName, constraintName string) error {
	dialect = strings.ToLower(strings.TrimSpace(dialect))
	qTable := quoteIdent(dialect, tableName)
	qName := quoteIdent(dialect, constraintName)
	switch dialect {
	case "postgres":
		return execSQLFn(db, fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`, qTable, qName))
	case "mysql", "mariadb":
		// MySQL/MariaDB don't support IF EXISTS for DROP CHECK. Check existence first.
		count, err := mysqlCheckConstraintCountFn(db, tableName, constraintName)
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		// Prefer DROP CHECK (MySQL 8.0.16+ / MariaDB). Fall back to DROP CONSTRAINT.
		dropSQL := fmt.Sprintf("ALTER TABLE %s DROP CHECK %s", qTable, qName)
		if err := execSQLFn(db, dropSQL); err == nil {
			return nil
		}
		return execSQLFn(db, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", qTable, qName))
	case "sqlserver":
		// SQL Server doesn't have DROP CONSTRAINT IF EXISTS; guard with sys.check_constraints.
		guarded := fmt.Sprintf(
			"IF EXISTS (SELECT 1 FROM sys.check_constraints WHERE name = N'%s') ALTER TABLE %s DROP CONSTRAINT %s;",
			escapeSQLServerStringLiteral(constraintName),
			qTable,
			qName,
		)
		return execSQLFn(db, guarded)
	case "sqlite":
		return nil
	default:
		return nil
	}
}

// Overridable seams for dialect paths that need a live MySQL/SQL Server in production.
var execSQLFn = func(db *gorm.DB, sql string) error {
	return db.Exec(sql).Error
}

var mysqlCheckConstraintCountFn = func(db *gorm.DB, tableName, constraintName string) (int64, error) {
	var count int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ? AND constraint_type = 'CHECK'`,
		tableName,
		constraintName,
	).Scan(&count).Error
	return count, err
}

func ensureMySQLCheckConstraint(db *gorm.DB, tableName, constraintName, expr string) error {
	expr = normalizeCheckExpr(expr)
	if expr == "" {
		return nil
	}

	// Drop+add to be idempotent and allow expression updates.
	if err := dropCheckConstraintBestEffort(db, "mysql", tableName, constraintName); err != nil {
		return err
	}
	return db.Exec(
		fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` CHECK %s", tableName, constraintName, expr),
	).Error
}

func ensureSQLServerCheckConstraint(db *gorm.DB, tableName, constraintName, expr string) error {
	expr = normalizeCheckExpr(expr)
	if expr == "" {
		return nil
	}

	if err := dropCheckConstraintBestEffort(db, "sqlserver", tableName, constraintName); err != nil {
		return err
	}

	// WITH CHECK ensures existing rows are validated.
	return db.Exec(
		fmt.Sprintf("ALTER TABLE [%s] WITH CHECK ADD CONSTRAINT [%s] CHECK %s;", tableName, constraintName, expr),
	).Error
}

func escapeSQLServerStringLiteral(s string) string {
	// Escape single quotes in string literal context.
	return strings.ReplaceAll(s, "'", "''")
}
