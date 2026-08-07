// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

// DualStoreRawEntities returns declaration-layer tables only.
func DualStoreRawEntities() []any {
	return []any{
		&rawModel{},
		&rawField{},
		&rawService{},
		&rawTypeParameter{},
		&rawParameter{},
		&rawDecorator{},
		&rawArgument{},
	}
}

// OpsEntities returns module admin/ops satellite tables; not dual-store shape.
// Includes meta_lock_lease (LockLease); acquire/renew/release stay in internal/state/lease.
func OpsEntities() []any {
	return []any{
		&ModuleIndex{},
		&ModelData{},
		&Setting{},
		&ModuleManagementLog{},
		&ModuleMigrationHistory{},
		&LockLease{},
	}
}

// CatalogEntities returns pkg/meta Entities plus DualStoreRawEntities and
// OpsEntities for a single AutoMigrate of the full catalog (effective
// projection + declaration tables + module admin/ops satellite tables).
func CatalogEntities() []any {
	out := append([]any{}, pkgmeta.Entities()...)
	out = append(out, DualStoreRawEntities()...)
	return append(out, OpsEntities()...)
}
