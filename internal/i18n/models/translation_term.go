// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

const (
	// KindLiteral is the default term kind (source literal; ≠ programming type).
	KindLiteral = "literal"

	// SourcePackaged is set when the term comes from module PO import.
	SourcePackaged = "packaged"
	// SourceOverride is set when the term was edited via Terminology Editor.
	SourceOverride = "override"

	coreApplication = "core"
)

// NormalizeKind returns a known kind or KindLiteral when empty/unknown-blank.
func NormalizeKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return KindLiteral
	}
	return kind
}

// TranslationTerm is the per-application terminology storage row.
// Physical table name is {application}_translation_term (see TranslationTermTableName).
// Runtime tables are created by TranslationTerm MetaModel AutoMigrate (unique key on
// Module, Lang, Scope, Src, Kind is ensured by the TS model / migrate path).
type TranslationTerm struct {
	meta.BaseModel `gorm:"embedded"`

	Application string `gorm:"column:application;type:varchar(255);not null;index"`
	Module      string `gorm:"column:module;type:varchar(255);not null;index"`
	Lang        string `gorm:"column:lang;type:varchar(32);not null;index"`
	Scope       string `gorm:"column:scope;type:varchar(512);not null;index"`
	Src         string `gorm:"column:src;type:text;not null"`
	Value       string `gorm:"column:value;type:text"`
	Kind        string `gorm:"column:kind;type:varchar(64);not null;default:literal;index"`
	Source      string `gorm:"column:source;type:varchar(32);not null;default:packaged"`
	Comments    string `gorm:"column:comments;type:text"`
}

// ValidApplicationIdentifier reports whether application is safe to use in
// physical table/index names interpolated into DDL (alnum + underscore).
func ValidApplicationIdentifier(application string) bool {
	application = strings.TrimSpace(application)
	if application == "" {
		return false
	}
	for _, r := range application {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// TranslationTermTableName returns {application}_translation_term.
func TranslationTermTableName(application string) string {
	return strings.TrimSpace(application) + "_translation_term"
}

// MigrateTranslationTermTable creates the physical table for unit-test fixtures.
// Production installs rely on TranslationTerm MetaModel AutoMigrate only.
func MigrateTranslationTermTable(runtimeScope scope.Scope, application string) error {
	application = strings.TrimSpace(application)
	if application == "" || application == coreApplication {
		return nil
	}
	if !ValidApplicationIdentifier(application) {
		return fmt.Errorf("invalid application name: %q", application)
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}

	tableName := TranslationTermTableName(application)
	db := runtimeScope.Session().Table(tableName)
	if err := db.AutoMigrate(&TranslationTerm{}); err != nil {
		return fmt.Errorf("migrate %s: %w", tableName, err)
	}
	if err := ensureTranslationTermUniqueIndex(runtimeScope, tableName); err != nil {
		return fmt.Errorf("migrate %s unique index: %w", tableName, err)
	}
	return nil
}

func translationTermUniqueIndexName(tableName string) string {
	return "uq_" + tableName + "_key"
}

func ensureTranslationTermUniqueIndex(runtimeScope scope.Scope, tableName string) error {
	db := runtimeScope.Session().DB
	indexName := translationTermUniqueIndexName(tableName)
	if db.Migrator().HasIndex(tableName, indexName) {
		return nil
	}
	sql := createUniqueIndexSQL(db.Dialector.Name(), tableName, indexName)
	return db.Exec(sql).Error
}

func createUniqueIndexSQL(dialect, tableName, indexName string) string {
	cols := "module, lang, scope, src, kind"
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres":
		return fmt.Sprintf(`CREATE UNIQUE INDEX "%s" ON "%s" (%s)`, indexName, tableName, cols)
	case "mysql":
		mysqlCols := "module(64), lang, scope(255), src(255), kind"
		return fmt.Sprintf("CREATE UNIQUE INDEX `%s` ON `%s` (%s)", indexName, tableName, mysqlCols)
	default: // sqlite and others
		return fmt.Sprintf("CREATE UNIQUE INDEX `%s` ON `%s` (%s)", indexName, tableName, cols)
	}
}
