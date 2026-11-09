// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"strings"

	"golang.org/x/mod/semver"
)

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	if semver.IsValid("v" + v) {
		return "v" + v
	}
	return v
}

// displayVersion returns the npm/package.json form (no leading "v") for catalog UI fields.
// Internal Module / snapshot rows may still store the v-prefixed SemVer used by Go tooling.
func displayVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		return v
	}
	rest := strings.TrimPrefix(v, "v")
	if semver.IsValid(v) || semver.IsValid("v"+rest) {
		return rest
	}
	return v
}

// compareVersion returns -1/0/1 like strings.Compare; prefers semver when both valid.
func compareVersion(a, b string) int {
	a = normalizeVersion(a)
	b = normalizeVersion(b)
	aValid := semver.IsValid(a)
	bValid := semver.IsValid(b)
	if aValid && bValid {
		return semver.Compare(a, b)
	}
	return strings.Compare(a, b)
}

func versionNewer(candidate, baseline string) bool {
	if strings.TrimSpace(candidate) == "" {
		return false
	}
	if strings.TrimSpace(baseline) == "" {
		return true
	}
	return compareVersion(candidate, baseline) > 0
}
