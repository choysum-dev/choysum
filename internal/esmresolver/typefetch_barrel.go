// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"regexp"
	"strings"
)

// Huge icon barrels (e.g. @vicons/material) rewrite to thousands of
// `export { default as X } from "./leaf.d.ts"` lines. TypeScript then opens
// every leaf when resolving the package entry, exhausting IDE file descriptors.
const largeDefaultAsReexportThreshold = 100

var defaultAsLocalReexportPattern = regexp.MustCompile(
	`(?m)^\s*export\s*\{\s*default\s+as\s+([A-Za-z_$][\w$]*)\s*\}\s*from\s*["'](\./[^"']+)["']\s*;?\s*$`,
)

// collapseLargeDefaultAsReexportBarrel rewrites a declaration file that is only
// local `export { default as Name } from "./…"` re-exports into one self-contained
// .d.ts, so the IDE does not open every leaf file.
//
// Leaf Vue icon components share the same default-export shape; collapsed exports
// reuse that shared type instead of following each relative path.
func collapseLargeDefaultAsReexportBarrel(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return content
	}

	matches := defaultAsLocalReexportPattern.FindAllStringSubmatch(content, -1)
	if len(matches) < largeDefaultAsReexportThreshold {
		return content
	}

	// Require the file to be exclusively these re-exports (plus blank/comment lines).
	withoutMatches := defaultAsLocalReexportPattern.ReplaceAllString(content, "")
	for _, line := range strings.Split(withoutMatches, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "*/") {
			continue
		}
		return content
	}

	names := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) < largeDefaultAsReexportThreshold {
		return content
	}

	var b strings.Builder
	b.Grow(len(names)*48 + 256)
	b.WriteString("// Collapsed local default-as re-export barrel: keep named exports without opening each leaf .d.ts.\n")
	b.WriteString("declare const __choysumCollapsedIcon: import(\"vue\").DefineComponent<{}, {}, {}, {}, {}, import(\"vue\").ComponentOptionsMixin, import(\"vue\").ComponentOptionsMixin, {}, string, import(\"vue\").PublicProps, Readonly<{}> & Readonly<{}>, {}, {}, {}, {}, string, import(\"vue\").ComponentProvideOptions, true, {}, any>;\n")
	for _, name := range names {
		b.WriteString("export declare const ")
		b.WriteString(name)
		b.WriteString(": typeof __choysumCollapsedIcon;\n")
	}
	return b.String()
}
