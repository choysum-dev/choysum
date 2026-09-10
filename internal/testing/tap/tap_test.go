// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tap

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteNilSafe(t *testing.T) {
	Write(nil, &Report{Total: 1})
	Write(&bytes.Buffer{}, nil)
}

func TestWritePlanAndCases(t *testing.T) {
	var buf bytes.Buffer
	Write(&buf, &Report{
		Total: 2,
		Cases: []Case{
			{Name: "pass", OK: true},
			{Name: "fail", OK: false, Error: &CaseError{Message: "boom", Stack: "stack"}},
		},
	})
	out := buf.String()
	if !strings.HasPrefix(out, "1..2\n") {
		t.Fatalf("plan: %q", out)
	}
	if !strings.Contains(out, "ok 1 - pass\n") {
		t.Fatalf("missing pass: %q", out)
	}
	if !strings.Contains(out, "not ok 2 - fail\n") {
		t.Fatalf("missing fail: %q", out)
	}
	if !strings.Contains(out, "message:") || !strings.Contains(out, "boom") {
		t.Fatalf("missing diagnostic: %q", out)
	}
}
