// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
)

// StableTopoSortApplicationsForTargets returns a stable topological order for target apps.
//
// Rules:
// - Only dependencies among targets are considered (deps outside targets do not constrain order).
// - Within the same level, order is alphabetical.
// - If a cycle exists, it returns alphabetical order and cyclic=true.
func StableTopoSortApplicationsForTargets(installedModules []meta.Module, targets []string) (order []string, cyclic bool) {
	set := map[string]bool{}
	for _, t := range targets {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		set[t] = true
	}
	apps := make([]string, 0, len(set))
	for app := range set {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	if len(apps) <= 1 {
		return apps, false
	}

	// Build module lookup by name.
	byName := map[string]meta.Module{}
	for _, m := range installedModules {
		if strings.TrimSpace(m.Name) == "" {
			continue
		}
		byName[m.Name] = m
	}

	// Build app->deps (edges: dep -> app).
	deps := map[string]map[string]bool{}
	for _, app := range apps {
		deps[app] = map[string]bool{}
	}
	for _, m := range installedModules {
		app := strings.TrimSpace(m.ApplicationStr)
		if app == "" || !set[app] {
			continue
		}
		var dependsArr []string
		if len(m.DependsStr) > 0 {
			_ = json.Unmarshal(m.DependsStr, &dependsArr)
		}
		for _, depName := range dependsArr {
			depName = strings.TrimSpace(depName)
			if depName == "" {
				continue
			}
			depMod, ok := byName[depName]
			if !ok {
				continue
			}
			depApp := strings.TrimSpace(depMod.ApplicationStr)
			if depApp == "" || !set[depApp] || depApp == app {
				continue
			}
			deps[app][depApp] = true
		}
	}

	// Kahn with stable queue.
	indeg := map[string]int{}
	out := map[string][]string{}
	for _, app := range apps {
		indeg[app] = 0
		out[app] = nil
	}
	for app, depSet := range deps {
		for dep := range depSet {
			// dep -> app
			indeg[app]++
			out[dep] = append(out[dep], app)
		}
	}
	for k := range out {
		sort.Strings(out[k])
	}

	ready := make([]string, 0, len(apps))
	for _, app := range apps {
		if indeg[app] == 0 {
			ready = append(ready, app)
		}
	}
	sort.Strings(ready)

	for len(ready) > 0 {
		cur := ready[0]
		ready = ready[1:]
		order = append(order, cur)
		for _, nxt := range out[cur] {
			indeg[nxt]--
			if indeg[nxt] == 0 {
				ready = append(ready, nxt)
			}
		}
		sort.Strings(ready)
	}

	if len(order) != len(apps) {
		// Cycle: deterministic fallback.
		sort.Strings(apps)
		return apps, true
	}
	return order, false
}
