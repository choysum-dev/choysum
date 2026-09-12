// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/ettle/strcase"
)

// appendJoinTablesFromModels derives JoinTableSpec entries from ManyToMany fields.
// Join models must already contribute columns to desired.Tables (fail closed otherwise).
// Join models with AutoMigrate=false or Readonly are skipped (not an error).
func appendJoinTablesFromModels(desired *DesiredSchema, models []*meta.Model) error {
	if desired == nil {
		return fmt.Errorf("desired schema is nil")
	}
	byKey := indexModelsByKey(models)
	seen := map[string]JoinTableSpec{}
	for _, jt := range desired.JoinTables {
		key := strings.ToLower(strings.TrimSpace(jt.Table))
		if key != "" {
			seen[key] = jt
		}
	}
	for _, model := range models {
		if model == nil || model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		parentTable := strings.TrimSpace(model.ModelTable)
		for _, field := range model.Fields {
			if field == nil {
				continue
			}
			joinRef, joinField, inverseJoinField, targetRef, ok := manyToManyJoinMeta(field)
			if !ok {
				if fieldIsExplicitManyToMany(field) {
					return fmt.Errorf("ManyToMany %s.%s is missing joinModel",
						model.Name, field.Name)
				}
				continue
			}
			joinModel := resolveModelRef(byKey, joinRef)
			if joinModel == nil {
				return fmt.Errorf("ManyToMany %s.%s joinModel %q not found (or ambiguous) among migrate models; qualify as application.ModelName",
					model.Name, field.Name, joinRef)
			}
			if joinModel.Readonly || (joinModel.AutoMigrate != nil && !*joinModel.AutoMigrate) {
				continue
			}
			joinTable := strings.TrimSpace(joinModel.ModelTable)
			if joinTable == "" {
				return fmt.Errorf("ManyToMany %s.%s joinModel %q has empty ModelTable",
					model.Name, field.Name, joinRef)
			}
			if parentTable == "" {
				return fmt.Errorf("ManyToMany %s.%s parent model has empty ModelTable",
					model.Name, field.Name)
			}
			joinField = strings.TrimSpace(joinField)
			inverseJoinField = strings.TrimSpace(inverseJoinField)
			if joinField == "" || inverseJoinField == "" {
				return fmt.Errorf("ManyToMany %s.%s requires joinField and inverseJoinField",
					model.Name, field.Name)
			}
			cols := desired.Tables[joinTable]
			if len(cols) == 0 {
				return fmt.Errorf("ManyToMany %s.%s join table %q has no desired columns",
					model.Name, field.Name, joinTable)
			}
			leftCol, okLeft := resolveJoinColumnName(cols, joinField)
			rightCol, okRight := resolveJoinColumnName(cols, inverseJoinField)
			if !okLeft {
				return fmt.Errorf("ManyToMany %s.%s join column %q missing from desired table %s",
					model.Name, field.Name, joinField, joinTable)
			}
			if !okRight {
				return fmt.Errorf("ManyToMany %s.%s join column %q missing from desired table %s",
					model.Name, field.Name, inverseJoinField, joinTable)
			}
			referRight := ""
			if targetRef != "" {
				target := resolveModelRef(byKey, targetRef)
				if target == nil {
					return fmt.Errorf("ManyToMany %s.%s targetModel %q not found among migrate models",
						model.Name, field.Name, targetRef)
				}
				referRight = strings.TrimSpace(target.ModelTable)
				if referRight == "" {
					return fmt.Errorf("ManyToMany %s.%s targetModel %q has empty ModelTable",
						model.Name, field.Name, targetRef)
				}
			}
			candidate := JoinTableSpec{
				Table: joinTable,
				Left: JoinEnd{
					Column:      leftCol,
					ReferTable:  parentTable,
					ReferColumn: "id",
				},
				Right: JoinEnd{
					Column:      rightCol,
					ReferTable:  referRight,
					ReferColumn: "id",
				},
			}
			key := strings.ToLower(joinTable)
			if prev, ok := seen[key]; ok {
				if !joinTableSpecsEqual(prev, candidate) {
					return fmt.Errorf("ManyToMany %s.%s conflicts with existing JoinTableSpec for %s",
						model.Name, field.Name, joinTable)
				}
				continue
			}
			seen[key] = candidate
			desired.JoinTables = append(desired.JoinTables, candidate)
		}
	}
	return nil
}

func joinTableSpecsEqual(a, b JoinTableSpec) bool {
	if !strings.EqualFold(a.Table, b.Table) {
		return false
	}
	// Bidirectional ManyToMany (User.Roles ↔ Role.Users) swaps Left/Right.
	return (joinEndsEqual(a.Left, b.Left) && joinEndsEqual(a.Right, b.Right)) ||
		(joinEndsEqual(a.Left, b.Right) && joinEndsEqual(a.Right, b.Left))
}

func joinEndsEqual(a, b JoinEnd) bool {
	return strings.EqualFold(a.Column, b.Column) &&
		strings.EqualFold(a.ReferTable, b.ReferTable) &&
		strings.EqualFold(a.ReferColumn, b.ReferColumn)
}

