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

// Matches @Migration({ ... phase: 'end' ... }) / phase: "end" in module sources.
// Intentionally conservative: false positives only cost a full RunPhase; false
// negatives would skip real end migrations.
var endMigrationPhasePattern = regexp.MustCompile(`(?i)@Migration\s*\(\s*\{[^}]*\bphase\s*:\s*['"]end['"]`)

// moduleSourceDeclaresEndMigration reports whether module sources declare an
// @Migration with phase end. Used to O(1)-skip PhaseEnd without semantic build
// or executor load when no end migrations exist.
func moduleSourceDeclaresEndMigration(module *meta.Module) bool {
	if module == nil {
		return false
	}
	root := strings.TrimSpace(module.Path)
	if root == "" {
		return false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || found {
			return walkErr
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
			return nil
		}
		if endMigrationPhasePattern.Match(raw) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
