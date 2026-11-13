// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"path/filepath"
	"strings"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/module/policy"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// TerminologyReloadStats aggregates packaged PO import counters for hotreload logs.
type TerminologyReloadStats struct {
	Upserted        int
	SkippedOverride int
	RejectedNoCtxt  int
	SkippedObsolete int
	PurgedRetired   int
}

// frameworkModuleName is the shared platform module. Its PO catalogs are written
// to core_translation_term (Scheme B); host apps import only their own PO.
const frameworkModuleName = "core"

// ReloadModuleTerminology re-imports modules/<module>/i18n/*.po into TranslationTerm
// without a full module Upgrade. Used by server hotreload when a .po file changes.
func ReloadModuleTerminology(runtimeScope scope.Scope, moduleName string) (TerminologyReloadStats, error) {
	out := TerminologyReloadStats{}
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return out, xfmt.Errorf("module name is required")
	}
	if runtimeScope == nil {
		return out, xfmt.Errorf("runtime scope is nil")
	}

	modulesPath := strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).modulesPath)
	application, err := policy.ReadModuleApplicationFromPackageJSON(modulesPath, moduleName)
	if err != nil {
		return out, xfmt.Errorf("resolve application for module %s: %w", moduleName, err)
	}
	application = strings.TrimSpace(application)

	acc := &i18nimport.ImportStats{}
	ctx := runtimeScope.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = i18nimport.ContextWithStatsAccumulator(ctx, acc)
	scoped := runtimeScope.WithContext(ctx)

	mod := &meta.Module{
		Name:           moduleName,
		ApplicationStr: application,
		Path:           filepath.Join(modulesPath, moduleName),
	}
	if err := importModuleTerminology(scoped, mod, modulesPath); err != nil {
		return out, err
	}
	out.Upserted = acc.Upserted
	out.SkippedOverride = acc.SkippedOverride
	out.RejectedNoCtxt = acc.RejectedNoCtxt
	out.SkippedObsolete = acc.SkippedObsolete
	out.PurgedRetired = acc.PurgedRetired
	return out, nil
}

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
	// Scheme A left Module=core rows in host tables; purge on every host import/upgrade.
	return purgeStaleFrameworkTermsFromHost(runtimeScope, application)
}

// purgeStaleFrameworkTermsFromHost deletes Module=core rows from a host app table.
func purgeStaleFrameworkTermsFromHost(runtimeScope scope.Scope, hostApplication string) error {
	hostApplication = strings.TrimSpace(hostApplication)
	if hostApplication == "" || hostApplication == frameworkModuleName {
		return nil
	}
	reg := store.RegistryFor(runtimeScope)
	if err := i18nimport.DeleteModuleTerms(runtimeScope, reg, hostApplication, frameworkModuleName); err != nil {
		return xfmt.Errorf("purge stale framework terminology from %s: %w", hostApplication, err)
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
