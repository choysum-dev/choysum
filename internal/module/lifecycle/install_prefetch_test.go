// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type prefetchLogCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *prefetchLogCapture) Enabled(context.Context, slog.Level) bool { return true }

func (h *prefetchLogCapture) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *prefetchLogCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *prefetchLogCapture) WithGroup(string) slog.Handler      { return h }

func (h *prefetchLogCapture) levelsFor(msg string) []slog.Level {
	h.mu.Lock()
	defer h.mu.Unlock()
	var levels []slog.Level
	for _, r := range h.records {
		if r.Message == msg {
			levels = append(levels, r.Level)
		}
	}
	return levels
}

type countingPrefetchOriginCoordinator struct {
	modules map[string]*meta.Module
	fetches []string
	// onFetch is invoked before each ResolveInstallModule lookup (tests may capture spinner state).
	onFetch func(input string)
}

func (c *countingPrefetchOriginCoordinator) Peek(_ context.Context, input string) (*meta.Module, error) {
	return c.lookup(input), nil
}

func (c *countingPrefetchOriginCoordinator) ResolveInstallModule(_ context.Context, input string) (*meta.Module, error) {
	c.fetches = append(c.fetches, input)
	if c.onFetch != nil {
		c.onFetch(input)
	}
	return c.lookup(input), nil
}

func (c *countingPrefetchOriginCoordinator) Fetch(ctx context.Context, input string) (*meta.Module, error) {
	return c.ResolveInstallModule(ctx, input)
}

func (c *countingPrefetchOriginCoordinator) Purge(context.Context, string) error {
	return nil
}

func (c *countingPrefetchOriginCoordinator) lookup(input string) *meta.Module {
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
	t.Parallel()
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	dependsRaw, err := json.Marshal([]string{"core"})
	if err != nil {
		t.Fatalf("marshal depends: %v", err)
	}
	core := &meta.Module{Name: "core", Version: "1.0.0", Path: filepath.Join(modulesPath, "core")}
	doc := &meta.Module{
		Name:       "document",
		Version:    "1.0.0",
		Path:       filepath.Join(modulesPath, "document"),
		DependsStr: dependsRaw,
	}
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.Module{
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

func TestPrefetchInstallModulesLogLevelUsesProgressDetailChannel(t *testing.T) {
	t.Parallel()
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
	locker := &moduleIndexSyncTestLocker{}

	t.Run("default context stays Info", func(t *testing.T) {
		capture := &prefetchLogCapture{}
		runtimeScope := newModuleIndexSyncScope(modulesPath, db)
		runtimeScope.logger = slog.New(capture)
		manager := NewModuleManager(
			runtimeScope,
			&moduleManagerNoopScriptExecutor{},
			WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
			WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
		)
		manager.bootstrapOnce.Do(func() {})

		if _, err := manager.PrefetchInstallModules(context.Background(), "solo"); err != nil {
			t.Fatalf("PrefetchInstallModules() error = %v", err)
		}
		levels := capture.levelsFor("install modules prefetched")
		if len(levels) != 1 || levels[0] != slog.LevelInfo {
			t.Fatalf("levels=%v, want [Info]", levels)
		}
	})

	t.Run("quiet progress demotes to Debug", func(t *testing.T) {
		capture := &prefetchLogCapture{}
		runtimeScope := newModuleIndexSyncScope(modulesPath, db)
		runtimeScope.logger = slog.New(capture)
		manager := NewModuleManager(
			runtimeScope,
			&moduleManagerNoopScriptExecutor{},
			WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
			WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
		)
		manager.bootstrapOnce.Do(func() {})

		ctx := logutil.WithQuietProgress(context.Background())
		if _, err := manager.PrefetchInstallModules(ctx, "solo"); err != nil {
			t.Fatalf("PrefetchInstallModules() error = %v", err)
		}
		levels := capture.levelsFor("install modules prefetched")
		if len(levels) != 1 || levels[0] != slog.LevelDebug {
			t.Fatalf("levels=%v, want [Debug]", levels)
		}
	})
}

func TestPrefetchInstallContextHelpers(t *testing.T) {
	t.Parallel()
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
	ctx = WithPrefetchedInstallModules(context.Background(), map[string]*meta.Module{
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
