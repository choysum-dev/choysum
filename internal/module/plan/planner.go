// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"golang.org/x/exp/errors/fmt"
)

type Resolver interface {
	// Peek returns a module described by its manifest without persisting it to modules/.
	Peek(ctx context.Context, name string) (*meta.Module, error)
	// Load returns the module from DB if it exists (typically installed).
	Load(name string) (*meta.Module, error)
}

func BuildPlan(ctx context.Context, op OpType, root *meta.Module, r Resolver) (Plan, error) {
	if root == nil {
		return Plan{}, fmt.Errorf("root module is nil")
	}
	if r == nil {
		return Plan{}, fmt.Errorf("resolver is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	plan := Plan{Op: op}

	apps := map[string]bool{}
	needsGlobalWebBuild := false
	addApp := func(mod *meta.Module) {
		if mod == nil {
			return
		}

		if strings.EqualFold(strings.TrimSpace(mod.Name), "web") || strings.TrimSpace(mod.WebEntryPoint) != "" {
			needsGlobalWebBuild = true
		}

		name := strings.TrimSpace(mod.ApplicationStr)
		if name == "" {
			return
		}
		// "web" application is handled by the global web build stage (dist/web).
		// Including it in app-stage would collide with dist/web publication.
		if strings.EqualFold(name, "web") {
			return
		}
		apps[name] = true
	}

	switch op {
	case OpInstall:
		reportBuildPlanProgress(ctx, BuildPlanProgress{Step: "resolve_dependencies", CurrentModule: strings.TrimSpace(root.Name)})
		order, err := topoByDependsStr(ctx, root, r, addApp)
		if err != nil {
			return Plan{}, err
		}
		plan.ModuleOrder = order
	case OpUninstall:
		order, err := reverseTopoByDependents(root, r, addApp)
		if err != nil {
			return Plan{}, err
		}
		plan.ModuleOrder = order
	case OpUpgrade:
		addApp(root)
		plan.ModuleOrder = []string{root.Name}
	default:
		return Plan{}, fmt.Errorf("unknown op: %q", string(op))
	}

	// If a web module is currently installed, keep global web build enabled.
	// This is intentionally conservative because dist/web aggregates imports across modules/apps.
	reportBuildPlanProgress(ctx, BuildPlanProgress{Step: "resolve_web_build"})
	if !needsGlobalWebBuild {
		webMod, err := r.Load("web")
		if err != nil {
			return Plan{}, fmt.Errorf("load web module for plan: %w", err)
		}
		if webMod != nil && webMod.Status == meta.Installed && strings.TrimSpace(webMod.WebEntryPoint) != "" {
			needsGlobalWebBuild = true
		}
	}
	plan.NeedsGlobalWebBuild = needsGlobalWebBuild

	for app := range apps {
		plan.AffectedApps = append(plan.AffectedApps, app)
	}
	sort.Strings(plan.AffectedApps)

	return plan, nil
}

func topoByDependsStr(ctx context.Context, root *meta.Module, r Resolver, addApp func(*meta.Module)) ([]string, error) {
	visited := map[string]bool{}
	stack := []string{}
	onStack := map[string]bool{}
	order := []string{}
	resolvedModules := 0
	resolvedDependencies := 0

	var dfs func(mod *meta.Module) error
	dfs = func(mod *meta.Module) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if mod == nil {
			return nil
		}
		name := mod.Name
		if name == "" {
			return nil
		}
		if visited[name] {
			return nil
		}
		if onStack[name] {
			// build cycle path
			idx := -1
			for i, v := range stack {
				if v == name {
					idx = i
					break
				}
			}
			cycle := append([]string{}, stack...)
			if idx >= 0 {
				cycle = append([]string{}, stack[idx:]...)
			}
			cycle = append(cycle, name)
			return fmt.Errorf("dependency cycle detected: %s", strings.Join(cycle, " -> "))
		}

		onStack[name] = true
		stack = append(stack, name)
		defer func() {
			onStack[name] = false
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}()

		addApp(mod)

		dependsArr := make([]string, 0)
		if len(mod.DependsStr) > 0 {
			if err := json.Unmarshal(mod.DependsStr, &dependsArr); err != nil {
				return fmt.Errorf("unmarshal depends for %s: %w", mod.Name, err)
			}
		}
		seen := map[string]bool{}
		for _, dep := range dependsArr {
			dep = strings.TrimSpace(dep)
			if dep == "" || seen[dep] {
				continue
			}
			seen[dep] = true
			resolvedDependencies++
			reportBuildPlanProgress(ctx, BuildPlanProgress{
				Step:                 "resolve_dependencies",
				CurrentModule:        dep,
				ResolvedModules:      resolvedModules,
				ResolvedDependencies: resolvedDependencies,
			})

			depMod, err := r.Load(dep)
			if err != nil {
				return fmt.Errorf("load dependency %s: %w", dep, err)
			}
			if depMod == nil {
				depMod, err = r.Peek(ctx, dep)
				if err != nil {
					return fmt.Errorf("peek dependency %s: %w", dep, err)
				}
			}
			if err := dfs(depMod); err != nil {
				return err
			}
		}

		visited[name] = true
		order = append(order, name)
		resolvedModules++
		reportBuildPlanProgress(ctx, BuildPlanProgress{
			Step:                 "resolve_modules",
			CurrentModule:        name,
			ResolvedModules:      resolvedModules,
			ResolvedDependencies: resolvedDependencies,
		})
		return nil
	}

	if err := dfs(root); err != nil {
		return nil, err
	}
	return order, nil
}

func reverseTopoByDependents(root *meta.Module, r Resolver, addApp func(*meta.Module)) ([]string, error) {
	visited := map[string]bool{}
	stack := []string{}
	onStack := map[string]bool{}
	order := []string{}

	var dfs func(name string) error
	dfs = func(name string) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		if visited[name] {
			return nil
		}
		if onStack[name] {
			idx := -1
			for i, v := range stack {
				if v == name {
					idx = i
					break
				}
			}
			cycle := append([]string{}, stack...)
			if idx >= 0 {
				cycle = append([]string{}, stack[idx:]...)
			}
			cycle = append(cycle, name)
			return fmt.Errorf("dependent cycle detected: %s", strings.Join(cycle, " -> "))
		}

		onStack[name] = true
		stack = append(stack, name)
		defer func() {
			onStack[name] = false
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}()

		mod, err := r.Load(name)
		if err != nil {
			return fmt.Errorf("load module %s: %w", name, err)
		}
		if mod == nil {
			// If module isn't found (already uninstalled), treat as no-op.
			visited[name] = true
			return nil
		}

		addApp(mod)

		seen := map[string]bool{}
		for _, depd := range mod.Dependents {
			if depd == nil {
				continue
			}
			depName := strings.TrimSpace(depd.Name)
			if depName == "" || seen[depName] {
				continue
			}
			seen[depName] = true
			if err := dfs(depName); err != nil {
				return err
			}
		}

		visited[name] = true
		order = append(order, name)
		return nil
	}

	if root == nil {
		return nil, nil
	}
	if err := dfs(root.Name); err != nil {
		return nil, err
	}
	return order, nil
}
