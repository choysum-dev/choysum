// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DeclarationQuery filters declaration-layer (meta_raw_*) rows for ListDeclarations.
// Callers receive []*Model; Path / ModuleId / Id are preserved from raw.
type DeclarationQuery struct {
	Application string
	Name        string
	ModuleID    string
	Path        string
	Abstract    *bool
	// PreloadTree loads Fields / Services / Decorators trees (DDL, merge).
	PreloadTree bool
}

// ListDeclarations loads declaration rows matching q and returns them as Model trees.
func ListDeclarations(db *gorm.DB, q DeclarationQuery) ([]*Model, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	query := db.Model(&RawModel{})
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

	var raws []*RawModel
	if err := query.Order("id DESC").Find(&raws).Error; err != nil {
		return nil, fmt.Errorf("list declarations: %w", err)
	}
	return RawModelsAsModels(raws), nil
}

// UpsertDeclaration ensures a declaration at (application, path) exists.
// Missing rows are created via PersistModelTreeAsRaw; existing rows update ModuleId
// when empty and create any Services listed on src that are not yet present by name.
func UpsertDeclaration(db *gorm.DB, src *Model) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if src == nil {
		return fmt.Errorf("declaration model is nil")
	}
	path := strings.TrimSpace(src.Path)
	app := strings.TrimSpace(src.Application)
	if path == "" || app == "" {
		return fmt.Errorf("upsert declaration requires path and application")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var existing RawModel
		err := tx.Where("path = ? AND application = ?", path, app).
			Order("created_at DESC, id DESC").
			Take(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PersistModelTreeAsRaw(tx, src)
		}
		if err != nil {
			return fmt.Errorf("lookup declaration %s: %w", path, err)
		}

		if src.ModuleId.Valid && strings.TrimSpace(src.ModuleId.String) != "" &&
			(!existing.ModuleId.Valid || strings.TrimSpace(existing.ModuleId.String) == "") {
			if saveErr := tx.Model(&existing).Update("module_id", src.ModuleId.String).Error; saveErr != nil {
				return fmt.Errorf("update declaration module: %w", saveErr)
			}
			existing.ModuleId = src.ModuleId
		}

		for _, s := range src.Services {
			if s == nil || strings.TrimSpace(s.Name) == "" {
				continue
			}
			var svc RawService
			takeErr := tx.Where("model_id = ? AND name = ?", existing.Id.String, s.Name).Take(&svc).Error
			if takeErr == nil {
				continue
			}
			if !errors.Is(takeErr, gorm.ErrRecordNotFound) {
				return fmt.Errorf("lookup declaration service %s: %w", s.Name, takeErr)
			}
			ns := *s
			if strings.TrimSpace(ns.OriginModelPath) == "" {
				ns.OriginModelPath = existing.Path
			}
			if strings.TrimSpace(ns.AccessibilityModifier) == "" {
				ns.AccessibilityModifier = "public"
			}
			rs := rawServiceFromService(&ns, existing.Id)
			ensureBaseModelID(&rs.BaseModel)
			if createErr := tx.Create(rs).Error; createErr != nil {
				return fmt.Errorf("create declaration service %s: %w", s.Name, createErr)
			}
		}
		return nil
	})
}

