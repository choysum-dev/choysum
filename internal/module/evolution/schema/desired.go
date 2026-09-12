// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/ettle/strcase"
)

// buildDesired builds DesiredSchema from models (fields must carry ResolvedSpec).
func buildDesired(models []*meta.Model) (DesiredSchema, error) {
	out := DesiredSchema{Tables: make(map[string][]ColumnSpec)}
	seenByTable := make(map[string]map[string]ColumnSpec) // table → column → first spec
	for _, model := range models {
		if model == nil {
			continue
		}
		if model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		table := strings.TrimSpace(model.ModelTable)
		if table == "" {
			continue
		}
		seen := seenByTable[table]
		if seen == nil {
			seen = make(map[string]ColumnSpec)
			seenByTable[table] = seen
		}
		for _, field := range model.Fields {
			col, err := columnSpecFromField(field, model)
			if err != nil {
				return DesiredSchema{}, err
			}
			if col == nil {
				continue
			}
			key := strings.ToLower(col.Name)
			if prev, ok := seen[key]; ok {
				if columnSpecsEquivalent(prev, *col) {
					continue
				}
				return DesiredSchema{}, fmt.Errorf(
					"conflicting column definitions for %s.%s (fields %q and %q)",
					table, col.Name, prev.FieldName, col.FieldName,
				)
			}
			seen[key] = *col
			out.Tables[table] = append(out.Tables[table], *col)
		}
	}
	return out, nil
}

func columnSpecsEquivalent(a, b ColumnSpec) bool {
	if a.Name != b.Name ||
		a.PhysicalType != b.PhysicalType ||
		a.NotNull != b.NotNull ||
		a.PrimaryKey != b.PrimaryKey ||
		a.Unique != b.Unique ||
		a.Indexed != b.Indexed ||
		a.IndexName != b.IndexName ||
		a.UniqueIndex != b.UniqueIndex ||
		a.Trigram != b.Trigram ||
		a.StorageKind != b.StorageKind ||
		a.CheckExpr != b.CheckExpr ||
		a.RenameFrom != b.RenameFrom ||
		a.DropAfter != b.DropAfter {
		return false
	}
	if !intPtrEqual(a.Size, b.Size) || !stringPtrEqual(a.Default, b.Default) {
		return false
	}
	if len(a.UniqueIndexNames) != len(b.UniqueIndexNames) {
		return false
	}
	for i := range a.UniqueIndexNames {
		if a.UniqueIndexNames[i] != b.UniqueIndexNames[i] {
			return false
		}
	}
	return true
}

func intPtrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func stringPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// columnSpecFromField returns nil when the field should not create a physical column.
func columnSpecFromField(field *meta.Field, model *meta.Model) (*ColumnSpec, error) {
	if field == nil {
		return nil, nil
	}
	resolved, err := field.GetResolvedSpec()
	if err != nil {
		return nil, fmt.Errorf("error unmarshal field resolved spec: %w", err)
	}
	if resolved == nil {
		return nil, nil
	}
	if !resolved.Migration.ShouldCreateColumn {
		return nil, nil
	}

	typeStr := strings.TrimSpace(resolved.Structural.FieldType)
	if typeStr == "" {
		return nil, nil
	}

	if (typeStr == "binary" || typeStr == "image") && model != nil && !isStorageBlobCarrierModel(model) {
		return nil, nil
	}

	physical := strings.TrimSpace(resolved.Migration.ResolvedColumnType)
	if physical == "" {
		physical = typeStr
	}
	storageKind := strings.TrimSpace(resolved.Migration.StorageKind)
	if storageKind == "" {
		storageKind = StoragePhysical
	}

	col := &ColumnSpec{
		Table:        strings.TrimSpace(model.ModelTable),
		Name:         strcase.ToSnake(field.Name),
		FieldName:    field.Name,
		LogicalType:  typeStr,
		PhysicalType: physical,
		StorageKind:  storageKind,
	}
	if rf := strings.TrimSpace(resolved.Structural.RenameFrom); rf != "" {
		// Authors pass the prior TS field name; physical column is snake_case.
		col.RenameFrom = strcase.ToSnake(rf)
	}
	if da := strings.TrimSpace(resolved.Structural.DropAfter); da != "" {
		col.DropAfter = da
	}

	if hints := resolved.Structural.StorageHints; hints != nil {
		if hints.Required != nil {
			col.NotNull = *hints.Required
		}
		if hints.Index != nil && strings.TrimSpace(*hints.Index) != "" {
			col.Indexed = true
			col.IndexName = strings.TrimSpace(*hints.Index)
		} else if hints.Indexed != nil {
			col.Indexed = *hints.Indexed
		}
		if hints.Size != nil {
			size := *hints.Size
			col.Size = &size
		}
		if hints.PrimaryKey != nil {
			col.PrimaryKey = *hints.PrimaryKey
			if col.PrimaryKey {
				// PK columns are NOT NULL in SQL; keep Desired aligned with live inspect.
				col.NotNull = true
			}
		}
		if hints.Unique != nil {
			col.Unique = *hints.Unique
		}
		if hints.UniqueIndexEnabled != nil {
			col.UniqueIndex = *hints.UniqueIndexEnabled
		}
		if hints.UniqueIndex != nil && strings.TrimSpace(*hints.UniqueIndex) != "" {
			col.UniqueIndex = true
			for _, part := range strings.Fields(strings.TrimSpace(*hints.UniqueIndex)) {
				if part != "" {
					col.UniqueIndexNames = append(col.UniqueIndexNames, part)
				}
			}
		}
		if hints.Default != nil {
			trimmed := strings.TrimSpace(*hints.Default)
			if trimmed != "" && !isJSFunctionDefaultLiteral(trimmed) {
				col.Default = &trimmed
			}
		}
	}

	// Trust ResolvedColumnType from parser; only apply schema-side physical overrides
	// that are already encoded in Migration for translate/companyDependent, plus
	// attachment blob on carrier models.
	if resolved.Structural.Translate != nil && *resolved.Structural.Translate {
		col.PhysicalType = "jsonobject"
		col.StorageKind = StorageJSONMap
		col.Size = nil
		col.Unique = false
		col.UniqueIndex = false
		col.UniqueIndexNames = nil
		col.Indexed = false
		col.IndexName = ""
		if hints := resolved.Structural.StorageHints; hints != nil && hints.Index != nil {
			if strings.EqualFold(strings.TrimSpace(*hints.Index), translatedTrigramIndexKind) {
				col.Trigram = true
			}
		}
	}

	if typeStr == "ManyToOne" || typeStr == "ManyToOneRef" {
		if col.PhysicalType == typeStr || col.PhysicalType == "" {
			col.PhysicalType = "char"
		}
		if col.Size == nil && col.PhysicalType == "char" {
			size := 20
			col.Size = &size
		}
		if !col.NotNull && field.NotNull {
			col.NotNull = true
		}
		if !col.Indexed && !col.Unique && !col.UniqueIndex {
			col.Indexed = true
		}
	}
	if typeStr == "ManyToManyRef" && (col.PhysicalType == typeStr || col.PhysicalType == "") {
		col.PhysicalType = "jsonobject"
	}
	if typeStr == "properties" && (col.PhysicalType == typeStr || col.PhysicalType == "") {
		col.PhysicalType = "jsonobject"
	}
	if typeStr == "selection" {
		if col.PhysicalType == typeStr || col.PhysicalType == "" {
			col.PhysicalType = "varchar"
		}
		if col.Size == nil && col.PhysicalType == "varchar" {
			size := 255
			col.Size = &size
		}
	}
	if typeStr == "binary" || typeStr == "image" {
		col.PhysicalType = "blob"
	}

	if resolved.Structural.CompanyDependent != nil && *resolved.Structural.CompanyDependent {
		col.PhysicalType = "jsonobject"
		col.StorageKind = StorageJSONMap
		col.Size = nil
		col.Unique = false
		col.UniqueIndex = false
		col.UniqueIndexNames = nil
		col.Indexed = false
		col.IndexName = ""
	}

	if cc := strings.TrimSpace(resolved.Structural.CheckConstraint); cc != "" {
		col.CheckExpr = cc
	}

	if getDefaultValue(col.PhysicalType) == nil {
		// Unsupported physical type: skip rather than produce invalid DDL.
		return nil, nil
	}

	return col, nil
}
