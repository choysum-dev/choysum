// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import "testing"

func TestSearchCondition_nilMapDomain(t *testing.T) {
	cond, err := SearchCondition(Plan{Domain: `null`})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	and, _ := cond["And"].([]any)
	if len(and) != 0 {
		t.Fatalf("cond = %#v", cond)
	}
}

func TestSearchCondition_whitespaceDomain(t *testing.T) {
	cond, err := SearchCondition(Plan{Domain: "   "})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	and, _ := cond["And"].([]any)
	if len(and) != 0 {
		t.Fatalf("cond = %#v", cond)
	}
}
