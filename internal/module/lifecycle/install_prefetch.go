// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type prefetchedInstallModulesKey struct{}

// PrefetchedInstall holds modules materialized for an install before any outer
// database transaction is opened (Phase 1 Prepare–Commit split).
type PrefetchedInstall struct {
	RootName string
	Modules  map[string]*meta.IrModule
}

// WithPrefetchedInstallModules attaches a PrefetchInstallModules result so
// Install / pipeline resolution consumes it instead of fetching again.
func WithPrefetchedInstallModules(ctx context.Context, modules map[string]*meta.IrModule) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(modules) == 0 {
		return ctx
	}
	copied := make(map[string]*meta.IrModule, len(modules))
	for name, mod := range modules {
		name = strings.TrimSpace(name)
		if name == "" || mod == nil {
			continue
		}
		copied[name] = mod
	}
	if len(copied) == 0 {
		return ctx
	}
	return context.WithValue(ctx, prefetchedInstallModulesKey{}, copied)
}

// PrefetchedInstallModulesFromContext returns modules attached by
// WithPrefetchedInstallModules, or nil when absent.
func PrefetchedInstallModulesFromContext(ctx context.Context) map[string]*meta.IrModule {
	if ctx == nil {
		return nil
	}
	modules, _ := ctx.Value(prefetchedInstallModulesKey{}).(map[string]*meta.IrModule)
	return modules
}

// PrefetchInstallModules resolves the install root and Fetches every module in
// the install plan order. Callers should run this outside the outer install
// transaction, then pass the result via WithPrefetchedInstallModules.
func PrefetchInstallModules(ctx context.Context, runtimeScope scope.Scope, input string, opts ...Option) (*PrefetchedInstall, error) {
	if runtimeScope == nil {
		return nil, xfmt.Errorf("scope is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, xfmt.Errorf("module name is empty")
	}

	manager := NewModuleManager(runtimeScope, nil, opts...)
	return manager.PrefetchInstallModules(ctx, input)
}

// PrefetchInstallModules resolves the install root and Fetches every module in
// the install plan order onto ModulesPath (no outer DB transaction).
func (m *ModuleManager) PrefetchInstallModules(ctx context.Context, input string) (*PrefetchedInstall, error) {
	if m == nil {
		return nil, xfmt.Errorf("module manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, xfmt.Errorf("module name is empty")
	}

	root, err := m.fetchInstallModuleFromOrigin(ctx, input)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, xfmt.Errorf("module %s not found in origin", input)
	}
	rootName := strings.TrimSpace(root.Name)
	if rootName == "" {
		return nil, xfmt.Errorf("resolved module name is empty")
	}

	opPlan, err := plan.BuildPlan(ctx, plan.OpInstall, root, m)
	if err != nil {
		return nil, xfmt.Errorf("build install plan for prefetch: %w", err)
	}

	modules := make(map[string]*meta.IrModule, len(opPlan.ModuleOrder)+1)
	storePrefetchedModule(modules, input, root)
	storePrefetchedModule(modules, rootName, root)

	for _, name := range opPlan.ModuleOrder {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if existing := lookupPrefetchedModule(modules, name); existing != nil {
			continue
		}
		mod, fetchErr := m.fetchInstallModuleFromOrigin(ctx, name)
		if fetchErr != nil {
			return nil, fetchErr
		}
		if mod == nil {
			return nil, xfmt.Errorf("module %s not found in origin", name)
		}
		storePrefetchedModule(modules, name, mod)
		storePrefetchedModule(modules, strings.TrimSpace(mod.Name), mod)
	}

	if logger := m.runtimeScope.Logger(); logger != nil {
		logger.Info("install modules prefetched",
			"root", rootName,
			"modules_count", len(opPlan.ModuleOrder),
		)
	}

	return &PrefetchedInstall{RootName: rootName, Modules: modules}, nil
}

func storePrefetchedModule(dst map[string]*meta.IrModule, key string, mod *meta.IrModule) {
	key = strings.TrimSpace(key)
	if dst == nil || key == "" || mod == nil {
		return
	}
	dst[key] = mod
}

func lookupPrefetchedModule(modules map[string]*meta.IrModule, name string) *meta.IrModule {
	name = strings.TrimSpace(name)
	if name == "" || len(modules) == 0 {
		return nil
	}
	if mod := modules[name]; mod != nil {
		return mod
	}
	return nil
}

// fetchInstallModuleFromOrigin always resolves from origin (no prefetch cache).
func (m *ModuleManager) fetchInstallModuleFromOrigin(ctx context.Context, name string) (*meta.IrModule, error) {
	if err := m.ensureMetaTables(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, xfmt.Errorf("module name is empty")
	}

	coordinator := m.newOriginCoordinator()
	module, err := coordinator.ResolveInstallModule(ctx, name)
	if err != nil {
		return nil, xfmt.Errorf("error resolving module %s from origin: %w", name, err)
	}
	if module == nil {
		return nil, nil
	}

	if meta.IsCoreModule(module.Name) {
		if err := m.migrateBaseModule(); err != nil {
			return nil, err
		}
	}
	return module, nil
}
