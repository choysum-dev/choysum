// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pipeline

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	planner "github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/meta"
	"golang.org/x/exp/errors/fmt"
)

type Callbacks struct {
	Logger *slog.Logger
	// OnInstallProgress receives install-loop progress events for rendering
	// interactive UI feedback (for example a single-line spinner).
	OnInstallProgress func(progress ModuleInstallProgress)
	// OnProgress receives unified stage progress events across pipeline execution.
	OnProgress func(event ProgressEvent)

	ResolveInstallModuleFromOrigin func(ctx context.Context, name string) (*meta.IrModule, error)
	ResolveInstalledModule         func(name string) (*meta.IrModule, error)

	Install   func(module *meta.IrModule) error
	Uninstall func(module *meta.IrModule) error
	Upgrade   func(module *meta.IrModule) error

	// Targets returns root directories for app-stage outputs.
	// distAppDir is typically: dist/apps/<app> (may be empty to skip per-app dist publishing).
	// proto/web/service dirs are typically:
	// <default_choysum_path>/generated/{proto,web,service}/<app>
	AppTargets func(appName string) (distAppDir string, modulesTargets ModulesAppTargets, err error)

	// BundlesTarget returns the root directory for backend bundles (typically dist/bundles).
	BundlesTarget func() (distBundlesDir string, err error)

	// WebTarget returns the root directory for global web assets build (typically dist/web).
	WebTarget func() (distWebDir string, err error)

	// DistManifestTarget returns the manifest file path (typically dist/<distmanifest.DistManifestFileName>).
	DistManifestTarget func() (distManifestPath string, err error)
	// WriteDistManifest writes the dist manifest to the provided staging file path.
	// The caller is responsible for producing correct content based on the final installed state.
	WriteDistManifest func(ctx context.Context, distManifestStagingPath string) error

	// App stage
	BuildBackendApp     func(ctx context.Context, appName string, distAppStagingDir string) error
	GenerateApp         func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error
	BuildBackendBundles func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error
	GlobalWebBuild      func(ctx context.Context, distWebStagingDir string) error
}

// ModuleInstallProgressStage labels a discrete phase inside a module
// install pipeline execution.
type ModuleInstallProgressStage string

const (
	ModuleInstallProgressStageStarted   ModuleInstallProgressStage = "started"
	ModuleInstallProgressStageCompleted ModuleInstallProgressStage = "completed"
	ModuleInstallProgressStageFailed    ModuleInstallProgressStage = "failed"
)

// ModuleInstallProgress carries progress information for a single module
// inside an install/uninstall/upgrade pipeline run.
type ModuleInstallProgress struct {
	Current  int
	Total    int
	Module   string
	Stage    ModuleInstallProgressStage
	Duration time.Duration
	Err      error
}

// ProgressStage labels coarse-grained pipeline stages for UX progress rendering.
type ProgressStage string

const (
	ProgressStageModuleInstallStarted     ProgressStage = "module.install.started"
	ProgressStageModuleInstallCompleted   ProgressStage = "module.install.completed"
	ProgressStageModuleInstallFailed      ProgressStage = "module.install.failed"
	ProgressStageModuleUninstallStarted   ProgressStage = "module.uninstall.started"
	ProgressStageModuleUninstallCompleted ProgressStage = "module.uninstall.completed"
	ProgressStageModuleUninstallFailed    ProgressStage = "module.uninstall.failed"
	ProgressStageModuleUpgradeStarted     ProgressStage = "module.upgrade.started"
	ProgressStageModuleUpgradeCompleted   ProgressStage = "module.upgrade.completed"
	ProgressStageModuleUpgradeFailed      ProgressStage = "module.upgrade.failed"
	ProgressStageAppStageStarted          ProgressStage = "app.stage.started"
	ProgressStageAppStageCompleted        ProgressStage = "app.stage.completed"
	ProgressStageAppBuildStarted          ProgressStage = "app.build.started"
	ProgressStageAppBuildCompleted        ProgressStage = "app.build.completed"
	ProgressStageAppGenerateStarted       ProgressStage = "app.generate.started"
	ProgressStageAppGenerateCompleted     ProgressStage = "app.generate.completed"
	ProgressStageBundlesBuildStarted      ProgressStage = "bundles.build.started"
	ProgressStageBundlesBuildCompleted    ProgressStage = "bundles.build.completed"
	ProgressStageWebBuildStarted          ProgressStage = "web.build.started"
	ProgressStageWebBuildCompleted        ProgressStage = "web.build.completed"
)

// ProgressEvent carries stage transition details for user-facing progress views.
type ProgressEvent struct {
	Stage    ProgressStage
	Current  int
	Total    int
	App      string
	Module   string
	Duration time.Duration
	Err      error
}

