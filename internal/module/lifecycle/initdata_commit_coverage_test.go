// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
)

func lifecycleCommitModule(t *testing.T, name string) (*meta.Module, string) {
	t.Helper()
	modulePath := t.TempDir()
	i18nDir := filepath.Join(modulePath, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`msgid ""
msgstr ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{
		Name:           name,
		Version:        "1.0.0",
		Status:         meta.ToInstall,
		Path:           modulePath,
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	return mod, modulePath
}

func TestCommitInstall_applyInitdataWithDemo(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod, _ := lifecycleCommitModule(t, "demo_with_demo")
	opCtx := newOpContext()
	opCtx.withDemo = true
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           opCtx,
	}
	if err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall with withDemo: %v", err)
	}
}

func TestCommitInstall_applyInitdataNilCtx(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod, _ := lifecycleCommitModule(t, "demo_nil_ctx")
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           nil,
	}
	if err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall with nil ctx: %v", err)
	}
}

func TestCommitUpgrade_applyInitdataWithDemo(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	mod := &meta.Module{
		Name:    "demo_upgrade_demo",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{Name: "demo_upgrade_demo", Version: "2.0.0", Status: meta.Installed, Path: modulePath}
	target.Id = mod.Id

	opCtx := newOpContext()
	opCtx.withDemo = true
	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           opCtx,
	}
	installer := &moduleInstaller{
		module:        target,
		runtimeScope:  runtimeScope,
		moduleManager: upgrader.moduleManager,
		ctx:           opCtx,
	}
	if _, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false); err != nil {
		t.Fatalf("commitUpgrade with withDemo: %v", err)
	}
}

func TestCommitUpgrade_applyInitdataNilCtx(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	mod := &meta.Module{
		Name:    "demo_upgrade_nil_ctx",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{Name: "demo_upgrade_nil_ctx", Version: "2.0.0", Status: meta.Installed, Path: modulePath}
	target.Id = mod.Id

	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           nil,
	}
	installer := &moduleInstaller{
		module:        target,
		runtimeScope:  runtimeScope,
		moduleManager: upgrader.moduleManager,
		ctx:           nil,
	}
	if _, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false); err != nil {
		t.Fatalf("commitUpgrade with nil ctx: %v", err)
	}
}
