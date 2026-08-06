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
}

func TestAppendTranslationTermOwnersFromInstalled_EmptyEntryApp(t *testing.T) {
	existing := []*meta.Module{
		{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts"},
	}
	installed := []meta.Module{
		{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts", Status: meta.Installed},
		{Name: "web", Path: "/m/web", ApplicationStr: "web", ServiceEntryPoint: "", Status: meta.Installed},
		{Name: "core", Path: "/m/core", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts", Status: meta.Installed},
	}
	got := appendTranslationTermOwnersFromInstalled(existing, installed)
	if len(got) != 2 {
		t.Fatalf("owners = %#v, want auth + web", got)
	}
	apps := map[string]bool{}
	for _, m := range got {
		apps[m.ApplicationStr] = true
	}
	if !apps["auth"] || !apps["web"] {
		t.Fatalf("expected auth and web owners, got %#v", got)
	}
}
