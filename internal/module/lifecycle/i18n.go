// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"path/filepath"
	"strings"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// frameworkModuleName is the shared platform module. Its PO catalogs are written
// to core_translation_term (Scheme B); host apps import only their own PO.
const frameworkModuleName = "core"

// importModuleTerminology loads modules/<m>/i18n/*.po into the application table.
// Missing i18n dir is a no-op.
//
// Scheme B:
//   - application "core" and module "core": import into core_translation_term only.
//   - Real apps: import only that module's PO into {app}_translation_term (no fan-out).
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
	if application == "" {
		return nil
	}
	if application == frameworkModuleName {
		if moduleName != frameworkModuleName {
			return nil
		}
		moduleRoot := resolveModuleRoot(mod, modulesPath, moduleName)
		if err := runTerminologyImport(runtimeScope, frameworkModuleName, frameworkModuleName, moduleRoot); err != nil {
			return xfmt.Errorf("import terminology for module %s: %w", moduleName, err)
		}
		return nil
	}

	moduleRoot := resolveModuleRoot(mod, modulesPath, moduleName)
	if err := runTerminologyImport(runtimeScope, application, moduleName, moduleRoot); err != nil {
		return xfmt.Errorf("import terminology for module %s: %w", moduleName, err)
	}
	return nil
}

// deleteModuleTerminology removes TranslationTerm rows for the module and invalidates cache.
func deleteModuleTerminology(runtimeScope scope.Scope, mod *meta.Module) error {
	if mod == nil {
		return nil
	}
	application := strings.TrimSpace(mod.ApplicationStr)
	moduleName := strings.TrimSpace(mod.Name)
	if application == "" || moduleName == "" {
		return nil
	}
	if application == frameworkModuleName {
		if moduleName != frameworkModuleName {
			return nil
		}
	} else if moduleName == frameworkModuleName {
		return nil
	}
	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.DeleteModuleTerms(runtimeScope, reg, application, moduleName); err != nil {
		return xfmt.Errorf("delete terminology for module %s: %w", moduleName, err)
	}
	return nil
}

func runTerminologyImport(runtimeScope scope.Scope, application, moduleName, moduleRoot string) error {
	ctx := context.Background()
	if runtimeScope != nil && runtimeScope.Context() != nil {
		ctx = runtimeScope.Context()
	}
	spec := importpkg.Spec{
		Profile:     importpkg.ProfileTerminology,
		Caller:      importpkg.CallerLifecycle,
		Policy:      importpkg.PolicyAtomic,
		Module:      moduleName,
		Application: application,
		Source: importpkg.Source{
			Format: "po",
			Path:   moduleRoot,
		},
	}
	_, err := importpkg.Run(ctx, runtimeScope, spec)
	return err
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
