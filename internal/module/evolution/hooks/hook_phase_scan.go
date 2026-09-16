// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/meta"
)

// Canonical @Hook* decorator forms used by module sources. Aliased / dynamic
// registration is not detected (authors should use the canonical decorator names).
// Generic args use non-greedy [^\n]*? so nested brackets and parentheses inside
// type arguments (e.g. <Map<string, number>>, <() => void>) still match.
var hookPhaseDecoratorPatterns = map[Phase]*regexp.Regexp{
	PhasePreInit:       regexp.MustCompile(`@HookPreInit\s*(?:<[^\n]*?>)?\s*\(`),
	PhasePostInit:      regexp.MustCompile(`@HookPostInit\s*(?:<[^\n]*?>)?\s*\(`),
	PhasePreUpgrade:    regexp.MustCompile(`@HookPreUpgrade\s*(?:<[^\n]*?>)?\s*\(`),
	PhasePostUpgrade:   regexp.MustCompile(`@HookPostUpgrade\s*(?:<[^\n]*?>)?\s*\(`),
	PhasePreUninstall:  regexp.MustCompile(`@HookPreUninstall\s*(?:<[^\n]*?>)?\s*\(`),
	PhasePostUninstall: regexp.MustCompile(`@HookPostUninstall\s*(?:<[^\n]*?>)?\s*\(`),
}

var hookMarker = []byte("@Hook")

func isIgnoredHookScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".git", "coverage", "demo", "__tests__":
		return true
	default:
		return false
	}
}

func isHookScanTestFile(base string) bool {
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return strings.HasSuffix(stem, ".test") || strings.HasSuffix(stem, ".spec")
}

// Overridable in tests for fail-open branches after EvalSymlinks.
var hookPhaseScanStat = os.Stat

// Per resolved module path: one tree walk serves all lifecycle phases.
var hookPhaseScanMemo sync.Map // map[string]*hookPhaseScanCache

type hookPhaseScanCache struct {
	indeterminate bool
	phases        map[Phase]struct{}
}

func (c *hookPhaseScanCache) has(phase Phase) bool {
	if c == nil || c.indeterminate {
		return true
	}
	_, ok := c.phases[phase]
	return ok
}

// moduleSourceDeclaresHookPhase reports whether module sources declare a
// canonical @Hook* decorator for phase. Used to skip RunPhase without Bundle /
// executor Reload when the registry would be empty.
//
// Returns true (fail open) when sources cannot be inspected — missing/blank Path,
// Stat/EvalSymlinks errors, walk I/O errors, or an unexpected symlink — so
// required phases still load JS rather than silently skipping real hooks.
// A nil module returns false (no sources to run); that is not an inspection failure.
func moduleSourceDeclaresHookPhase(module *meta.Module, phase Phase) bool {
	if module == nil {
		return false
	}
	if _, ok := hookPhaseDecoratorPatterns[phase]; !ok {
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
	info, err := hookPhaseScanStat(resolved)
	if err != nil {
		return true
	}
	if !info.IsDir() {
		return true
	}
	if cached, ok := hookPhaseScanMemo.Load(resolved); ok {
		return cached.(*hookPhaseScanCache).has(phase)
	}
	cache := scanModuleHookPhases(resolved)
	hookPhaseScanMemo.Store(resolved, cache)
	return cache.has(phase)
}

// scanModuleHookPhases walks module sources once and records every phase whose
// canonical decorator appears.
func scanModuleHookPhases(root string) *hookPhaseScanCache {
	cache := &hookPhaseScanCache{phases: make(map[Phase]struct{})}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			cache.indeterminate = true
			return filepath.SkipAll
		}
		// WalkDir does not follow symlinks. Skip known-irrelevant names first
		// (e.g. pnpm symlinked node_modules) so they do not force fail-open.
		if d.Type()&os.ModeSymlink != 0 {
			if isIgnoredHookScanDir(d.Name()) {
				return nil
			}
			cache.indeterminate = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			// Do not SkipDir the module root when its basename is an ignore name
			// (e.g. a module checked out as .../demo).
			if path != root && isIgnoredHookScanDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		base := strings.ToLower(d.Name())
		if isHookScanTestFile(base) {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts":
		default:
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			cache.indeterminate = true
			return filepath.SkipAll
		}
		// Every pattern requires "@Hook"; skip the comment-strip copy otherwise.
		if !bytes.Contains(raw, hookMarker) {
			return nil
		}
		stripped := stripLineComments(raw)
		for phase, pattern := range hookPhaseDecoratorPatterns {
			if _, already := cache.phases[phase]; already {
				continue
			}
			if pattern.Match(stripped) {
				cache.phases[phase] = struct{}{}
			}
		}
		return nil
	})
	return cache
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
