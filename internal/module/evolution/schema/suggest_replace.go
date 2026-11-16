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
		)
		if kind, ok := suggestCopyCastKind(detail); ok {
			out = append(out, fmt.Sprintf("$choysum.schema.replaceColumnCopyCast(%q, %q, %q, %q);", table, obsolete, col, kind))
		} else {
			out = append(out, fmt.Sprintf("// choose replaceColumnCopyCast kind or custom SQL for %s.%s", table, col))
		}
		out = append(out, fmt.Sprintf("$choysum.schema.replaceColumnDropObsolete(%q, %q);", table, col))
	}
	return out
}

// suggestCopyCastKind picks a built-in CopyCast kind from the plan detail, or ok=false
// when authors must supply an explicit conversion (e.g. varchar → int).
func suggestCopyCastKind(detail string) (kind string, ok bool) {
	d := strings.ToLower(strings.TrimSpace(detail))
	if d == "" {
		return "", false
	}
	if strings.Contains(d, "narrow size") {
		return "as_text", true
	}
	target := d
	for _, sep := range []string{"→", "->"} {
		if i := strings.LastIndex(d, sep); i >= 0 {
			target = strings.TrimSpace(d[i+len(sep):])
			break
		}
	}
	if strings.Contains(target, "varchar") || strings.Contains(target, "text") || strings.Contains(target, "char") {
		src := d
		if i := strings.Index(d, "→"); i >= 0 {
			src = strings.TrimSpace(d[:i])
		} else if i := strings.Index(d, "->"); i >= 0 {
			src = strings.TrimSpace(d[:i])
		}
		if sourceLooksLikeIntegerType(src) {
			return "int_to_varchar", true
		}
		return "as_text", true
	}
	return "", false
}

func sourceLooksLikeIntegerType(src string) bool {
	// Longer tokens first so "integer" wins over bare "int".
	for _, tok := range []string{"integer", "bigint", "smallint", "tinyint", "int"} {
		if sqliteSQLMentionsColumn(strings.ToLower(src), tok) {
			return true
		}
	}
	return false
}
