// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type Migrator interface {
	Migrate() error
}

func NewMigrator(runtimeScope scope.Scope, module *meta.Module) (Migrator, error) {
	return newMigrator(runtimeScope, module)
}

func newMigrator(runtimeScope scope.Scope, module *meta.Module) (*migrator, error) {
	models, err := getModuleModels(runtimeScope, module)
	if err != nil {
		return nil, err
	}
	return &migrator{
		runtimeScope:       runtimeScope,
		module:             module,
		modelMigrator:      newModelMigrator(runtimeScope, module, models),
		foreignKeyMigrator: newForeignKeyMigrator(runtimeScope, module, models),
	}, nil
}

func getModuleModels(runtimeScope scope.Scope, module *meta.Module) ([]*meta.Model, error) {
	absFalse := false
	moduleModels, err := meta.ListDeclarations(runtimeScope.Session().DB, meta.DeclarationQuery{
		ModuleID:    module.Id.String,
		Abstract:    &absFalse,
		PreloadTree: true,
	})
	if err != nil {
		return nil, xfmt.Errorf("error getting models by module id: %w", err)
	}

	// Declaration-only raw rows omit inherited columns; expand Extends in memory for DDL.
	if err := meta.ExpandModelsAlongExtends(runtimeScope.Session().DB, moduleModels); err != nil {
		return nil, xfmt.Errorf("error expanding model extends for schema: %w", err)
	}

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

	// 2. Seed terminology.editor ACL for TranslationTerm (table via model migrate).
	application := ""
	if m.module != nil {
		application = m.module.ApplicationStr
	}
	if err := i18nmodels.EnsureTerminologyEditorAllows(m.runtimeScope, application); err != nil {
		return fmt.Errorf("ensure terminology editor allows: %w", err)
	}

	// 3. Apply foreign key constraints.
	if err := m.foreignKeyMigrator.MigrateForeignKeys(); err != nil {
		return fmt.Errorf("migrate foreign keys: %w", err)
	}

	return nil
}