// resolveJoinColumnName maps a joinField / inverseJoinField ref onto a desired physical column.
// Prefers snake_case of the field name, then exact/case-insensitive Name or FieldName match.
func resolveJoinColumnName(cols []ColumnSpec, fieldRef string) (string, bool) {
	fieldRef = strings.TrimSpace(fieldRef)
	if fieldRef == "" {
		return "", false
	}
	snake := strcase.ToSnake(fieldRef)
	for _, col := range cols {
		name := strings.TrimSpace(col.Name)
		if name == "" {
			continue
		}
		if strings.EqualFold(name, snake) {
			return name, true
		}
	}
	for _, col := range cols {
		name := strings.TrimSpace(col.Name)
		if name == "" {
			continue
		}
		if strings.EqualFold(name, fieldRef) {
			return name, true
		}
		if fn := strings.TrimSpace(col.FieldName); fn != "" && strings.EqualFold(fn, fieldRef) {
			return name, true
		}
	}
	return "", false
}

// fieldIsExplicitManyToMany reports FieldType ManyToMany (not ManyToManyRef).
// Used to fail closed when joinModel is missing on a real M2M relation field.
func fieldIsExplicitManyToMany(field *meta.Field) bool {
	if field == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(field.Relation), "ManyToMany") {
		return true
	}
	spec, err := field.GetResolvedSpec()
	if err != nil || spec == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(spec.Structural.FieldType), "ManyToMany")
}

// manyToManyJoinMeta returns join identity for a ManyToMany field, or ok=false when not M2M / no joinModel.
func manyToManyJoinMeta(field *meta.Field) (joinRef, joinField, inverseJoinField, targetRef string, ok bool) {
	joinRef = strings.TrimSpace(field.RelationJoinModel)
	joinField = strings.TrimSpace(field.RelationJoinField)
	inverseJoinField = strings.TrimSpace(field.RelationInverseJoinField)
	targetRef = strings.TrimSpace(field.RelationModel)
	isM2M := strings.EqualFold(strings.TrimSpace(field.Relation), "ManyToMany")
	if spec, err := field.GetResolvedSpec(); err == nil && spec != nil {
		if strings.EqualFold(strings.TrimSpace(spec.Structural.FieldType), "ManyToMany") {
			isM2M = true
		}
		if rel := spec.Structural.Relation; rel != nil {
			if joinRef == "" {
				if v, found := rel["joinModel"]; found {
					joinRef = strings.TrimSpace(fmt.Sprintf("%v", v))
				}
			}
			if joinField == "" {
				if v, found := rel["joinField"]; found {
					joinField = strings.TrimSpace(fmt.Sprintf("%v", v))
				}
			}
			if inverseJoinField == "" {
				if v, found := rel["inverseJoinField"]; found {
					inverseJoinField = strings.TrimSpace(fmt.Sprintf("%v", v))
				}
			}
			if targetRef == "" {
				if v, found := rel["targetModel"]; found {
					targetRef = strings.TrimSpace(fmt.Sprintf("%v", v))
				}
			}
		}
	}
	if !isM2M || joinRef == "" {
		return "", "", "", "", false
	}
	return joinRef, joinField, inverseJoinField, targetRef, true
}

func indexModelsByKey(models []*meta.Model) map[string]*meta.Model {
	out := map[string]*meta.Model{}
	for _, model := range models {
		if model == nil {
			continue
		}
		name := strings.TrimSpace(model.Name)
		app := strings.TrimSpace(model.Application)
		if name != "" {
			key := strings.ToLower(name)
			// Same unqualified name in two apps is ambiguous; resolveModelRef must not guess.
			if existing, ok := out[key]; ok && existing != model {
				out[key] = nil
			} else {
				out[key] = model
			}
		}
		if app != "" && name != "" {
			key := strings.ToLower(app + "." + name)
			// Duplicate application+name is ambiguous; resolveModelRef must not guess.
			if existing, ok := out[key]; ok && existing != model {
				out[key] = nil
			} else {
				out[key] = model
			}
		}
	}
	return out
}

func resolveModelRef(byKey map[string]*meta.Model, ref string) *meta.Model {
	ref = normalizeModelRefLiteral(ref)
	if ref == "" {
		return nil
	}
	return byKey[strings.ToLower(ref)]
}

// normalizeModelRefLiteral turns decorator model refs into lookup keys.
// Authors write joinModel/targetModel as () => UserRole; metadata stores that literal.
func normalizeModelRefLiteral(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if i := strings.Index(ref, "=>"); i >= 0 {
		ref = strings.TrimSpace(ref[i+2:])
	}
	ref = strings.TrimSpace(strings.TrimSuffix(ref, ";"))
	// Quotes and optional wrapping parens: () => (UserRole) / () => "UserRole".
	ref = strings.Trim(ref, `"'`+"`()")
	ref = strings.TrimSpace(ref)
	// Keep only a dotted identifier (UserRole / auth.UserRole); drop call suffixes.
	if cut := strings.IndexAny(ref, "({[\n\r\t "); cut >= 0 {
		ref = strings.TrimSpace(ref[:cut])
	}
	return ref
}
