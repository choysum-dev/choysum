// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"os"
	"path/filepath"
	"strings"
)

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
