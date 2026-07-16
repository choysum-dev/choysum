// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"path/filepath"
	"strings"
)

// FormatScope builds Scope = path[@location], matching modules/core/service/i18n/scope.ts.
func FormatScope(path string, location string) string {
	p := strings.TrimSpace(path)
	loc := strings.TrimSpace(location)
	if p == "" {
		return loc
	}
	if loc == "" {
		return p
	}
	return p + "@" + loc
}

// ResolveI18nScope mirrors runtime resolveI18nScope:
// explicit scope → withI18nScope stack top → path[@location].
func ResolveI18nScope(manualScope string, scopeStack []string, path string, location string) string {
	if manual := strings.TrimSpace(manualScope); manual != "" {
		return manual
	}
	if len(scopeStack) > 0 {
		return scopeStack[len(scopeStack)-1]
	}
	return FormatScope(path, location)
}

// ScopePathFromRelPath converts a module-relative file path into the auto Scope path
// (no extension, forward slashes), e.g. web/pages/Login.vue → web/pages/Login.
func ScopePathFromRelPath(relPath string) string {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	relPath = strings.TrimPrefix(relPath, "./")
	ext := filepath.Ext(relPath)
	if ext != "" {
		relPath = strings.TrimSuffix(relPath, ext)
	}
	return relPath
}
