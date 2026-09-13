// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/shopspring/decimal"
)

var defaultValues = map[string]interface{}{
	"char":     "",
	"varchar":  "",
	"text":     "",
	"html":     "",
	"int":      0,
	"bigint":   int64(0),
	"float":    0.0,
	"number":   0.0,
	"decimal":  decimal.Decimal{},
	"monetary": decimal.Decimal{},
	"bool":     false,
	"boolean":  false,
	"time":     time.Time{},
	"datetime": time.Time{},
	// date: use string to avoid sqlite DATE NUMERIC affinity quirks and to align with API contract (YYYY-MM-DD).
	"date":       "",
	"bytes":      []byte{},
	"blob":       []byte{},
	"jsonobject": datatypes.JSON([]byte("{}")),
}

var dialectTypeMappings = map[string]map[string]string{
	"postgres": {
		"datetime":   "timestamp with time zone",
		"time":       "time without time zone",
		"date":       "date",
		"bool":       "boolean",
		"int":        "integer",
		"bigint":     "bigint",
		"float":      "double precision",
		"number":     "double precision",
		"text":       "text",
		"html":       "text",
		"blob":       "bytea",
		"jsonobject": "jsonb",
		"varchar":    "varchar",
	},
	"mysql": {
		"datetime":   "datetime",
		"time":       "time",
		"date":       "date",
		"bool":       "tinyint(1)",
		"int":        "int",
		"bigint":     "bigint",
		"float":      "double",
		"number":     "double",
		"text":       "longtext",
		"html":       "longtext",
		"blob":       "longblob",
		"jsonobject": "json",
		"varchar":    "varchar",
	},
	"sqlite": {
		"datetime": "datetime",
		"time":     "time",
		// SQLite has no real DATE type; using DATE yields NUMERIC affinity which breaks comparisons/unique indexes for YYYY-MM-DD.
		"date":       "text",
		"bool":       "boolean",
		"int":        "integer",
		"bigint":     "bigint",
		"float":      "real",
		"number":     "real",
		"text":       "text",
		"html":       "text",
		"blob":       "blob",
		"jsonobject": "json",
		"varchar":    "varchar",
	},
	"sqlserver": {
		"datetime":   "datetime2",
		"time":       "time",
		"date":       "date",
		"bool":       "bit",
		"int":        "int",
		"bigint":     "bigint",
		"float":      "float",
		"number":     "float",
		"text":       "nvarchar(max)",
		"html":       "nvarchar(max)",
		"blob":       "varbinary(max)",
		"jsonobject": "nvarchar(max)",
		"varchar":    "nvarchar",
	},
}

// getDefaultValue returns the default value for the given column type.
func getDefaultValue(columnType string) interface{} {
	if v, ok := defaultValues[columnType]; ok {
		return v
	}
	return nil
}

// buildColumnTypeTag builds the gorm type tag for a column type.
func buildColumnTypeTag(dialect string, columnType string, meta map[string]interface{}) string {

	if dialectMappings, ok := dialectTypeMappings[dialect]; ok {
		if mappedType, ok := dialectMappings[columnType]; ok {
			columnType = mappedType
		}
	}

	switch columnType {
	case "char":
		if size, ok := meta["size"]; ok {
			sizeInt, err := strconv.Atoi(fmt.Sprintf("%v", size))
			if err == nil && sizeInt > 0 {
				return fmt.Sprintf("type:%s(%d)", columnType, sizeInt)
			}
		}
		// Missing or invalid size: use the smallest valid length.
		return fmt.Sprintf("type:%s(1)", columnType)

	case "varchar":
		if size, ok := meta["size"]; ok {
			sizeInt, err := strconv.Atoi(fmt.Sprintf("%v", size))
			if err == nil && sizeInt > 0 {
				return fmt.Sprintf("type:%s(%d)", columnType, sizeInt)
			}
		}
		// Cross-dialect safe default when size is omitted.
		// - MySQL/SQL Server require length: use 255.
		// - PostgreSQL/SQLite allow no length, but use 255 for consistent behavior.
		return fmt.Sprintf("type:%s(255)", columnType)

	case "decimal", "monetary":
		// Cross-dialect convention: fixed DECIMAL(38,18), ignore meta precision/scale.
		// monetary shares physical storage with decimal (logical FieldType may stay monetary).
		return "type:decimal(38,18)"

	case "bigint":
		return "type:bigint"

	default:
		return "type:" + columnType
	}
}

func isJSFunctionDefaultLiteral(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	// Quoted string literals are not JS functions.
	if (strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) ||
		(strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) ||
		(strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`")) {
		return false
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "=>") {
		return true
	}
	if strings.HasPrefix(lower, "async function") || strings.HasPrefix(lower, "function") {
		return strings.Contains(lower, "(") || strings.Contains(lower, "{")
	}
	return false
}

func isQuotedStringLiteral(value string) bool {
	return (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
		(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`"))
}

func isKnownSQLDefaultKeyword(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NULL", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME", "TRUE", "FALSE":
		return true
	default:
		return false
	}
}

func normalizeDefaultStringLiteral(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if isQuotedStringLiteral(trimmed) {
		return trimmed
	}
	if isKnownSQLDefaultKeyword(trimmed) {
		return trimmed
	}
	if strings.ContainsAny(trimmed, "()") {
		return trimmed
	}
	escaped := strings.ReplaceAll(trimmed, "'", "''")
	return "'" + escaped + "'"
}

type ModelMigrator interface {
	MigrateSchema() error
	PlanSchema() (SchemaPlan, error)
}

type ForeignKeyMigrator interface {
	MigrateForeignKeys() error
}
