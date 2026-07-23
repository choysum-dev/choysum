// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/ettle/strcase"
	"gorm.io/gorm"
)

// Fixed L2 whitelist (data-i18n-design D17). Not read from Language / config.
var translatedL2Langs = []string{"en_US", "zh_CN"}

func translatedL2IndexName(tableName, columnName, lang string) string {
	table := strings.TrimSpace(tableName)
	column := strings.TrimSpace(columnName)
	suffix := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(lang), "-", "_"))
	return fmt.Sprintf("idx_%s_%s_%s", table, column, suffix)
}

func createTranslatedL2IndexSQL(dialect, tableName, columnName, indexName, lang string) (string, bool) {
	table := strings.TrimSpace(tableName)
	column := strings.TrimSpace(columnName)
	index := strings.TrimSpace(indexName)
	langKey := strings.TrimSpace(lang)
	if table == "" || column == "" || index == "" || langKey == "" {
		return "", false
	}
	path := "$." + langKey
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "sqlite":
		// Expression index; equality on json_extract can use it.
		return fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS "%s" ON "%s" (json_extract("%s", '%s'))`,
			index, table, column, path,
		), true
	case "mysql", "mariadb":
		// Functional index (MySQL 8.0.13+). CAST keeps a bounded btree key.
		return fmt.Sprintf(
			"CREATE INDEX `%s` ON `%s` ((CAST(JSON_UNQUOTE(JSON_EXTRACT(`%s`, '%s')) AS CHAR(512))))",
			index, table, column, path,
		), true
	case "sqlserver", "mssql":
		// No portable expression-index DDL without persisted computed columns; skip (D17 best-effort).
		return "", false
	default:
		return "", false
	}
}

// ensureTranslatedL2Indexes creates fixed-whitelist (en_US, zh_CN) expression indexes on non-PG dialects.
// Postgres uses L1 trigram instead. Unknown / unsupported dialects are no-ops.
func ensureTranslatedL2Indexes(db *gorm.DB, dialect, tableName, fieldName string) error {
	if db == nil {
		return nil
	}
	d := strings.ToLower(strings.TrimSpace(dialect))
	if d == "postgres" || d == "postgresql" || d == "" || d == "unknown" {
		return nil
	}
	table := strings.TrimSpace(tableName)
	column := strcase.ToSnake(strings.TrimSpace(fieldName))
	if table == "" || column == "" {
		return nil
	}
	for _, lang := range translatedL2Langs {
		indexName := translatedL2IndexName(table, column, lang)
		if db.Migrator().HasIndex(table, indexName) {
			continue
		}
		sql, ok := createTranslatedL2IndexSQL(d, table, column, indexName, lang)
		if !ok || sql == "" {
			continue
		}
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("ensure translated L2 index %s: %w", indexName, err)
		}
	}
	return nil
}

func (m *modelMigrator) applyTableTranslatedL2Indexes(tableName string, model *meta.IrModel) error {
	dialect := m.getDialect()
	if dialect == "postgres" || dialect == "unknown" {
		return nil
	}
	db := m.runtimeScope.Session().DB
	for _, field := range model.Fields {
		// Same opt-in as L1: translate + index:'trigram' (§7.3).
		if !isTranslatedTrigramField(field) {
			continue
		}
		if err := ensureTranslatedL2Indexes(db, dialect, tableName, field.Name); err != nil {
			return err
		}
	}
	return nil
}
