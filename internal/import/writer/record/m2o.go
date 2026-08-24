// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"
	"fmt"
	"strings"

	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"gorm.io/gorm"
)

// ResolveM2O resolves a Many2One CSV field path such as DefaultCurrencyId/Code via ORM Search.
// The comodel comes from meta.Field.RelationModel on unit.Model; lookup fields are validated
// against the target model's field metadata (Id is always allowed).
func ResolveM2O(ctx context.Context, db *gorm.DB, modelCaller importcaller.Caller, unit recordplan.Unit, fieldPath, raw string) (string, error) {
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

	srcModel, err := LookupModel(db, unit.Model)
	if err != nil {
		return "", rowError(unit, fieldPath, importpkg.CodeModelNotFound, fmt.Sprintf("model %s not found", unit.Model))
	}
	srcField, err := LookupField(db, srcModel, fieldName)
	if err != nil {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, fmt.Sprintf("unknown field %s", fieldName))
	}
	if !fieldIsManyToOne(srcField) {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, fmt.Sprintf("field %s is not ManyToOne", fieldName))
	}
	targetModel, err := fieldRelationTarget(srcField)
	if err != nil {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, err.Error())
	}

	if lookupField != "Id" {
		targetMeta, err := LookupModel(db, targetModel)
		if err != nil {
			return "", rowError(unit, fieldPath, importpkg.CodeModelNotFound, fmt.Sprintf("related model %s not found", targetModel))
		}
		if _, err := LookupField(db, targetMeta, lookupField); err != nil {
			return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat,
				fmt.Sprintf("lookup field %s not found on %s", lookupField, targetModel))
		}
	}

	result, err := modelCaller.Call(ctx, importcaller.CallRequest{
		Model:  targetModel,
		Method: "Search",
		Args: []any{
			map[string]any{"And": []any{[]any{lookupField, "=", raw}}},
			map[string]any{"fields": []string{"Id"}, "limit": 1},
		},
	})
	if err != nil {
		return "", mapORMError(unit, fieldPath, err)
	}
	resID := firstRecordID(result)
	if resID == "" {
		return "", rowError(unit, fieldPath, importpkg.CodeExternalIDNotFound, fmt.Sprintf("related record not found for %s=%q", lookupField, raw))
	}
	return resID, nil
}
