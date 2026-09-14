// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
)

// Matches @Migration / @Migration<...>({ ... phase: 'end'|Phase.End ... }),
// quoted keys ('phase'/"phase"), object spreads (...foo), or a non-literal
// options argument (@Migration(opts)). Uses a non-greedy span so nested
// `{ ... }` before `phase` still match. Identifier / spread / variable args
// fail open (false positives only cost a full RunPhase).
var endMigrationPhasePattern = regexp.MustCompile(`(?is)@Migration(?:\s*<[^>]*>)?\s*\(\s*(?:\{.*?(?:(?:\bphase\b|\[\s*['"]phase['"]\s*\]|['"]phase['"])\s*:\s*(?:['"]end['"]|[A-Za-z_$][\w$.]*)|\.\.\.)|[A-Za-z_$])`)

// moduleSourceDeclaresEndMigration reports whether module sources declare an
// @Migration with phase end. Used to O(1)-skip PhaseEnd without semantic build
// or executor load when no end migrations exist.
//
// Returns true (fail open) when sources cannot be inspected so runtime-bundle
// PhaseEnd migrations are still discovered via resolveScripts.
func moduleSourceDeclaresEndMigration(module *meta.Module) bool {
	if module == nil {
		return false
	}
	root := strings.TrimSpace(module.Path)
	if root == "" {
		return true
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return true
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Unreadable trees are treated as "maybe has end migrations" so we
			// never silently skip PhaseEnd on I/O failure.
			found = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "dist" || name == ".git" || name == "coverage" || name == "demo" || name == "__tests__" {
				return filepath.SkipDir
			}
			return nil
		}
		base := strings.ToLower(d.Name())
		if strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") ||
			strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".spec.ts") ||
			strings.HasSuffix(base, ".spec.tsx") || strings.HasSuffix(base, ".spec.js") {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts":
		default:
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			found = true
			return filepath.SkipAll
		}
		if endMigrationPhasePattern.Match(raw) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
