// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
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

func NewMigrator(runtimeScope scope.Scope, module *meta.IrModule) Migrator {
	return newMigrator(runtimeScope, module)
}

func newMigrator(runtimeScope scope.Scope, module *meta.IrModule) *migrator {
	models, _ := getModuleModels(runtimeScope, module)
	return &migrator{
		runtimeScope:       runtimeScope,
		module:             module,
		modelMigrator:      newModelMigrator(runtimeScope, module, models),
		foreignKeyMigrator: newForeignKeyMigrator(runtimeScope, module, models),
	}
}

func getModuleModels(runtimeScope scope.Scope, module *meta.IrModule) ([]*meta.IrModel, error) {
	var moduleModels []*meta.IrModel

	if result := runtimeScope.Session().
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Fields.Decorators", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Fields.Decorators.Arguments", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Where(&meta.IrModel{ModuleId: module.Id}).
		Where("abstract = ?", false).
		Find(&moduleModels); result.Error != nil {
		return nil, xfmt.Errorf("error getting models by module id: %w", result.Error)
	}

	filteredModels := make([]*meta.IrModel, 0, len(moduleModels))
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
	module             *meta.IrModule
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
	if m.module != nil {
		application = m.module.ApplicationStr
	}
	if err := i18nmodels.EnsureTranslationTermTable(m.runtimeScope, application); err != nil {
		return fmt.Errorf("ensure translation term table: %w", err)
	}

	// 3. Apply foreign key constraints.
	if err := m.foreignKeyMigrator.MigrateForeignKeys(); err != nil {
		return fmt.Errorf("migrate foreign keys: %w", err)
	}

	return nil
}
