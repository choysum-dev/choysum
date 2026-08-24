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
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

const (
	countryModelFull  = "base.Country"
	currencyModelFull = "base.Currency"
)

// UpsertCountry writes one CSV row through TS ORM Create / UpdateById.
func UpsertCountry(ctx context.Context, txScope scope.Scope, unit recordplan.Unit) error {
	caller, ok := orm.CallerFromContext(ctx)
	if !ok {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "orm caller is required for record writer")
	}
	if txScope == nil || txScope.Session() == nil || txScope.Session().DB == nil {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "database session is required")
	}
	db := txScope.Session().DB

	model := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "base", "Country").First(model).Error; err != nil {
		return rowError(unit, "", importpkg.CodeModelNotFound, "base.Country model metadata not found")
	}

	vals, err := buildCountryVals(ctx, caller, unit)
	if err != nil {
		return err
	}
	if len(vals) == 0 {
		return rowError(unit, "", importpkg.CodeEmptyRequired, "row has no importable values")
	}

	externalKey, hasExternalID, err := parseUnitExternalID(unit)
	if err != nil {
		return err
	}
	if hasExternalID {
		if err := AssertExternalIDWritable(db, externalKey, unit.UnitIndex()); err != nil {
			return err
		}
		return upsertCountryByExternalID(ctx, caller, db, model, unit, externalKey, vals)
	}
	return upsertCountryByCode(ctx, caller, unit, vals)
}

func parseUnitExternalID(unit recordplan.Unit) (MetaModelDataKey, bool, error) {
	raw := strings.TrimSpace(unit.ExternalID)
	if raw == "" {
		return MetaModelDataKey{}, false, nil
	}
	key, err := ParseMetaModelDataKey(raw)
	if err != nil {
		return MetaModelDataKey{}, false, &importpkg.Error{
			Code:  importpkg.CodeInvalidFormat,
			Text:  err.Error(),
			Row:   unit.UnitIndex(),
			Field: "id",
		}
	}
	return key, true, nil
}

func buildCountryVals(ctx context.Context, caller orm.Caller, unit recordplan.Unit) (map[string]any, error) {
	out := make(map[string]any, len(unit.Values))
	for fieldPath, raw := range unit.Values {
		fieldPath = strings.TrimSpace(fieldPath)
		raw = strings.TrimSpace(raw)
		if fieldPath == "" {
			continue
		}
		if raw == "" {
			return nil, rowError(unit, fieldPath, importpkg.CodeEmptyRequired, "required value is empty")
		}
		if strings.Contains(fieldPath, "/") {
			resolved, err := ResolveM2O(ctx, caller, unit, fieldPath, raw)
			if err != nil {
				return nil, err
			}
			baseField := strings.Split(fieldPath, "/")[0]
			out[baseField] = resolved
			continue
		}
		if strings.Contains(fieldPath, ".") {
			return nil, rowError(unit, fieldPath, importpkg.CodeInvalidFormat, "O2M field paths are not supported in V1 record import")
		}
		value, err := parseScalarValue(unit, fieldPath, raw)
		if err != nil {
			return nil, err
		}
		out[fieldPath] = value
	}
	return out, nil
}

func parseScalarValue(unit recordplan.Unit, fieldPath, raw string) (any, error) {
	switch fieldPath {
	case "ZipRequired", "StateRequired", "IsActive":
		parsed, ok := parseBool(raw)
		if !ok {
			return nil, rowError(unit, fieldPath, importpkg.CodeInvalidFormat, fmt.Sprintf("invalid boolean value %q", raw))
		}
		return parsed, nil
	case "Code":
		return normalizeCountryCode(raw), nil
	default:
		return raw, nil
	}
}

func normalizeCountryCode(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "t", "yes", "y":
		return true, true
	case "0", "false", "f", "no", "n":
		return false, true
	default:
		return false, false
	}
}

