// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// AbstractModelSpec describes a Go-side abstract declaration (e.g. I18n) to ensure.
type AbstractModelSpec struct {
	Name         string
	Path         string
	Application  string
	ModuleID     sql.NullString
	ServiceNames []string
}

// EnsureAbstractModel upserts an abstract declaration and prefers its tip.
// Does not FlushEffective — callers flush at the install/persist boundary before
// reading effective rows (EDS-opt-2 / EDS-opt-3).
func EnsureAbstractModel(db *gorm.DB, spec AbstractModelSpec) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	name := strings.TrimSpace(spec.Name)
	path := strings.TrimSpace(spec.Path)
	app := strings.TrimSpace(spec.Application)
	if name == "" || path == "" || app == "" {
		return fmt.Errorf("EnsureAbstractModel requires name, path, and application")
	}
	decl := NewAbstractReadonlyDeclaration(name, path, app, spec.ModuleID, spec.ServiceNames)
	if err := UpsertDeclaration(db, decl); err != nil {
		return err
	}
	if err := PreferDeclarationTip(db, app, name, path); err != nil {
		return err
	}
	return nil
}

// ReplaceModuleDeclarations rewrites all declaration trees for moduleID from models
// and returns the logical keys that must be flushed afterward.
func ReplaceModuleDeclarations(db *gorm.DB, moduleID string, models []*Model) ([]LogicalKey, error) {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	orderedPaths := make([]string, 0, len(models))
	modelByPath := make(map[string]*Model, len(models))
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

	var prevEff []Model
	if err := db.Model(&Model{}).
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
		if err := DeleteRawModelsForModule(tx, moduleID); err != nil {
			return fmt.Errorf("delete previous raw models: %w", err)
		}
		for _, path := range orderedPaths {
			m := modelByPath[path]
			if err := PersistModelTreeAsRaw(tx, m); err != nil {
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
