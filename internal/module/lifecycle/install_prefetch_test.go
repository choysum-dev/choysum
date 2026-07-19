// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type countingPrefetchOriginCoordinator struct {
	modules map[string]*meta.IrModule
	fetches []string
}

func (c *countingPrefetchOriginCoordinator) Peek(_ context.Context, input string) (*meta.IrModule, error) {
	return c.lookup(input), nil
}

func (c *countingPrefetchOriginCoordinator) ResolveInstallModule(_ context.Context, input string) (*meta.IrModule, error) {
	c.fetches = append(c.fetches, input)
	return c.lookup(input), nil
}

func (c *countingPrefetchOriginCoordinator) Fetch(ctx context.Context, input string) (*meta.IrModule, error) {
	return c.ResolveInstallModule(ctx, input)
}

func (c *countingPrefetchOriginCoordinator) Purge(context.Context, string) error {
	return nil
}

func (c *countingPrefetchOriginCoordinator) lookup(input string) *meta.IrModule {
	if c == nil || c.modules == nil {
		return nil
	}
	if mod := c.modules[input]; mod != nil {
		cloned := *mod
		return &cloned
	}
	return nil
}

func TestPrefetchInstallModulesThenResolveUsesCache(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	dependsRaw, err := json.Marshal([]string{"core"})
	if err != nil {
		t.Fatalf("marshal depends: %v", err)
	}
	core := &meta.IrModule{Name: "core", Version: "1.0.0", Path: filepath.Join(modulesPath, "core")}
	doc := &meta.IrModule{
		Name:       "document",
		Version:    "1.0.0",
		Path:       filepath.Join(modulesPath, "document"),
		DependsStr: dependsRaw,
	}
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.IrModule{
			"core":     core,
			"document": doc,
		},
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	locker := &moduleIndexSyncTestLocker{}
	manager := NewModuleManager(
		runtimeScope,
		&moduleManagerNoopScriptExecutor{},
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
	)
	manager.bootstrapOnce.Do(func() {})

	prefetched, err := manager.PrefetchInstallModules(context.Background(), "document")
	if err != nil {
		t.Fatalf("PrefetchInstallModules() error = %v", err)
	}
	if prefetched.RootName != "document" {
		t.Fatalf("RootName = %q, want document", prefetched.RootName)
	}
	if lookupPrefetchedModule(prefetched.Modules, "core") == nil {
		t.Fatal("expected core in prefetched modules")
	}
	prefetchFetches := len(origin.fetches)
	if prefetchFetches < 2 {
		t.Fatalf("expected at least root+dep fetches during prefetch, got %v", origin.fetches)
	}

	ctx := WithPrefetchedInstallModules(context.Background(), prefetched.Modules)
	gotRoot, err := manager.resolveInstallModuleFromOrigin(ctx, "document")
	if err != nil {
		t.Fatalf("resolve root from cache: %v", err)
	}
	if gotRoot == nil || gotRoot.Name != "document" {
		t.Fatalf("cached root = %#v", gotRoot)
	}
	gotDep, err := manager.resolveInstallModuleFromOrigin(ctx, "core")
	if err != nil {
		t.Fatalf("resolve dep from cache: %v", err)
	}
	if gotDep == nil || gotDep.Name != "core" {
		t.Fatalf("cached dep = %#v", gotDep)
	}
	if len(origin.fetches) != prefetchFetches {
		t.Fatalf("expected no additional origin fetches after prefetch, before=%d after=%d fetches=%v",
			prefetchFetches, len(origin.fetches), origin.fetches)
	}

	if _, err := manager.resolveInstallModuleFromOrigin(ctx, "missing"); err == nil {
		t.Fatal("expected error for module missing from prefetch cache")
	}
}

func TestPrefetchInstallContextHelpers(t *testing.T) {
	if PrefetchedInstallModulesFromContext(nil) != nil {
		t.Fatal("nil ctx")
	}
	if PrefetchedInstallModulesFromContext(context.Background()) != nil {
		t.Fatal("empty ctx")
	}
	ctx := WithPrefetchedInstallModules(nil, nil)
	if PrefetchedInstallModulesFromContext(ctx) != nil {
		t.Fatal("empty modules")
	}
	ctx = WithPrefetchedInstallModules(context.Background(), map[string]*meta.IrModule{
		"":     {Name: "skip"},
		"auth": nil,
		"web":  {Name: "web"},
	})
	got := PrefetchedInstallModulesFromContext(ctx)
	if len(got) != 1 || got["web"] == nil || got["web"].Name != "web" {
		t.Fatalf("got=%#v", got)
	}
	if _, err := PrefetchInstallModules(context.Background(), nil, "auth"); err == nil {
		t.Fatal("nil scope")
	}
}
