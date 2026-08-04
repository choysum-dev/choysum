// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"database/sql"
	"fmt"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

type Migrator interface {
	Migrate() error
}

func NewMigrator(runtimeScope scope.Scope, module *meta.Module) Migrator {
	return newMigrator(runtimeScope, module)
}

func newMigrator(runtimeScope scope.Scope, module *meta.Module) *migrator {
	models, _ := getModuleModels(runtimeScope, module)
	return &migrator{
		runtimeScope:       runtimeScope,
		module:             module,
		modelMigrator:      newModelMigrator(runtimeScope, module, models),
		foreignKeyMigrator: newForeignKeyMigrator(runtimeScope, module, models),
	}
}

func getModuleModels(runtimeScope scope.Scope, module *meta.Module) ([]*meta.Model, error) {
	var rawModels []*meta.RawModel

	if result := runtimeScope.Session().
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Fields.Decorators", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Fields.Decorators.Arguments", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Where(&meta.RawModel{ModuleId: module.Id}).
		Where("abstract = ?", false).
		Find(&rawModels); result.Error != nil {
		return nil, xfmt.Errorf("error getting models by module id: %w", result.Error)
	}

	moduleModels := meta.RawModelsAsModels(rawModels)
	filteredModels := make([]*meta.Model, 0, len(moduleModels))
	for _, model := range moduleModels {
		if model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		filteredModels = append(filteredModels, model)
	}

	return filteredModels, nil
}

type migrator struct {
	runtimeScope       scope.Scope
	module             *meta.Module
	modelMigrator      ModelMigrator
	foreignKeyMigrator ForeignKeyMigrator
}

func (m *migrator) Migrate() error {
	// 1. Migrate model schemas.
	if err := m.modelMigrator.MigrateSchema(); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	// 2. Ensure per-application terminology table (skip application == "core").
	application := ""
	var moduleID sql.NullString
	if m.module != nil {
		application = m.module.ApplicationStr
		moduleID = m.module.Id
	}
	if err := i18nmodels.EnsureTranslationTermTable(m.runtimeScope, application); err != nil {
		return fmt.Errorf("ensure translation term table: %w", err)
	}
	if err := i18nmodels.EnsureI18nMeta(m.runtimeScope, application, moduleID); err != nil {
		return fmt.Errorf("ensure i18n ir meta: %w", err)
	}

	// 3. Apply foreign key constraints.
	if err := m.foreignKeyMigrator.MigrateForeignKeys(); err != nil {
		return fmt.Errorf("migrate foreign keys: %w", err)
	}

	return nil
}
