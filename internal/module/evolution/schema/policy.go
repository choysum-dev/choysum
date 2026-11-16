// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
)

// replaceColumnHint is appended when ValidatePlan rejects incompatible in-place column changes.
const replaceColumnHint = "hint: incompatible in-place change; use ReplaceColumn " +
	"($choysum.schema.replaceColumnPre / replaceColumnCopyCast / replaceColumnDropObsolete; " +
	"see .dev/docs/meta/schema_column_replace_plan.md). " +
	"Target @Field must be nullable for this upgrade."

// ValidatePlan fails closed on any Guarded or Manual op unless an Intent already covers it.
func ValidatePlan(plan SchemaPlan, intents IntentBag) error {
	var guarded []string
	hintReplace := false
	for _, op := range plan.Ops {
		switch op.Safety {
		case SafetyAuto:
			continue
		case SafetyGuarded, SafetyManual:
			if IntentSatisfies(op, intents) {
				continue
			}
			detail := strings.TrimSpace(op.Detail)
			if detail == "" {
				detail = string(op.Kind)
			}
			if needsReplaceColumnHint(detail) {
				hintReplace = true
			}
			guarded = append(guarded, fmt.Sprintf("%s.%s (%s, %s)", op.Table, detail, op.Kind, op.Safety))
		default:
			guarded = append(guarded, fmt.Sprintf("%s.%s (unknown safety %q)", op.Table, op.Kind, op.Safety))
		}
	}
	if len(guarded) == 0 {
		return nil
	}
	msg := "schema plan has guarded/manual operations requiring an explicit script or safe change: " + strings.Join(guarded, "; ")
	if hintReplace {
		msg += "; " + replaceColumnHint
	}
	return fmt.Errorf("%s", msg)
}

func needsReplaceColumnHint(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "type change") || strings.Contains(lower, "narrow size")
}
