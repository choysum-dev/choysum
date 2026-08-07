// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"

)

type fastFailOriginCoordinator struct {
	peekErr              error
	peekModule           *meta.Module
	peekInputs           []string
	resolveInstallErr    error
	resolveInstallModule *meta.Module
	resolveInstallInputs []string
}

func (f *fastFailOriginCoordinator) Peek(_ context.Context, input string) (*meta.Module, error) {
	f.peekInputs = append(f.peekInputs, input)
	if f.peekErr != nil {
		return nil, f.peekErr
	}
	if f.peekModule != nil {
		return f.peekModule, nil
	}
	return &meta.Module{Name: "task"}, nil
}

func (f *fastFailOriginCoordinator) ResolveInstallModule(_ context.Context, input string) (*meta.Module, error) {
	f.resolveInstallInputs = append(f.resolveInstallInputs, input)
	if f.resolveInstallErr != nil {
		return nil, f.resolveInstallErr
	}
	if f.resolveInstallModule != nil {
		return f.resolveInstallModule, nil
	}
	return nil, errors.New("not implemented")
}

func (f *fastFailOriginCoordinator) Fetch(context.Context, string) (*meta.Module, error) {
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
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
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
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
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

func TestModuleManagerMigrateBaseModuleSucceedsWithValidDB(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	if err := manager.migrateBaseModule(); err != nil {
		t.Fatalf("migrateBaseModule() error = %v", err)
	}
	// Second call must be idempotent.
	if err := manager.migrateBaseModule(); err != nil {
		t.Fatalf("second migrateBaseModule() error = %v", err)
	}
}

func TestModuleManagerUpgradeFastFailWhenRegistryPeekFails(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}
	coordinator := &fastFailOriginCoordinator{peekErr: errors.New("module catalog source not found")}

	manager := NewModuleManager(
		runtimeScope,
		nil,
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return coordinator }),
	)

	err := manager.Upgrade(context.Background(), "task@1.0.0")
	if err == nil || !strings.Contains(err.Error(), "module catalog source not found") {
		t.Fatalf("Upgrade() error = %v, want registry peek error", err)
	}
	if locker.acquired != 0 {
		t.Fatalf("locker Acquire calls = %d, want 0", locker.acquired)
	}
	if len(coordinator.peekInputs) != 1 || coordinator.peekInputs[0] != "task@1.0.0" {
		t.Fatalf("peek inputs = %#v, want [task@1.0.0]", coordinator.peekInputs)
	}
}

func TestModuleManagerPeekDoesNotFallbackToResolveInstallWhenLocalMissing(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	coordinator := &fastFailOriginCoordinator{
		peekErr:              errors.New("module core not found in modules path"),
		resolveInstallModule: &meta.Module{Name: "core", Version: "v1.0.0"},
	}

	manager := NewModuleManager(
		runtimeScope,
		nil,
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return coordinator }),
	)
	manager.bootstrapOnce.Do(func() {})

	mod, err := manager.Peek(context.Background(), "core")
	if err == nil || !strings.Contains(err.Error(), "module core not found in modules path") {
		t.Fatalf("Peek() error = %v, want local missing module error", err)
	}
	if mod != nil {
		t.Fatalf("Peek() module = %#v, want nil", mod)
	}
	if len(coordinator.peekInputs) != 1 || coordinator.peekInputs[0] != "core" {
		t.Fatalf("peek inputs = %#v, want [core]", coordinator.peekInputs)
	}
	if len(coordinator.resolveInstallInputs) != 0 {
		t.Fatalf("resolve install inputs = %#v, want none", coordinator.resolveInstallInputs)
	}
}

func TestModuleManagerUninstallFastFailWhenLoadReturnsError(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}

	manager := NewModuleManager(runtimeScope, nil, WithLockerFactory(func(scope.Scope) statepkg.Locker {
		return locker
	}))
	manager.bootstrapOnce.Do(func() {})

	err := manager.Uninstall(context.Background(), "auth")
	if err == nil || !strings.Contains(err.Error(), "error loading module auth") {
		t.Fatalf("Uninstall() error = %v, want load error", err)
	}
	if locker.acquired != 0 {
		t.Fatalf("locker Acquire calls = %d, want 0", locker.acquired)
	}
}

func TestModuleManagerUpgradeFastFailWhenLocalLoadReturnsError(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}

	manager := NewModuleManager(runtimeScope, nil, WithLockerFactory(func(scope.Scope) statepkg.Locker {
		return locker
	}))
	manager.bootstrapOnce.Do(func() {})

	err := manager.Upgrade(context.Background(), "auth")
	if err == nil || !strings.Contains(err.Error(), "error loading module auth") {
		t.Fatalf("Upgrade() error = %v, want load error", err)
	}
	if locker.acquired != 0 {
		t.Fatalf("locker Acquire calls = %d, want 0", locker.acquired)
	}
}

func TestModuleManagerUpgradeRegistryEntersLeaseBeforeOriginSwitchFailure(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	locker := &moduleIndexSyncTestLocker{}
	coordinator := &fastFailOriginCoordinator{}

	manager := NewModuleManager(
		runtimeScope,
		nil,
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return coordinator }),
	)
	manager.bootstrapOnce.Do(func() {})

	err := manager.Upgrade(context.Background(), "task@1.0.0")
	if err == nil || !strings.Contains(err.Error(), "scope config is not initialized") {
		t.Fatalf("Upgrade() error = %v, want origin switch config error", err)
	}
	if locker.acquired != 1 {
		t.Fatalf("locker Acquire calls = %d, want 1", locker.acquired)
	}
	if locker.released != 1 {
		t.Fatalf("locker Release calls = %d, want 1", locker.released)
	}
}
