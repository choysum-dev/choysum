// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"
)

// ValidatePlan fails closed on any Guarded or Manual op (P0 has no Intent bag).
func ValidatePlan(plan SchemaPlan) error {
	var guarded []string
	for _, op := range plan.Ops {
		switch op.Safety {
		case SafetyAuto:
			continue
		case SafetyGuarded, SafetyManual:
			detail := strings.TrimSpace(op.Detail)
			if detail == "" {
				detail = string(op.Kind)
			}
			guarded = append(guarded, fmt.Sprintf("%s.%s (%s, %s)", op.Table, detail, op.Kind, op.Safety))
		default:
			guarded = append(guarded, fmt.Sprintf("%s.%s (unknown safety %q)", op.Table, op.Kind, op.Safety))
		}
	}
	if len(guarded) == 0 {
		return nil
	}
	return fmt.Errorf("schema plan has guarded/manual operations requiring an explicit script or safe change: %s", strings.Join(guarded, "; "))
}
