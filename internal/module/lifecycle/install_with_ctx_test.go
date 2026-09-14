// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

func TestInstallWithCtxMarksTouchedOnlyOnRealWork(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	manager := NewModuleManager(runtimeScope, &moduleManagerNoopScriptExecutor{})

	ctx := newOpContext()
	fresh := &meta.Module{
		Name: "touch_install_demo", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	if err := manager.installWithCtx(fresh, ctx); err != nil {
		t.Fatalf("installWithCtx fresh: %v", err)
	}
	if !ctx.isInstallTouched("touch_install_demo") {
		t.Fatal("fresh install should mark installTouched")
	}

	// already_installed short-circuit must not mark touched.
	ctx2 := newOpContext()
	already := &meta.Module{
		Name: "touch_install_demo", Version: "1.0.0", Status: meta.Installed,
		Path: fresh.Path, ApplicationStr: "auth",
	}
	if err := manager.installWithCtx(already, ctx2); err != nil {
		t.Fatalf("installWithCtx already: %v", err)
	}
	if ctx2.isInstallTouched("touch_install_demo") {
		t.Fatal("already_installed must not mark installTouched")
	}
	if !ctx2.isInstallDone("touch_install_demo") {
		t.Fatal("already_installed should still mark installDone")
	}

	// install() error path wraps and does not mark touched.
	ctx3 := newOpContext()
	bad := &meta.Module{
		Name: "touch_install_bad", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
		DependsStr: datatypes.JSON([]byte(`["missing_dep"]`)),
	}
	if err := manager.installWithCtx(bad, ctx3); err == nil {
		t.Fatal("expected dependency error")
	}
	if ctx3.isInstallTouched("touch_install_bad") || ctx3.isInstallDone("touch_install_bad") {
		t.Fatal("failed install must not mark done/touched")
	}
}
