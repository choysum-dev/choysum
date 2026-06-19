// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package discovery

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// ResolveTestApps resolves the user-supplied target (app name or "all") into
// a concrete list of module apps that have runnable tests under the requested scopes.
func ResolveTestApps(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if runtimeScope == nil || !runtimeOpts.hasConfig {
		return nil, xfmt.Errorf("scope is not initialized")
	}
	modulesPath := strings.TrimSpace(runtimeOpts.modulesPath)
	if modulesPath == "" {
		return nil, xfmt.Errorf("config missing modules_path")
	}
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return nil, xfmt.Errorf("missing app")
	}

	// Single app.
	if arg != "all" {
		appDir := filepath.Join(modulesPath, arg)
		if st, err := os.Stat(appDir); err != nil || !st.IsDir() {
			return nil, xfmt.Errorf(testsemantics.UnknownAppMessage(arg))
		}

		ok := false
		if runBE {
			has, err := HasAnyBackendTests(modulesPath, arg)
			if err != nil {
				return nil, err
			}
			ok = ok || has
		}
		if runFE {
			has, err := HasAnyFrontendTests(modulesPath, arg)
			if err != nil {
				return nil, err
			}
			ok = ok || has
		}
		if ok {
			return []string{arg}, nil
		}
		return nil, nil
	}

	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, xfmt.Errorf("read modules dir: %w", err)
	}

	apps := make([]string, 0)
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if shouldSkipAppName(name) {
			continue
		}

		ok := false
		if runBE {
			has, err := HasAnyBackendTests(modulesPath, name)
			if err != nil {
				return nil, err
			}
			ok = ok || has
		}
		if runFE {
			has, err := HasAnyFrontendTests(modulesPath, name)
			if err != nil {
				return nil, err
			}
			ok = ok || has
		}
		if ok {
			apps = append(apps, name)
		}
	}

	return apps, nil
}

func HasAnyBackendTests(modulesPath string, app string) (bool, error) {
	serviceDir := filepath.Join(modulesPath, app, "service")
	st, err := os.Stat(serviceDir)
	if err != nil || !st.IsDir() {
		return false, nil
	}
	return hasAnyTestFiles(serviceDir, func(name string) bool {
		return strings.HasSuffix(name, ".test.ts")
	})
}

func HasAnyFrontendTests(modulesPath string, app string) (bool, error) {
	webDir := filepath.Join(modulesPath, app, "web")
	st, err := os.Stat(webDir)
	if err != nil || !st.IsDir() {
		return false, nil
	}
	return hasAnyTestFiles(webDir, func(name string) bool {
		return strings.HasSuffix(name, ".test.ts") ||
			strings.HasSuffix(name, ".test.tsx") ||
			strings.HasSuffix(name, ".spec.ts") ||
			strings.HasSuffix(name, ".spec.tsx")
	})
}

func hasAnyTestFiles(root string, match func(name string) bool) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		_ = path
		if d.IsDir() {
			if shouldSkipTestScanDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if match(d.Name()) {
			found = true
			return errors.New("__found__")
		}
		return nil
	})
	if err != nil {
		if err.Error() == "__found__" {
			return true, nil
		}
		return false, err
	}
	return found, nil
}

func shouldSkipTestScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".choysum", "tmp":
		return true
	default:
		return false
	}
}

func shouldSkipAppName(name string) bool {
	switch name {
	case ".choysum", "tmp":
		return true
	default:
		return false
	}
}
