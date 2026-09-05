// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatStderr_StableShape(t *testing.T) {
	var buf bytes.Buffer
	FormatStderr(&buf, []Diagnostic{{
		File:     "modules/demo/service/bad.ts",
		Line:     1,
		Column:   7,
		Code:     2322,
		Category: "error",
		Message:  "Type 'number' is not assignable to type 'string'.",
	}})
	got := buf.String()
	wantSub := "modules/demo/service/bad.ts:1:7 - error TS2322:"
	if !strings.Contains(got, wantSub) {
		t.Fatalf("format = %q, want substring %q", got, wantSub)
	}
}
