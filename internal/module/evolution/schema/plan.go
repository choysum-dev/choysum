// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
)

// buildPlan diffs desired vs live. P0 emits create_table / add_column (Auto) and
// alter_column (Guarded) for type/null mismatches; leftovers are reported only.
func buildPlan(moduleName string, desired DesiredSchema, live LiveSchema, dialect string) SchemaPlan {
	plan := SchemaPlan{
		Module:   strings.TrimSpace(moduleName),
		Ops:      nil,
		Leftover: nil,
	}

	for table, cols := range desired.Tables {
		exists := live.Tables[table]
		if !exists {
			copied := append([]ColumnSpec(nil), cols...)
			plan.Ops = append(plan.Ops, PlanOp{
				Kind:    OpCreateTable,
				Safety:  SafetyAuto,
				Table:   table,
				Detail:  "create table",
				Columns: copied,
			})
			continue
		}

		liveCols := live.Columns[table]
		rowCount := live.RowCount[table]
		desiredNames := map[string]struct{}{}
		for i := range cols {
			col := cols[i]
			desiredNames[strings.ToLower(col.Name)] = struct{}{}
			liveCol, ok := liveCols[strings.ToLower(col.Name)]
			if !ok {
				safety := SafetyAuto
				detail := "add column"
				if col.NotNull && rowCount > 0 {
					safety = SafetyGuarded
					detail = "add NOT NULL column on non-empty table"
				}
				colCopy := col
				plan.Ops = append(plan.Ops, PlanOp{
					Kind:   OpAddColumn,
					Safety: safety,
					Table:  table,
					Detail: detail,
					Column: &colCopy,
				})
				continue
			}
			if mismatch, reason := columnMismatch(col, liveCol, dialect); mismatch {
				colCopy := col
				plan.Ops = append(plan.Ops, PlanOp{
					Kind:   OpAlterColumn,
					Safety: SafetyGuarded,
					Table:  table,
					Detail: reason,
					Column: &colCopy,
				})
			}
		}

		for name := range liveCols {
			if _, ok := desiredNames[name]; ok {
				continue
			}
			plan.Leftover = append(plan.Leftover, Leftover{
				Kind:  LeftoverColumn,
				Table: table,
				Name:  liveCols[name].Name,
			})
		}
	}

	return plan
}

func columnMismatch(desired ColumnSpec, live LiveColumn, dialect string) (bool, string) {
	wantType := normalizeDBType(mapPhysicalToDialectType(dialect, desired.PhysicalType))
	haveType := normalizeDBType(live.DatabaseTypeName)
	if wantType != "" && haveType != "" && wantType != haveType {
		// SQLite affinity: many types surface as TEXT/INTEGER/REAL/BLOB/NUMERIC.
		if dialect == "sqlite" && sqliteTypeCompatible(wantType, haveType) {
			// continue to nullability / size checks
		} else {
			return true, fmt.Sprintf("type change %s → %s (desired physical %s)", haveType, wantType, desired.PhysicalType)
		}
	}
	if live.Nullable != nil {
		liveNullable := *live.Nullable
		if desired.NotNull && liveNullable {
			return true, "tighten nullability to NOT NULL"
		}
		if !desired.NotNull && !liveNullable {
			return true, "loosen nullability to NULL"
		}
	}
	if lengthMeaningful(wantType, desired.PhysicalType) &&
		desired.Size != nil && live.Length != nil && *live.Length > 0 &&
		int64(*desired.Size) != *live.Length {
		return true, fmt.Sprintf("size change %d → %d", *live.Length, *desired.Size)
	}
	return false, ""
}

func lengthMeaningful(normalizedType, physical string) bool {
	switch normalizedType {
	case "varchar", "char":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(physical)) {
	case "varchar", "char":
		return true
	default:
		return false
	}
}

func mapPhysicalToDialectType(dialect, physical string) string {
	physical = strings.TrimSpace(physical)
	if physical == "" {
		return ""
	}
	if mappings, ok := dialectTypeMappings[dialect]; ok {
		if mapped, ok := mappings[physical]; ok {
			return mapped
		}
	}
	if physical == "decimal" || physical == "monetary" {
		return "decimal"
	}
	if physical == "char" {
		return "char"
	}
	return physical
}

func normalizeDBType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "("); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	switch s {
	case "character varying", "nvarchar", "varchar":
		return "varchar"
	case "character", "nchar", "char":
		return "char"
	case "integer", "int", "int4", "mediumint", "smallint", "tinyint":
		return "int"
	case "bigint", "int8":
		return "bigint"
	case "boolean", "bool", "bit":
		return "bool"
	case "double precision", "double", "float", "float8", "real", "float4":
		return "float"
	case "bytea", "longblob", "mediumblob", "blob", "varbinary":
		return "blob"
	case "jsonb", "json":
		return "jsonobject"
	case "longtext", "mediumtext", "text", "clob":
		return "text"
	case "timestamp with time zone", "timestamptz", "timestamp", "datetime", "datetime2":
		return "datetime"
	case "time without time zone", "time":
		return "time"
	case "date":
		return "date"
	case "numeric", "decimal", "number":
		return "decimal"
	default:
		return s
	}
}

func sqliteTypeCompatible(want, have string) bool {
	// SQLite ColumnTypes often report affinity buckets.
	switch have {
	case "text":
		return want == "text" || want == "varchar" || want == "char" || want == "jsonobject" || want == "date" || want == "datetime" || want == "time" || want == "html"
	case "integer":
		return want == "int" || want == "bigint" || want == "bool"
	case "real":
		return want == "float" || want == "decimal"
	case "blob":
		return want == "blob"
	case "numeric":
		return want == "decimal" || want == "float" || want == "bool" || want == "date" || want == "datetime"
	default:
		return want == have
	}
}
