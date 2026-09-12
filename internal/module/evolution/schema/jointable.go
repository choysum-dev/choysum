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
func appendJoinTablesFromModels(desired *DesiredSchema, models []*meta.Model) error {
	if desired == nil {
		return fmt.Errorf("desired schema is nil")
	}
	byKey := indexModelsByKey(models)
	seen := map[string]struct{}{}
	for _, jt := range desired.JoinTables {
		seen[strings.ToLower(strings.TrimSpace(jt.Table))] = struct{}{}
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
				continue
			}
			joinModel := resolveModelRef(byKey, joinRef)
			if joinModel == nil {
				return fmt.Errorf("ManyToMany %s.%s joinModel %q not found among migrate models",
					model.Name, field.Name, joinRef)
			}
			joinTable := strings.TrimSpace(joinModel.ModelTable)
			if joinTable == "" {
				return fmt.Errorf("ManyToMany %s.%s joinModel %q has empty ModelTable",
					model.Name, field.Name, joinRef)
			}
			cols := desired.Tables[joinTable]
			if len(cols) == 0 {
				return fmt.Errorf("ManyToMany %s.%s join table %q has no desired columns",
					model.Name, field.Name, joinTable)
			}
			key := strings.ToLower(joinTable)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			leftCol := strcase.ToSnake(joinField)
			rightCol := strcase.ToSnake(inverseJoinField)
			referRight := ""
			if target := resolveModelRef(byKey, targetRef); target != nil {
				referRight = strings.TrimSpace(target.ModelTable)
			}
			desired.JoinTables = append(desired.JoinTables, JoinTableSpec{
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
			})
		}
	}
	return nil
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
			out[strings.ToLower(name)] = model
		}
		if app != "" && name != "" {
			out[strings.ToLower(app+"."+name)] = model
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
	ref = strings.Trim(ref, `"'`+"`")
	ref = strings.TrimSpace(ref)
	// Keep only a dotted identifier (UserRole / auth.UserRole); drop call suffixes.
	if cut := strings.IndexAny(ref, "({[\n\r\t "); cut >= 0 {
		ref = strings.TrimSpace(ref[:cut])
	}
	return ref
}
