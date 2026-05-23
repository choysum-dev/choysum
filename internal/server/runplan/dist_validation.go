// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	xfmt "golang.org/x/exp/errors/fmt"
)

// ValidateDistForTargets enforces the frozen runtime rules in docs/backend_bundle_mode_merged.md.
// It is intentionally a pure filesystem check so unit tests can cover error classification.
func ValidateDistForTargets(bundleMode string, distRoot string, targets []string) error {
	mode := strings.ToLower(strings.TrimSpace(bundleMode))
	if mode == "" {
		mode = "bundle"
	}

	backendTargets := make([]string, 0, len(targets))
	needsWeb := false
	for _, target := range targets {
		name := strings.TrimSpace(target)
		if name == "" {
			continue
		}
		if name == "web" {
			needsWeb = true
			continue
		}
		backendTargets = append(backendTargets, name)
	}

	if needsWeb {
		webDir := filepath.Join(distRoot, "web")
		if st, err := os.Stat(webDir); err != nil || !st.IsDir() {
			return xfmt.Errorf("web dist missing: %s", webDir)
		}
	}

	if len(backendTargets) == 0 {
		return nil
	}

	switch mode {
	case "bundle":
		bundlesDir := filepath.Join(distRoot, "bundles")
		st, err := os.Stat(bundlesDir)
		if err != nil || !st.IsDir() {
			return xfmt.Errorf("bundles dir missing: %s", bundlesDir)
		}
		bundlesIndex := filepath.Join(bundlesDir, "index.js")
		if st, err := os.Stat(bundlesIndex); err != nil || st.IsDir() {
			return xfmt.Errorf("bundles index missing: %s", bundlesIndex)
		}
		for _, app := range backendTargets {
			protoDir := config.APIAppProtoDir(distRoot, app)
			if st, err := os.Stat(protoDir); err != nil || !st.IsDir() {
				return xfmt.Errorf("api proto assets missing: %s", protoDir)
			}
		}
		return nil
	case "application":
		appsDir := filepath.Join(distRoot, "apps")
		for _, app := range backendTargets {
			appDir := filepath.Join(appsDir, app)
			indexJS := filepath.Join(appDir, "index.js")
			if st, err := os.Stat(indexJS); err != nil || st.IsDir() {
				return xfmt.Errorf("app index missing: %s", indexJS)
			}
			assetsDir := filepath.Join(appDir, "assets")
			if st, err := os.Stat(assetsDir); err != nil || !st.IsDir() {
				return xfmt.Errorf("app proto assets missing: %s", assetsDir)
			}
			hasAny := false
			_ = filepath.WalkDir(assetsDir, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				hasAny = true
				return fs.SkipAll
			})
			if !hasAny {
				return xfmt.Errorf("app proto assets missing: %s", assetsDir)
			}
		}
		return nil
	default:
		return xfmt.Errorf("invalid compile.bundleMode: %q (allowed: %q | %q)", bundleMode, "bundle", "application")
	}
}

func resolveDefaultTargetsFromDist(bundleMode string, distRoot string) ([]string, error) {
	mode := strings.ToLower(strings.TrimSpace(bundleMode))
	if mode == "" {
		mode = "bundle"
	}

	backend := make([]string, 0)
	switch mode {
	case "bundle":
		apiRoot := config.APIRootFromDist(distRoot)
		ents, err := os.ReadDir(apiRoot)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				break
			}
			return nil, xfmt.Errorf("read api root dir: %w", err)
		}
		for _, entry := range ents {
			if !entry.IsDir() {
				continue
			}
			protoDir := filepath.Join(apiRoot, entry.Name(), "proto")
			if st, err := os.Stat(protoDir); err == nil && st.IsDir() {
				backend = append(backend, entry.Name())
			}
		}
	case "application":
		appsDir := filepath.Join(distRoot, "apps")
		ents, err := os.ReadDir(appsDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				break
			}
			return nil, xfmt.Errorf("read apps dist dir: %w", err)
		}
		for _, entry := range ents {
			if entry.IsDir() {
				backend = append(backend, entry.Name())
			}
		}
	default:
		return nil, xfmt.Errorf("invalid compile.bundleMode: %q (allowed: %q | %q)", bundleMode, "bundle", "application")
	}

	sort.Strings(backend)

	out := make([]string, 0, len(backend)+1)
	out = append(out, backend...)
	if st, err := os.Stat(filepath.Join(distRoot, "web")); err == nil && st.IsDir() {
		out = append(out, "web")
	}
	return out, nil
}
