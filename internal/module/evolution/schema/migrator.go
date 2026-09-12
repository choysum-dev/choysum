// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type Migrator interface {
	Migrate() error
	PlanOnly() (SchemaPlan, error)
}

func NewMigrator(runtimeScope scope.Scope, module *meta.Module) (Migrator, error) {
	return newMigrator(runtimeScope, module)
}

func newMigrator(runtimeScope scope.Scope, module *meta.Module) (*migrator, error) {
	models, err := loadModelsForSchema(runtimeScope, module)
	if err != nil {
		return nil, err
	}
	return &migrator{
		modelMigrator:      newModelMigrator(runtimeScope, module, models),
		foreignKeyMigrator: newForeignKeyMigrator(runtimeScope, module, models),
	}, nil
}

type migrator struct {
	modelMigrator      ModelMigrator
	foreignKeyMigrator ForeignKeyMigrator
}

func (m *migrator) Migrate() error {
	if err := m.modelMigrator.MigrateSchema(); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	if err := m.foreignKeyMigrator.MigrateForeignKeys(); err != nil {
		return fmt.Errorf("migrate foreign keys: %w", err)
	}
	return nil
}

func (m *migrator) PlanOnly() (SchemaPlan, error) {
	plan, err := m.modelMigrator.PlanSchema()
	if err != nil {
		return plan, fmt.Errorf("plan schema: %w", err)
	}
	return plan, nil
}
