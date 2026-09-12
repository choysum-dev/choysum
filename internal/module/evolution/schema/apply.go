// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	"gorm.io/gorm"
)

// Overridable for tests that need to force post-AddColumn index failures.
var ensureIndexesForColumnFn = ensureIndexesForColumn

// Overridable so tests can force DropIndex failures without a broken migrator.
var dropIndexFn = func(mig gorm.Migrator, value any, name string) error {
	return mig.DropIndex(value, name)
}

// columnNeedsIndex reports whether ensureIndexesForColumn should run for col.
func columnNeedsIndex(col ColumnSpec) bool {
	return col.Indexed || col.Unique || col.UniqueIndex || len(col.UniqueIndexNames) > 0
}

// applyPlan executes Auto ops only (caller must Validate first).
func applyPlan(runtimeScope scope.Scope, dialect string, plan SchemaPlan) error {
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return fmt.Errorf("runtime scope is nil")
	}
	db := runtimeScope.Session()
	for _, op := range plan.Ops {
		if op.Safety != SafetyAuto {
			continue
		}
		switch op.Kind {
		case OpCreateTable, OpCreateJoinTable:
			inst, err := structForCreateTable(op.Table, op.Columns, dialect)
			if err != nil {
				return fmt.Errorf("build %s struct %s: %w", op.Kind, op.Table, err)
			}
			if err := db.Table(op.Table).Migrator().CreateTable(inst); err != nil {
				return fmt.Errorf("%s %s: %w", op.Kind, op.Table, err)
			}
			if op.Kind == OpCreateJoinTable {
				for _, col := range op.Columns {
					if !columnNeedsIndex(col) {
						continue
					}
					if err := ensureIndexesForColumnFn(db.DB, op.Table, col, dialect); err != nil {
						return fmt.Errorf("ensure join table index %s.%s: %w", op.Table, col.Name, err)
					}
				}
			}
		case OpAddColumn:
			if op.Column == nil {
				return fmt.Errorf("add_column missing column for table %s", op.Table)
			}
			inst, err := structForAddColumn(op.Table, *op.Column, dialect)
			if err != nil {
				return fmt.Errorf("build add_column struct %s.%s: %w", op.Table, op.Column.Name, err)
			}
			fieldName := exportIdent(op.Column.FieldName)
			if err := db.Table(op.Table).Migrator().AddColumn(inst, fieldName); err != nil {
				return fmt.Errorf("add column %s.%s: %w", op.Table, op.Column.Name, err)
			}
			// Indexes are applied via separate OpAddIndex ops (with populated-table safety).
		case OpAlterColumn:
			if op.Column == nil {
				return fmt.Errorf("alter_column missing column for table %s", op.Table)
			}
			if !strings.Contains(strings.ToLower(op.Detail), "widen size") {
				return fmt.Errorf("alter_column %s.%s is not an auto widen (%s)", op.Table, op.Column.Name, op.Detail)
			}
			if err := applyAlterColumnWiden(db.DB, op.Table, *op.Column, dialect); err != nil {
				return fmt.Errorf("alter column %s.%s: %w", op.Table, op.Column.Name, err)
			}
		case OpAddIndex:
			if op.Column == nil {
				return fmt.Errorf("add_index missing column for table %s", op.Table)
			}
			if err := ensureIndexesForColumnFn(db.DB, op.Table, *op.Column, dialect); err != nil {
				return fmt.Errorf("add index on %s.%s: %w", op.Table, op.Column.Name, err)
			}
		case OpEnsureCheck:
			expr := strings.TrimSpace(op.CheckExpr)
			name := strings.TrimSpace(op.CheckName)
			if expr == "" || name == "" {
				return fmt.Errorf("ensure_check missing name/expr for table %s", op.Table)
			}
			// Only rewrite the chk_ constraint prefix; table/column names may contain "chk_".
			if strings.HasPrefix(name, "chk_") {
				legacy := "ck_" + strings.TrimPrefix(name, "chk_")
				_ = dropCheckConstraintBestEffort(db.DB, dialect, op.Table, legacy)
			}
			if err := ensureCheckConstraint(db.DB, dialect, op.Table, name, expr); err != nil {
				return fmt.Errorf("ensure check %s on %s: %w", name, op.Table, err)
			}
		case OpRenameColumn:
			if op.Column == nil {
				return fmt.Errorf("rename_column missing column for table %s", op.Table)
			}
			fromName := strings.TrimSpace(op.FromName)
			if fromName == "" {
				return fmt.Errorf("rename_column missing from name for table %s", op.Table)
			}
			if err := renameColumn(db.DB, op.Table, fromName, *op.Column, dialect); err != nil {
				return fmt.Errorf("rename column %s.%s → %s: %w", op.Table, fromName, op.Column.Name, err)
			}
		default:
			// Never apply drop/manual here.
		}
	}
	return nil
}

