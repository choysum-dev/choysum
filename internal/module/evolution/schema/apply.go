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
		case OpCreateTable:
			inst, err := structForCreateTable(op.Table, op.Columns, dialect)
			if err != nil {
				return fmt.Errorf("build create_table struct %s: %w", op.Table, err)
			}
			if err := db.Table(op.Table).Migrator().CreateTable(inst); err != nil {
				return fmt.Errorf("create table %s: %w", op.Table, err)
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
			// AddColumn does not create indexes from gorm tags; create them explicitly.
			if err := ensureIndexesForColumnFn(db.DB, op.Table, *op.Column, dialect); err != nil {
				return err
			}
		default:
			// P0: never apply alter/drop here.
		}
	}
	return nil
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
	if !col.Indexed && !col.UniqueIndex {
		return nil
	}
	inst, err := structForAddColumn(table, col, dialect)
	if err != nil {
		return fmt.Errorf("build index struct %s.%s: %w", table, col.Name, err)
	}
	mig := db.Table(table).Migrator()
	for _, name := range indexLookupNames(col) {
		if mig.HasIndex(inst, name) {
			continue
		}
		if err := mig.CreateIndex(inst, name); err != nil {
			return fmt.Errorf("create index %s on %s.%s: %w", name, table, col.Name, err)
		}
	}
	return nil
}

// indexLookupNames returns names suitable for Migrator.HasIndex/CreateIndex LookIndex.
func indexLookupNames(col ColumnSpec) []string {
	seen := map[string]struct{}{}
	var names []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if col.Indexed {
		if col.IndexName != "" && !strings.EqualFold(col.IndexName, translatedTrigramIndexKind) {
			add(col.IndexName)
		} else {
			add(exportIdent(col.FieldName))
		}
	}
	if col.UniqueIndex {
		if len(col.UniqueIndexNames) > 0 {
			for _, name := range col.UniqueIndexNames {
				add(name)
			}
		} else {
			add(exportIdent(col.FieldName))
		}
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
	// P0: do not emit check tags; CHECK is applied via ensureCheckConstraint after create/add.
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
