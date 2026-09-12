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
)

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
			if err := db.Table(op.Table).Migrator().AddColumn(inst, exportIdent(op.Column.FieldName)); err != nil {
				return fmt.Errorf("add column %s.%s: %w", op.Table, op.Column.Name, err)
			}
		default:
			// P0: never apply alter/drop here.
		}
	}
	return nil
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

func exportIdent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Col"
	}
	runes := []rune(name)
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		return "F" + name
	}
	runes[0] = unicode.ToUpper(runes[0])
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "Col"
	}
	return string(out)
}
