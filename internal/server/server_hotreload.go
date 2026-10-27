// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/oerrors"
)

// choyTailwindGeneratedCSSName matches the kit Tailwind emit basename under
// <default_choysum_path>/generated/styles. Kept in sync with web build.
const choyTailwindGeneratedCSSName = "choy-tailwind.generated.css"

// hotreloadLeaseRetryBase is the first backoff after module_management LEASE_CONFLICT.
// CLI upgrade / overlapping hotreload hold the lease briefly; waiting the full
// 60s TTL (retry_after_ms) would strand deferred module saves.
var hotreloadLeaseRetryBase = 500 * time.Millisecond

// hotreloadLeaseRetryMax caps automatic requeues after LEASE_CONFLICT.
// Zero disables retries (tests that only assert the defer path).
var hotreloadLeaseRetryMax = 8

// shouldIgnoreWatchPath reports paths that must not trigger module Upgrade.
// Kit Tailwind atomic writes under generated/styles (and legacy modules kit
// styles leftovers) run during install/web_build; upgrading on those events
// races the module_management lease and must be skipped.
// Unit/FE test sources are also ignored: they do not need a module Upgrade and
// often fire right after a product-file save, colliding with the same lease.
func shouldIgnoreWatchPath(path string) bool {
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)
	switch {
	case strings.HasSuffix(lowerBase, ".test.ts"),
		strings.HasSuffix(lowerBase, ".test.tsx"),
		strings.HasSuffix(lowerBase, ".test.js"),
		strings.HasSuffix(lowerBase, ".test.jsx"),
		strings.HasSuffix(lowerBase, ".spec.ts"),
		strings.HasSuffix(lowerBase, ".spec.tsx"),
		strings.HasSuffix(lowerBase, ".spec.js"),
		strings.HasSuffix(lowerBase, ".spec.jsx"):
		return true
	}

	return isChoyTailwindGeneratedWatchPath(path)
}

func isChoyTailwindGeneratedWatchPath(path string) bool {
	base := filepath.Base(path)
	isTmp := strings.HasPrefix(base, ".choy-tailwind-") && strings.HasSuffix(base, ".tmp")
	isCSS := base == choyTailwindGeneratedCSSName
	if !isTmp && !isCSS {
		return false
	}
	slash := filepath.ToSlash(filepath.Clean(path))
	// Current emit: .../generated/styles/choy-tailwind.generated.css (+ atomic tmp)
	if strings.Contains(slash, "/generated/styles/") {
		return true
	}
	// Legacy in-module emit (and leftover files during cutover).
	const kitStyles = "modules/web/web/styles"
	dir := filepath.ToSlash(filepath.Dir(path))
	return dir == kitStyles || strings.HasSuffix(dir, "/"+kitStyles)
}

// isModuleLeaseConflict reports meta.lock LEASE_CONFLICT through wrapped errors.
func isModuleLeaseConflict(err error) bool {
	return oerrors.Is(err, "meta.lock", "LEASE_CONFLICT")
}

// resolveWatchPath canonicalizes symlinked watch roots and event paths while
// still tolerating missing leaf paths from remove/rename events.
func resolveWatchPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		return resolvedPath, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	current := absPath
	missingSegments := []string{}
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missingSegments = append(missingSegments, filepath.Base(current))
		current = parent

		resolvedParent, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr == nil {
			for i := len(missingSegments) - 1; i >= 0; i-- {
				resolvedParent = filepath.Join(resolvedParent, missingSegments[i])
			}
			return resolvedParent, nil
		}
		if !os.IsNotExist(resolveErr) {
			return "", resolveErr
		}
	}
}

func isWatchedPath(moduleDir string, file string) (bool, error) {
	relPath, err := filepath.Rel(moduleDir, file)
	if err != nil {
		return false, err
	}
	if relPath == "." {
		return true, nil
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return false, nil
	}
	return true, nil
}

// resolveWatchModule returns the module name that owns the given resolved
// file path, or "" when no registered watch target contains the file.
// Callers must pass an already-resolved path (e.g. from enqueueWatchEvent).
func (s *GRPCWebServer) resolveWatchModule(resolvedFile string) string {
	if s == nil {
		return ""
	}
	for _, target := range s.hotreload.watchTargetsSnapshot() {
		contained, err := isWatchedPath(target.root, resolvedFile)
		if err != nil {
			continue
		}
		if contained {
			return target.moduleName
		}
	}
	return ""
}
