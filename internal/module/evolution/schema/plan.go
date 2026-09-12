// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
)

// buildPlan diffs desired vs live.
// Auto: create_table, add_column (nullable / empty table), varchar widen, add_index,
// ensure_check (idempotent on postgres/mysql/sqlserver; omitted for existing sqlite).
// Guarded: type/null/default changes, varchar narrow, unique on populated tables.
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
				plan.Ops = append(plan.Ops, checkOpsForColumn(table, col, dialect, false)...)
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
				plan.Ops = append(plan.Ops, indexOpsForColumn(table, col, live, rowCount, desiredIndexKeys)...)
				plan.Ops = append(plan.Ops, checkOpsForColumn(table, col, dialect, true)...)
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

			plan.Ops = append(plan.Ops, indexOpsForColumn(table, col, live, rowCount, desiredIndexKeys)...)
			plan.Ops = append(plan.Ops, checkOpsForColumn(table, col, dialect, true)...)
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
		for _, idx := range live.Indexes[table] {
			idxName := strings.TrimSpace(idx.Name)
			if idxName == "" {
				continue
			}
			if _, ok := desiredIndexKeys[strings.ToLower(idxName)]; ok {
				continue
			}
			if indexCoveredByDesiredKey(idx, desiredIndexKeys) {
				continue
			}
			// Skip primary-key / sqlite autoindex leftovers (informational noise).
			if idx.PrimaryKey || strings.HasPrefix(strings.ToLower(idxName), "sqlite_autoindex_") {
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

func indexOpsForColumn(table string, col ColumnSpec, live LiveSchema, rowCount int64, desiredIndexKeys map[string]struct{}) []PlanOp {
	var ops []PlanOp
	candidates := indexLookupCandidates(col)
	// Only register the physical column name when this column declares an index.
	// Otherwise leftover indexes on declared-but-unindexed columns are swallowed.
	if col.Name != "" && len(candidates) > 0 {
		desiredIndexKeys[strings.ToLower(col.Name)] = struct{}{}
	}
	for _, cand := range candidates {
		desiredIndexKeys[strings.ToLower(cand.Name)] = struct{}{}
		if liveHasIndex(live, table, cand.Name, cand.Unique) {
			continue
		}
		// Default GORM lookup uses exportIdent(FieldName) (e.g. CreatedBy); also try the
		// physical column name (created_by). Custom IndexName/UniqueIndexNames match by name only.
		if indexCandidateUsesFieldLookup(col, cand.Name) &&
			col.Name != "" &&
			liveHasIndex(live, table, col.Name, cand.Unique) {
			continue
		}
		safety := SafetyAuto
		detail := "add index " + cand.Name
		if cand.Unique && rowCount > 0 {
			safety = SafetyGuarded
			detail = "add unique index on non-empty table"
		}
		colCopy := indexPlanColumn(col)
		ops = append(ops, PlanOp{
			Kind:      OpAddIndex,
			Safety:    safety,
			Table:     table,
			Detail:    detail,
			Column:    &colCopy,
			IndexName: cand.Name,
		})
	}
	return ops
}

// indexCandidateUsesFieldLookup reports whether name came from the field export default
// (not an explicit IndexName / UniqueIndexNames entry).
func indexCandidateUsesFieldLookup(col ColumnSpec, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if col.IndexName != "" && strings.EqualFold(name, col.IndexName) {
		return false
	}
	for _, n := range col.UniqueIndexNames {
		if strings.EqualFold(name, n) {
			return false
		}
	}
	return strings.EqualFold(name, exportIdent(col.FieldName))
}

// indexPlanColumn normalizes Unique into UniqueIndex so CreateIndex applies reliably.
func indexPlanColumn(col ColumnSpec) ColumnSpec {
	out := col
	if out.Unique && !out.UniqueIndex {
		out.UniqueIndex = true
	}
	return out
}

func checkOpsForColumn(table string, col ColumnSpec, dialect string, tableExists bool) []PlanOp {
	expr := strings.TrimSpace(col.CheckExpr)
	if expr == "" {
		return nil
	}
	// SQLite cannot ALTER TABLE ADD CHECK. New tables get CHECK from gorm tags on
	// CREATE TABLE; do not emit ensure_check for existing sqlite tables (would be a
	// no-op Auto or a permanent Guarded block on every subsequent migrate).
	if strings.EqualFold(strings.TrimSpace(dialect), "sqlite") && tableExists {
		return nil
	}
	columnName := col.Name
	if columnName == "" {
		columnName = strings.ToLower(col.FieldName)
	}
	name := fmt.Sprintf("chk_%s_%s", table, columnName)
	// ensureCheckConstraint is idempotent (drop+add) on postgres/mysql/sqlserver.
	// Keep Auto so populated tables do not permanently fail ValidatePlan / MigrateSchema.
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

func liveHasIndex(live LiveSchema, table, name string, wantUnique bool) bool {
	if live.Indexes == nil {
		return false
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, idx := range live.Indexes[table] {
		if strings.EqualFold(strings.TrimSpace(idx.Name), name) {
			if wantUnique && !idx.Unique {
				continue
			}
			return true
		}
		// Column fallback: only exact single-column indexes (composites are not equivalent).
		if len(idx.Columns) != 1 {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(idx.Columns[0]), name) {
			continue
		}
		if wantUnique && !idx.Unique {
			continue
		}
		return true
	}
	return false
}

func indexCoveredByDesiredKey(idx LiveIndex, desiredIndexKeys map[string]struct{}) bool {
	// Composites only match by index name (handled by the caller). Column-key
	// coverage applies only to exact single-column indexes so a desired pair of
	// single-column indexes does not swallow a stale (a,b) composite.
	if len(idx.Columns) != 1 {
		return false
	}
	_, ok := desiredIndexKeys[strings.ToLower(strings.TrimSpace(idx.Columns[0]))]
	return ok
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
		// Desired removed an explicit default while live still has one.
		if live.Default == nil || isAbsentLiveDefault(*live.Default) {
			return false
		}
		return true
	}
	want := strings.TrimSpace(*desired.Default)
	if want == "" {
		if live.Default == nil || isAbsentLiveDefault(*live.Default) {
			return false
		}
		return true
	}
	if live.Default == nil {
		// Unknown live default: do not fail closed (dialects often omit default metadata).
		return false
	}
	have := strings.TrimSpace(*live.Default)
	return !strings.EqualFold(normalizeDefaultLiteral(want), normalizeDefaultLiteral(have))
}

// isAbsentLiveDefault reports dialect/engine sentinel defaults that are not an
// application-authored ColumnSpec.Default (so MigrateSchema is not permanently blocked).
func isAbsentLiveDefault(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" || s == "null" {
		return true
	}
	if strings.HasPrefix(s, "nextval(") {
		return true
	}
	if strings.HasPrefix(s, "current_timestamp") || s == "current_date" || s == "current_time" {
		return true
	}
	if strings.HasPrefix(s, "getdate(") || strings.HasPrefix(s, "now(") || strings.HasPrefix(s, "localtimestamp") {
		return true
	}
	return false
}

func normalizeDefaultLiteral(v string) string {
	v = strings.TrimSpace(v)
	for {
		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
				v = v[1 : len(v)-1]
				continue
			}
			// Only unwrap when the whole value is paren-wrapped (not values like foo()).
			if v[0] == '(' && v[len(v)-1] == ')' {
				v = v[1 : len(v)-1]
				continue
			}
		}
		if next, ok := stripOuterPostgresCast(v); ok {
			v = next
			continue
		}
		return v
	}
}

// stripOuterPostgresCast removes a trailing ::type when the left-hand side is a
// quoted literal or parenthesized expression (so values like a::b stay intact).
func stripOuterPostgresCast(v string) (string, bool) {
	idx := postgresCastIndexOutsideQuotes(v)
	if idx < 0 {
		return v, false
	}
	left := strings.TrimSpace(v[:idx])
	if len(left) < 2 {
		return v, false
	}
	if (left[0] == '\'' && left[len(left)-1] == '\'') ||
		(left[0] == '"' && left[len(left)-1] == '"') ||
		(left[0] == '(' && left[len(left)-1] == ')') {
		return left, true
	}
	return v, false
}

// postgresCastIndexOutsideQuotes returns the index of "::" outside quoted literals, or -1.
func postgresCastIndexOutsideQuotes(v string) int {
	inSingle, inDouble := false, false
	for i := 0; i+1 < len(v); i++ {
		c := v[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case !inSingle && !inDouble && c == ':' && v[i+1] == ':':
			return i
		}
	}
	return -1
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
