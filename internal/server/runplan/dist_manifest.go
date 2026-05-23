// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	xfmt "golang.org/x/exp/errors/fmt"
)

func LoadDistManifest(distRoot string) (*distmanifest.DistManifestV2, error) {
	path := filepath.Join(distRoot, distmanifest.DistManifestFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, xfmt.Errorf("read dist manifest %s: %w", path, err)
	}

	var manifest distmanifest.DistManifestV2
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, xfmt.Errorf("parse dist manifest %s: %w", path, err)
	}
	if manifest.SchemaVersion != distmanifest.SchemaVersion {
		return nil, xfmt.Errorf("unsupported dist manifest schemaVersion: %d", manifest.SchemaVersion)
	}

	manifest.CompileBundleMode = strings.ToLower(strings.TrimSpace(manifest.CompileBundleMode))
	if manifest.CompileBundleMode == "" {
		return nil, xfmt.Errorf("dist manifest compileBundleMode is required")
	}
	if manifest.Apps == nil {
		manifest.Apps = map[string]distmanifest.DistManifestApp{}
	}
	manifest.BackendTopoOrder = normalizeStringList(manifest.BackendTopoOrder)

	return &manifest, nil
}

func normalizeStringList(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// orderServeTargetsByTopo filters and orders serve targets according to topoOrder.
// Targets not present in topoOrder are appended in alphabetical order.
func orderServeTargetsByTopo(topoOrder []string, targets []string) (ordered []string, unknown []string) {
	set := map[string]bool{}
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		set[target] = true
	}

	for _, app := range topoOrder {
		if set[app] {
			ordered = append(ordered, app)
			delete(set, app)
		}
	}

	for app := range set {
		unknown = append(unknown, app)
	}
	sort.Strings(unknown)
	ordered = append(ordered, unknown...)
	return ordered, unknown
}

// computeMissingAppDeps returns Closure(requested) - requested using manifest app deps.
// The result is alphabetical and stable.
func computeMissingAppDeps(manifest *distmanifest.DistManifestV2, requested []string) []string {
	if manifest == nil {
		return nil
	}

	requestedSet := map[string]bool{}
	queue := make([]string, 0, len(requested))
	for _, app := range requested {
		app = strings.TrimSpace(app)
		if app == "" || requestedSet[app] {
			continue
		}
		requestedSet[app] = true
		queue = append(queue, app)
	}
	closure := map[string]bool{}
	for app := range requestedSet {
		closure[app] = true
	}

	for index := 0; index < len(queue); index++ {
		app := queue[index]
		info, ok := manifest.Apps[app]
		if !ok {
			continue
		}
		for _, dep := range info.Deps.Apps {
			dep = strings.TrimSpace(dep)
			if dep == "" || closure[dep] {
				continue
			}
			closure[dep] = true
			queue = append(queue, dep)
		}
	}

	missing := make([]string, 0)
	for dep := range closure {
		if !requestedSet[dep] {
			missing = append(missing, dep)
		}
	}
	sort.Strings(missing)
	return missing
}
