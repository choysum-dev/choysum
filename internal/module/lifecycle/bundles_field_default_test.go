// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestPickFieldDefaultOwnerModule_LastEligible(t *testing.T) {
	mods := []*meta.Module{
		{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "skip", Path: "", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		{Name: "partner_bank", Path: "/virtual/modules/partner_bank", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
	}
	owner := pickFieldDefaultOwnerModule("partner", mods)
	if owner == nil || owner.Name != "partner_bank" {
		t.Fatalf("expected last eligible owner partner_bank, got %#v", owner)
	}
	if pickFieldDefaultOwnerModule("core", mods) != nil {
		t.Fatal("core must not pick an owner")
	}
	if pickFieldDefaultOwnerModule("partner", nil) != nil {
		t.Fatal("empty mods must return nil")
	}
}
