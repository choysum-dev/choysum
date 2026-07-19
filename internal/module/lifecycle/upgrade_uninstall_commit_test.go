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
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
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
	if err := runtimeScope.Session().AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}

func TestModuleUpgraderCommitRollsBackOnFailure(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	mod := &meta.IrModule{
		Name:    "demo",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	target := &meta.IrModule{
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

	var got meta.IrModule
	if err := runtimeScope.Session().Where("name = ?", "demo").Take(&got).Error; err != nil {
		t.Fatalf("load module: %v", err)
	}
	if got.Version != "1.0.0" {
		t.Fatalf("module version after rollback = %q, want 1.0.0", got.Version)
	}
}

func TestModuleUninstallerCommitRollsBackOnFailure(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.IrModule{
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

	var got meta.IrModule
	if err := runtimeScope.Session().Where("name = ?", "demo").Take(&got).Error; err != nil {
		t.Fatalf("load module: %v", err)
	}
	if got.Status != meta.Installed {
		t.Fatalf("module status after rollback = %q, want %q", got.Status, meta.Installed)
	}
}
