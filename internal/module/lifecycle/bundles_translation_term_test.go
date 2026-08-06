// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestPickTranslationTermOwnerModule_EmptyEntryAllowed(t *testing.T) {
	mods := []*meta.Module{
		nil,
		{Name: "web", Path: "/virtual/modules/web", ApplicationStr: "web", ServiceEntryPoint: ""},
	}
	owner := pickTranslationTermOwnerModule("web", mods)
	if owner == nil || owner.Name != "web" {
		t.Fatalf("expected empty-entry web owner, got %#v", owner)
	}
	if pickTranslationTermOwnerModule("core", mods) != nil {
		t.Fatal("core must not pick an owner")
	}

	both := []*meta.Module{
		{Name: "web_fe", Path: "/m/web_fe", ApplicationStr: "web", ServiceEntryPoint: ""},
		{Name: "web_be", Path: "/m/web_be", ApplicationStr: "web", ServiceEntryPoint: "service/index.ts"},
	}
	pref := pickTranslationTermOwnerModule("web", both)
	if pref == nil || pref.Name != "web_be" {
		t.Fatalf("expected with-entry preference, got %#v", pref)
	}
}

func TestAppendTranslationTermOwnersFromInstalled_EmptyEntryApp(t *testing.T) {
	existing := []*meta.Module{
		nil,
		{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts"},
	}
	installed := []meta.Module{
		{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts", Status: meta.Installed},
		{Name: "web", Path: "/m/web", ApplicationStr: "web", ServiceEntryPoint: "", Status: meta.Installed},
		{Name: "core", Path: "/m/core", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts", Status: meta.Installed},
		{Name: "ghost", Path: "", ApplicationStr: "ghost", ServiceEntryPoint: "", Status: meta.Installed},
	}
	got := appendTranslationTermOwnersFromInstalled(existing, installed)
	apps := map[string]bool{}
	for _, m := range got {
		if m == nil {
			continue
		}
		apps[m.ApplicationStr] = true
	}
	if !apps["auth"] || !apps["web"] || apps["core"] || apps["ghost"] {
		t.Fatalf("expected auth+web only, got %#v", got)
	}
}
