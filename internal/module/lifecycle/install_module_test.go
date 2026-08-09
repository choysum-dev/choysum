// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"

)

func TestPrepareInstallAliasesPrefetchInstallModules(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	dependsRaw, err := json.Marshal([]string{})
	if err != nil {
		t.Fatalf("marshal depends: %v", err)
	}
	mod := &meta.Module{
		Name:       "solo",
		Version:    "1.0.0",
		Path:       filepath.Join(modulesPath, "solo"),
		DependsStr: dependsRaw,
	}
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.Module{"solo": mod},
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	locker := &moduleIndexSyncTestLocker{}
	opts := []Option{
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
	}

	prepared, err := PrepareInstall(context.Background(), runtimeScope, "solo", opts...)
	if err != nil {
		t.Fatalf("PrepareInstall() error = %v", err)
	}
	if prepared.RootName != "solo" {
		t.Fatalf("RootName = %q, want solo", prepared.RootName)
	}
	if lookupPrefetchedModule(prepared.Modules, "solo") == nil {
		t.Fatal("expected solo in prepared modules")
	}
}

func TestInstallModulePropagatesSkipWebShell(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	dependsRaw, err := json.Marshal([]string{})
	if err != nil {
		t.Fatalf("marshal depends: %v", err)
	}
	mod := &meta.Module{
		Name:           "solo_skip_web",
		Version:        "1.0.0",
		Path:           filepath.Join(modulesPath, "solo_skip_web"),
		DependsStr:     dependsRaw,
		WebEntryPoint:  "web/index.ts",
		ApplicationStr: "solo",
	}
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.Module{"solo_skip_web": mod},
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	locker := &moduleIndexSyncTestLocker{}
	opts := []Option{
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
	}

	// PrepareInstall succeeds; Install fails later without a JS executor — SkipWebShell
	// wiring on InstallModuleRequest / InstallRequest still executes.
	err = InstallModule(context.Background(), runtimeScope, nil, InstallModuleRequest{
		Input:        "solo_skip_web",
		SkipWebShell: true,
	}, opts...)
	if err == nil {
		t.Fatal("expected InstallModule to fail without executor/full install")
	}
}
