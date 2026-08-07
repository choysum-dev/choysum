// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// DeclarationQuery filters declaration-layer (meta_raw_*) rows for ListDeclarations.
// Callers receive []*pkgmeta.Model; Path / ModuleId / Id are preserved from raw.
type DeclarationQuery struct {
	Application string
	Name        string
	ModuleID    string
	Path        string
	Abstract    *bool
	// PreloadTree loads Fields / Services / Decorators trees (DDL, merge).
	PreloadTree bool
}

// ListDeclarations loads declaration rows matching q and returns them as pkgmeta.Model trees.
func ListDeclarations(db *gorm.DB, q DeclarationQuery) ([]*pkgmeta.Model, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	query := db.Model(&rawModel{})
	if app := strings.TrimSpace(q.Application); app != "" {
		query = query.Where("application = ?", app)
	}
	if name := strings.TrimSpace(q.Name); name != "" {
		query = query.Where("name = ?", name)
	}
	if moduleID := strings.TrimSpace(q.ModuleID); moduleID != "" {
		query = query.Where("module_id = ?", moduleID)
	}
	if path := strings.TrimSpace(q.Path); path != "" {
		query = query.Where("path = ?", path)
	}
	if q.Abstract != nil {
		query = query.Where("abstract = ?", *q.Abstract)
	}
	if q.PreloadTree {
		orderID := func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }
		query = query.
			Preload("Fields", orderID).
			Preload("Fields.Decorators", orderID).
			Preload("Fields.Decorators.Arguments", orderID).
			Preload("Services", orderID).
			Preload("Services.Parameters", orderID).
			Preload("Services.TypeParameters", orderID).
			Preload("Services.Decorators", orderID).
			Preload("Services.Decorators.Arguments", orderID).
			Preload("Decorators", orderID).
			Preload("Decorators.Arguments", orderID)
	}

	var raws []*rawModel
	if err := query.Order("id DESC").Find(&raws).Error; err != nil {
		return nil, fmt.Errorf("list declarations: %w", err)
	}
	return rawModelsAsModels(raws), nil
}

// DeleteDeclarationTrees hard-deletes raw catalog trees for the given model ids
// (cascade: arguments → decorators → type/params → services → fields → models).
func DeleteDeclarationTrees(db *gorm.DB, modelIDs []string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	ids := make([]string, 0, len(modelIDs))
	seen := map[string]struct{}{}
	for _, id := range modelIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		fresh := func() *gorm.DB { return tx.Session(&gorm.Session{NewDB: true}).Unscoped() }

		var serviceIDs []string
		if err := fresh().Model(&rawService{}).Where("model_id IN ?", ids).Pluck("id", &serviceIDs).Error; err != nil {
			return fmt.Errorf("load declaration services: %w", err)
		}
		var fieldIDs []string
		if err := fresh().Model(&rawField{}).Where("model_id IN ?", ids).Pluck("id", &fieldIDs).Error; err != nil {
			return fmt.Errorf("load declaration fields: %w", err)
		}
		decoratorQ := fresh().Model(&rawDecorator{}).Where("model_id IN ?", ids)
		if len(serviceIDs) > 0 {
			decoratorQ = decoratorQ.Or("service_id IN ?", serviceIDs)
		}
		if len(fieldIDs) > 0 {
			decoratorQ = decoratorQ.Or("field_id IN ?", fieldIDs)
		}
		var decoratorIDs []string
		if err := decoratorQ.Pluck("id", &decoratorIDs).Error; err != nil {
			return fmt.Errorf("load declaration decorators: %w", err)
		}

		if len(decoratorIDs) > 0 {
			if err := deleteWhereFn(fresh(), &rawArgument{}, "decorator_id IN ?", decoratorIDs); err != nil {
				return fmt.Errorf("delete declaration arguments: %w", err)
			}
			if err := deleteWhereFn(fresh(), &rawDecorator{}, "id IN ?", decoratorIDs); err != nil {
				return fmt.Errorf("delete declaration decorators: %w", err)
			}
		}
		if len(serviceIDs) > 0 {
			if err := deleteWhereFn(fresh(), &rawTypeParameter{}, "service_id IN ?", serviceIDs); err != nil {
				return fmt.Errorf("delete declaration type parameters: %w", err)
			}
			if err := deleteWhereFn(fresh(), &rawParameter{}, "service_id IN ?", serviceIDs); err != nil {
				return fmt.Errorf("delete declaration parameters: %w", err)
			}
			if err := deleteWhereFn(fresh(), &rawService{}, "id IN ?", serviceIDs); err != nil {
				return fmt.Errorf("delete declaration services: %w", err)
			}
		}
		if len(fieldIDs) > 0 {
			if err := deleteWhereFn(fresh(), &rawField{}, "id IN ?", fieldIDs); err != nil {
				return fmt.Errorf("delete declaration fields: %w", err)
			}
		}
		if err := deleteWhereFn(fresh(), &rawModel{}, "id IN ?", ids); err != nil {
			return fmt.Errorf("delete declaration models: %w", err)
		}
		return nil
	})
}

