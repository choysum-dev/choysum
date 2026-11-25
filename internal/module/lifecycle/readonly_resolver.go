// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"fmt"

	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/meta"
)

// ReadOnlyResolver implements plan.Resolver without AutoMigrate on the warm path.
// Catalog DDL runs only when meta_module is missing (cold start), or via write
// paths that call migrateBaseModule / ensureMetaTables explicitly.
type ReadOnlyResolver struct {
	m *ModuleManager
}

// NewReadOnlyResolver wraps a ModuleManager for Plan-style read-only resolution.
func NewReadOnlyResolver(m *ModuleManager) *ReadOnlyResolver {
	if m == nil {
		return nil
	}
	return &ReadOnlyResolver{m: m}
}

// AsReadOnlyResolver returns a Plan-safe resolver. ModuleManager becomes
// ReadOnlyResolver; an existing ReadOnlyResolver is returned as-is; other
// resolvers (tests) are unchanged.
func AsReadOnlyResolver(r plan.Resolver) plan.Resolver {
	if r == nil {
		return nil
	}
	if ro, ok := r.(*ReadOnlyResolver); ok {
		return ro
	}
	if m, ok := r.(*ModuleManager); ok {
		return NewReadOnlyResolver(m)
	}
	return r
}

func (r *ReadOnlyResolver) Load(name string) (*meta.Module, error) {
	if r == nil || r.m == nil {
		return nil, fmt.Errorf("readonly resolver: module manager is nil")
	}
	if err := r.ensureMetaModuleTable(); err != nil {
		return nil, err
	}
	return r.m.loadModuleRecord(name)
}

func (r *ReadOnlyResolver) Peek(ctx context.Context, name string) (*meta.Module, error) {
	if r == nil || r.m == nil {
		return nil, fmt.Errorf("readonly resolver: module manager is nil")
	}
	if err := r.ensureMetaModuleTable(); err != nil {
		return nil, err
	}
	return r.m.peekModuleRecord(ctx, name)
}

// ensureMetaModuleTable migrates CatalogEntities only when meta_module is absent.
func (r *ReadOnlyResolver) ensureMetaModuleTable() error {
	if r.m.runtimeScope == nil || r.m.runtimeScope.Session() == nil || r.m.runtimeScope.Session().DB == nil {
		return fmt.Errorf("readonly resolver: session is unavailable")
	}
	db := r.m.runtimeScope.Session().DB
	if db.Migrator().HasTable(&meta.Module{}) {
		return nil
	}
	return ensureCatalogSchemaMigrated(db)
}
