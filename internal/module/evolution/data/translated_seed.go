// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"errors"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

const translatedBaseLang = "en_US"

const LoadErrorCodeTranslatedLangUnknown = "translated_lang_unknown"
const LoadErrorCodeTranslatedSeedInvalid = "translated_seed_invalid"

func (l *Loader) lookupIrField(tx *gorm.DB, model *meta.IrModel, fieldName string) (*meta.IrField, error) {
	if model == nil || strings.TrimSpace(model.Id.String) == "" {
		return nil, nil
	}
	name := strings.TrimSpace(fieldName)
	if name == "" {
		return nil, nil
	}
	field := &meta.IrField{}
	err := tx.Where("model_id = ? AND name = ?", model.Id.String, name).First(field).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return field, nil
}

func isTranslateField(field *meta.IrField) bool {
	if field == nil {
		return false
	}
	spec, err := field.GetResolvedSpec()
	if err != nil || spec == nil {
		return false
	}
	return spec.Structural.Translate != nil && *spec.Structural.Translate
}

func seedSelfLanguageCode(values map[string]any) string {
	if values == nil {
		return ""
	}
	raw, ok := values["Code"]
	if !ok {
		return ""
	}
	if s, ok := raw.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func (l *Loader) languageCodeExists(tx *gorm.DB, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	model := &meta.IrModel{}
	if err := tx.Where("application = ? AND name = ?", "base", "Language").First(model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	table := strings.TrimSpace(model.ModelTable)
	if table == "" {
		return false, nil
	}
	var count int64
	if err := tx.Table(table).Where("code = ?", code).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (l *Loader) assertSeedLanguageCode(
	tx *gorm.DB,
	filePath string,
	recordIndex int,
	rec record,
	fieldPath string,
	lang string,
	selfCode string,
) error {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return &LoadError{
			Kind:        LoadErrorKindValidation,
			Code:        LoadErrorCodeTranslatedSeedInvalid,
			FilePath:    filePath,
			RecordIndex: recordIndex,
			Module:      strings.TrimSpace(rec.Module),
			ExternalID:  strings.TrimSpace(rec.ExternalID),
			Model:       strings.TrimSpace(rec.Model),
			FieldPath:   fieldPath,
			Message:     "translated lang map key must be a non-empty terminology code",
		}
	}
	if strings.Contains(lang, "-") {
		return &LoadError{
			Kind:        LoadErrorKindValidation,
			Code:        LoadErrorCodeTranslatedSeedInvalid,
			FilePath:    filePath,
			RecordIndex: recordIndex,
			Module:      strings.TrimSpace(rec.Module),
			ExternalID:  strings.TrimSpace(rec.ExternalID),
			Model:       strings.TrimSpace(rec.Model),
			FieldPath:   fieldPath,
			Message:     "translated lang map key looks like a UI locale; use a terminology code such as zh_CN",
		}
	}
	if lang == translatedBaseLang {
		return nil
	}
	if selfCode != "" && lang == selfCode {
		return nil
	}
	ok, err := l.languageCodeExists(tx, lang)
	if err != nil {
		return wrapLoadErrorWithCode(err, filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBError, "lookup Language.Code")
	}
	if !ok {
		return &LoadError{
			Kind:        LoadErrorKindValidation,
			Code:        LoadErrorCodeTranslatedLangUnknown,
			FilePath:    filePath,
			RecordIndex: recordIndex,
			Module:      strings.TrimSpace(rec.Module),
			ExternalID:  strings.TrimSpace(rec.ExternalID),
			Model:       strings.TrimSpace(rec.Model),
			FieldPath:   fieldPath,
			Message:     "unknown Language.Code in translated seed map: " + lang,
		}
	}
	return nil
}

func stringifySeedScalar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// normalizeTranslatedSeedValue rewrites translate-field seed inputs:
// scalar → {"en_US": scalar}; lang map → validated map; rejects { "_t": ... }.
func (l *Loader) normalizeTranslatedSeedValue(
	tx *gorm.DB,
	filePath string,
	recordIndex int,
	rec record,
	model *meta.IrModel,
	fieldName string,
	value any,
	rawValues map[string]any,
) (any, error) {
	field, err := l.lookupIrField(tx, model, fieldName)
	if err != nil {
		return nil, wrapLoadErrorWithCode(err, filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBError, "lookup IrField")
	}
	if !isTranslateField(field) {
		return value, nil
	}

	fieldPath := "values." + fieldName
	selfCode := ""
	if strings.EqualFold(strings.TrimSpace(rec.Model), "base.Language") {
		selfCode = seedSelfLanguageCode(rawValues)
	}

	if value == nil {
		return nil, nil
	}

	switch t := value.(type) {
	case map[string]any:
		if _, hasTerm := t["_t"]; hasTerm {
			return nil, &LoadError{
				Kind:        LoadErrorKindValidation,
				Code:        LoadErrorCodeTranslatedSeedInvalid,
				FilePath:    filePath,
				RecordIndex: recordIndex,
				Module:      strings.TrimSpace(rec.Module),
				ExternalID:  strings.TrimSpace(rec.ExternalID),
				Model:       strings.TrimSpace(rec.Model),
				FieldPath:   fieldPath,
				Message:     `translated seed must not use { "_t": ... }; use a lang map or English scalar`,
			}
		}
		if _, ok, _ := parseRefQuerySpec(t); ok {
			return value, nil
		}
		out := make(map[string]any, len(t))
		for k, v := range t {
			lang := strings.TrimSpace(k)
			if err := l.assertSeedLanguageCode(tx, filePath, recordIndex, rec, fieldPath, lang, selfCode); err != nil {
				return nil, err
			}
			if v == nil {
				return nil, &LoadError{
					Kind:        LoadErrorKindValidation,
					Code:        LoadErrorCodeTranslatedSeedInvalid,
					FilePath:    filePath,
					RecordIndex: recordIndex,
					Module:      strings.TrimSpace(rec.Module),
					ExternalID:  strings.TrimSpace(rec.ExternalID),
					Model:       strings.TrimSpace(rec.Model),
					FieldPath:   fieldPath,
					Message:     "translated lang map values must be strings",
				}
			}
			switch vv := v.(type) {
			case string:
				out[lang] = vv
			case float64, bool, int, int64, int32:
				out[lang] = stringifySeedScalar(vv)
			default:
				return nil, &LoadError{
					Kind:        LoadErrorKindValidation,
					Code:        LoadErrorCodeTranslatedSeedInvalid,
					FilePath:    filePath,
					RecordIndex: recordIndex,
					Module:      strings.TrimSpace(rec.Module),
					ExternalID:  strings.TrimSpace(rec.ExternalID),
					Model:       strings.TrimSpace(rec.Model),
					FieldPath:   fieldPath,
					Message:     "translated lang map values must be strings",
				}
			}
		}
		if len(out) == 0 {
			return nil, &LoadError{
				Kind:        LoadErrorKindValidation,
				Code:        LoadErrorCodeTranslatedSeedInvalid,
				FilePath:    filePath,
				RecordIndex: recordIndex,
				Module:      strings.TrimSpace(rec.Module),
				ExternalID:  strings.TrimSpace(rec.ExternalID),
				Model:       strings.TrimSpace(rec.Model),
				FieldPath:   fieldPath,
				Message:     "translated lang map must not be empty",
			}
		}
		return out, nil
	case string:
		return map[string]any{translatedBaseLang: t}, nil
	case float64, bool, int, int64, int32:
		return map[string]any{translatedBaseLang: stringifySeedScalar(t)}, nil
	default:
		return nil, &LoadError{
			Kind:        LoadErrorKindValidation,
			Code:        LoadErrorCodeTranslatedSeedInvalid,
			FilePath:    filePath,
			RecordIndex: recordIndex,
			Module:      strings.TrimSpace(rec.Module),
			ExternalID:  strings.TrimSpace(rec.ExternalID),
			Model:       strings.TrimSpace(rec.Model),
			FieldPath:   fieldPath,
			Message:     "translated seed expects a string or lang map object",
		}
	}
}