// PreferDeclarationTip bumps created_at/updated_at on the declaration at path so
// pickTipRaw and E2 tip scalars prefer that row among same-name siblings.
func PreferDeclarationTip(db *gorm.DB, application, name, path string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	application = strings.TrimSpace(application)
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if application == "" || name == "" || path == "" {
		return fmt.Errorf("prefer tip requires application, name, and path")
	}

	var canonical RawModel
	if err := db.Where("path = ? AND application = ?", path, application).
		Order("created_at DESC, id DESC").
		Take(&canonical).Error; err != nil {
		return fmt.Errorf("lookup tip declaration: %w", err)
	}
	if !canonical.Id.Valid {
		return fmt.Errorf("tip declaration id is empty")
	}

	var siblings []RawModel
	if err := db.Select("id", "created_at", "updated_at").
		Where("application = ? AND name = ? AND id <> ?", application, name, canonical.Id.String).
		Find(&siblings).Error; err != nil {
		return fmt.Errorf("load tip siblings: %w", err)
	}
	tip := time.Now().UTC()
	for _, s := range siblings {
		if !tip.After(s.UpdatedAt) {
			tip = s.UpdatedAt.Add(time.Millisecond)
		}
		if !tip.After(s.CreatedAt) {
			tip = s.CreatedAt.Add(time.Millisecond)
		}
	}
	if err := db.Model(&canonical).UpdateColumns(map[string]any{
		"created_at": tip,
		"updated_at": tip,
	}).Error; err != nil {
		return fmt.Errorf("promote declaration tip: %w", err)
	}
	return nil
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
		if err := fresh().Model(&RawService{}).Where("model_id IN ?", ids).Pluck("id", &serviceIDs).Error; err != nil {
			return fmt.Errorf("load declaration services: %w", err)
		}
		var fieldIDs []string
		if err := fresh().Model(&RawField{}).Where("model_id IN ?", ids).Pluck("id", &fieldIDs).Error; err != nil {
			return fmt.Errorf("load declaration fields: %w", err)
		}
		decoratorQ := fresh().Model(&RawDecorator{}).Where("model_id IN ?", ids)
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
			if err := deleteWhereFn(fresh(), &RawArgument{}, "decorator_id IN ?", decoratorIDs); err != nil {
				return fmt.Errorf("delete declaration arguments: %w", err)
			}
			if err := deleteWhereFn(fresh(), &RawDecorator{}, "id IN ?", decoratorIDs); err != nil {
				return fmt.Errorf("delete declaration decorators: %w", err)
			}
		}
		if len(serviceIDs) > 0 {
			if err := deleteWhereFn(fresh(), &RawTypeParameter{}, "service_id IN ?", serviceIDs); err != nil {
				return fmt.Errorf("delete declaration type parameters: %w", err)
			}
			if err := deleteWhereFn(fresh(), &RawParameter{}, "service_id IN ?", serviceIDs); err != nil {
				return fmt.Errorf("delete declaration parameters: %w", err)
			}
			if err := deleteWhereFn(fresh(), &RawService{}, "id IN ?", serviceIDs); err != nil {
				return fmt.Errorf("delete declaration services: %w", err)
			}
		}
		if len(fieldIDs) > 0 {
			if err := deleteWhereFn(fresh(), &RawField{}, "id IN ?", fieldIDs); err != nil {
				return fmt.Errorf("delete declaration fields: %w", err)
			}
		}
		if err := deleteWhereFn(fresh(), &RawModel{}, "id IN ?", ids); err != nil {
			return fmt.Errorf("delete declaration models: %w", err)
		}
		return nil
	})
}

// HasDeclarationCatalog reports whether meta_raw_model / meta_raw_service exist.
func HasDeclarationCatalog(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	m := db.Migrator()
	return m.HasTable((&RawModel{}).TableName()) && m.HasTable((&RawService{}).TableName())
}

// HasEffectiveCatalog reports whether meta_model / meta_service exist.
func HasEffectiveCatalog(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	m := db.Migrator()
	return m.HasTable((&Model{}).TableName()) && m.HasTable((&Service{}).TableName())
}

// NewAbstractReadonlyDeclaration builds a minimal declaration Model used by builtins (e.g. I18n).
func NewAbstractReadonlyDeclaration(name, path, application string, moduleID sql.NullString, serviceNames []string) *Model {
	falseVal := false
	m := &Model{
		Name:        name,
		Path:        path,
		Application: application,
		ClassName:   name,
		Abstract:    true,
		Readonly:    true,
		AutoMigrate: &falseVal,
		ModuleId:    moduleID,
	}
	for _, methodName := range serviceNames {
		methodName = strings.TrimSpace(methodName)
		if methodName == "" {
			continue
		}
		m.Services = append(m.Services, &Service{
			Name:                  methodName,
			OriginModelPath:       path,
			AccessibilityModifier: "public",
			IsStatic:              true,
		})
	}
	return m
}
