// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type fastFailOriginCoordinator struct {
	peekErr    error
	peekInputs []string
}

func (f *fastFailOriginCoordinator) Peek(_ context.Context, input string) (*meta.IrModule, error) {
	f.peekInputs = append(f.peekInputs, input)
	if f.peekErr != nil {
		return nil, f.peekErr
	}
	return &meta.IrModule{Name: "task"}, nil
}

func (f *fastFailOriginCoordinator) ResolveInstallModule(context.Context, string) (*meta.IrModule, error) {
	return nil, errors.New("not implemented")
}

func (f *fastFailOriginCoordinator) Fetch(context.Context, string) (*meta.IrModule, error) {
	return nil, errors.New("not implemented")
}

func (f *fastFailOriginCoordinator) Purge(context.Context, string) error {
	return errors.New("not implemented")
}

func TestModuleManagerUninstallFastFailWhenNameEmpty(t *testing.T) {
	err := (&ModuleManager{}).Uninstall(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("Uninstall() error = %v, want module name is empty", err)
	}
}

func TestModuleManagerUninstallFastFailWhenModuleMissing(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}

	manager := NewModuleManager(runtimeScope, nil, WithLockerFactory(func(scope.Scope) statepkg.Locker {
		return locker
	}))
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}
	manager.bootstrapOnce.Do(func() {})

	err := manager.Uninstall(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "module missing not found") {
		t.Fatalf("Uninstall() error = %v, want missing module error", err)
	}
	if locker.acquired != 0 {
		t.Fatalf("locker Acquire calls = %d, want 0", locker.acquired)
	}
}

func TestModuleManagerUpgradeFastFailWhenInputEmpty(t *testing.T) {
	err := (&ModuleManager{}).Upgrade(context.Background(), "\t\n")
	if err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("Upgrade() error = %v, want module name is empty", err)
	}
}

func TestModuleManagerUpgradeFastFailWhenInputInvalid(t *testing.T) {
	err := (&ModuleManager{}).Upgrade(context.Background(), "registry/module@")
	if err == nil || !strings.Contains(err.Error(), "error parsing module input") {
		t.Fatalf("Upgrade() error = %v, want parse input error", err)
	}
}

func TestModuleManagerUpgradeFastFailWhenLocalModuleMissing(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}

	manager := NewModuleManager(runtimeScope, nil, WithLockerFactory(func(scope.Scope) statepkg.Locker {
		return locker
	}))
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}
	manager.bootstrapOnce.Do(func() {})

	err := manager.Upgrade(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "module missing not found") {
		t.Fatalf("Upgrade() error = %v, want missing module error", err)
	}
	if locker.acquired != 0 {
		t.Fatalf("locker Acquire calls = %d, want 0", locker.acquired)
	}
}

func TestModuleManagerUpgradeFastFailWhenRegistryPeekFails(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}
	coordinator := &fastFailOriginCoordinator{peekErr: errors.New("registry alias not found")}

	manager := NewModuleManager(
		runtimeScope,
		nil,
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return coordinator }),
	)

	err := manager.Upgrade(context.Background(), "registry/task@1.0.0")
	if err == nil || !strings.Contains(err.Error(), "registry alias not found") {
		t.Fatalf("Upgrade() error = %v, want registry peek error", err)
	}
	if locker.acquired != 0 {
		t.Fatalf("locker Acquire calls = %d, want 0", locker.acquired)
	}
	if len(coordinator.peekInputs) != 1 || coordinator.peekInputs[0] != "registry/task@1.0.0" {
		t.Fatalf("peek inputs = %#v, want [registry/task@1.0.0]", coordinator.peekInputs)
	}
}