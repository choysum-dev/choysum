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
	"int":      0,
	"bigint":   int64(0),
	"float":    0.0,
	"number":   0.0,
	"decimal":  decimal.Decimal{},
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

	case "decimal":
		// Cross-dialect convention: fixed DECIMAL(38,18), ignore meta precision/scale.
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
		(strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) {
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

// addStandardTags appends standard gorm tags.
func addStandardTags(tags *[]string, meta map[string]interface{}) {
	// Primary key
	if v, ok := meta["primaryKey"].(bool); ok && v {
		*tags = append(*tags, "primaryKey")
	}

	// Not-null constraint
	if v, ok := meta["notNull"].(bool); ok && v {
		*tags = append(*tags, "not null")
	}

	// Unique constraint
	if v, ok := meta["unique"].(bool); ok && v {
		*tags = append(*tags, "unique")
	}

	// Index handling: supports string and boolean values.
	if v, ok := meta["index"]; ok {
		switch val := v.(type) {
		case bool:
			if val {
				*tags = append(*tags, "index")
			}
		case string:
			if val != "" {
				*tags = append(*tags, fmt.Sprintf("index:%s", val))
			}
		}
	}

	// Unique index handling: supports string and boolean values.
	if v, ok := meta["uniqueIndex"]; ok {
		switch val := v.(type) {
		case bool:
			if val {
				*tags = append(*tags, "uniqueIndex")
			}
		case string:
			if val != "" {
				// Support multiple unique index names in one string, separated by whitespace.
				// This matches TS decorator usage like: uniqueIndex: "idx_a idx_b" on a shared column.
				for _, part := range strings.Fields(val) {
					if part == "" {
						continue
					}
					*tags = append(*tags, fmt.Sprintf("uniqueIndex:%s", part))
				}
			}
		}
	}

	// Default value handling.
	if v, ok := meta["default"]; ok {
		switch val := v.(type) {
		case string:
			trimmed := strings.TrimSpace(val)
			if trimmed != "" && !isJSFunctionDefaultLiteral(trimmed) {
				*tags = append(*tags, fmt.Sprintf("default:%s", trimmed))
			}
		case bool, int, int32, int64, uint, uint32, uint64, float32, float64:
			*tags = append(*tags, fmt.Sprintf("default:%v", val))
		}
	}

	// CHECK constraint via gorm tag (helps MySQL/SQLite/SQLServer create it during migration).
	// - Use `check:,<expr>` to force default naming: chk_<table>_<column>.
	// - Normalize expressions to avoid SQL syntax errors caused by template quoting/whitespace.
	if v, ok := meta["checkConstraint"].(string); ok {
		expr := normalizeCheckExpr(v)
		if expr != "" {
			*tags = append(*tags, "check:,"+expr)
		}
	}
}

type ModelMigrator interface {
	MigrateSchema() error
}

type JoinTableMigrator interface {
	MigrateJoinTables() error
}

type ForeignKeyMigrator interface {
	MigrateForeignKeys() error
}
