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

// UpsertRecord writes one CSV row for unit.Model through TS ORM Create / UpdateById.
func UpsertRecord(ctx context.Context, txScope scope.Scope, unit recordplan.Unit) error {
	caller, ok := orm.CallerFromContext(ctx)
	if !ok {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "orm caller is required for record writer")
	}
	if txScope == nil || txScope.Session() == nil || txScope.Session().DB == nil {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "database session is required")
	}
	db := txScope.Session().DB

	modelFull := strings.TrimSpace(unit.Model)
	if modelFull == "" {
		return rowError(unit, "", importpkg.CodeModelNotFound, "model is required")
	}
	model, err := LookupModel(db, modelFull)
	if err != nil {
		return rowError(unit, "", importpkg.CodeModelNotFound, fmt.Sprintf("model %s metadata not found", modelFull))
	}
	fields, err := ListFields(db, model)
	if err != nil {
		return importpkg.ErrorfWrap(importpkg.CodeConstraint, "list model fields", err)
	}
	fieldByName := make(map[string]*meta.Field, len(fields))
	for i := range fields {
		fieldByName[fields[i].Name] = &fields[i]
	}

	vals, err := buildRecordVals(ctx, db, caller, unit, fieldByName)
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
		return upsertByExternalID(ctx, caller, db, model, modelFull, unit, externalKey, vals)
	}
	return upsertByUniqueKeys(ctx, caller, unit, modelFull, fieldByName, vals)
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

func buildRecordVals(
	ctx context.Context,
	db *gorm.DB,
	caller orm.Caller,
	unit recordplan.Unit,
	fieldByName map[string]*meta.Field,
) (map[string]any, error) {
	out := make(map[string]any, len(unit.Values))
	for fieldPath, raw := range unit.Values {
		fieldPath = strings.TrimSpace(fieldPath)
		raw = strings.TrimSpace(raw)
		if fieldPath == "" {
			continue
		}
		if strings.Contains(fieldPath, ".") {
			return nil, rowError(unit, fieldPath, importpkg.CodeInvalidFormat, "O2M field paths are not supported in V1 record import")
		}
		if strings.Contains(fieldPath, "/") {
			if raw == "" {
				return nil, rowError(unit, fieldPath, importpkg.CodeEmptyRequired, "required value is empty")
			}
			resolved, err := ResolveM2O(ctx, db, caller, unit, fieldPath, raw)
			if err != nil {
				return nil, err
			}
			baseField := strings.Split(fieldPath, "/")[0]
			out[baseField] = resolved
			continue
		}
		field := fieldByName[fieldPath]
		if field == nil {
			return nil, rowError(unit, fieldPath, importpkg.CodeInvalidFormat, fmt.Sprintf("unknown field %s", fieldPath))
		}
		if raw == "" {
			if field.NotNull {
				return nil, rowError(unit, fieldPath, importpkg.CodeEmptyRequired, "required value is empty")
			}
			continue
		}
		value, err := parseScalarValue(unit, field, raw)
		if err != nil {
			return nil, err
		}
		out[fieldPath] = value
	}
	return out, nil
}

func parseScalarValue(unit recordplan.Unit, field *meta.Field, raw string) (any, error) {
	if fieldIsBoolean(field) {
		parsed, ok := parseBool(raw)
		if !ok {
			return nil, rowError(unit, field.Name, importpkg.CodeInvalidFormat, fmt.Sprintf("invalid boolean value %q", raw))
		}
		return parsed, nil
	}
	return raw, nil
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

func upsertByExternalID(
	ctx context.Context,
	caller orm.Caller,
	db *gorm.DB,
	model *meta.Model,
	modelFull string,
	unit recordplan.Unit,
	key MetaModelDataKey,
	vals map[string]any,
) error {
	mapping, err := lookupExternalID(db, key)
	if err != nil {
		return importpkg.ErrorfWrap(importpkg.CodeConstraint, "lookup external id mapping", err)
	}
	if mapping != nil {
		exists, err := recordExistsByID(ctx, caller, unit, modelFull, mapping.ResID)
		if err != nil {
			return err
		}
		if exists {
			if _, err := caller.Call(ctx, orm.CallRequest{
				Model:  modelFull,
				Method: "UpdateById",
				Args:   []any{mapping.ResID, vals, []string{"Id"}},
			}); err != nil {
				return mapORMError(unit, "", err)
			}
			return upsertExternalIDMapping(db, key, model, mapping.ResID)
		}
	}

	created, err := caller.Call(ctx, orm.CallRequest{
		Model:  modelFull,
		Method: "Create",
		Args:   []any{vals, []string{"Id"}},
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

func recordExistsByID(ctx context.Context, caller orm.Caller, unit recordplan.Unit, modelFull, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	result, err := caller.Call(ctx, orm.CallRequest{
		Model:  modelFull,
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

func upsertByUniqueKeys(
	ctx context.Context,
	caller orm.Caller,
	unit recordplan.Unit,
	modelFull string,
	fieldByName map[string]*meta.Field,
	vals map[string]any,
) error {
	domain := make([]any, 0, 4)
	for name, field := range fieldByName {
		if !fieldIsUnique(field) || fieldIsManyToOne(field) {
			continue
		}
		v, ok := vals[name]
		if !ok {
			continue
		}
		domain = append(domain, []any{name, "=", v})
	}
	if len(domain) == 0 {
		return rowError(unit, "", importpkg.CodeEmptyRequired, "id column or unique business key is required")
	}

	existingID, err := searchRecordID(ctx, caller, unit, modelFull, domain)
	if err != nil {
		return err
	}
	if existingID != "" {
		if _, err := caller.Call(ctx, orm.CallRequest{
			Model:  modelFull,
			Method: "UpdateById",
			Args:   []any{existingID, vals, []string{"Id"}},
		}); err != nil {
			return mapORMError(unit, "", err)
		}
		return nil
	}

	if _, err := caller.Call(ctx, orm.CallRequest{
		Model:  modelFull,
		Method: "Create",
		Args:   []any{vals, []string{"Id"}},
	}); err != nil {
		return mapORMError(unit, "", err)
	}
	return nil
}

func searchRecordID(ctx context.Context, caller orm.Caller, unit recordplan.Unit, modelFull string, domain []any) (string, error) {
	result, err := caller.Call(ctx, orm.CallRequest{
		Model:  modelFull,
		Method: "Search",
		Args: []any{
			map[string]any{"And": domain},
			map[string]any{"fields": []string{"Id"}, "limit": 1},
		},
	})
	if err != nil {
		return "", mapORMError(unit, "", err)
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