// applyAlterColumnWiden applies Auto varchar/char size increases via GORM AlterColumn.
// Only type/size are applied so FullDataTypeOf cannot emit unvalidated NOT NULL/DEFAULT,
// except on dialects where MODIFY COLUMN rewrites the whole definition.
func applyAlterColumnWiden(db *gorm.DB, table string, col ColumnSpec, dialect string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	sizeOnly := widenColumnSpec(col, dialect)
	inst, err := structForAddColumn(table, sizeOnly, dialect)
	if err != nil {
		return err
	}
	fieldName := exportIdent(col.FieldName)
	if err := db.Table(table).Migrator().AlterColumn(inst, fieldName); err != nil {
		return err
	}
	return nil
}

func renameColumn(db *gorm.DB, table, fromCol string, to ColumnSpec, dialect string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	fromCol = strings.TrimSpace(fromCol)
	if fromCol == "" || strings.TrimSpace(to.Name) == "" {
		return fmt.Errorf("rename requires from and to column names")
	}
	if strings.TrimSpace(to.FieldName) == "" {
		to.FieldName = to.Name
	}
	inst, err := structForAddColumn(table, to, dialect)
	if err != nil {
		return fmt.Errorf("build rename struct %s.%s: %w", table, to.Name, err)
	}
	toField := exportIdent(to.FieldName)
	if err := db.Table(table).Migrator().RenameColumn(inst, fromCol, toField); err != nil {
		return err
	}
	return nil
}

// widenColumnSpec builds the AlterColumn payload. MySQL/SQL Server MODIFY COLUMN
// rewrites the full definition, so nullability/default must be preserved there.
func widenColumnSpec(col ColumnSpec, dialect string) ColumnSpec {
	sizeOnly := ColumnSpec{
		Name:         col.Name,
		FieldName:    col.FieldName,
		PhysicalType: col.PhysicalType,
		Size:         col.Size,
	}
	if dialect == "mysql" || dialect == "sqlserver" {
		sizeOnly.NotNull = col.NotNull
		sizeOnly.Default = col.Default
	}
	return sizeOnly
}

