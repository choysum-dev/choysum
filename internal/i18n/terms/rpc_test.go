// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"strings"
	"testing"
)

func TestParseTermItemDefaults(t *testing.T) {
	if got := ParseTermItem("auth", nil); got.Application != "auth" {
		t.Fatalf("nil map = %#v", got)
	}
	item := ParseTermItem("auth", map[string]any{
		"Module":   "auth",
		"Scope":    "a@b",
		"Src":      "Hi",
		"Value":    "你好",
		"Kind":     "",
		"Comments": "a.ts:1",
	})
	if item.Kind != "literal" || item.Status != "translated" || item.Comments != "a.ts:1" {
		t.Fatalf("item = %#v", item)
	}
	if ParseTermItem("auth", map[string]any{"Value": "  "}).Status != "missing" {
		t.Fatal("blank value should be missing")
	}
}

func TestParseSearchResultSkipsBadRows(t *testing.T) {
	got := ParseSearchResult("auth", "zh_CN", 10, 0, 1, map[string]any{
		"result": []any{
			map[string]any{"Src": "ok", "Value": "好"},
			"bad-row",
		},
	})
	if len(got.Items) != 1 || got.Items[0].Src != "ok" {
		t.Fatalf("items = %#v", got.Items)
	}
}

func TestToInt64Coercion(t *testing.T) {
	if ToInt64(int(7)) != 7 || ToInt64(int64(8)) != 8 || ToInt64(float64(9)) != 9 {
		t.Fatal("expected int/int64/float64 coercion")
	}
	if ToInt64(int32(3)) != 3 || ToInt64(float32(4)) != 4 || ToInt64("5") != 5 {
		t.Fatal("expected numeric coercion")
	}
	if ToInt64(nil) != 0 || ToInt64("<nil>") != 0 {
		t.Fatal("expected zero for nil-like values")
	}
}

func TestTermStatus(t *testing.T) {
	if TermStatus("x") != "translated" || TermStatus("") != "missing" {
		t.Fatal("unexpected term status")
	}
}

func TestBuildSearchCondition(t *testing.T) {
	empty := BuildSearchCondition("zh_CN", nil, "")
	and, ok := empty["And"].([]any)
	if !ok || len(and) != 1 {
		t.Fatalf("empty modules: %#v", empty)
	}
	one := BuildSearchCondition("zh_CN", []string{"auth"}, "")
	and, ok = one["And"].([]any)
	if !ok || len(and) != 2 {
		t.Fatalf("one module: %#v", one)
	}
	many := BuildSearchCondition("zh_CN", []string{"auth", "web"}, "q")
	and, ok = many["And"].([]any)
	if !ok || len(and) != 3 {
		t.Fatalf("many modules + q: %#v", many)
	}
	if !strings.Contains(many["And"].([]any)[2].(map[string]any)["Or"].([]any)[0].([]any)[2].(string), "q") {
		t.Fatalf("q filter missing: %#v", many)
	}
}

func TestMapStringIgnoresBlank(t *testing.T) {
	if mapString(map[string]any{"k": "<nil>"}, "k") != "" {
		t.Fatal("expected blank for nil marker")
	}
}
