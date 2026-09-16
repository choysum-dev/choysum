// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
)

// Canonical @Hook* decorator forms used by module sources. Aliased / dynamic
// registration is not detected (authors should use the canonical decorator names).
// Generic args use [^()\n]* so nested brackets like <Map<string, number>> still match.
var hookPhaseDecoratorPatterns = map[Phase]*regexp.Regexp{
	PhasePreInit:       regexp.MustCompile(`@HookPreInit\s*(?:<[^()\n]*>)?\s*\(`),
	PhasePostInit:      regexp.MustCompile(`@HookPostInit\s*(?:<[^()\n]*>)?\s*\(`),
	PhasePreUpgrade:    regexp.MustCompile(`@HookPreUpgrade\s*(?:<[^()\n]*>)?\s*\(`),
	PhasePostUpgrade:   regexp.MustCompile(`@HookPostUpgrade\s*(?:<[^()\n]*>)?\s*\(`),
	PhasePreUninstall:  regexp.MustCompile(`@HookPreUninstall\s*(?:<[^()\n]*>)?\s*\(`),
	PhasePostUninstall: regexp.MustCompile(`@HookPostUninstall\s*(?:<[^()\n]*>)?\s*\(`),
}

func isIgnoredHookScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".git", "coverage", "demo", "__tests__":
		return true
	default:
		return false
	}
}

// moduleSourceDeclaresHookPhase reports whether module sources declare a
// canonical @Hook* decorator for phase. Used to skip RunPhase without Bundle /
// executor Reload when the registry would be empty.
//
// Returns true (fail open) when sources cannot be inspected — missing Path,
// Stat/EvalSymlinks errors, walk I/O errors, or an unexpected symlink — so
// required phases still load JS rather than silently skipping real hooks.
func moduleSourceDeclaresHookPhase(module *meta.Module, phase Phase) bool {
	if module == nil {
		return false
	}
	pattern := hookPhaseDecoratorPatterns[phase]
	if pattern == nil {
		return true
	}
	root := strings.TrimSpace(module.Path)
	if root == "" {
		return true
	}
	// os.Stat follows symlinks but WalkDir Lstats the root, so a symlinked
	// module dir would otherwise look like a non-dir and never be scanned.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return true
	}
	root = resolved
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return true
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			found = true
			return filepath.SkipAll
		}
		// WalkDir does not follow symlinks. Skip known-irrelevant names first
		// (e.g. pnpm symlinked node_modules) so they do not force fail-open.
		if d.Type()&os.ModeSymlink != 0 {
			if isIgnoredHookScanDir(d.Name()) {
				return nil
			}
			found = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			if isIgnoredHookScanDir(d.Name()) {
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
		if pattern.Match(stripLineComments(raw)) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// stripLineComments drops //... tails per line so disabled decorators like
// `// @HookPostInit()` do not keep the phase warm. Does not attempt string-
// aware or block-comment parsing (false positives only cost a full RunPhase).
func stripLineComments(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	out := make([]byte, 0, len(raw))
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\n' {
			continue
		}
		out = append(out, stripLineCommentSegment(raw[start:i])...)
		out = append(out, '\n')
		start = i + 1
	}
	if start < len(raw) {
		out = append(out, stripLineCommentSegment(raw[start:])...)
	}
	return out
}

func stripLineCommentSegment(line []byte) []byte {
	if idx := indexLineComment(line); idx >= 0 {
		return line[:idx]
	}
	return line
}

func indexLineComment(line []byte) int {
	for i := 0; i+1 < len(line); i++ {
		if line[i] == '/' && line[i+1] == '/' {
			return i
		}
	}
	return -1
}
