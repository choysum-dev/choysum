// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openPostgres wraps the upstream postgres dialector so AutoMigrate treats
// PostgreSQL's udt_name "bpchar" as the same type as gorm tags type:char(N) /
// type:character(N). Without this alias, every migrateBaseModule AutoMigrate
// rewrites already-char(20) columns (AccessExclusiveLock) and can hang Plan
// behind any idle-in-transaction reader.
//
// Embed *postgres.Dialector (not gorm.Dialector) so optional interfaces such as
// SavePointerDialectorInterface and ErrorTranslator stay on the concrete type;
// wrapping the interface alone drops them and nested db.Transaction fails with
// "unsupported driver".
func openPostgres(dsn string) gorm.Dialector {
	inner, ok := postgres.Open(dsn).(*postgres.Dialector)
	if !ok {
		return postgres.Open(dsn)
	}
	return postgresDialector{Dialector: inner}
}

type postgresDialector struct {
	*postgres.Dialector
}

func (d postgresDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return bpcharAwareMigrator{Migrator: d.Dialector.Migrator(db)}
}

type bpcharAwareMigrator struct {
	gorm.Migrator
}

func (m bpcharAwareMigrator) GetTypeAliases(databaseTypeName string) []string {
	aliases := m.Migrator.GetTypeAliases(databaseTypeName)
	if strings.EqualFold(strings.TrimSpace(databaseTypeName), "bpchar") {
		return appendCharAliasesForBPChar(aliases)
	}
	return aliases
}

func appendCharAliasesForBPChar(aliases []string) []string {
	hasChar, hasCharacter := false, false
	for _, alias := range aliases {
		switch strings.ToLower(strings.TrimSpace(alias)) {
		case "char":
			hasChar = true
		case "character":
			hasCharacter = true
		}
	}
	out := make([]string, 0, len(aliases)+2)
	out = append(out, aliases...)
	if !hasChar {
		out = append(out, "char")
	}
	if !hasCharacter {
		out = append(out, "character")
	}
	return out
}
