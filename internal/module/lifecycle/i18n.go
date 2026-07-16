// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"path/filepath"
	"strings"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// importModuleTerminology loads modules/<m>/i18n/*.po into the application table.
// Missing i18n dir is a no-op. core / empty application skips.
func importModuleTerminology(runtimeScope scope.Scope, mod *meta.IrModule, modulesPath string) error {
	if mod == nil {
		return nil
	}
	application := strings.TrimSpace(mod.ApplicationStr)
	moduleName := strings.TrimSpace(mod.Name)
	if application == "" || application == "core" || moduleName == "" {
		return nil
	}

	moduleRoot := strings.TrimSpace(mod.Path)
	if moduleRoot == "" {
		modulesPath = strings.TrimSpace(modulesPath)
		if modulesPath == "" {
			return nil
		}
		moduleRoot = filepath.Join(modulesPath, moduleName)
	}

	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.ImportModuleI18nDir(runtimeScope, reg, application, moduleName, moduleRoot); err != nil {
		return xfmt.Errorf("import terminology for module %s: %w", moduleName, err)
	}
	return nil
}

// deleteModuleTerminology removes TranslationTerm rows for the module and invalidates cache.
func deleteModuleTerminology(runtimeScope scope.Scope, mod *meta.IrModule) error {
	if mod == nil {
		return nil
	}
	application := strings.TrimSpace(mod.ApplicationStr)
	moduleName := strings.TrimSpace(mod.Name)
	if application == "" || application == "core" || moduleName == "" {
		return nil
	}
	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.DeleteModuleTerms(runtimeScope, reg, application, moduleName); err != nil {
		return xfmt.Errorf("delete terminology for module %s: %w", moduleName, err)
	}
	return nil
}