// ModulesAppTargets describes per-application module output directories.
// Under the new layout, these directories are staged and committed
// independently so each can be swapped atomically.
type ModulesAppTargets struct {
	ProtoDir        string
	WebDir          string
	ServiceDir      string
	RuntimeProtoDir string
}

const infoSummaryNameListLimit = 8

func summarizeInfoNames(values []string) (int, []string) {
	if len(values) == 0 {
		return 0, nil
	}
	seen := make(map[string]struct{}, len(values))
	compact := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		compact = append(compact, value)
	}
	if len(compact) == 0 {
		return 0, nil
	}
	if len(compact) > infoSummaryNameListLimit {
		return len(compact), nil
	}
	return len(compact), compact
}

func mirrorDir(srcDir string, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dstDir, 0o755)
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func Execute(ctx context.Context, plan planner.Plan, root *meta.IrModule, cb Callbacks) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := staging.OpIDFromContext(ctx); !ok {
		ctx = staging.WithOpID(ctx, staging.NewOpID())
	}
	logger := cb.Logger
	normalizeLogAttrs := func(attrs []any) []any {
		if len(attrs) == 0 {
			return nil
		}
		normalized := make([]any, 0, len(attrs))
		for i := 0; i < len(attrs); i++ {
			key, ok := attrs[i].(string)
			if !ok || i+1 >= len(attrs) {
				normalized = append(normalized, attrs[i])
				continue
			}

			value := attrs[i+1]
			if key == "duration" {
				key = "duration_ms"
				if duration, ok := value.(time.Duration); ok {
					value = duration.Milliseconds()
				}
			}

			normalized = append(normalized, key, value)
			i++
		}
		return normalized
	}
	logStep := func(level slog.Level, msg string, attrs ...any) {
		if logger == nil {
			return
		}
		logger.Log(ctx, level, msg, normalizeLogAttrs(attrs)...)
	}
	emitProgress := func(event ProgressEvent) {
		if cb.OnProgress == nil {
			return
		}
		cb.OnProgress(event)
	}

	checkCtx := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	modulesCount, infoModules := summarizeInfoNames(plan.ModuleOrder)
	logModuleStageStarted := func() time.Time {
		if modulesCount == 0 {
			return time.Time{}
		}
		attrs := []any{"modules_count", modulesCount}
		if len(infoModules) > 0 {
			attrs = append(attrs, "modules", infoModules)
		}
		logStep(slog.LevelDebug, "pipeline module stage started", attrs...)
		return time.Now()
	}
	logModuleStageCompleted := func(started time.Time) {
		if modulesCount == 0 || started.IsZero() {
			return
		}
		attrs := []any{"modules_count", modulesCount}
		if len(infoModules) > 0 {
			attrs = append(attrs, "modules", infoModules)
		}
		attrs = append(attrs, "duration", time.Since(started))
		logStep(slog.LevelDebug, "pipeline module stage completed", attrs...)
	}

	// generateModulesForApp stages and commits api outputs (proto/web/service/runtimeProto) for a single app.
	// This is used during install so downstream module builds can resolve generated
	// side-effect entry imports even if outputs were deleted before install starts.
	generateModulesForApp := func(app string) error {
		app = strings.TrimSpace(app)
		if app == "" {
			return nil
		}
		// "web" application is handled by the global web build stage.
		if strings.EqualFold(app, "web") {
			return nil
		}
		if err := checkCtx(); err != nil {
			return err
		}
		if cb.AppTargets == nil {
			return fmt.Errorf("AppTargets callback is required")
		}
		if cb.GenerateApp == nil {
			return fmt.Errorf("GenerateApp callback is required")
		}

		_, modulesTargets, err := cb.AppTargets(app)
		if err != nil {
			return err
		}
		if strings.TrimSpace(modulesTargets.ProtoDir) == "" || strings.TrimSpace(modulesTargets.WebDir) == "" || strings.TrimSpace(modulesTargets.ServiceDir) == "" {
			return fmt.Errorf("AppTargets callback returned empty proto/web/service dir for app %s", app)
		}

		prepareStarted := time.Now()
		protoStage, err := staging.PrepareDir(ctx, modulesTargets.ProtoDir)
		if err != nil {
			return err
		}
		webStage, err := staging.PrepareDir(ctx, modulesTargets.WebDir)
		if err != nil {
			_ = protoStage.Abort()
			return err
		}
		serviceStage, err := staging.PrepareDir(ctx, modulesTargets.ServiceDir)
		if err != nil {
			_ = protoStage.Abort()
			_ = webStage.Abort()
			return err
		}
		var runtimeProtoStage *staging.DirStaging
		if strings.TrimSpace(modulesTargets.RuntimeProtoDir) != "" {
			runtimeProtoStage, err = staging.PrepareDir(ctx, modulesTargets.RuntimeProtoDir)
			if err != nil {
				_ = protoStage.Abort()
				_ = webStage.Abort()
				_ = serviceStage.Abort()
				return err
			}
		}
		logStep(slog.LevelDebug, "pipeline prepared modules staging", "app", app, "duration", time.Since(prepareStarted), "proto_target", modulesTargets.ProtoDir, "web_target", modulesTargets.WebDir, "service_target", modulesTargets.ServiceDir, "runtime_proto_target", modulesTargets.RuntimeProtoDir)

		genStarted := time.Now()
		modulesStaging := ModulesAppTargets{ProtoDir: protoStage.StagingDir, WebDir: webStage.StagingDir, ServiceDir: serviceStage.StagingDir}
		if runtimeProtoStage != nil {
			modulesStaging.RuntimeProtoDir = runtimeProtoStage.StagingDir
		}
		if err := cb.GenerateApp(ctx, app, modulesStaging, ""); err != nil {
			_ = protoStage.Abort()
			_ = webStage.Abort()
			_ = serviceStage.Abort()
			if runtimeProtoStage != nil {
				_ = runtimeProtoStage.Abort()
			}
			logStep(slog.LevelError, "pipeline modules generation failed", "app", app, "duration", time.Since(genStarted), "error", err)
			return err
		}
		if runtimeProtoStage != nil {
			if err := mirrorDir(protoStage.StagingDir, runtimeProtoStage.StagingDir); err != nil {
				_ = protoStage.Abort()
				_ = webStage.Abort()
				_ = serviceStage.Abort()
				_ = runtimeProtoStage.Abort()
				logStep(slog.LevelError, "pipeline runtime proto sync failed", "app", app, "duration", time.Since(genStarted), "error", err)
				return err
			}
		}
		logStep(slog.LevelDebug, "pipeline modules generated", "app", app, "duration", time.Since(genStarted))

		committed := []*staging.DirStaging{}
		rollbackCommitted := func() {
			for i := len(committed) - 1; i >= 0; i-- {
				s := committed[i]
				started := time.Now()
				err := s.Rollback()
				if err != nil {
					logStep(slog.LevelWarn, "pipeline rollback failed", "target", s.TargetDir, "duration", time.Since(started), "error", err)
				} else {
					logStep(slog.LevelWarn, "pipeline rolled back", "target", s.TargetDir, "duration", time.Since(started))
				}
			}
			committed = nil
		}
		finalizeCommitted := func() {
			for _, s := range committed {
				started := time.Now()
				_ = s.Finalize()
				logStep(slog.LevelDebug, "pipeline finalized", "target", s.TargetDir, "duration", time.Since(started))
			}
			committed = nil
		}

		commitOne := func(label string, s *staging.DirStaging) error {
			started := time.Now()
			if err := s.CommitKeepBackup(); err != nil {
				logStep(slog.LevelError, "pipeline modules commit failed", "app", app, "stage", label, "duration", time.Since(started), "error", err)
				rollbackCommitted()
				return err
			}
			committed = append(committed, s)
			logStep(slog.LevelDebug, "pipeline modules committed", "app", app, "stage", label, "duration", time.Since(started))
			return nil
		}

		// Commit proto/runtimeProto/web/service. If commit fails, roll back already-committed stages.
		if err := commitOne("proto", protoStage); err != nil {
			_ = webStage.Abort()
			_ = serviceStage.Abort()
			if runtimeProtoStage != nil {
				_ = runtimeProtoStage.Abort()
			}
			return err
		}
		if runtimeProtoStage != nil {
			if err := commitOne("runtime_proto", runtimeProtoStage); err != nil {
				_ = webStage.Abort()
				_ = serviceStage.Abort()
				return err
			}
		}
		if err := commitOne("web", webStage); err != nil {
			_ = serviceStage.Abort()
			return err
		}
		if err := commitOne("service", serviceStage); err != nil {
			return err
		}
		finalizeCommitted()
		return nil
	}

	runAppStage := func() error {
		stageCtx := ctx
		if err := checkCtx(); err != nil {
			return err
		}

		if len(plan.AffectedApps) > 0 && cb.AppTargets == nil {
			return fmt.Errorf("AppTargets callback is required")
		}
		// BuildBackendApp is conditionally required depending on whether per-app dist targets are provided.
		if len(plan.AffectedApps) > 0 && cb.GenerateApp == nil {
			return fmt.Errorf("GenerateApp callback is required")
		}

		seen := map[string]bool{}
		apps := append([]string(nil), plan.AffectedApps...)
		sort.Strings(apps)
		appsCount, infoApps := summarizeInfoNames(apps)
		appStageStarted := time.Now()
		if appsCount > 0 {
			attrs := []any{"apps_count", appsCount}
			if len(infoApps) > 0 {
				attrs = append(attrs, "apps", infoApps)
			}
			logStep(slog.LevelDebug, "pipeline app stage started", attrs...)
			emitProgress(ProgressEvent{Stage: ProgressStageAppStageStarted, Total: appsCount})
		}

		type appStages struct {
			app               string
			distStage         *staging.DirStaging
			protoStage        *staging.DirStaging
			runtimeProtoStage *staging.DirStaging
			webStage          *staging.DirStaging
			serviceStage      *staging.DirStaging
		}
		var stages []appStages
		var committed []*staging.DirStaging
		var committedManifest *staging.FileStaging

		abortStages := func(s appStages) {
			if s.distStage != nil {
				_ = s.distStage.Abort()
			}
			if s.protoStage != nil {
				_ = s.protoStage.Abort()
			}
			if s.runtimeProtoStage != nil {
				_ = s.runtimeProtoStage.Abort()
			}
			if s.webStage != nil {
				_ = s.webStage.Abort()
			}
			if s.serviceStage != nil {
				_ = s.serviceStage.Abort()
			}
		}

		rollbackCommitted := func() {
			if committedManifest != nil {
				started := time.Now()
				err := committedManifest.Rollback()
				if err != nil {
					logStep(slog.LevelWarn, "pipeline rollback failed", "target", committedManifest.TargetPath, "duration", time.Since(started), "error", err)
				} else {
					logStep(slog.LevelWarn, "pipeline rolled back", "target", committedManifest.TargetPath, "duration", time.Since(started))
				}
				committedManifest = nil
			}
			for i := len(committed) - 1; i >= 0; i-- {
				s := committed[i]
				started := time.Now()
				err := s.Rollback()
				if err != nil {
					logStep(slog.LevelWarn, "pipeline rollback failed", "target", s.TargetDir, "duration", time.Since(started), "error", err)
				} else {
					logStep(slog.LevelWarn, "pipeline rolled back", "target", s.TargetDir, "duration", time.Since(started))
				}
			}
			committed = nil
		}

		finalizeCommitted := func() {
			if committedManifest != nil {
				started := time.Now()
				_ = committedManifest.Finalize()
				logStep(slog.LevelDebug, "pipeline finalized", "target", committedManifest.TargetPath, "duration", time.Since(started))
				committedManifest = nil
			}
			for _, s := range committed {
				started := time.Now()
				_ = s.Finalize()
				logStep(slog.LevelDebug, "pipeline finalized", "target", s.TargetDir, "duration", time.Since(started))
			}
			committed = nil
		}

		commitManifest := func() error {
			if cb.WriteDistManifest == nil {
				return nil
			}
			if cb.DistManifestTarget == nil {
				return fmt.Errorf("DistManifestTarget callback is required")
			}
			path, err := cb.DistManifestTarget()
			if err != nil {
				return err
			}
			manifestStage, err := staging.PrepareFile(stageCtx, path)
			if err != nil {
				return err
			}
			defer manifestStage.Abort()
			writeStarted := time.Now()
			if err := cb.WriteDistManifest(stageCtx, manifestStage.StagingPath); err != nil {
				logStep(slog.LevelError, "pipeline dist manifest write failed", "duration", time.Since(writeStarted), "error", err)
				return err
			}
			logStep(slog.LevelDebug, "pipeline dist manifest written", "duration", time.Since(writeStarted))

			commitStarted := time.Now()
			if err := manifestStage.CommitKeepBackup(); err != nil {
				logStep(slog.LevelError, "pipeline dist manifest commit failed", "duration", time.Since(commitStarted), "error", err)
				return err
			}
			committedManifest = manifestStage
			logStep(slog.LevelDebug, "pipeline dist manifest committed", "duration", time.Since(commitStarted))
			return nil
		}

		// Phase 1: prepare + build/generate into staging dirs (no commits).
		appIndex := 0
		for _, app := range apps {
			if err := checkCtx(); err != nil {
				return err
			}
			app = strings.TrimSpace(app)
			if app == "" || seen[app] {
				continue
			}
			seen[app] = true
			appIndex++

			distAppDir, modulesTargets, err := cb.AppTargets(app)
			if err != nil {
				return err
			}

			prepareStarted := time.Now()
			var distStage *staging.DirStaging
			if strings.TrimSpace(distAppDir) != "" {
				if cb.BuildBackendApp == nil {
					return fmt.Errorf("BuildBackendApp callback is required")
				}
				distStage, err = staging.PrepareDir(stageCtx, distAppDir)
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(modulesTargets.ProtoDir) == "" || strings.TrimSpace(modulesTargets.WebDir) == "" || strings.TrimSpace(modulesTargets.ServiceDir) == "" {
				if distStage != nil {
					_ = distStage.Abort()
				}
				return fmt.Errorf("AppTargets callback returned empty proto/web/service dir for app %s", app)
			}

			protoStage, err := staging.PrepareDir(stageCtx, modulesTargets.ProtoDir)
			if err != nil {
				if distStage != nil {
					_ = distStage.Abort()
				}
				return err
			}
			webStage, err := staging.PrepareDir(stageCtx, modulesTargets.WebDir)
			if err != nil {
				if distStage != nil {
					_ = distStage.Abort()
				}
				_ = protoStage.Abort()
				return err
			}
			serviceStage, err := staging.PrepareDir(stageCtx, modulesTargets.ServiceDir)
			if err != nil {
				if distStage != nil {
					_ = distStage.Abort()
				}
				_ = protoStage.Abort()
				_ = webStage.Abort()
				return err
			}
			var runtimeProtoStage *staging.DirStaging
			if strings.TrimSpace(modulesTargets.RuntimeProtoDir) != "" {
				runtimeProtoStage, err = staging.PrepareDir(stageCtx, modulesTargets.RuntimeProtoDir)
				if err != nil {
					if distStage != nil {
						_ = distStage.Abort()
					}
					_ = protoStage.Abort()
					_ = webStage.Abort()
					_ = serviceStage.Abort()
					return err
				}
			}
			logStep(
				slog.LevelDebug,
				"pipeline prepared app staging",
				"app",
				app,
				"duration",
				time.Since(prepareStarted),
				"dist_target",
				distAppDir,
				"proto_target",
				modulesTargets.ProtoDir,
				"runtime_proto_target",
				modulesTargets.RuntimeProtoDir,
				"web_target",
				modulesTargets.WebDir,
				"service_target",
				modulesTargets.ServiceDir,
			)

			stages = append(stages, appStages{app: app, distStage: distStage, protoStage: protoStage, runtimeProtoStage: runtimeProtoStage, webStage: webStage, serviceStage: serviceStage})

			if distStage != nil {
				emitProgress(ProgressEvent{Stage: ProgressStageAppBuildStarted, App: app, Current: appIndex, Total: appsCount})
				buildStarted := time.Now()
				if err := cb.BuildBackendApp(stageCtx, app, distStage.StagingDir); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					emitProgress(ProgressEvent{Stage: ProgressStageAppBuildCompleted, App: app, Current: appIndex, Total: appsCount, Duration: time.Since(buildStarted), Err: err})
					logStep(slog.LevelError, "pipeline backend build failed", "app", app, "duration", time.Since(buildStarted), "error", err)
					return err
				}
				emitProgress(ProgressEvent{Stage: ProgressStageAppBuildCompleted, App: app, Current: appIndex, Total: appsCount, Duration: time.Since(buildStarted)})
				logStep(slog.LevelDebug, "pipeline backend built", "app", app, "duration", time.Since(buildStarted))
			}

			emitProgress(ProgressEvent{Stage: ProgressStageAppGenerateStarted, App: app, Current: appIndex, Total: appsCount})
			genStarted := time.Now()
			distStagingDir := ""
			if distStage != nil {
				distStagingDir = distStage.StagingDir
			}
			modulesStaging := ModulesAppTargets{ProtoDir: protoStage.StagingDir, WebDir: webStage.StagingDir, ServiceDir: serviceStage.StagingDir}
			if runtimeProtoStage != nil {
				modulesStaging.RuntimeProtoDir = runtimeProtoStage.StagingDir
			}
			if err := cb.GenerateApp(stageCtx, app, modulesStaging, distStagingDir); err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				emitProgress(ProgressEvent{Stage: ProgressStageAppGenerateCompleted, App: app, Current: appIndex, Total: appsCount, Duration: time.Since(genStarted), Err: err})
				logStep(slog.LevelError, "pipeline app generation failed", "app", app, "duration", time.Since(genStarted), "error", err)
				return err
			}
			if runtimeProtoStage != nil {
				if err := mirrorDir(protoStage.StagingDir, runtimeProtoStage.StagingDir); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					emitProgress(ProgressEvent{Stage: ProgressStageAppGenerateCompleted, App: app, Current: appIndex, Total: appsCount, Duration: time.Since(genStarted), Err: err})
					logStep(slog.LevelError, "pipeline runtime proto sync failed", "app", app, "duration", time.Since(genStarted), "error", err)
					return err
				}
			}
			emitProgress(ProgressEvent{Stage: ProgressStageAppGenerateCompleted, App: app, Current: appIndex, Total: appsCount, Duration: time.Since(genStarted)})
			logStep(slog.LevelDebug, "pipeline app generated", "app", app, "duration", time.Since(genStarted))
		}
		if appsCount > 0 {
			attrs := []any{"apps_count", appsCount}
			if len(infoApps) > 0 {
				attrs = append(attrs, "apps", infoApps)
			}
			attrs = append(attrs, "duration", time.Since(appStageStarted))
			logStep(slog.LevelDebug, "pipeline app stage completed", attrs...)
			emitProgress(ProgressEvent{Stage: ProgressStageAppStageCompleted, Total: appsCount, Duration: time.Since(appStageStarted)})
		}

		// Phase 2: commit all apps after all succeeded.
		// Commit dist first (runtime-critical), then bundles (if any), then modules.
		for _, s := range stages {
			if err := checkCtx(); err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				rollbackCommitted()
				return err
			}
			if s.distStage != nil {
				commitStarted := time.Now()
				if err := s.distStage.CommitKeepBackup(); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					logStep(slog.LevelError, "pipeline dist commit failed", "app", s.app, "duration", time.Since(commitStarted), "error", err)
					rollbackCommitted()
					return err
				}
				committed = append(committed, s.distStage)
				logStep(slog.LevelDebug, "pipeline dist committed", "app", s.app, "duration", time.Since(commitStarted))
			}
		}

		if cb.BuildBackendBundles != nil {
			if cb.BundlesTarget == nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				rollbackCommitted()
				return fmt.Errorf("BundlesTarget callback is required")
			}
			distBundlesDir, err := cb.BundlesTarget()
			if err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				rollbackCommitted()
				return err
			}
			prepareBundlesStarted := time.Now()
			bundlesStage, err := staging.PrepareDir(stageCtx, distBundlesDir)
			if err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				rollbackCommitted()
				return err
			}
			defer bundlesStage.Abort()
			logStep(slog.LevelDebug, "pipeline prepared bundles staging", "duration", time.Since(prepareBundlesStarted), "bundles_target", distBundlesDir)

			affectedProtoStaging := map[string]string{}
			for _, s := range stages {
				if s.protoStage != nil {
					affectedProtoStaging[s.app] = s.protoStage.StagingDir
				}
			}
			emitProgress(ProgressEvent{Stage: ProgressStageBundlesBuildStarted})
			bundlesBuildStarted := time.Now()
			if err := cb.BuildBackendBundles(stageCtx, bundlesStage.StagingDir, affectedProtoStaging); err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				emitProgress(ProgressEvent{Stage: ProgressStageBundlesBuildCompleted, Duration: time.Since(bundlesBuildStarted), Err: err})
				logStep(slog.LevelError, "pipeline bundles build failed", "duration", time.Since(bundlesBuildStarted), "error", err)
				rollbackCommitted()
				return err
			}
			emitProgress(ProgressEvent{Stage: ProgressStageBundlesBuildCompleted, Duration: time.Since(bundlesBuildStarted)})
			logStep(slog.LevelInfo, "pipeline bundles built", "duration", time.Since(bundlesBuildStarted))

			bundlesCommitStarted := time.Now()
			if err := bundlesStage.CommitKeepBackup(); err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				logStep(slog.LevelError, "pipeline bundles commit failed", "duration", time.Since(bundlesCommitStarted), "error", err)
				rollbackCommitted()
				return err
			}
			committed = append(committed, bundlesStage)
			logStep(slog.LevelDebug, "pipeline bundles committed", "duration", time.Since(bundlesCommitStarted))
		}
		for _, s := range stages {
			if err := checkCtx(); err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				rollbackCommitted()
				return err
			}
			if s.protoStage != nil {
				commitStarted := time.Now()
				if err := s.protoStage.CommitKeepBackup(); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					logStep(slog.LevelError, "pipeline proto commit failed", "app", s.app, "duration", time.Since(commitStarted), "error", err)
					rollbackCommitted()
					return err
				}
				committed = append(committed, s.protoStage)
				logStep(slog.LevelDebug, "pipeline proto committed", "app", s.app, "duration", time.Since(commitStarted))
			}
			if s.runtimeProtoStage != nil {
				commitStarted := time.Now()
				if err := s.runtimeProtoStage.CommitKeepBackup(); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					logStep(slog.LevelError, "pipeline runtime proto commit failed", "app", s.app, "duration", time.Since(commitStarted), "error", err)
					rollbackCommitted()
					return err
				}
				committed = append(committed, s.runtimeProtoStage)
				logStep(slog.LevelDebug, "pipeline runtime proto committed", "app", s.app, "duration", time.Since(commitStarted))
			}
			if s.webStage != nil {
				commitStarted := time.Now()
				if err := s.webStage.CommitKeepBackup(); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					logStep(slog.LevelError, "pipeline web commit failed", "app", s.app, "duration", time.Since(commitStarted), "error", err)
					rollbackCommitted()
					return err
				}
				committed = append(committed, s.webStage)
				logStep(slog.LevelDebug, "pipeline web committed", "app", s.app, "duration", time.Since(commitStarted))
			}
			if s.serviceStage != nil {
				commitStarted := time.Now()
				if err := s.serviceStage.CommitKeepBackup(); err != nil {
					for i := len(stages) - 1; i >= 0; i-- {
						abortStages(stages[i])
					}
					logStep(slog.LevelError, "pipeline service commit failed", "app", s.app, "duration", time.Since(commitStarted), "error", err)
					rollbackCommitted()
					return err
				}
				committed = append(committed, s.serviceStage)
				logStep(slog.LevelDebug, "pipeline service committed", "app", s.app, "duration", time.Since(commitStarted))
			}
		}

		// If no global web build is needed, finalize after writing manifest.
		if !plan.NeedsGlobalWebBuild {
			if err := commitManifest(); err != nil {
				for i := len(stages) - 1; i >= 0; i-- {
					abortStages(stages[i])
				}
				rollbackCommitted()
				return err
			}
			finalizeCommitted()
			return nil
		}
		if cb.WebTarget == nil {
			rollbackCommitted()
			return fmt.Errorf("WebTarget callback is required")
		}
		if cb.GlobalWebBuild == nil {
			rollbackCommitted()
			return fmt.Errorf("GlobalWebBuild callback is required")
		}
		distWebDir, err := cb.WebTarget()
		if err != nil {
			rollbackCommitted()
			return err
		}

		prepareWebStarted := time.Now()
		webStage, err := staging.PrepareDir(stageCtx, distWebDir)
		if err != nil {
			rollbackCommitted()
			return err
		}
		defer webStage.Abort()
		logStep(slog.LevelDebug, "pipeline prepared web staging", "duration", time.Since(prepareWebStarted), "web_target", distWebDir)

		emitProgress(ProgressEvent{Stage: ProgressStageWebBuildStarted})
		webBuildStarted := time.Now()
		if err := cb.GlobalWebBuild(stageCtx, webStage.StagingDir); err != nil {
			emitProgress(ProgressEvent{Stage: ProgressStageWebBuildCompleted, Duration: time.Since(webBuildStarted), Err: err})
			logStep(slog.LevelError, "pipeline web build failed", "duration", time.Since(webBuildStarted), "error", err)
			rollbackCommitted()
			return err
		}
		emitProgress(ProgressEvent{Stage: ProgressStageWebBuildCompleted, Duration: time.Since(webBuildStarted)})
		logStep(slog.LevelInfo, "pipeline web built", "duration", time.Since(webBuildStarted))

		webCommitStarted := time.Now()
		if err := webStage.CommitKeepBackup(); err != nil {
			logStep(slog.LevelError, "pipeline web commit failed", "duration", time.Since(webCommitStarted), "error", err)
			rollbackCommitted()
			return err
		}
		committed = append(committed, webStage)
		logStep(slog.LevelDebug, "pipeline web committed", "duration", time.Since(webCommitStarted))

		if err := commitManifest(); err != nil {
			rollbackCommitted()
			return err
		}

		finalizeCommitted()
		return nil
	}

	switch plan.Op {
	case planner.OpInstall:
		if cb.ResolveInstallModuleFromOrigin == nil {
			return fmt.Errorf("ResolveInstallModuleFromOrigin callback is required for install")
		}
		if cb.Install == nil {
			return fmt.Errorf("Install callback is required for install")
		}
		moduleStageStarted := logModuleStageStarted()
		generated := map[string]bool{}
		totalModules := len(plan.ModuleOrder)
		for index, name := range plan.ModuleOrder {
			if err := checkCtx(); err != nil {
				return err
			}
			var mod *meta.IrModule
			if root != nil && name == root.Name {
				mod = root
			} else {
				m, err := cb.ResolveInstallModuleFromOrigin(ctx, name)
				if err != nil {
					return fmt.Errorf("resolve install module from origin %s: %w", name, err)
				}
				mod = m
			}
			if mod == nil {
				continue
			}
			moduleName := strings.TrimSpace(mod.Name)
			if moduleName == "" {
				moduleName = strings.TrimSpace(name)
			}
			if cb.OnInstallProgress != nil {
				cb.OnInstallProgress(ModuleInstallProgress{
					Current: index + 1,
					Total:   totalModules,
					Module:  moduleName,
					Stage:   ModuleInstallProgressStageStarted,
				})
			}
			emitProgress(ProgressEvent{Stage: ProgressStageModuleInstallStarted, Current: index + 1, Total: totalModules, Module: moduleName})
			installStarted := time.Now()
			if err := cb.Install(mod); err != nil {
				emitProgress(ProgressEvent{Stage: ProgressStageModuleInstallFailed, Current: index + 1, Total: totalModules, Module: moduleName, Duration: time.Since(installStarted), Err: err})
				if cb.OnInstallProgress != nil {
					cb.OnInstallProgress(ModuleInstallProgress{
						Current:  index + 1,
						Total:    totalModules,
						Module:   moduleName,
						Stage:    ModuleInstallProgressStageFailed,
						Duration: time.Since(installStarted),
						Err:      err,
					})
				}
				return err
			}
			installDuration := time.Since(installStarted)
			emitProgress(ProgressEvent{Stage: ProgressStageModuleInstallCompleted, Current: index + 1, Total: totalModules, Module: moduleName, Duration: installDuration})
			logStep(slog.LevelInfo, "module installed",
				"installed_module", mod.Name,
				"duration_ms", installDuration.Milliseconds(),
			)
			app := strings.TrimSpace(mod.ApplicationStr)
			if app != "" && !generated[app] {
				if err := generateModulesForApp(app); err != nil {
					if cb.OnInstallProgress != nil {
						cb.OnInstallProgress(ModuleInstallProgress{
							Current:  index + 1,
							Total:    totalModules,
							Module:   moduleName,
							Stage:    ModuleInstallProgressStageFailed,
							Duration: installDuration,
							Err:      err,
						})
					}
					return err
				}
				generated[app] = true
			}
			if cb.OnInstallProgress != nil {
				cb.OnInstallProgress(ModuleInstallProgress{
					Current:  index + 1,
					Total:    totalModules,
					Module:   moduleName,
					Stage:    ModuleInstallProgressStageCompleted,
					Duration: installDuration,
				})
			}
		}
		logModuleStageCompleted(moduleStageStarted)
		return runAppStage()

	case planner.OpUninstall:
		if cb.ResolveInstalledModule == nil {
			return fmt.Errorf("ResolveInstalledModule callback is required for uninstall")
		}
		if cb.Uninstall == nil {
			return fmt.Errorf("Uninstall callback is required for uninstall")
		}
		moduleStageStarted := logModuleStageStarted()
		totalModules := len(plan.ModuleOrder)
		for index, name := range plan.ModuleOrder {
			if err := checkCtx(); err != nil {
				return err
			}
			var mod *meta.IrModule
			if root != nil && name == root.Name {
				mod = root
			} else {
				m, err := cb.ResolveInstalledModule(name)
				if err != nil {
					return fmt.Errorf("resolve installed module %s: %w", name, err)
				}
				mod = m
			}
			if mod == nil {
				continue
			}
			moduleName := strings.TrimSpace(mod.Name)
			if moduleName == "" {
				moduleName = strings.TrimSpace(name)
			}
			uninstallStarted := time.Now()
			emitProgress(ProgressEvent{Stage: ProgressStageModuleUninstallStarted, Current: index + 1, Total: totalModules, Module: moduleName})
			if err := cb.Uninstall(mod); err != nil {
				emitProgress(ProgressEvent{Stage: ProgressStageModuleUninstallFailed, Current: index + 1, Total: totalModules, Module: moduleName, Duration: time.Since(uninstallStarted), Err: err})
				return err
			}
			emitProgress(ProgressEvent{Stage: ProgressStageModuleUninstallCompleted, Current: index + 1, Total: totalModules, Module: moduleName, Duration: time.Since(uninstallStarted)})
		}
		logModuleStageCompleted(moduleStageStarted)
		return runAppStage()

	case planner.OpUpgrade:
		if cb.ResolveInstalledModule == nil {
			return fmt.Errorf("ResolveInstalledModule callback is required for upgrade")
		}
		if cb.Upgrade == nil {
			return fmt.Errorf("Upgrade callback is required for upgrade")
		}
		moduleStageStarted := logModuleStageStarted()
		totalModules := len(plan.ModuleOrder)
		for index, name := range plan.ModuleOrder {
			if err := checkCtx(); err != nil {
				return err
			}
			var mod *meta.IrModule
			if root != nil && name == root.Name {
				mod = root
			} else {
				m, err := cb.ResolveInstalledModule(name)
				if err != nil {
					return fmt.Errorf("resolve installed module %s: %w", name, err)
				}
				mod = m
			}
			if mod == nil {
				continue
			}
			moduleName := strings.TrimSpace(mod.Name)
			if moduleName == "" {
				moduleName = strings.TrimSpace(name)
			}
			emitProgress(ProgressEvent{Stage: ProgressStageModuleUpgradeStarted, Current: index + 1, Total: totalModules, Module: moduleName})
			upgradeStarted := time.Now()
			if err := cb.Upgrade(mod); err != nil {
				emitProgress(ProgressEvent{Stage: ProgressStageModuleUpgradeFailed, Current: index + 1, Total: totalModules, Module: moduleName, Duration: time.Since(upgradeStarted), Err: err})
				return err
			}
			emitProgress(ProgressEvent{Stage: ProgressStageModuleUpgradeCompleted, Current: index + 1, Total: totalModules, Module: moduleName, Duration: time.Since(upgradeStarted)})
			logStep(slog.LevelInfo, "module upgraded",
				"module", mod.Name,
				"duration_ms", time.Since(upgradeStarted).Milliseconds(),
			)
		}
		logModuleStageCompleted(moduleStageStarted)
		return runAppStage()

	default:
		return fmt.Errorf("unknown plan op: %q", string(plan.Op))
	}
}
