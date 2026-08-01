// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"path/filepath"
	"sort"
	"strings"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// frameworkModuleName is the shared platform module. Its PO catalogs are hosted
// in each real application's {app}_translation_term with Module=core (Scheme A).
// There is no core_translation_term / core.I18n.
const frameworkModuleName = "core"

// importModuleTerminology loads modules/<m>/i18n/*.po into the application table.
// Missing i18n dir is a no-op.
//
// Scheme A:
//   - Real modules: import into their ApplicationStr table, then also import
//     modules/core/i18n into that same host table as Module=core.
//   - Framework module (application == "core"): fan-out core PO into every
//     installed host application table; never create core_translation_term.
func importModuleTerminology(runtimeScope scope.Scope, mod *meta.Module, modulesPath string) error {
	if mod == nil {
		return nil
	}
	application := strings.TrimSpace(mod.ApplicationStr)
	moduleName := strings.TrimSpace(mod.Name)
	if moduleName == "" {
		return nil
	}

	modulesPath = strings.TrimSpace(modulesPath)
	if application == "" || application == frameworkModuleName {
		if moduleName != frameworkModuleName {
			return nil
		}
		return importFrameworkTerminologyIntoAllApps(runtimeScope, modulesPath, mod)
	}

	moduleRoot := resolveModuleRoot(mod, modulesPath, moduleName)
	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.ImportModuleI18nDir(runtimeScope, reg, application, moduleName, moduleRoot); err != nil {
		return xfmt.Errorf("import terminology for module %s: %w", moduleName, err)
	}
	if err := importFrameworkTerminology(runtimeScope, application, modulesPath, mod); err != nil {
		return err
	}
	return nil
}

// deleteModuleTerminology removes TranslationTerm rows for the module and invalidates cache.
// Framework module uninstall does not purge Module=core rows from host app tables
// (those stay until the host application itself is removed / cleaned).
func deleteModuleTerminology(runtimeScope scope.Scope, mod *meta.Module) error {
	if mod == nil {
		return nil
	}
	application := strings.TrimSpace(mod.ApplicationStr)
	moduleName := strings.TrimSpace(mod.Name)
	if application == "" || application == frameworkModuleName || moduleName == "" || moduleName == frameworkModuleName {
		return nil
	}
	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.DeleteModuleTerms(runtimeScope, reg, application, moduleName); err != nil {
		return xfmt.Errorf("delete terminology for module %s: %w", moduleName, err)
	}
	return nil
}

func importFrameworkTerminology(runtimeScope scope.Scope, hostApplication, modulesPath string, hint *meta.Module) error {
	hostApplication = strings.TrimSpace(hostApplication)
	if hostApplication == "" || hostApplication == frameworkModuleName {
		return nil
	}
	coreRoot := resolveFrameworkModuleRoot(modulesPath, hint)
	if coreRoot == "" {
		return nil
	}
	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.ImportModuleI18nDir(runtimeScope, reg, hostApplication, frameworkModuleName, coreRoot); err != nil {
		return xfmt.Errorf("import framework terminology into %s: %w", hostApplication, err)
	}
	return nil
}

func importFrameworkTerminologyIntoAllApps(runtimeScope scope.Scope, modulesPath string, hint *meta.Module) error {
	hosts, err := listHostApplications(runtimeScope)
	if err != nil {
		return err
	}
	for _, app := range hosts {
		if err := importFrameworkTerminology(runtimeScope, app, modulesPath, hint); err != nil {
			return err
		}
	}
	return nil
}

func listHostApplications(runtimeScope scope.Scope) ([]string, error) {
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil, nil
	}
	session := runtimeScope.Session()
	if !session.Migrator().HasTable((&meta.Module{}).TableName()) {
		return nil, nil
	}
	var modules []meta.Module
	if err := session.Where("status = ?", meta.Installed).Find(&modules).Error; err != nil {
		return nil, xfmt.Errorf("list host applications for framework i18n: %w", err)
	}
	seen := map[string]struct{}{}
	for _, mod := range modules {
		app := strings.TrimSpace(mod.ApplicationStr)
		if app == "" || app == frameworkModuleName {
			continue
		}
		seen[app] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for app := range seen {
		out = append(out, app)
	}
	sort.Strings(out)
	return out, nil
}

func resolveModuleRoot(mod *meta.Module, modulesPath, moduleName string) string {
	if mod != nil {
		if root := strings.TrimSpace(mod.Path); root != "" {
			return root
		}
	}
	if modulesPath == "" || moduleName == "" {
		return ""
	}
	return filepath.Join(modulesPath, moduleName)
}

func resolveFrameworkModuleRoot(modulesPath string, hint *meta.Module) string {
	if hint != nil && strings.TrimSpace(hint.Name) == frameworkModuleName {
		if root := strings.TrimSpace(hint.Path); root != "" {
			return root
		}
	}
	if modulesPath == "" {
		return ""
	}
	return filepath.Join(modulesPath, frameworkModuleName)
}
