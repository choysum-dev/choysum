// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestPickTranslationTermOwnerModule_EmptyEntryAllowed(t *testing.T) {
	mods := []*meta.Module{
		nil,
		{Name: "web", Path: "/virtual/modules/web", ApplicationStr: "web", ServiceEntryPoint: ""},
	}
	owner := pickTranslationTermOwnerModule("web", mods)
	if owner == nil || owner.Name != "web" {
		t.Fatalf("expected empty-entry web owner, got %#v", owner)
	}
	if pickTranslationTermOwnerModule("core", mods) != nil {
		t.Fatal("core must not pick an owner")
	}

	both := []*meta.Module{
		{Name: "web_fe", Path: "/m/web_fe", ApplicationStr: "web", ServiceEntryPoint: ""},
		{Name: "web_be", Path: "/m/web_be", ApplicationStr: "web", ServiceEntryPoint: "service/index.ts"},
	}
	pref := pickTranslationTermOwnerModule("web", both)
	if pref == nil || pref.Name != "web_be" {
		t.Fatalf("expected with-entry preference, got %#v", pref)
	}
}

func TestAppendTranslationTermOwnersFromInstalled_EmptyEntryApp(t *testing.T) {
	existing := []*meta.Module{
		nil,
		{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts"},
	}
	installed := []meta.Module{
		{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts", Status: meta.Installed},
		{Name: "web", Path: "/m/web", ApplicationStr: "web", ServiceEntryPoint: "", Status: meta.Installed},
		{Name: "core", Path: "/m/core", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts", Status: meta.Installed},
		{Name: "ghost", Path: "", ApplicationStr: "ghost", ServiceEntryPoint: "", Status: meta.Installed},
		{Name: "base", Path: "/m/base", ApplicationStr: "base", ServiceEntryPoint: "", Status: meta.Installed},
	}
	got := appendTranslationTermOwnersFromInstalled(existing, installed)
	apps := map[string]bool{}
	var order []string
	for _, m := range got {
		if m == nil {
			continue
		}
		apps[m.ApplicationStr] = true
		if m.ApplicationStr != "auth" {
			order = append(order, m.ApplicationStr)
		}
	}
	if !apps["auth"] || !apps["web"] || !apps["base"] || apps["core"] || apps["ghost"] {
		t.Fatalf("expected auth+web+base only, got %#v", got)
	}
	// Appended Ensure-only apps follow installed order (web then base).
	if len(order) < 2 || order[0] != "web" || order[1] != "base" {
		t.Fatalf("expected stable append order web,base; got %#v", order)
	}
}

func TestPickBackendBundleRepresentative_FallsBackToTranslationTermOwner(t *testing.T) {
	if got := pickBackendBundleRepresentative(nil, nil, nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	tt := &meta.Module{Name: "web", Path: "/m/web", ApplicationStr: "web"}
	if got := pickBackendBundleRepresentative(nil, nil, []*meta.Module{nil, tt}); got != tt {
		t.Fatalf("expected TT fallback, got %#v", got)
	}
	// orderedApps non-empty but modsByApp missing/empty → still fall through to TT.
	if got := pickBackendBundleRepresentative([]string{"auth"}, map[string][]*meta.Module{"auth": {}}, []*meta.Module{tt}); got != tt {
		t.Fatalf("expected TT fallback for empty mods, got %#v", got)
	}
	if got := pickBackendBundleRepresentative([]string{"auth"}, nil, []*meta.Module{tt}); got != tt {
		t.Fatalf("expected TT fallback for nil modsByApp, got %#v", got)
	}
	be := &meta.Module{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "s"}
	modsByApp := map[string][]*meta.Module{"auth": {be}}
	if got := pickBackendBundleRepresentative([]string{"auth"}, modsByApp, []*meta.Module{tt}); got != be {
		t.Fatalf("expected backend rep, got %#v", got)
	}
}

type stubBundlerToDir struct {
	err error
	n   int
	dir string
}

func (s *stubBundlerToDir) BundleToDirCtx(_ context.Context, distAppDir string) (*module.BuildResult, error) {
	s.n++
	s.dir = distAppDir
	if s.err != nil {
		return nil, s.err
	}
	return &module.BuildResult{}, nil
}

func TestWriteBackendBundleToDir(t *testing.T) {
	if err := writeBackendBundleToDir(context.Background(), struct{}{}, t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "does not support BundleToDirCtx") {
		t.Fatalf("expected BundlerToDir !ok error, got %v", err)
	}

	fail := &stubBundlerToDir{err: errors.New("esbuild boom")}
	err := writeBackendBundleToDir(context.Background(), fail, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "backend bundle failed for bundles") {
		t.Fatalf("expected wrap, got %v", err)
	}

	ok := &stubBundlerToDir{}
	dir := t.TempDir()
	if err := writeBackendBundleToDir(context.Background(), ok, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok.n != 1 || ok.dir != dir {
		t.Fatalf("expected one BundleToDirCtx(%q), got n=%d dir=%q", dir, ok.n, ok.dir)
	}
}

func TestBuildBackendBundlesToDir_NoRepresentativeReturnsNil(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	if err := manager.buildBackendBundlesToDir(context.Background(), t.TempDir(), nil); err != nil {
		t.Fatalf("expected nil with no reps, got %v", err)
	}
}

func TestBuildBackendBundlesToDir_SuccessfulWrite(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	modPath := filepath.Join(modulesPath, "core")
	if err := os.MkdirAll(filepath.Join(modPath, "service"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modPath, "service", "index.ts"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	// core-only: no C2 owners, so BundleInjectAppModels runs with an empty list.
	if err := db.Create(&meta.Module{
		Name:              "core",
		ApplicationStr:    "core",
		Status:            meta.Installed,
		ServiceEntryPoint: "service/index.ts",
		Path:              modPath,
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	prev := writeBackendBundleToDirFn
	writeBackendBundleToDirFn = func(context.Context, any, string) error { return nil }
	t.Cleanup(func() { writeBackendBundleToDirFn = prev })

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	if err := manager.buildBackendBundlesToDir(context.Background(), t.TempDir(), nil); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestBuildBackendBundlesToDir_EmptyEntryOnlyStillInjectsTranslationTerm(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	modPath := filepath.Join(modulesPath, "web")
	if err := os.MkdirAll(modPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// canEnsureServiceEntry requires Spec base model on disk when ModulesPath exists.
	baseModel := filepath.Join(modulesPath, "core", "service", "orm", "model", "translation_term_base_model.ts")
	if err := os.MkdirAll(filepath.Dir(baseModel), 0o755); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}
	if err := os.WriteFile(baseModel, []byte("export default class TranslationTermBaseModel {}\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := db.Create(&meta.Module{
		Name:              "web",
		ApplicationStr:    "web",
		Status:            meta.Installed,
		ServiceEntryPoint: "",
		Path:              modPath,
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	distBundlesDir := t.TempDir()

	// Drop meta_raw_model so BundleInjectAppModels fails after owners were collected
	// and the Ensure-only representative path was selected.
	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	err := manager.buildBackendBundlesToDir(context.Background(), distBundlesDir, nil)
	if err == nil || !strings.Contains(err.Error(), "inject app models for bundles") {
		t.Fatalf("expected BundleInject error for Ensure-only apps, got %v", err)
	}
}
