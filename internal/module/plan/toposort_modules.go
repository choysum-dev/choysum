// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
)

// StableTopoSortModules returns a stable topological order for the given modules
// using DependsStr edges among that set only (deps outside the set are ignored).
// Within the same level, order is alphabetical by module name.
// If a cycle exists, it returns alphabetical order and cyclic=true.
func StableTopoSortModules(modules []meta.Module) (order []meta.Module, cyclic bool) {
	byName := map[string]meta.Module{}
	names := make([]string, 0, len(modules))
	for _, m := range modules {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		if _, exists := byName[name]; exists {
			continue
		}
		byName[name] = m
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) <= 1 {
		out := make([]meta.Module, 0, len(names))
		for _, name := range names {
			out = append(out, byName[name])
		}
		return out, false
	}

	deps := map[string]map[string]bool{}
	for _, name := range names {
		deps[name] = map[string]bool{}
	}
	for _, name := range names {
		m := byName[name]
		var dependsArr []string
		if len(m.DependsStr) > 0 {
			_ = json.Unmarshal(m.DependsStr, &dependsArr)
		}
		for _, depName := range dependsArr {
			depName = strings.TrimSpace(depName)
			if depName == "" || depName == name {
				continue
			}
			if _, ok := byName[depName]; !ok {
				continue
			}
			deps[name][depName] = true
		}
	}

	indeg := map[string]int{}
	outEdges := map[string][]string{}
	for _, name := range names {
		indeg[name] = 0
		outEdges[name] = nil
	}
	for name, depSet := range deps {
		for dep := range depSet {
			indeg[name]++
			outEdges[dep] = append(outEdges[dep], name)
		}
	}
	for k := range outEdges {
		sort.Strings(outEdges[k])
	}

	ready := make([]string, 0, len(names))
	for _, name := range names {
		if indeg[name] == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	orderedNames := make([]string, 0, len(names))
	for len(ready) > 0 {
		cur := ready[0]
		ready = ready[1:]
		orderedNames = append(orderedNames, cur)
		for _, nxt := range outEdges[cur] {
			indeg[nxt]--
			if indeg[nxt] == 0 {
				ready = append(ready, nxt)
			}
		}
		sort.Strings(ready)
	}

	if len(orderedNames) != len(names) {
		sort.Strings(names)
		orderedNames = names
		cyclic = true
	}
	order = make([]meta.Module, 0, len(orderedNames))
	for _, name := range orderedNames {
		order = append(order, byName[name])
	}
	return order, cyclic
}
