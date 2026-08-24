// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// SplitModelFullName parses "app.Model" into application + name.
func SplitModelFullName(full string) (application, name string, err error) {
	full = strings.TrimSpace(full)
	dot := strings.Index(full, ".")
	if dot <= 0 || dot == len(full)-1 {
		return "", "", fmt.Errorf("model must be app.Model, got %q", full)
	}
	application = strings.TrimSpace(full[:dot])
	name = strings.TrimSpace(full[dot+1:])
	if application == "" || name == "" || strings.Contains(name, ".") {
		return "", "", fmt.Errorf("model must be app.Model, got %q", full)
	}
	return application, name, nil
}

// LookupModel loads meta.Model by full name "app.Model".
func LookupModel(db *gorm.DB, fullName string) (*meta.Model, error) {
	app, name, err := SplitModelFullName(fullName)
	if err != nil {
		return nil, err
	}
	model := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", app, name).First(model).Error; err != nil {
		return nil, err
	}
	return model, nil
}

// LookupField loads a field by PascalCase name on the given model.
func LookupField(db *gorm.DB, model *meta.Model, fieldName string) (*meta.Field, error) {
	fieldName = strings.TrimSpace(fieldName)
	if model == nil || fieldName == "" {
		return nil, gorm.ErrRecordNotFound
	}
	field := &meta.Field{}
	if err := db.Where("model_id = ? AND name = ?", model.Id.String, fieldName).First(field).Error; err != nil {
		return nil, err
	}
	return field, nil
}

// ListFields returns all fields for a model.
func ListFields(db *gorm.DB, model *meta.Model) ([]meta.Field, error) {
	if model == nil {
		return nil, fmt.Errorf("model is required")
	}
	var fields []meta.Field
	if err := db.Where("model_id = ?", model.Id.String).Find(&fields).Error; err != nil {
		return nil, err
	}
	return fields, nil
}

func fieldIsManyToOne(field *meta.Field) bool {
	if field == nil {
		return false
	}
	ft := strings.TrimSpace(field.FieldType)
	return ft == "ManyToOne" || ft == "ManyToOneRef" || strings.EqualFold(field.Relation, "ManyToOne")
}

func fieldRelationTarget(field *meta.Field) (string, error) {
	if field == nil {
		return "", fmt.Errorf("field is required")
	}
	target := strings.TrimSpace(field.RelationModel)
	if target == "" {
		if spec, err := field.GetResolvedSpec(); err == nil && spec != nil && spec.Structural.Relation != nil {
			if v, ok := spec.Structural.Relation["targetModel"].(string); ok {
				target = strings.TrimSpace(v)
			}
		}
	}
	if target == "" {
		return "", fmt.Errorf("ManyToOne field %s has no RelationModel", field.Name)
	}
	if !strings.Contains(target, ".") {
		return "", fmt.Errorf("ManyToOne target %q must be app.Model", target)
	}
	return target, nil
}

func fieldIsUnique(field *meta.Field) bool {
	if field == nil || field.Name == "Id" {
		return false
	}
	spec, err := field.GetResolvedSpec()
	if err != nil || spec == nil || spec.Structural.StorageHints == nil {
		return false
	}
	hints := spec.Structural.StorageHints
	if hints.Unique != nil && *hints.Unique {
		return true
	}
	if hints.UniqueIndexEnabled != nil && *hints.UniqueIndexEnabled {
		return true
	}
	if strings.TrimSpace(ptrString(hints.UniqueIndex)) != "" {
		return true
	}
	return false
}

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func fieldIsBoolean(field *meta.Field) bool {
	if field == nil {
		return false
	}
	ft := strings.ToLower(strings.TrimSpace(field.FieldType))
	return ft == "boolean" || ft == "bool"
}
