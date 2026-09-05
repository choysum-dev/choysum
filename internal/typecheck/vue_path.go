// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import "strings"

// ScriptKindStrategy documents how .vue files participate in the TypeScript Program.
//
// Frozen for PR-3 as Strategy B:
//   - Disk / import paths stay "*.vue".
//   - Program roots use a sibling virtual path "*.vue.ts" holding the service script.
//   - Host overlays both "*.vue" and "*.vue.ts" with the same service-script text so
//     `import x from './Y.vue'` resolves while Program accepts a supported extension.
//   - Diagnostics on "*.vue.ts" are reported as "*.vue" after SpanMapping remap.
//
// Strategy A (Program root is literally "*.vue") is not viable with the current
// typescript-go-internal extension allow-list (TS6054).
const ScriptKindStrategy = "B"

func toVueProgramPath(vuePath string) string {
	return vuePath + ".ts"
}

func fromVueProgramPath(path string) (vuePath string, ok bool) {
	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".vue.ts") {
		return "", false
	}
	return path[:len(path)-len(".ts")], true
}

func rewriteVueRootsToProgramPaths(files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".vue") {
			out = append(out, toVueProgramPath(f))
			continue
		}
		out = append(out, f)
	}
	return out
}
