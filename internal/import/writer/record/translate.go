// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"encoding/json"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

const translatedBaseLang = "en_US"

func fieldTranslateEnabled(db *gorm.DB, modelID, fieldName string) (bool, error) {
	modelID = strings.TrimSpace(modelID)
	fieldName = strings.TrimSpace(fieldName)
	if modelID == "" || fieldName == "" || db == nil {
		return false, nil
	}
	var field meta.Field
	if err := db.Where("model_id = ? AND name = ?", modelID, fieldName).First(&field).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	spec, err := field.GetResolvedSpec()
	if err != nil || spec == nil || spec.Structural.Translate == nil {
		return false, err
	}
	return *spec.Structural.Translate, nil
}

func encodeTranslatedScalar(raw string) (string, error) {
	payload := map[string]string{
		translatedBaseLang: raw,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