func upsertCountryByExternalID(
	ctx context.Context,
	caller orm.Caller,
	db *gorm.DB,
	model *meta.Model,
	unit recordplan.Unit,
	key MetaModelDataKey,
	vals map[string]any,
) error {
	mapping, err := lookupExternalID(db, key)
	if err != nil {
		return importpkg.ErrorfWrap(importpkg.CodeConstraint, "lookup external id mapping", err)
	}
	if mapping != nil {
		exists, err := countryExistsByID(ctx, caller, unit, mapping.ResID)
		if err != nil {
			return err
		}
		if exists {
			if _, err := caller.Call(ctx, orm.CallRequest{
				Model:  countryModelFull,
				Method: "UpdateById",
				Args:   []any{mapping.ResID, vals, []string{"Id", "Code"}},
			}); err != nil {
				return mapORMError(unit, "", err)
			}
			return upsertExternalIDMapping(db, key, model, mapping.ResID)
		}
		// Mapping points at a deleted row — recreate and remap.
	}

	created, err := caller.Call(ctx, orm.CallRequest{
		Model:  countryModelFull,
		Method: "Create",
		Args:   []any{vals, []string{"Id", "Code"}},
	})
	if err != nil {
		return mapORMError(unit, "", err)
	}
	resID, ok := recordIDFromResult(created)
	if !ok || resID == "" {
		return rowError(unit, "", importpkg.CodeConstraint, "Create did not return Id")
	}
	return upsertExternalIDMapping(db, key, model, resID)
}

func countryExistsByID(ctx context.Context, caller orm.Caller, unit recordplan.Unit, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	result, err := caller.Call(ctx, orm.CallRequest{
		Model:  countryModelFull,
		Method: "Search",
		Args: []any{
			map[string]any{"And": []any{[]any{"Id", "=", id}}},
			map[string]any{"fields": []string{"Id"}, "limit": 1},
		},
	})
	if err != nil {
		return false, mapORMError(unit, "", err)
	}
	return firstRecordID(result) != "", nil
}

func upsertCountryByCode(ctx context.Context, caller orm.Caller, unit recordplan.Unit, vals map[string]any) error {
	code, _ := vals["Code"].(string)
	if strings.TrimSpace(code) == "" {
		return rowError(unit, "Code", importpkg.CodeEmptyRequired, "Code is required when id column is absent")
	}

	existingID, err := searchCountryIDByCode(ctx, caller, unit, code)
	if err != nil {
		return err
	}
	if existingID != "" {
		if _, err := caller.Call(ctx, orm.CallRequest{
			Model:  countryModelFull,
			Method: "UpdateById",
			Args:   []any{existingID, vals, []string{"Id", "Code"}},
		}); err != nil {
			return mapORMError(unit, "Code", err)
		}
		return nil
	}

	if _, err := caller.Call(ctx, orm.CallRequest{
		Model:  countryModelFull,
		Method: "Create",
		Args:   []any{vals, []string{"Id", "Code"}},
	}); err != nil {
		return mapORMError(unit, "Code", err)
	}
	return nil
}

func searchCountryIDByCode(ctx context.Context, caller orm.Caller, unit recordplan.Unit, code string) (string, error) {
	result, err := caller.Call(ctx, orm.CallRequest{
		Model:  countryModelFull,
		Method: "Search",
		Args: []any{
			map[string]any{"And": []any{[]any{"Code", "=", code}}},
			map[string]any{"fields": []string{"Id", "Code"}, "limit": 1},
		},
	})
	if err != nil {
		return "", mapORMError(unit, "Code", err)
	}
	return firstRecordID(result), nil
}

func firstRecordID(result any) string {
	switch rows := result.(type) {
	case []any:
		if len(rows) == 0 {
			return ""
		}
		id, _ := recordIDFromResult(rows[0])
		return id
	case []map[string]any:
		if len(rows) == 0 {
			return ""
		}
		id, _ := recordIDFromResult(rows[0])
		return id
	default:
		return ""
	}
}

func recordIDFromResult(result any) (string, bool) {
	switch row := result.(type) {
	case map[string]any:
		if id, ok := row["Id"].(string); ok && id != "" {
			return id, true
		}
		if id, ok := row["id"].(string); ok && id != "" {
			return id, true
		}
	}
	return "", false
}

func mapORMError(unit recordplan.Unit, field string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	code := importpkg.CodeConstraint
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "unique") || strings.Contains(lower, "duplicate") {
		code = importpkg.CodeDuplicateKey
	}
	if strings.Contains(lower, "required") || strings.Contains(lower, "not null") || strings.Contains(lower, "empty") {
		code = importpkg.CodeEmptyRequired
	}
	return &importpkg.Error{
		Code:  code,
		Text:  msg,
		Row:   unit.UnitIndex(),
		Field: field,
	}
}

func rowError(unit recordplan.Unit, field, code, text string) *importpkg.Error {
	return &importpkg.Error{
		Code:  code,
		Text:  text,
		Row:   unit.UnitIndex(),
		Field: strings.TrimSpace(field),
	}
}
