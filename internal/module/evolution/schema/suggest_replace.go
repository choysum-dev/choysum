// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
)

// FormatReplaceColumnSuggestions returns dry-run guidance for guarded type-change /
// narrow-size ops. Empty when the plan has no such ops.
func FormatReplaceColumnSuggestions(plan SchemaPlan) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, op := range plan.Ops {
		if op.Safety != SafetyGuarded && op.Safety != SafetyManual {
			continue
		}
		detail := strings.TrimSpace(op.Detail)
		if !needsReplaceColumnHint(detail) {
			continue
		}
		col := ""
		if op.Column != nil {
			col = strings.TrimSpace(op.Column.Name)
		}
		if col == "" {
			continue
		}
		table := strings.TrimSpace(op.Table)
		if table == "" {
			continue
		}
		key := strings.ToLower(table + "." + col)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		obsolete := ObsoleteColumnName(col)
		out = append(out,
			fmt.Sprintf("// ReplaceColumn for %s.%s (%s)", table, col, detail),
			fmt.Sprintf("$choysum.schema.replaceColumnPre(%q, %q);", table, col),
			fmt.Sprintf("$choysum.schema.replaceColumnCopyCast(%q, %q, %q, %q);", table, obsolete, col, "int_to_varchar"),
			fmt.Sprintf("$choysum.schema.replaceColumnDropObsolete(%q, %q);", table, col),
		)
	}
	return out
}
