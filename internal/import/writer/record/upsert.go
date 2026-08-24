// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"fmt"
	"strings"
	"time"

	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

const (
	countryModelFull  = "base.Country"
	countryTable      = "base_country"
	currencyModelFull = "base.Currency"
	currencyTable     = "base_currency"
)

// UpsertCountry writes one CSV row to base.Country.
func UpsertCountry(txScope scope.Scope, unit recordplan.Unit) error {
	txScope = WithImportFileContext(txScope)
	db := txScope.Session().DB
	if db == nil {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "database session is required")
	}

	model := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "base", "Country").First(model).Error; err != nil {
		return rowError(unit, "", importpkg.CodeModelNotFound, "base.Country model metadata not found")
	}

	columns, err := mapCountryValues(db, model, unit)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
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
	}

	now := time.Now().UTC()
	if hasExternalID {
		return upsertCountryByExternalID(db, model, unit, externalKey, columns, now)
	}
	return upsertCountryByCode(db, model, unit, columns, now)
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

func mapCountryValues(db *gorm.DB, model *meta.Model, unit recordplan.Unit) (map[string]any, error) {
	out := make(map[string]any, len(unit.Values))
	modelID := ""
	if model != nil {
		modelID = strings.TrimSpace(model.Id.String)
	}
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
			resolved, err := ResolveM2O(db, unit, countryModelFull, fieldPath, raw)
			if err != nil {
				return nil, err
			}
			baseField := strings.Split(fieldPath, "/")[0]
			out[strcase.ToSnake(baseField)] = resolved
			continue
		}
		if strings.Contains(fieldPath, ".") {
			return nil, rowError(unit, fieldPath, importpkg.CodeInvalidFormat, "O2M field paths are not supported in V1 record import")
		}
		value, err := parseScalarValue(unit, fieldPath, raw)
		if err != nil {
			return nil, err
		}
		if translate, err := fieldTranslateEnabled(db, modelID, fieldPath); err != nil {
			return nil, mapDBError(unit, fieldPath, err)
		} else if translate {
			if strVal, ok := value.(string); ok {
				encoded, encErr := encodeTranslatedScalar(strVal)
				if encErr != nil {
					return nil, rowError(unit, fieldPath, importpkg.CodeInvalidFormat, encErr.Error())
				}
				value = encoded
			}
		}
		out[strcase.ToSnake(fieldPath)] = value
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

func upsertCountryByExternalID(db *gorm.DB, model *meta.Model, unit recordplan.Unit, key MetaModelDataKey, columns map[string]any, now time.Time) error {
	mapping, err := lookupExternalID(db, key)
	if err != nil {
		return importpkg.ErrorfWrap(importpkg.CodeConstraint, "lookup external id mapping", err)
	}
	if mapping != nil {
		columns["updated_at"] = now
		if err := db.Table(countryTable).Where("id = ?", mapping.ResID).Updates(columns).Error; err != nil {
			return mapDBError(unit, "", err)
		}
		return upsertExternalIDMapping(db, key, model, mapping.ResID)
	}

	resID := xid.New().String()
	columns["id"] = resID
	columns["created_at"] = now
	columns["updated_at"] = now
	if err := db.Table(countryTable).Create(columns).Error; err != nil {
		return mapDBError(unit, "", err)
	}
	return upsertExternalIDMapping(db, key, model, resID)
}

func upsertCountryByCode(db *gorm.DB, model *meta.Model, unit recordplan.Unit, columns map[string]any, now time.Time) error {
	code, ok := columns["code"].(string)
	if !ok || strings.TrimSpace(code) == "" {
		return rowError(unit, "Code", importpkg.CodeEmptyRequired, "Code is required when id column is absent")
	}

	var existingID string
	err := db.Table(countryTable).Select("id").Where("code = ?", code).Limit(1).Scan(&existingID).Error
	if err != nil {
		return mapDBError(unit, "Code", err)
	}
	if existingID != "" {
		columns["updated_at"] = now
		delete(columns, "id")
		delete(columns, "created_at")
		if err := db.Table(countryTable).Where("id = ?", existingID).Updates(columns).Error; err != nil {
			return mapDBError(unit, "Code", err)
		}
		return nil
	}

	resID := xid.New().String()
	columns["id"] = resID
	columns["created_at"] = now
	columns["updated_at"] = now
	if err := db.Table(countryTable).Create(columns).Error; err != nil {
		return mapDBError(unit, "Code", err)
	}
	_ = model
	return nil
}

func mapDBError(unit recordplan.Unit, field string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	code := importpkg.CodeConstraint
	if strings.Contains(strings.ToLower(msg), "unique") || strings.Contains(strings.ToLower(msg), "duplicate") {
		code = importpkg.CodeDuplicateKey
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
