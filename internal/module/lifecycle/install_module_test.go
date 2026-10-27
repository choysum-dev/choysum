// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestEnsureInstallPrepareProgressTicker(t *testing.T) {
	t.Parallel()

	t.Run("no progress line leaves ctx unchanged", func(t *testing.T) {
		ctx := context.Background()
		gotCtx, ticker, owns := ensureInstallPrepareProgressTicker(ctx)
		if gotCtx != ctx || ticker != nil || owns {
			t.Fatalf("ctx=%v ticker=%v owns=%v", gotCtx != ctx, ticker != nil, owns)
		}
	})

	t.Run("creates ticker when ProgressLine present", func(t *testing.T) {
		line := logutil.NewProgressLine(&bytes.Buffer{})
		ctx := logutil.WithProgressLine(context.Background(), line)
		gotCtx, ticker, owns := ensureInstallPrepareProgressTicker(ctx)
		if ticker == nil || !owns {
			t.Fatalf("ticker=%v owns=%v", ticker != nil, owns)
		}
		if logutil.ProgressTickerFromContext(gotCtx) != ticker {
			t.Fatal("expected ticker stored on context")
		}
		ticker.Stop()
	})

	t.Run("reuses existing ticker", func(t *testing.T) {
		existing := logutil.NewProgressTicker(nil, logutil.ProgressTickerOptions{})
		ctx := logutil.WithProgressTicker(context.Background(), existing)
		gotCtx, ticker, owns := ensureInstallPrepareProgressTicker(ctx)
		if gotCtx != ctx || ticker != existing || owns {
			t.Fatalf("reuse failed: sameCtx=%v tickerMatch=%v owns=%v", gotCtx == ctx, ticker == existing, owns)
		}
	})
}

func TestSetInstallResolvingModulesSpinner(t *testing.T) {
	t.Parallel()
	ticker := logutil.NewProgressTicker(nil, logutil.ProgressTickerOptions{})
	setInstallResolvingModulesSpinner(ticker, "partner")
	if got := ticker.Message(); got != "partner: resolving modules" {
		t.Fatalf("message=%q", got)
	}
	setInstallResolvingModulesSpinner(ticker, "  ")
	if got := ticker.Message(); got != "unknown: resolving modules" {
		t.Fatalf("empty input message=%q", got)
	}
	setInstallResolvingModulesSpinner(nil, "partner") // no panic
}

func TestInstallModuleSetsResolvingModulesSpinnerBeforePrepare(t *testing.T) {
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
		Name:       "solo_spinner",
		Version:    "1.0.0",
		Path:       filepath.Join(modulesPath, "solo_spinner"),
		DependsStr: dependsRaw,
	}
	line := logutil.NewProgressLine(&bytes.Buffer{})
	ticker := logutil.NewProgressTicker(line, logutil.ProgressTickerOptions{})
	defer ticker.Stop()
	var captured string
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.Module{"solo_spinner": mod},
		onFetch: func(string) {
			if captured == "" {
				captured = ticker.Message()
			}
		},
	}
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	locker := &moduleIndexSyncTestLocker{}
	opts := []Option{
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
	}

	ctx := logutil.WithProgressLine(context.Background(), line)
	ctx = logutil.WithProgressTicker(ctx, ticker)

	err = InstallModule(ctx, runtimeScope, nil, InstallModuleRequest{
		Input:        "solo_spinner",
		SkipWebShell: true,
	}, opts...)
	if err == nil {
		t.Fatal("expected InstallModule to fail without executor/full install")
	}
	if len(origin.fetches) == 0 || origin.fetches[0] != "solo_spinner" {
		t.Fatalf("fetches=%v", origin.fetches)
	}
	if captured != "solo_spinner: resolving modules" {
		t.Fatalf("spinner during prepare = %q, want resolving modules", captured)
	}
}

