// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import pkgmeta "github.com/choysum-dev/choysum/pkg/meta"

// ModuleCatalogDepend is the join row for MetaModuleCatalog self-ManyToMany Depends.
// ModuleId is the depender; DependId is the dependee.
type ModuleCatalogDepend struct {
	pkgmeta.BaseModel `gorm:"embedded"`

	ModuleID string `gorm:"column:module_id;type:char(20);not null;uniqueIndex:uidx_meta_module_catalog_depend;index"`
	DependID string `gorm:"column:depend_id;type:char(20);not null;uniqueIndex:uidx_meta_module_catalog_depend;index"`
}

func (ModuleCatalogDepend) TableName() string {
	return "meta_module_catalog_depend"
}
