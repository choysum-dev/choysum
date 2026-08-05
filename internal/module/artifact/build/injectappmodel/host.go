// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// Host is implemented by backend ModuleBuilder via a thin adapter (no circular import).
type Host interface {
	Module() *meta.Module
	SessionDB() *gorm.DB // may be nil
	ModulesPath() string
	EntryPointImports() []string
	SetEntryPointImports([]string)
	RegisterVirtualSource(path, contents string)
}
