// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CollectRootFiles returns absolute slash-normalized TypeScript roots for scope.
func CollectRootFiles(ctx context.Context, modulesPath, app string, scope Scope) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	case ScopeService, ScopeNoVue, ScopeAll:
		if err := collectAppRootTS(ctx, appRoot, add); err != nil {
			return nil, err
		}
		if err := walkTSTree(ctx, filepath.Join(appRoot, "service"), add, false, false); err != nil {
			return nil, err
		}
		if scope == ScopeNoVue || scope == ScopeAll {
			allowVue := scope == ScopeAll
			if err := walkTSTree(ctx, filepath.Join(appRoot, "web"), add, true, allowVue); err != nil {
				return nil, err
			}
		}
	default:
		return nil, ErrUnsupportedScope
	}

	if len(files) == 0 {
		return nil, ErrNoRootFiles
	}
	return files, nil
}

func collectAppRootTS(ctx context.Context, appRoot string, add func(string)) error {
	entries, err := readDir(appRoot)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isCollectableTSName(name, false, false) || shouldSkipTSFileName(name) {
			continue
		}
		add(filepath.Join(appRoot, name))
	}
	return nil
}

// walkTSTree walks root for .ts / .d.ts (and optionally .tsx / .vue). Missing root is OK.
func walkTSTree(ctx context.Context, root string, add func(string), allowTSX, allowVue bool) error {
	st, err := stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !st.IsDir() {
		return nil
	}
	return walkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipScanDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !isCollectableTSName(name, allowTSX, allowVue) || shouldSkipTSFileName(name) {
			return nil
		}
		add(path)
		return nil
	})
}

func isCollectableTSName(name string, allowTSX, allowVue bool) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".vue") {
		return allowVue
	}
	if allowTSX && strings.HasSuffix(lower, ".tsx") {
		return true
	}
	return strings.HasSuffix(lower, ".ts")
}

// Test hooks for hard-to-trigger filesystem failures.
var (
	absPath = filepath.Abs
	readDir = os.ReadDir
	walkDir = filepath.WalkDir
	stat    = os.Stat
)

func shouldSkipScanDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "node_modules", "dist", "tmp", "test", "tests", "__tests__", "coverage", "e2e":
		return true
	default:
		return false
	}
}

func shouldSkipTSFileName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".test.ts") || strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".test.tsx") || strings.HasSuffix(lower, ".spec.tsx") ||
		strings.HasSuffix(lower, ".test.d.ts") || strings.HasSuffix(lower, ".spec.d.ts") ||
		strings.HasSuffix(lower, ".test.vue") || strings.HasSuffix(lower, ".spec.vue") {
		return true
	}
	if strings.Contains(lower, ".gen.") {
		return true
	}
	return false
}
