// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"fmt"
	"strings"

	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/ettle/strcase"
	"gorm.io/gorm"
)

// ResolveM2O resolves a Many2One CSV field path such as DefaultCurrencyId/Code.
func ResolveM2O(db *gorm.DB, unit recordplan.Unit, modelFull, fieldPath, raw string) (string, error) {
	parts := strings.Split(fieldPath, "/")
	if len(parts) != 2 {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, "invalid M2O field path")
	}
	fieldName := strings.TrimSpace(parts[0])
	lookupField := strings.TrimSpace(parts[1])
	if fieldName == "" || lookupField == "" {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, "invalid M2O field path")
	}
	if lookupField == "id" {
		lookupField = "Id"
	}

	targetModel, err := resolveRelationTarget(db, modelFull, fieldName)
	if err != nil {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, err.Error())
	}
	tableName := strings.TrimSpace(targetModel.ModelTable)
	if tableName == "" {
		return "", rowError(unit, fieldPath, importpkg.CodeModelNotFound, "related model has empty table")
	}

	column := strcase.ToSnake(lookupField)
	var resID string
	err = db.Table(tableName).Select("id").Where(column+" = ?", raw).Limit(1).Scan(&resID).Error
	if err != nil {
		return "", mapDBError(unit, fieldPath, err)
	}
	if resID == "" {
		return "", rowError(unit, fieldPath, importpkg.CodeExternalIDNotFound, fmt.Sprintf("related record not found for %s=%q", lookupField, raw))
	}
	return resID, nil
}

func resolveRelationTarget(db *gorm.DB, modelFull, fieldName string) (*meta.Model, error) {
	app, name, ok := strings.Cut(modelFull, ".")
	if !ok {
		return nil, fmt.Errorf("invalid model name %q", modelFull)
	}
	var owner meta.Model
	if err := db.Where("application = ? AND name = ?", app, name).First(&owner).Error; err != nil {
		return nil, fmt.Errorf("resolve model %s: %w", modelFull, err)
	}
	var field meta.Field
	if err := db.Where("model_id = ? AND name = ?", owner.Id.String, fieldName).First(&field).Error; err != nil {
		return nil, fmt.Errorf("resolve field %s on %s: %w", fieldName, modelFull, err)
	}
	if !strings.EqualFold(strings.TrimSpace(field.FieldType), "ManyToOne") {
		return nil, fmt.Errorf("field %s is not ManyToOne", fieldName)
	}
	switch fieldName {
	case "DefaultCurrencyId":
		target := &meta.Model{}
		if err := db.Where("application = ? AND name = ?", "base", "Currency").First(target).Error; err != nil {
			return nil, fmt.Errorf("resolve target model base.Currency: %w", err)
		}
		return target, nil
	default:
		return nil, fmt.Errorf("unsupported M2O field %s for V1 record import", fieldName)
	}
}
