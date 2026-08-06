// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	be := &meta.Module{Name: "auth", Path: "/m/auth", ApplicationStr: "auth", ServiceEntryPoint: "s"}
	modsByApp := map[string][]*meta.Module{"auth": {be}}
	if got := pickBackendBundleRepresentative([]string{"auth"}, modsByApp, []*meta.Module{tt}); got != be {
		t.Fatalf("expected backend rep, got %#v", got)
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
