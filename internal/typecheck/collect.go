// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CollectRootFiles returns absolute slash-normalized TypeScript roots for scope.
func CollectRootFiles(modulesPath, app string, scope Scope) ([]string, error) {
	app = strings.TrimSpace(app)
	if strings.TrimSpace(modulesPath) == "" {
		return nil, ErrModulesPathRequired
	}
	if app == "" {
		return nil, ErrAppRequired
	}
	modulesPath = filepath.Clean(modulesPath)
	appRoot := filepath.Join(modulesPath, app)
	st, err := stat(appRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoRootFiles
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, ErrNoRootFiles
	}

	var files []string
	seen := make(map[string]struct{})
	add := func(path string) {
		abs, err := absPath(path)
		if err != nil {
			abs = path
		}
		key := filepath.ToSlash(abs)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		files = append(files, key)
	}

	switch scope {
	case ScopeService:
		entries, err := readDir(appRoot)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			lower := strings.ToLower(name)
			switch {
			case strings.HasSuffix(lower, ".d.ts"):
				add(filepath.Join(appRoot, name))
			case strings.HasSuffix(lower, ".ts"):
				if shouldSkipTSFileName(name) {
					continue
				}
				add(filepath.Join(appRoot, name))
			}
		}

		serviceRoot := filepath.Join(appRoot, "service")
		st, err := stat(serviceRoot)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else if st.IsDir() {
			err := walkDir(serviceRoot, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if shouldSkipScanDir(d.Name()) {
						return fs.SkipDir
					}
					return nil
				}
				name := d.Name()
				lower := strings.ToLower(name)
				switch {
				case strings.HasSuffix(lower, ".d.ts"):
					add(path)
				case strings.HasSuffix(lower, ".ts"):
					if shouldSkipTSFileName(name) {
						return nil
					}
					add(path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.New("typecheck: unsupported scope")
	}

	if len(files) == 0 {
		return nil, ErrNoRootFiles
	}
	return files, nil
}

// Test hooks for hard-to-trigger filesystem failures.
var (
	absPath = filepath.Abs
	readDir = os.ReadDir
	walkDir = filepath.WalkDir
	stat    = os.Stat
)

func shouldSkipScanDir(name string) bool {
	switch strings.ToLower(name) {
	case "node_modules", "dist", ".choysum", "tmp", "tests", "__tests__", "coverage", "e2e":
		return true
	default:
		return false
	}
}

func shouldSkipTSFileName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".d.ts") {
		return false
	}
	if strings.HasSuffix(lower, ".test.ts") || strings.HasSuffix(lower, ".spec.ts") {
		return true
	}
	if strings.Contains(lower, ".gen.") {
		return true
	}
	return false
}