// RemoveModuleDeclarations hard-deletes all declaration trees for moduleID and returns
// the logical keys that must be flushed afterward. Does not call FlushEffective.
func RemoveModuleDeclarations(db *gorm.DB, moduleID string) ([]LogicalKey, error) {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	decls, err := ListDeclarations(db, DeclarationQuery{ModuleID: moduleID})
	if err != nil {
		return nil, fmt.Errorf("list previous raw models: %w", err)
	}
	keys := make([]LogicalKey, 0, len(decls))
	seen := map[string]struct{}{}
	for _, row := range decls {
		if row == nil {
			continue
		}
		k := LogicalKey{Application: row.Application, Name: row.Name}.Normalized()
		if !k.Valid() {
			continue
		}
		id := k.Application + "\x00" + k.Name
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		keys = append(keys, k)
	}

	if err := deleteRawModelsForModule(db, moduleID); err != nil {
		return nil, fmt.Errorf("delete raw models: %w", err)
	}
	return keys, nil
}

// ReplaceModuleDeclarations rewrites all declaration trees for moduleID from models
// and returns the logical keys that must be flushed afterward.
func ReplaceModuleDeclarations(db *gorm.DB, moduleID string, models []*pkgmeta.Model) ([]LogicalKey, error) {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	orderedPaths := make([]string, 0, len(models))
	modelByPath := make(map[string]*pkgmeta.Model, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		path := strings.TrimSpace(m.Path)
		if path == "" {
			continue
		}
		if _, exists := modelByPath[path]; !exists {
			orderedPaths = append(orderedPaths, path)
		}
		modelByPath[path] = m
	}

	keys := make([]LogicalKey, 0)
	appendKey := func(application, name string) {
		k := LogicalKey{Application: application, Name: name}.Normalized()
		if k.Valid() {
			keys = append(keys, k)
		}
	}

	prevDecls, err := ListDeclarations(db, DeclarationQuery{ModuleID: moduleID})
	if err != nil {
		return nil, fmt.Errorf("list previous raw models: %w", err)
	}
	for _, row := range prevDecls {
		appendKey(row.Application, row.Name)
	}

	var prevEff []pkgmeta.Model
	if err := db.Model(&pkgmeta.Model{}).
		Select("id, application, name").
		Where("module_id = ?", moduleID).
		Find(&prevEff).Error; err != nil {
		return nil, fmt.Errorf("list previous effective models: %w", err)
	}
	for _, row := range prevEff {
		appendKey(row.Application, row.Name)
	}

	for _, path := range orderedPaths {
		m := modelByPath[path]
		m.ModuleId = sql.NullString{String: moduleID, Valid: true}
		appendKey(m.Application, m.Name)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := deleteRawModelsForModule(tx, moduleID); err != nil {
			return fmt.Errorf("delete previous raw models: %w", err)
		}
		for _, path := range orderedPaths {
			m := modelByPath[path]
			if err := persistModelTreeAsRaw(tx, m); err != nil {
				return fmt.Errorf("persist raw model %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}
