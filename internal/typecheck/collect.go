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
	modulesPath = filepath.Clean(modulesPath)
	app = strings.TrimSpace(app)
	if modulesPath == "" || app == "" {
		return nil, ErrAppRequired
	}
	appRoot := filepath.Join(modulesPath, app)
	st, err := os.Stat(appRoot)
	if err != nil || !st.IsDir() {
		return nil, ErrNoRootFiles
	}

	var files []string
	seen := make(map[string]struct{})
	add := func(path string) {
		abs, err := filepath.Abs(path)
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
		entries, err := os.ReadDir(appRoot)
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
		if st, err := os.Stat(serviceRoot); err == nil && st.IsDir() {
			err := filepath.WalkDir(serviceRoot, func(path string, d fs.DirEntry, err error) error {
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
