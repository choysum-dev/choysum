// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

const importNamespace = "import"

// MetaModelDataKey holds parsed external id parts.
type MetaModelDataKey struct {
	Module string
	Name   string
}

// ParseMetaModelDataKey parses module.name external id keys.
func ParseMetaModelDataKey(raw string) (MetaModelDataKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return MetaModelDataKey{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "external id must not be empty")
	}
	dot := strings.Index(raw, ".")
	if dot < 0 {
		module := importNamespace
		name := raw
		if name == "" {
			return MetaModelDataKey{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "external id name must not be empty")
		}
		return MetaModelDataKey{Module: module, Name: name}, nil
	}
	module := strings.TrimSpace(raw[:dot])
	name := strings.TrimSpace(raw[dot+1:])
	if module == "" || name == "" {
		return MetaModelDataKey{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "external id must be module.name")
	}
	return MetaModelDataKey{Module: module, Name: name}, nil
}

// AssertExternalIDWritable rejects protected initdata xml_ids before overwrite.
func AssertExternalIDWritable(tx *gorm.DB, key MetaModelDataKey, row int) error {
	if tx == nil {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "database session is required")
	}
	var mapping modmeta.ModelData
	err := tx.Where("module = ? AND name = ?", key.Module, key.Name).First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return importpkg.ErrorfWrap(importpkg.CodeConstraint, "lookup external id mapping", err)
	}
	if mapping.NoUpdate {
		return &importpkg.Error{
			Code:      importpkg.CodeExternalIDProtected,
			Text:      "external id is protected by noupdate",
			Row:       row,
			Field:     "id",
			RecordRef: key.Module + "." + key.Name,
		}
	}
	if key.Module != importNamespace && isInstalledModuleNamespace(tx, key.Module) {
		return &importpkg.Error{
			Code:      importpkg.CodeExternalIDProtected,
			Text:      "external id belongs to an installed module namespace",
			Row:       row,
			Field:     "id",
			RecordRef: key.Module + "." + key.Name,
		}
	}
	return nil
}

func isInstalledModuleNamespace(tx *gorm.DB, moduleName string) bool {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return false
	}
	var count int64
	if err := tx.Model(&meta.Module{}).Where("name = ?", moduleName).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func lookupExternalID(tx *gorm.DB, key MetaModelDataKey) (*modmeta.ModelData, error) {
	var mapping modmeta.ModelData
	err := tx.Where("module = ? AND name = ?", key.Module, key.Name).First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &mapping, nil
}

func upsertExternalIDMapping(tx *gorm.DB, key MetaModelDataKey, model *meta.Model, resID string) error {
	existing, err := lookupExternalID(tx, key)
	if err != nil {
		return err
	}
	if existing != nil {
		updates := map[string]any{
			"application": model.Application,
			"model_id":    model.Id.String,
			"model_name":  model.Name,
			"res_id":      resID,
		}
		return tx.Model(existing).Updates(updates).Error
	}
	mapping := &modmeta.ModelData{
		Module:      key.Module,
		Name:        key.Name,
		Application: model.Application,
		ModelId:     model.Id.String,
		ModelName:   model.Name,
		ResID:       resID,
	}
	return tx.Create(mapping).Error
}
