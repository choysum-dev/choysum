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

const translatedTrigramIndexKind = "trigram"

// hasTrigram reports whether the Postgres pg_trgm extension is installed.
// Non-Postgres dialects always return false. Lookup failures return false (skip indexes; D11).
func hasTrigram(db *gorm.DB) bool {
	if db == nil || db.Dialector == nil {
		return false
	}
	dialect := strings.ToLower(strings.TrimSpace(db.Dialector.Name()))
	if dialect != "postgres" && dialect != "postgresql" {
		return false
	}
	var ok bool
	err := db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm')`).Scan(&ok).Error
	if err != nil {
		return false
	}
	return ok
}

func translatedTrigramIndexName(tableName, columnName string) string {
	table := strings.TrimSpace(tableName)
	column := strings.TrimSpace(columnName)
	return fmt.Sprintf("idx_%s_%s_trgm", table, column)
}

func isTranslatedTrigramField(field *meta.Field) bool {
	if field == nil {
		return false
	}
	spec, err := field.GetResolvedSpec()
	if err != nil || spec == nil {
		return false
	}
	if spec.Structural.Translate == nil || !*spec.Structural.Translate {
		return false
	}
	hints := spec.Structural.StorageHints
	if hints == nil || hints.Index == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(*hints.Index), translatedTrigramIndexKind)
}

func createTranslatedTrigramIndexSQL(tableName, columnName, indexName string) string {
	// Quote identifiers for Postgres; expression matches data-i18n-design §7.1.
	return fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS "%s" ON "%s" USING gin ((jsonb_path_query_array("%s", '$.*')::text) gin_trgm_ops)`,
		indexName,
		tableName,
		columnName,
	)
}

// ensureTranslatedTrigramIndex creates the full-language GIN trigram index when pg_trgm is available.
// Missing extension or non-Postgres dialect is a no-op (no fatal).
func ensureTranslatedTrigramIndex(db *gorm.DB, tableName, fieldName string) error {
	if db == nil {
		return nil
	}
	table := strings.TrimSpace(tableName)
	column := strcase.ToSnake(strings.TrimSpace(fieldName))
	if table == "" || column == "" {
		return nil
	}
	if !hasTrigram(db) {
		return nil
	}
	indexName := translatedTrigramIndexName(table, column)
	if db.Migrator().HasIndex(table, indexName) {
		return nil
	}
	sql := createTranslatedTrigramIndexSQL(table, column, indexName)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("ensure translated trigram index %s: %w", indexName, err)
	}
	return nil
}

func (m *modelMigrator) applyTableTranslatedTrigramIndexes(tableName string, model *meta.Model) error {
	if m.getDialect() != "postgres" {
		return nil
	}
	db := m.runtimeScope.Session().DB
	if !hasTrigram(db) {
		return nil
	}
	for _, field := range model.Fields {
		if !isTranslatedTrigramField(field) {
			continue
		}
		if err := ensureTranslatedTrigramIndex(db, tableName, field.Name); err != nil {
			return err
		}
	}
	return nil
}
