// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package manifest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

func WriteDistManifest(ctx context.Context, runtimeScope scope.Scope, compileBundleMode string, outPath string) error {
	outPath = strings.TrimSpace(outPath)
	if outPath == "" {
		return xfmt.Errorf("dist manifest path is empty")
	}
	if runtimeScope == nil {
		return xfmt.Errorf("module runtime scope is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var installed []meta.IrModule
	sess := runtimeScope.Session()
	if sess == nil {
		return xfmt.Errorf("scope session is nil")
	}
	if err := sess.WithContext(ctx).Where("status = ?", meta.Installed).Find(&installed).Error; err != nil {
		return xfmt.Errorf("list installed modules: %w", err)
	}

	byName := map[string]meta.IrModule{}
	for _, mod := range installed {
		name := strings.TrimSpace(mod.Name)
		if name == "" {
			continue
		}
		byName[name] = mod
	}

	appsSet := map[string]bool{}
	webEnabled := false
	for _, mod := range installed {
		if strings.TrimSpace(mod.WebEntryPoint) != "" {
			webEnabled = true
		}

		app := strings.TrimSpace(mod.ApplicationStr)
		if app == "" || strings.EqualFold(app, "web") {
			continue
		}
		appsSet[app] = true
	}

	apps := make([]string, 0, len(appsSet))
	for app := range appsSet {
		apps = append(apps, app)
	}
	sort.Strings(apps)

	appToModules := map[string][]string{}
	appDeps := map[string]map[string]bool{}
	for _, app := range apps {
		appToModules[app] = nil
		appDeps[app] = map[string]bool{}
	}

	for _, mod := range installed {
		app := strings.TrimSpace(mod.ApplicationStr)
		if app == "" || strings.EqualFold(app, "web") {
			continue
		}
		if !appsSet[app] {
			continue
		}

		modName := strings.TrimSpace(mod.Name)
		if modName != "" {
			appToModules[app] = append(appToModules[app], modName)
		}

		var depends []string
		if len(mod.DependsStr) > 0 {
			_ = json.Unmarshal(mod.DependsStr, &depends)
		}
		seen := map[string]bool{}
		for _, depName := range depends {
			depName = strings.TrimSpace(depName)
			if depName == "" || seen[depName] {
				continue
			}
			seen[depName] = true

			depMod, ok := byName[depName]
			if !ok {
				continue
			}
			depApp := strings.TrimSpace(depMod.ApplicationStr)
			if depApp == "" || strings.EqualFold(depApp, "web") || depApp == app {
				continue
			}
			if appsSet[depApp] {
				appDeps[app][depApp] = true
			}
		}
	}

	for app := range appToModules {
		mods := appToModules[app]
		uniq := map[string]bool{}
		out := make([]string, 0, len(mods))
		for _, value := range mods {
			value = strings.TrimSpace(value)
			if value == "" || uniq[value] {
				continue
			}
			uniq[value] = true
			out = append(out, value)
		}
		sort.Strings(out)
		appToModules[app] = out
	}

	topo, cyclic := stableTopoSortApps(apps, appDeps)
	if cyclic {
		if runtimeScope.Logger() != nil {
			runtimeScope.Logger().Warn("dist manifest backend topo order fallback", "reason", "application_dependency_cycle", "apps", topo)
		}
		sort.Strings(topo)
	}

	manifest := distmanifest.DistManifestV2{
		SchemaVersion:     distmanifest.SchemaVersion,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		CompileBundleMode: strings.ToLower(strings.TrimSpace(compileBundleMode)),
		HasWeb:            webEnabled,
		BackendTopoOrder:  topo,
		Apps:              map[string]distmanifest.DistManifestApp{},
	}

	for _, app := range apps {
		depsOut := make([]string, 0, len(appDeps[app]))
		for dep := range appDeps[app] {
			depsOut = append(depsOut, dep)
		}
		sort.Strings(depsOut)
		manifest.Apps[app] = distmanifest.DistManifestApp{
			Deps: distmanifest.DistManifestAppDeps{Apps: depsOut},
			Dev:  distmanifest.DistManifestAppDev{Modules: appToModules[app]},
		}
	}

	data, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return xfmt.Errorf("marshal dist manifest: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return xfmt.Errorf("mkdir dist manifest parent: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return xfmt.Errorf("write dist manifest %s: %w", outPath, err)
	}
	return nil
}

func stableTopoSortApps(apps []string, deps map[string]map[string]bool) (order []string, cyclic bool) {
	apps = append([]string{}, apps...)
	sort.Strings(apps)
	if len(apps) <= 1 {
		return apps, false
	}

	indeg := map[string]int{}
	out := map[string][]string{}
	for _, app := range apps {
		indeg[app] = 0
		out[app] = nil
	}
	for app, depSet := range deps {
		for dep := range depSet {
			if _, ok := indeg[dep]; !ok {
				continue
			}
			if _, ok := indeg[app]; !ok {
				continue
			}
			indeg[app]++
			out[dep] = append(out[dep], app)
		}
	}
	for key := range out {
		sort.Strings(out[key])
	}

	ready := make([]string, 0, len(apps))
	for _, app := range apps {
		if indeg[app] == 0 {
			ready = append(ready, app)
		}
	}
	sort.Strings(ready)

	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		order = append(order, current)
		for _, next := range out[current] {
			indeg[next]--
			if indeg[next] == 0 {
				ready = append(ready, next)
			}
		}
		sort.Strings(ready)
	}

	if len(order) != len(apps) {
		return apps, true
	}
	return order, false
}
