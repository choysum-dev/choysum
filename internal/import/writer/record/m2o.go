// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/orm"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// ResolveM2O resolves a Many2One CSV field path such as DefaultCurrencyId/Code via ORM Search.
func ResolveM2O(ctx context.Context, caller orm.Caller, unit recordplan.Unit, fieldPath, raw string) (string, error) {
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
	if !isAllowedM2OLookupField(lookupField) {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat,
			fmt.Sprintf("unsupported M2O lookup field %q (allowed: Id, Code)", lookupField))
	}

	targetModel, err := resolveM2OTargetModel(fieldName)
	if err != nil {
		return "", rowError(unit, fieldPath, importpkg.CodeInvalidFormat, err.Error())
	}

	result, err := caller.Call(ctx, orm.CallRequest{
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

func resolveM2OTargetModel(fieldName string) (string, error) {
	switch fieldName {
	case "DefaultCurrencyId":
		return currencyModelFull, nil
	default:
		return "", fmt.Errorf("unsupported M2O field %s for V1 record import", fieldName)
	}
}

func isAllowedM2OLookupField(lookupField string) bool {
	switch lookupField {
	case "Id", "Code":
		return true
	default:
		return false
	}
}
