// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
)

func newLifecycleCommitTestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "lifecycle-commit.db"),
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		logger,
	)
	if err := runtimeScope.Session().AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := runtimeScope.Session().AutoMigrate(&modmeta.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate ModelData: %v", err)
	}
	return runtimeScope
}

func TestModuleUpgraderCommitRollsBackOnFailure(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	mod := &meta.Module{
		Name:    "demo",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	target := &meta.Module{
		Name:    "demo",
		Version: "2.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	target.Id = mod.Id

	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	wantErr := errors.New("forced upgrade commit failure")
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, _ scope.Transaction) error {
		installer := &moduleInstaller{
			module:        target,
			runtimeScope:  txScope,
			moduleManager: upgrader.moduleManager,
			ctx:           upgrader.ctx,
		}
		committed := *upgrader
		committed.runtimeScope = txScope
		if _, err := committed.commitUpgrade(installer, "1.0.0", nil, false); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Required() error = %v, want %v", err, wantErr)
	}

	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "demo").Take(&got).Error; err != nil {
		t.Fatalf("load module: %v", err)
	}
	if got.Version != "1.0.0" {
		t.Fatalf("module version after rollback = %q, want 1.0.0", got.Version)
	}
}

func TestRunUpgradeCommitTX_WithAndWithoutManager(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "upgrade_tx_demo", Version: "1.0.0", Status: meta.Installed,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	target := &meta.Module{
		Name: "upgrade_tx_demo", Version: "2.0.0", Status: meta.Installed,
		Path: mod.Path, ApplicationStr: "auth",
	}
	target.Id = mod.Id

	mgr := &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}}
	upgrader := &moduleUpgrader{runtimeScope: runtimeScope, module: mod, moduleManager: mgr, ctx: newOpContext()}
	installer := &moduleInstaller{module: target, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
	var buildResult *module.BuildResult
	if err := upgrader.runUpgradeCommitTX(runtimeScope, runtimeScope.Context(), installer, "1.0.0", &buildResult, false); err != nil {
		t.Fatalf("with manager: %v", err)
	}
	buildResult = nil
	if err := upgrader.runUpgradeCommitTX(runtimeScope, nil, installer, "1.0.0", &buildResult, false); err != nil {
		t.Fatalf("nil ctx re-upgrade: %v", err)
	}
	badInstaller := &moduleInstaller{runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
	if err := upgrader.runUpgradeCommitTX(runtimeScope, context.Background(), badInstaller, "1.0.0", &buildResult, false); err == nil || !strings.Contains(err.Error(), "installer is nil") {
		t.Fatalf("nil module installer: %v", err)
	}
	if err := upgrader.runUpgradeCommitTX(nil, context.Background(), installer, "1.0.0", &buildResult, false); err == nil {
		t.Fatal("expected nil scope error")
	}
	if err := upgrader.runUpgradeCommitTX(runtimeScope, context.Background(), installer, "1.0.0", nil, false); err == nil {
		t.Fatal("expected nil build result slot error")
	}

	// commitErr inside Required: closed DB after installer is valid.
	closedScope := newLifecycleCommitTestScope(t)
	closedMod := &meta.Module{
		Name: "upgrade_tx_closed", Version: "1.0.0", Status: meta.Installed,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	closedMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := closedScope.Session().Create(closedMod).Error; err != nil {
		t.Fatal(err)
	}
	closedTarget := &meta.Module{
		Name: closedMod.Name, Version: "2.0.0", Status: meta.Installed,
		Path: closedMod.Path, ApplicationStr: "auth",
	}
	closedTarget.Id = closedMod.Id
	sqlDB, err := closedScope.Session().DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	closedUp := &moduleUpgrader{runtimeScope: closedScope, module: closedMod, moduleManager: mgr, ctx: newOpContext()}
	closedInst := &moduleInstaller{module: closedTarget, runtimeScope: closedScope, moduleManager: mgr, ctx: newOpContext()}
	sentinel := &module.BuildResult{}
	buildResult = sentinel
	origDeps := replaceModuleDependenciesFn
	t.Cleanup(func() { replaceModuleDependenciesFn = origDeps })
	replaceModuleDependenciesFn = func(*scope.Session, *meta.Module) error {
		return errors.New("dep replace boom")
	}
	closedTarget.Dependencies = []*meta.Module{{
		Name: "dep_for_fail", Version: "1.0.0", Status: meta.Installed, Path: t.TempDir(),
	}}
	closedTarget.Dependencies[0].Id = sql.NullString{String: xid.New().String(), Valid: true}
	// Use a live scope so Required enters the callback; fail inside commitUpgrade.
	liveScope := newLifecycleCommitTestScope(t)
	liveMod := &meta.Module{
		Name: "upgrade_tx_noclobber", Version: "1.0.0", Status: meta.Installed,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	liveMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := liveScope.Session().Create(liveMod).Error; err != nil {
		t.Fatal(err)
	}
	liveTarget := &meta.Module{
		Name: liveMod.Name, Version: "2.0.0", Status: meta.Installed,
		Path: liveMod.Path, ApplicationStr: "auth",
		Dependencies: closedTarget.Dependencies,
	}
	liveTarget.Id = liveMod.Id
	liveUp := &moduleUpgrader{runtimeScope: liveScope, module: liveMod, moduleManager: mgr, ctx: newOpContext()}
	liveInst := &moduleInstaller{module: liveTarget, runtimeScope: liveScope, moduleManager: mgr, ctx: newOpContext()}
	if err := liveUp.runUpgradeCommitTX(liveScope, nil, liveInst, "1.0.0", &buildResult, false); err == nil {
		t.Fatal("expected upgrade commit error")
	}
	if buildResult != sentinel {
		t.Fatal("failed upgrade commit must not publish build result")
	}

	buildResult = nil
	if err := closedUp.runUpgradeCommitTX(closedScope, nil, closedInst, "1.0.0", &buildResult, false); err == nil {
		t.Fatal("expected closed-db commit error")
	}

	if err := (*moduleUpgrader)(nil).upgradeAfterPrepare(installer, "1.0.0", nil, false); err == nil || !strings.Contains(err.Error(), "scope is nil") {
		t.Fatalf("nil upgrader: %v", err)
	}
	if err := (&moduleUpgrader{}).upgradeAfterPrepare(installer, "1.0.0", nil, false); err == nil || !strings.Contains(err.Error(), "scope is nil") {
		t.Fatalf("nil runtimeScope: %v", err)
	}
	if err := closedUp.upgradeAfterPrepare(closedInst, "1.0.0", nil, false); err == nil {
		t.Fatal("expected closed-db upgradeAfterPrepare error")
	}
}

func TestModuleUninstallerCommitRollsBackOnFailure(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:    "demo",
		Version: "1.0.0",
		Status:  meta.Installed,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	wantErr := errors.New("forced uninstall commit failure")
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, _ scope.Transaction) error {
		committed := *uninstaller
		committed.runtimeScope = txScope
		if err := committed.commitUninstall(); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Required() error = %v, want %v", err, wantErr)
	}

	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "demo").Take(&got).Error; err != nil {
		t.Fatalf("load module: %v", err)
	}
	if got.Status != meta.Installed {
		t.Fatalf("module status after rollback = %q, want %q", got.Status, meta.Installed)
	}
}