func TestInstallModuleRejectsEmptyPreparedRoot(t *testing.T) {
	// Not parallel: stubs package-level prepareInstallModules and would race
	// other InstallModule tests that expect real prefetch (fetches).
	prev := prepareInstallModules
	prepareInstallModules = func(context.Context, scope.Scope, string, ...Option) (*PrefetchedInstall, error) {
		return &PrefetchedInstall{RootName: "  "}, nil
	}
	t.Cleanup(func() { prepareInstallModules = prev })

	line := logutil.NewProgressLine(&bytes.Buffer{})
	ticker := logutil.NewProgressTicker(line, logutil.ProgressTickerOptions{})
	defer ticker.Stop()
	ticker.SetMessage("outer: installing batch")
	ctx := logutil.WithProgressLine(context.Background(), line)
	ctx = logutil.WithProgressTicker(ctx, ticker)

	err := InstallModule(ctx, newModuleIndexSyncScope(t.TempDir(), newModuleIndexSyncDB(t)), nil, InstallModuleRequest{
		Input: "solo_empty_root",
	})
	if err == nil || !strings.Contains(err.Error(), "prepared install root is empty") {
		t.Fatalf("InstallModule() error = %v, want empty root", err)
	}
	if got := ticker.Message(); got != "outer: installing batch" {
		t.Fatalf("caller spinner after empty root = %q, want restored", got)
	}
}

func TestInstallModuleRestoresCallerOwnedPrepareSpinnerOnPrepareFailure(t *testing.T) {
	t.Parallel()
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	line := logutil.NewProgressLine(&bytes.Buffer{})
	ticker := logutil.NewProgressTicker(line, logutil.ProgressTickerOptions{})
	defer ticker.Stop()
	ticker.SetMessage("outer: installing batch")
	var duringPrepare string
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.Module{},
		onFetch: func(string) {
			if duringPrepare == "" {
				duringPrepare = ticker.Message()
			}
		},
	}
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	ctx := logutil.WithProgressLine(context.Background(), line)
	ctx = logutil.WithProgressTicker(ctx, ticker)

	err := InstallModule(ctx, runtimeScope, nil, InstallModuleRequest{
		Input:        "missing_prepare",
		SkipWebShell: true,
	}, []Option{
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return &moduleIndexSyncTestLocker{} }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
	}...)
	if err == nil {
		t.Fatal("expected prepare failure for missing module")
	}
	if duringPrepare != "missing_prepare: resolving modules" {
		t.Fatalf("spinner during prepare = %q, want resolving modules", duringPrepare)
	}
	if got := ticker.Message(); got != "outer: installing batch" {
		t.Fatalf("caller spinner after prepare failure = %q, want restored outer message", got)
	}
}

func TestEnsureInstallPrepareProgressTickerNilContext(t *testing.T) {
	t.Parallel()
	gotCtx, ticker, owns := ensureInstallPrepareProgressTicker(nil)
	if gotCtx == nil || ticker != nil || owns {
		t.Fatalf("nil ctx: gotCtx=%v ticker=%v owns=%v", gotCtx != nil, ticker != nil, owns)
	}
}

func TestInstallModuleOwnsPrepareTickerWhenOnlyProgressLine(t *testing.T) {
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
		Name:       "solo_owned_ticker",
		Version:    "1.0.0",
		Path:       filepath.Join(modulesPath, "solo_owned_ticker"),
		DependsStr: dependsRaw,
	}
	origin := &countingPrefetchOriginCoordinator{
		modules: map[string]*meta.Module{"solo_owned_ticker": mod},
	}
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	locker := &moduleIndexSyncTestLocker{}
	// ProgressLine without ticker → InstallModule owns prepare ticker (Stop deferred).
	ctx := logutil.WithProgressLine(context.Background(), logutil.NewProgressLine(&bytes.Buffer{}))
	err = InstallModule(ctx, runtimeScope, nil, InstallModuleRequest{
		Input:        "solo_owned_ticker",
		SkipWebShell: true,
	}, []Option{
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return origin }),
	}...)
	if err == nil {
		t.Fatal("expected InstallModule to fail without executor/full install")
	}
	if len(origin.fetches) == 0 {
		t.Fatal("expected prefetch fetch")
	}
}

func TestPrepareInstallAliasesPrefetchInstallModules(t *testing.T) {
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

	// PrepareInstall runs BuildPlan with SkipWebShell; web must not be resolved.
	// Install then fails without a JS executor (expected harness limit).
	err = InstallModule(context.Background(), runtimeScope, nil, InstallModuleRequest{
		Input:        "solo_skip_web",
		SkipWebShell: true,
	}, opts...)
	if err == nil {
		t.Fatal("expected InstallModule to fail without executor/full install")
	}
	for _, name := range origin.fetches {
		if name == "web" {
			t.Fatalf("SkipWebShell should not resolve web shell, fetches=%v", origin.fetches)
		}
	}
	if len(origin.fetches) == 0 || origin.fetches[0] != "solo_skip_web" {
		t.Fatalf("fetches=%v, want solo_skip_web first", origin.fetches)
	}
}
