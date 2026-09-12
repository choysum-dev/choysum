// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
)

// buildPlan diffs desired vs live.
// Auto: create_table, add_column (nullable / empty table), varchar widen, add_index,
// ensure_check. Guarded: type/null/default changes, varchar narrow, unique on populated.
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
			for _, col := range cols {
				plan.Ops = append(plan.Ops, checkOpsForColumn(table, col)...)
			}
			continue
		}

		liveCols := live.Columns[table]
		rowCount := live.RowCount[table]
		desiredNames := map[string]struct{}{}
		desiredIndexKeys := map[string]struct{}{}

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
				// Indexes for new columns are created in apply after AddColumn.
				plan.Ops = append(plan.Ops, checkOpsForColumn(table, col)...)
				continue
			}

			diffs := columnDiffs(col, liveCol, dialect)
			for _, d := range diffs {
				colCopy := col
				plan.Ops = append(plan.Ops, PlanOp{
					Kind:   OpAlterColumn,
					Safety: d.Safety,
					Table:  table,
					Detail: d.Reason,
					Column: &colCopy,
				})
			}

			for _, name := range indexLookupNames(col) {
				desiredIndexKeys[strings.ToLower(name)] = struct{}{}
				if liveHasIndex(live, table, name) {
					continue
				}
				safety := SafetyAuto
				detail := "add index " + name
				if (col.UniqueIndex || col.Unique) && rowCount > 0 {
					safety = SafetyGuarded
					detail = "add unique index on non-empty table"
				}
				colCopy := col
				plan.Ops = append(plan.Ops, PlanOp{
					Kind:      OpAddIndex,
					Safety:    safety,
					Table:     table,
					Detail:    detail,
					Column:    &colCopy,
					IndexName: name,
				})
			}

			plan.Ops = append(plan.Ops, checkOpsForColumn(table, col)...)
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
		for idxName := range live.Indexes[table] {
			if _, ok := desiredIndexKeys[strings.ToLower(idxName)]; ok {
				continue
			}
			// Skip automatic primary-key / sqlite internal names from leftover noise when possible.
			if strings.EqualFold(idxName, "sqlite_autoindex_"+table+"_1") {
				continue
			}
			plan.Leftover = append(plan.Leftover, Leftover{
				Kind:  LeftoverIndex,
				Table: table,
				Name:  idxName,
			})
		}
	}

	return plan
}

func checkOpsForColumn(table string, col ColumnSpec) []PlanOp {
	expr := strings.TrimSpace(col.CheckExpr)
	if expr == "" {
		return nil
	}
	columnName := col.Name
	if columnName == "" {
		columnName = strings.ToLower(col.FieldName)
	}
	name := fmt.Sprintf("chk_%s_%s", table, columnName)
	return []PlanOp{{
		Kind:      OpEnsureCheck,
		Safety:    SafetyAuto,
		Table:     table,
		Detail:    "ensure check " + name,
		CheckName: name,
		CheckExpr: expr,
		Column:    &col,
	}}
}

func liveHasIndex(live LiveSchema, table, name string) bool {
	if live.Indexes == nil {
		return false
	}
	m := live.Indexes[table]
	if m == nil {
		return false
	}
	return m[strings.ToLower(name)]
}

type columnDiff struct {
	Reason string
	Safety SafetyClass
}

func columnDiffs(desired ColumnSpec, live LiveColumn, dialect string) []columnDiff {
	var out []columnDiff
	wantType := normalizeDBType(mapPhysicalToDialectType(dialect, desired.PhysicalType))
	haveType := normalizeDBType(live.DatabaseTypeName)
	if wantType != "" && haveType != "" && wantType != haveType {
		if dialect == "sqlite" && sqliteTypeCompatible(wantType, haveType) {
			// continue
		} else {
			out = append(out, columnDiff{
				Reason: fmt.Sprintf("type change %s → %s (desired physical %s)", haveType, wantType, desired.PhysicalType),
				Safety: SafetyGuarded,
			})
		}
	}
	if live.Nullable != nil {
		liveNullable := *live.Nullable
		if desired.NotNull && liveNullable {
			out = append(out, columnDiff{Reason: "tighten nullability to NOT NULL", Safety: SafetyGuarded})
		}
		if !desired.NotNull && !liveNullable {
			out = append(out, columnDiff{Reason: "loosen nullability to NULL", Safety: SafetyGuarded})
		}
	}
	if dialect != "sqlite" &&
		lengthMeaningful(wantType, desired.PhysicalType) &&
		desired.Size != nil && live.Length != nil && *live.Length > 0 {
		liveLen := int(*live.Length)
		wantLen := *desired.Size
		if wantLen > liveLen {
			out = append(out, columnDiff{
				Reason: fmt.Sprintf("widen size %d → %d", liveLen, wantLen),
				Safety: SafetyAuto,
			})
		} else if wantLen < liveLen {
			out = append(out, columnDiff{
				Reason: fmt.Sprintf("narrow size %d → %d", liveLen, wantLen),
				Safety: SafetyGuarded,
			})
		}
	}
	if defaultChanged(desired, live) {
		out = append(out, columnDiff{Reason: "change column default", Safety: SafetyGuarded})
	}
	return out
}

func defaultChanged(desired ColumnSpec, live LiveColumn) bool {
	if desired.Default == nil {
		return false
	}
	want := strings.TrimSpace(*desired.Default)
	if want == "" {
		return false
	}
	if live.Default == nil {
		// Unknown live default: do not fail closed (dialects often omit default metadata).
		return false
	}
	have := strings.TrimSpace(*live.Default)
	return !strings.EqualFold(strings.Trim(want, `"'`), strings.Trim(have, `"'`))
}

// columnMismatch is retained for tests; returns true when any guarded/auto alter is needed.
func columnMismatch(desired ColumnSpec, live LiveColumn, dialect string) (bool, string) {
	diffs := columnDiffs(desired, live, dialect)
	if len(diffs) == 0 {
		return false, ""
	}
	return true, diffs[0].Reason
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
