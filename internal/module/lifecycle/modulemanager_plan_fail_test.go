// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/artifact/pipeline"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func planFailTestManager(t *testing.T, moduleName string) (*ModuleManager, *meta.Module) {
	t.Helper()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	modulesPath := t.TempDir()
	defaultChoysumPath := t.TempDir()
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	runtimeScope.cfg = &config.Config{
		ModulesPath:        modulesPath,
		DistPath:           filepath.Join(defaultChoysumPath, "dist"),
		TmpPath:            filepath.Join(defaultChoysumPath, "tmp"),
		DefaultChoysumPath: defaultChoysumPath,
	}
	locker := &moduleIndexSyncTestLocker{}
	manager := NewModuleManager(runtimeScope, &moduleManagerNoopScriptExecutor{}, WithLockerFactory(func(scope.Scope) statepkg.Locker {
		return locker
	}))
	manager.bootstrapOnce.Do(func() {})

	mod := &meta.Module{
		Name:    moduleName,
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    filepath.Join(modulesPath, moduleName),
	}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	return manager, mod
}

func withProgressTicker(t *testing.T) (context.Context, *logutil.ProgressTicker) {
	t.Helper()
	line := logutil.NewProgressLine(&bytes.Buffer{})
	ticker := logutil.NewProgressTicker(line, logutil.ProgressTickerOptions{})
	t.Cleanup(ticker.Stop)
	ctx := logutil.WithProgressTicker(logutil.WithProgressLine(context.Background(), line), ticker)
	return ctx, ticker
}

