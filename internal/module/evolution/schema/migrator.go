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

// MigratorOption configures NewMigrator. Options may reject an unsupported migrator.
type MigratorOption func(*migrator) error

// WithIntentBag attaches the upgrade Intent bag used by ValidatePlan.
func WithIntentBag(bag IntentBag) MigratorOption {
	return func(m *migrator) error {
		if m == nil {
			return nil
		}
		mm, ok := m.modelMigrator.(*modelMigrator)
		if !ok {
			return fmt.Errorf("WithIntentBag requires *modelMigrator, got %T", m.modelMigrator)
		}
		mm.intents = bag
		return nil
	}
}

// WithToVersion sets the target module version for dropAfter leftover warnings.
func WithToVersion(version string) MigratorOption {
	return func(m *migrator) error {
		if m == nil {
			return nil
		}
		mm, ok := m.modelMigrator.(*modelMigrator)
		if !ok {
			return fmt.Errorf("WithToVersion requires *modelMigrator, got %T", m.modelMigrator)
		}
		mm.toVersion = version
		return nil
	}
}

func NewMigrator(runtimeScope scope.Scope, module *meta.Module, opts ...MigratorOption) (Migrator, error) {
	m, err := newMigrator(runtimeScope, module)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(m); err != nil {
			return nil, err
		}
	}
	return m, nil
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
