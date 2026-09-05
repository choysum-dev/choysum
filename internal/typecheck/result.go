// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"fmt"

	"github.com/buke/typescript-go-internal/v7/pkg/diagnostics"
)

// Diagnostic is a stable, remappable typecheck finding.
type Diagnostic struct {
	File     string
	Start    int
	Length   int
	Line     int // 1-based; 0 if unknown
	Column   int // 1-based; 0 if unknown
	Code     int32
	Category string
	Message  string
}

// Result holds diagnostics from Check.
type Result struct {
	Diagnostics []Diagnostic
}

// Err returns a non-nil error when Result contains at least one error-category diagnostic.
func (r Result) Err() error {
	var n int
	for _, d := range r.Diagnostics {
		if d.Category == "error" {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	return fmt.Errorf("typecheck: %d error(s)", n)
}

// HasErrors reports whether any diagnostic is an error.
func (r Result) HasErrors() bool {
	return r.Err() != nil
}

func normalizeCategory(c diagnostics.Category) string {
	switch c {
	case diagnostics.CategoryError:
		return "error"
	case diagnostics.CategoryWarning:
		return "warning"
	case diagnostics.CategorySuggestion:
		return "suggestion"
	case diagnostics.CategoryMessage:
		return "message"
	default:
		return "unknown"
	}
}
