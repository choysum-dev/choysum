// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	moduleplan "github.com/choysum-dev/choysum/internal/module/plan"
)

func TestMergeUniqueModuleNames(t *testing.T) {
	got := mergeUniqueModuleNames(
		[]string{"", " web ", "auth"},
		nil,
		[]string{"auth", "partner", "  "},
		[]string{"web"},
	)
	want := []string{"web", "auth", "partner"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if got := mergeUniqueModuleNames(); len(got) != 0 {
		t.Fatalf("empty merge = %v", got)
	}
}

func TestModuleOperationPlanInfoAttrsIncludesEnsure(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationPlanInfoAttrs(moduleplan.Plan{
		ModuleOrder:         []string{"partner"},
		EnsureOrder:         []string{"auth", "web"},
		AffectedApps:        []string{"partner"},
		NeedsGlobalWebBuild: true,
	}))
	if got := attrs["ensure_count"]; got != 2 {
		t.Fatalf("ensure_count=%#v, want 2", got)
	}
	ensure, ok := attrs["ensure"].([]string)
	if !ok || len(ensure) != 2 || ensure[0] != "auth" || ensure[1] != "web" {
		t.Fatalf("ensure=%#v, want [auth web]", attrs["ensure"])
	}
}