// ensureIndexesForDesired creates missing ordinary/unique indexes for desired columns.
// CreateTable already creates indexes from tags; this covers AddColumn and existing columns.
func ensureIndexesForDesired(db *gorm.DB, desired DesiredSchema, dialect string) error {
	if db == nil {
		return nil
	}
	for table, cols := range desired.Tables {
		for _, col := range cols {
			if err := ensureIndexesForColumn(db, table, col, dialect); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureIndexesForColumn(db *gorm.DB, table string, col ColumnSpec, dialect string) error {
	if db == nil {
		return nil
	}
	if col.Trigram || strings.EqualFold(col.IndexName, translatedTrigramIndexKind) {
		return nil
	}
	if !col.Indexed && !col.UniqueIndex && !col.Unique {
		return nil
	}
	col = indexPlanColumn(col)
	inst, err := structForAddColumn(table, col, dialect)
	if err != nil {
		return fmt.Errorf("build index struct %s.%s: %w", table, col.Name, err)
	}
	mig := db.Table(table).Migrator()
	for _, cand := range indexLookupCandidates(col) {
		if mig.HasIndex(inst, cand.Name) {
			if !cand.Unique {
				continue
			}
			// cand.Name may be a field export (Code) while the live index uses idx_table_col.
			unique, err := liveIndexUnique(db, table, cand.Name, col.Name)
			if err != nil {
				return err
			}
			if unique {
				continue
			}
			// Same lookup exists but is non-unique; replace so the unique requirement is enforced.
			if err := dropIndexFn(mig, inst, cand.Name); err != nil {
				return fmt.Errorf("drop non-unique index %s on %s.%s: %w", cand.Name, table, col.Name, err)
			}
		}
		if err := mig.CreateIndex(inst, cand.Name); err != nil {
			return fmt.Errorf("create index %s on %s.%s: %w", cand.Name, table, col.Name, err)
		}
	}
	return nil
}

// liveIndexUnique reports whether a live index matching indexName (or single-column colName) is unique.
func liveIndexUnique(db *gorm.DB, table, indexName, colName string) (bool, error) {
	indexes, err := getIndexes(db, table)
	if err != nil {
		return false, fmt.Errorf("inspect indexes for %s: %w", table, err)
	}
	for _, idx := range indexes {
		if idx == nil || !liveIndexMatches(idx, indexName, colName) {
			continue
		}
		u, ok := idx.Unique()
		return ok && u, nil
	}
	return false, nil
}

// liveIndexMatches is true when idx's name equals indexName, or it is a single-column
// index on colName / indexName (field-export lookups often differ from physical index names).
func liveIndexMatches(idx gorm.Index, indexName, colName string) bool {
	indexName = strings.TrimSpace(indexName)
	colName = strings.TrimSpace(colName)
	if indexName != "" && strings.EqualFold(strings.TrimSpace(idx.Name()), indexName) {
		return true
	}
	cols := idx.Columns()
	if len(cols) != 1 {
		return false
	}
	c := strings.TrimSpace(cols[0])
	if c == "" {
		return false
	}
	if colName != "" && strings.EqualFold(c, colName) {
		return true
	}
	return indexName != "" && strings.EqualFold(c, indexName)
}

type indexNameCandidate struct {
	Name   string
	Unique bool
}

// indexLookupCandidates returns Migrator.HasIndex/CreateIndex names with per-name uniqueness.
func indexLookupCandidates(col ColumnSpec) []indexNameCandidate {
	type entry struct {
		name   string
		unique bool
	}
	order := make([]string, 0, 4)
	byName := map[string]*entry{}
	add := func(name string, unique bool) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if prev, ok := byName[name]; ok {
			if unique {
				prev.unique = true
			}
			return
		}
		byName[name] = &entry{name: name, unique: unique}
		order = append(order, name)
	}
	if col.Indexed {
		if col.IndexName != "" && !strings.EqualFold(col.IndexName, translatedTrigramIndexKind) {
			add(col.IndexName, false)
		} else {
			add(exportIdent(col.FieldName), false)
		}
	}
	if col.UniqueIndex || col.Unique {
		if len(col.UniqueIndexNames) > 0 {
			for _, name := range col.UniqueIndexNames {
				add(name, true)
			}
		} else {
			add(exportIdent(col.FieldName), true)
		}
	}
	out := make([]indexNameCandidate, 0, len(order))
	for _, name := range order {
		e := byName[name]
		out = append(out, indexNameCandidate{Name: e.name, Unique: e.unique})
	}
	return out
}

// indexLookupNames returns names suitable for Migrator.HasIndex/CreateIndex LookIndex.
func indexLookupNames(col ColumnSpec) []string {
	cands := indexLookupCandidates(col)
	names := make([]string, 0, len(cands))
	for _, cand := range cands {
		names = append(names, cand.Name)
	}
	return names
}

func structForCreateTable(table string, cols []ColumnSpec, dialect string) (any, error) {
	fields := make([]reflect.StructField, 0, len(cols))
	seen := map[string]struct{}{}
	for _, col := range cols {
		sf, err := structFieldForColumn(col, dialect)
		if err != nil {
			return nil, err
		}
		name := sf.Name
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("duplicate exported field %s on table %s", name, table)
		}
		seen[name] = struct{}{}
		fields = append(fields, sf)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("create_table %s has no columns", table)
	}
	typ := reflect.StructOf(fields)
	return reflect.New(typ).Interface(), nil
}

func structForAddColumn(table string, col ColumnSpec, dialect string) (any, error) {
	sf, err := structFieldForColumn(col, dialect)
	if err != nil {
		return nil, err
	}
	typ := reflect.StructOf([]reflect.StructField{sf})
	_ = table
	return reflect.New(typ).Interface(), nil
}

func structFieldForColumn(col ColumnSpec, dialect string) (reflect.StructField, error) {
	dv := getDefaultValue(col.PhysicalType)
	if dv == nil {
		return reflect.StructField{}, fmt.Errorf("unsupported physical type %q for column %s", col.PhysicalType, col.Name)
	}
	gormParts := make([]string, 0, 8)
	gormParts = append(gormParts, "column:"+col.Name)
	typeTag := buildColumnTypeTagFromSpec(dialect, col)
	if typeTag != "" {
		gormParts = append(gormParts, typeTag)
	}
	addStandardTagsFromSpec(&gormParts, col)
	jsonName := col.Name
	if jsonName == "" {
		jsonName = strcase.ToSnake(col.FieldName)
	}
	tag := fmt.Sprintf(`gorm:"%s" json:"%s"`, strings.Join(gormParts, ";"), jsonName)
	return reflect.StructField{
		Name: exportIdent(col.FieldName),
		Type: reflect.TypeOf(dv),
		Tag:  reflect.StructTag(tag),
	}, nil
}

func buildColumnTypeTagFromSpec(dialect string, col ColumnSpec) string {
	metaMap := map[string]interface{}{}
	if col.Size != nil {
		metaMap["size"] = *col.Size
	}
	return buildColumnTypeTag(dialect, col.PhysicalType, metaMap)
}

func addStandardTagsFromSpec(tags *[]string, col ColumnSpec) {
	if col.PrimaryKey {
		*tags = append(*tags, "primaryKey")
	}
	if col.NotNull {
		*tags = append(*tags, "not null")
	}
	if col.Unique {
		*tags = append(*tags, "unique")
	}
	if col.Indexed {
		if col.IndexName != "" && !strings.EqualFold(col.IndexName, translatedTrigramIndexKind) {
			*tags = append(*tags, fmt.Sprintf("index:%s", col.IndexName))
		} else if !col.Trigram {
			*tags = append(*tags, "index")
		}
	}
	if col.UniqueIndex {
		if len(col.UniqueIndexNames) > 0 {
			for _, name := range col.UniqueIndexNames {
				*tags = append(*tags, fmt.Sprintf("uniqueIndex:%s", name))
			}
		} else {
			*tags = append(*tags, "uniqueIndex")
		}
	}
	if col.Default != nil {
		trimmed := strings.TrimSpace(*col.Default)
		if trimmed != "" && !isJSFunctionDefaultLiteral(trimmed) {
			*tags = append(*tags, fmt.Sprintf("default:%s", normalizeDefaultStringLiteral(trimmed)))
		}
	}
	if expr := strings.TrimSpace(col.CheckExpr); expr != "" {
		normalized := normalizeCheckExpr(expr)
		if normalized != "" {
			// Force default naming chk_<table>_<column> (same as OpEnsureCheck).
			*tags = append(*tags, "check:,"+normalized)
		}
	}
	// OpEnsureCheck applies CHECK via ALTER on postgres/mysql/sqlserver. SQLite cannot
	// ALTER TABLE ADD CONSTRAINT; new tables get CHECK from the gorm tag above.
}

// exportIdent returns a valid exported Go identifier for reflect.StructOf.
func exportIdent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Col"
	}
	sanitized := make([]rune, 0, len(name))
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			sanitized = append(sanitized, r)
		}
	}
	if len(sanitized) == 0 {
		return "Col"
	}
	// Must start with an uppercase letter (exported); prefix when first rune is not a letter.
	if !unicode.IsLetter(sanitized[0]) {
		sanitized = append([]rune{'F'}, sanitized...)
	} else {
		sanitized[0] = unicode.ToUpper(sanitized[0])
	}
	return string(sanitized)
}
