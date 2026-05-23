// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

func TestStableTopoSortApplicationsForTargets_DependencyOrder(t *testing.T) {
	mods := []meta.IrModule{
		{Name: "modB", ApplicationStr: "b", DependsStr: datatypes.JSON(`[]`)},
		{Name: "modA", ApplicationStr: "a", DependsStr: datatypes.JSON(`["modB"]`)},
	}

	order, cyclic := StableTopoSortApplicationsForTargets(mods, []string{"a", "b"})
	if cyclic {
		t.Fatalf("expected acyclic")
	}
	if len(order) != 2 || order[0] != "b" || order[1] != "a" {
		t.Fatalf("unexpected order: %#v", order)
	}
}

func TestStableTopoSortApplicationsForTargets_AlphabeticalWithinLevel(t *testing.T) {
	mods := []meta.IrModule{
		{Name: "mA", ApplicationStr: "a", DependsStr: datatypes.JSON(`[]`)},
		{Name: "mC", ApplicationStr: "c", DependsStr: datatypes.JSON(`[]`)},
		{Name: "mB", ApplicationStr: "b", DependsStr: datatypes.JSON(`[]`)},
	}

	order, cyclic := StableTopoSortApplicationsForTargets(mods, []string{"c", "b", "a"})
	if cyclic {
		t.Fatalf("expected acyclic")
	}
	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("unexpected order: %#v", order)
	}
}

func TestStableTopoSortApplicationsForTargets_CycleFallsBackAlphabetical(t *testing.T) {
	mods := []meta.IrModule{
		{Name: "modA", ApplicationStr: "a", DependsStr: datatypes.JSON(`["modB"]`)},
		{Name: "modB", ApplicationStr: "b", DependsStr: datatypes.JSON(`["modA"]`)},
	}

	order, cyclic := StableTopoSortApplicationsForTargets(mods, []string{"b", "a"})
	if !cyclic {
		t.Fatalf("expected cyclic")
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("unexpected order: %#v", order)
	}
}
