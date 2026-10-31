// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"encoding/json"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestStableTopoSortModules(t *testing.T) {
	t.Parallel()
	depends := func(names ...string) []byte {
		raw, err := json.Marshal(names)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	mods := []meta.Module{
		{Name: "document", DependsStr: depends("task", "core")},
		{Name: "task", DependsStr: depends("core")},
		{Name: "core"},
	}
	order, cyclic := StableTopoSortModules(mods)
	if cyclic {
		t.Fatal("expected acyclic")
	}
	if len(order) != 3 || order[0].Name != "core" || order[1].Name != "task" || order[2].Name != "document" {
		t.Fatalf("order = %#v", order)
	}
}

func TestStableTopoSortModulesCycleFallsBackAlphabetical(t *testing.T) {
	t.Parallel()
	depends := func(names ...string) []byte {
		raw, _ := json.Marshal(names)
		return raw
	}
	mods := []meta.Module{
		{Name: "a", DependsStr: depends("b")},
		{Name: "b", DependsStr: depends("a")},
	}
	order, cyclic := StableTopoSortModules(mods)
	if !cyclic {
		t.Fatal("expected cyclic")
	}
	if len(order) != 2 || order[0].Name != "a" || order[1].Name != "b" {
		t.Fatalf("order = %#v", order)
	}
}