func TestModuleManagerUninstallClearsSpinnerWhenPlanFails(t *testing.T) {
	manager, _ := planFailTestManager(t, "solo_unplan")
	ctx, ticker := withProgressTicker(t)

	prev := buildModuleOperationPlan
	buildModuleOperationPlan = func(context.Context, plan.OpType, *meta.Module, plan.Resolver, ...plan.BuildOption) (plan.Plan, error) {
		return plan.Plan{}, errors.New("forced uninstall plan failure")
	}
	t.Cleanup(func() { buildModuleOperationPlan = prev })

	err := manager.Uninstall(ctx, "solo_unplan")
	if err == nil || !strings.Contains(err.Error(), "forced uninstall plan failure") {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if got := ticker.Message(); got != "" {
		t.Fatalf("spinner message after failure = %q, want empty", got)
	}
}

func TestModuleManagerUpgradeClearsSpinnerWhenPlanFails(t *testing.T) {
	manager, _ := planFailTestManager(t, "solo_upplan")
	ctx, ticker := withProgressTicker(t)

	prev := buildModuleOperationPlan
	buildModuleOperationPlan = func(context.Context, plan.OpType, *meta.Module, plan.Resolver, ...plan.BuildOption) (plan.Plan, error) {
		return plan.Plan{}, errors.New("forced upgrade plan failure")
	}
	t.Cleanup(func() { buildModuleOperationPlan = prev })

	err := manager.Upgrade(ctx, "solo_upplan")
	if err == nil || !strings.Contains(err.Error(), "forced upgrade plan failure") {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if got := ticker.Message(); got != "" {
		t.Fatalf("spinner message after failure = %q, want empty", got)
	}
}

func TestModuleManagerUpgradeClearsSpinnerWhenListInstalledAppsFails(t *testing.T) {
	manager, _ := planFailTestManager(t, "core")
	ctx, ticker := withProgressTicker(t)

	prev := buildModuleOperationPlan
	buildModuleOperationPlan = func(context.Context, plan.OpType, *meta.Module, plan.Resolver, ...plan.BuildOption) (plan.Plan, error) {
		return plan.Plan{Op: plan.OpUpgrade, ModuleOrder: []string{"core"}}, nil
	}
	t.Cleanup(func() { buildModuleOperationPlan = prev })

	prevList := listInstalledApplicationNames
	listInstalledApplicationNames = func(*ModuleManager, context.Context) ([]string, error) {
		// Planning spinner is still active here; prove the error path clears it.
		ticker.SetMessage("forced list-apps spinner")
		return nil, errors.New("list installed apps: forced")
	}
	t.Cleanup(func() { listInstalledApplicationNames = prevList })

	err := manager.Upgrade(ctx, "core")
	if err == nil || !strings.Contains(err.Error(), "list installed apps") {
		t.Fatalf("Upgrade() error = %v, want list installed apps", err)
	}
	if got := ticker.Message(); got != "" {
		t.Fatalf("spinner message after list-apps failure = %q, want empty", got)
	}
}

func TestModuleManagerUpgradeClearsSpinnerWhenFinalizeLoadFails(t *testing.T) {
	manager, _ := planFailTestManager(t, "solo_finload")
	ctx, ticker := withProgressTicker(t)

	prevPlan := buildModuleOperationPlan
	buildModuleOperationPlan = func(context.Context, plan.OpType, *meta.Module, plan.Resolver, ...plan.BuildOption) (plan.Plan, error) {
		return plan.Plan{Op: plan.OpUpgrade, ModuleOrder: []string{"solo_finload"}}, nil
	}
	t.Cleanup(func() { buildModuleOperationPlan = prevPlan })

	prevExec := executeModulePipeline
	executeModulePipeline = func(context.Context, plan.Plan, *meta.Module, pipeline.Callbacks) error {
		return nil
	}
	t.Cleanup(func() { executeModulePipeline = prevExec })

	prevPhase := phaseEndModuleCandidates
	phaseEndModuleCandidates = func(plan.OpType, []string, []string, *opContext) []string {
		return []string{"solo_finload"}
	}
	t.Cleanup(func() { phaseEndModuleCandidates = prevPhase })

	prevLoad := loadModuleForLifecycle
	loadModuleForLifecycle = func(*ModuleManager, string) (*meta.Module, error) {
		return nil, errors.New("forced finalize load failure")
	}
	t.Cleanup(func() { loadModuleForLifecycle = prevLoad })

	err := manager.Upgrade(ctx, "solo_finload")
	if err == nil || !strings.Contains(err.Error(), "forced finalize load failure") {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if got := ticker.Message(); got != "" {
		t.Fatalf("spinner message after finalize failure = %q, want empty", got)
	}
}

func TestInstallHeartbeatStageLabelUsedUnderShortThreshold(t *testing.T) {
	prevThresh, prevInterval := setModuleOpStageHeartbeatConfig(time.Millisecond, time.Millisecond)
	t.Cleanup(func() { setModuleOpStageHeartbeatConfig(prevThresh, prevInterval) })

	manager, root := planFailTestManager(t, "solo_hb")
	var logBuf bytes.Buffer
	runtimeScope, ok := manager.runtimeScope.(*moduleIndexSyncTestScope)
	if !ok {
		t.Fatalf("runtimeScope type = %T, want *moduleIndexSyncTestScope", manager.runtimeScope)
	}
	runtimeScope.logger = slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	root.Status = meta.Uninstalled
	root.DependsStr = []byte(`[]`)
	if err := manager.runtimeScope.Session().DB.Save(root).Error; err != nil {
		t.Fatalf("save module: %v", err)
	}

	prevPlan := buildModuleOperationPlan
	buildModuleOperationPlan = func(ctx context.Context, op plan.OpType, rootMod *meta.Module, r plan.Resolver, opts ...plan.BuildOption) (plan.Plan, error) {
		time.Sleep(25 * time.Millisecond)
		return plan.Plan{Op: plan.OpInstall, ModuleOrder: []string{"solo_hb"}}, nil
	}
	t.Cleanup(func() { buildModuleOperationPlan = prevPlan })

	prevExec := executeModulePipeline
	executeModulePipeline = func(context.Context, plan.Plan, *meta.Module, pipeline.Callbacks) error {
		return nil
	}
	t.Cleanup(func() { executeModulePipeline = prevExec })

	origin := &countingPrefetchOriginCoordinator{modules: map[string]*meta.Module{"solo_hb": root}}
	manager.originCoordinatorFactory = func(scope.Scope) OriginCoordinator { return origin }

	ctx := WithPrefetchedInstallModules(context.Background(), map[string]*meta.Module{"solo_hb": root})
	_ = manager.Install(ctx, "solo_hb")
	out := logBuf.String()
	if !strings.Contains(out, `"msg":"module operation stage heartbeat"`) ||
		!strings.Contains(out, `"stage_label":"Planning module operation"`) {
		t.Fatalf("planning heartbeat missing expected label; log: %s", out)
	}
}
