// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/distmanifest"
)

func resolveServeTargets(distRoot string, configBundleMode string, logger *slog.Logger, manifest *distmanifest.DistManifestV2, serviceNames []string) (compileBundleMode string, serveTargets []string, err error) {
	if logger == nil {
		logger = slog.Default()
	}

	compileBundleMode = configBundleMode
	if manifest != nil && strings.TrimSpace(manifest.CompileBundleMode) != "" {
		compileBundleMode = manifest.CompileBundleMode
	}

	explicitArgs := len(serviceNames) > 0

	if len(serviceNames) == 0 {
		if manifest != nil && len(manifest.BackendTopoOrder) > 0 {
			serviceNames = append([]string{}, manifest.BackendTopoOrder...)
			if manifest.HasWeb {
				if st, err := os.Stat(filepath.Join(distRoot, "web")); err == nil && st.IsDir() {
					serviceNames = append(serviceNames, "web")
				}
			}
		} else {
			resolved, err := resolveDefaultTargetsFromDist(compileBundleMode, distRoot)
			if err != nil {
				return "", nil, err
			}
			serviceNames = resolved
		}
	}

	needsWeb := false
	backendSet := map[string]bool{}
	backend := make([]string, 0, len(serviceNames))
	for _, name := range serviceNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if name == "web" {
			needsWeb = true
			continue
		}
		if backendSet[name] {
			continue
		}
		backendSet[name] = true
		backend = append(backend, name)
	}

	if manifest != nil && len(manifest.BackendTopoOrder) > 0 {
		ordered, unknown := orderServeTargetsByTopo(manifest.BackendTopoOrder, backend)
		if len(unknown) > 0 {
			logger.Warn("serve targets order fallback", "reason", "requested_apps_missing_from_manifest", "order", "alphabetical", "unknown", unknown)
		}
		backend = ordered
	} else {
		if len(backend) > 1 {
			reason := "manifest_topo_order_unavailable"
			if manifest == nil {
				reason = "dist_manifest_missing"
			}
			logger.Warn("serve targets order fallback", "reason", reason, "order", "alphabetical", "apps", backend)
		}
		sort.Strings(backend)
	}

	if explicitArgs && manifest != nil {
		missing := computeMissingAppDeps(manifest, backend)
		if len(missing) > 0 {
			logger.Warn(
				"serve targets dependency warning",
				"reason", "requested_apps_missing_dependencies",
				"requested", backend,
				"missing", missing,
			)
		}
	}

	serveTargets = make([]string, 0, len(backend)+1)
	serveTargets = append(serveTargets, backend...)
	if needsWeb {
		serveTargets = append(serveTargets, "web")
	}
	return compileBundleMode, serveTargets, nil
}
