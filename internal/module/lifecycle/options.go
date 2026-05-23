// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

// Option configures a ModuleManager.
type Option func(m *ModuleManager)

// WithEntities replaces the default entity set.
func WithEntities(entities ...any) Option {
	return func(m *ModuleManager) {
		m.entities = append([]any(nil), entities...)
	}
}

// WithExtraEntities appends entities to the default set.
func WithExtraEntities(entities ...any) Option {
	return func(m *ModuleManager) {
		m.entities = append(m.entities, entities...)
	}
}

// WithOriginCoordinatorFactory injects an origin coordinator factory for origin resolution and fetch operations.
func WithOriginCoordinatorFactory(factory func(runtimeScope scope.Scope) OriginCoordinator) Option {
	return func(m *ModuleManager) {
		m.originCoordinatorFactory = factory
	}
}

// WithLockerFactory injects the lock implementation used by module-manager
// critical sections.
func WithLockerFactory(factory statepkg.LockerFactory) Option {
	return func(m *ModuleManager) {
		if factory == nil {
			return
		}
		m.lockerFactory = factory
	}
}
