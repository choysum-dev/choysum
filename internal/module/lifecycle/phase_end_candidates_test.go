// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"reflect"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/plan"
)

func TestPhaseEndCandidates_InstallSkipsAlreadyInstalledDeps(t *testing.T) {
	t.Parallel()
	ctx := newOpContext()
	ctx.markInstallTouched("partner")
	got := phaseEndCandidates(plan.OpInstall, []string{"core", "base", "partner"}, nil, ctx)
	want := []string{"partner"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("phaseEndCandidates(install) = %v, want %v", got, want)
	}
}

func TestPhaseEndCandidates_UpgradeOnlyTargetPlusNewEnsured(t *testing.T) {
	t.Parallel()
	ctx := newOpContext()
	ctx.markUpgradeTouched("partner")
	ctx.markInstallTouched("web")
	got := phaseEndCandidates(plan.OpUpgrade, []string{"partner", "other"}, []string{"web", "core"}, ctx)
	want := []string{"partner", "web"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("phaseEndCandidates(upgrade) = %v, want %v", got, want)
	}
}

func TestPhaseEndCandidates_NilCtxKeepsInstallOrder(t *testing.T) {
	t.Parallel()
	order := []string{"core", "partner"}
	got := phaseEndCandidates(plan.OpInstall, order, nil, nil)
	if !reflect.DeepEqual(got, order) {
		t.Fatalf("nil ctx install candidates = %v, want %v", got, order)
	}
}
