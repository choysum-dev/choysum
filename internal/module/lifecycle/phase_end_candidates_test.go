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

func TestPhaseEndCandidates_SkipsBlankNames(t *testing.T) {
	t.Parallel()
	ctx := newOpContext()
	ctx.markInstallTouched("partner")
	ctx.markUpgradeTouched("partner")
	got := phaseEndCandidates(plan.OpInstall, []string{"", "partner", " "}, nil, ctx)
	if !reflect.DeepEqual(got, []string{"partner"}) {
		t.Fatalf("install blanks = %v", got)
	}
	got = phaseEndCandidates(plan.OpUpgrade, []string{"", "partner"}, []string{"", "web"}, ctx)
	if !reflect.DeepEqual(got, []string{"partner"}) {
		t.Fatalf("upgrade blanks = %v", got)
	}
	ctx.markInstallTouched("web")
	got = phaseEndCandidates(plan.OpUpgrade, []string{"partner"}, []string{"", "web"}, ctx)
	if !reflect.DeepEqual(got, []string{"partner", "web"}) {
		t.Fatalf("upgrade blanks+web = %v", got)
	}
}

func TestPhaseEndCandidates_DefaultAndNilUpgrade(t *testing.T) {
	t.Parallel()
	got := phaseEndCandidates(plan.OpType("other"), []string{"a", "a", "b"}, nil, nil)
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default op = %v, want %v", got, want)
	}
	got = phaseEndCandidates(plan.OpInstall, []string{"a", "b"}, nil, nil)
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("nil ctx install = %v", got)
	}
	ctx := newOpContext()
	ctx.markUpgradeTouched("partner")
	got = phaseEndCandidates(plan.OpUpgrade, []string{"partner", "skip"}, nil, ctx)
	if !reflect.DeepEqual(got, []string{"partner"}) {
		t.Fatalf("upgrade filtered = %v", got)
	}
	got = phaseEndCandidates(plan.OpUpgrade, []string{"partner"}, []string{"web"}, nil)
	if !reflect.DeepEqual(got, []string{"partner"}) {
		t.Fatalf("nil ctx upgrade = %v", got)
	}
	ctx.markInstallTouched("")
	ctx.markUpgradeTouched("")
	if ctx.isInstallTouched("") || ctx.isUpgradeTouched("") {
		t.Fatal("empty name must not mark touched")
	}
}

func TestOpContextTouchedGuardsAndCycles(t *testing.T) {
	t.Parallel()
	var nilCtx *opContext
	nilCtx.markInstallTouched("x")
	nilCtx.markUpgradeTouched("x")
	if nilCtx.isInstallTouched("x") || nilCtx.isUpgradeTouched("x") {
		t.Fatal("nil ctx should not report touched")
	}
	ctx := &opContext{} // nil maps
	ctx.markInstallTouched("a")
	ctx.markUpgradeTouched("b")
	if !ctx.isInstallTouched("a") || !ctx.isUpgradeTouched("b") {
		t.Fatal("lazy map init failed")
	}
	if path := cyclePath([]string{"a", "b"}, "missing"); path != nil {
		t.Fatalf("missing cycle = %v", path)
	}
	if path := cyclePath([]string{"a", "b", "c"}, "b"); !reflect.DeepEqual(path, []string{"b", "c", "b"}) {
		t.Fatalf("cycle path = %v", path)
	}
	ctx = newOpContext()
	if cycle := ctx.pushInstall("a"); cycle != nil {
		t.Fatalf("first push: %v", cycle)
	}
	if cycle := ctx.pushInstall("a"); cycle == nil {
		t.Fatal("expected install cycle")
	}
	ctx.popInstall("a")
	if cycle := ctx.pushUninstall("u"); cycle != nil {
		t.Fatalf("uninstall push: %v", cycle)
	}
	if cycle := ctx.pushUninstall("u"); cycle == nil {
		t.Fatal("expected uninstall cycle")
	}
	ctx.popUninstall("u")
	if cycle := ctx.pushUpgrade("up"); cycle != nil {
		t.Fatalf("upgrade push: %v", cycle)
	}
	if cycle := ctx.pushUpgrade("up"); cycle == nil {
		t.Fatal("expected upgrade cycle")
	}
	ctx.popUpgrade("up")
	ctx.setFromVersion("", "1")
	ctx.setFromVersion("m", "1.0.0")
	if ctx.getFromVersion("m") != "1.0.0" || ctx.getFromVersion("") != "" {
		t.Fatalf("fromVersion helpers failed: %#v", ctx.fromVersion)
	}
	var nilGet *opContext
	if nilGet.getFromVersion("m") != "" {
		t.Fatal("nil getFromVersion")
	}
}
