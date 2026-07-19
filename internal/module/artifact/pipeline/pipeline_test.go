// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pipeline

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	planner "github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestExecuteValidationErrors(t *testing.T) {
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}

	if err := Execute(ctx, planner.Plan{Op: planner.OpType("unknown")}, root, Callbacks{}); err == nil {
		t.Fatal("expected unknown op error")
	}
	if err := Execute(ctx, planner.Plan{Op: planner.OpInstall}, root, Callbacks{}); err == nil || err.Error() != `ResolveInstallModuleFromOrigin callback is required for install` {
		t.Fatalf("unexpected install validation error: %v", err)
	}
	if err := Execute(ctx, planner.Plan{Op: planner.OpInstall}, root, Callbacks{ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return root, nil }}); err == nil || err.Error() != `Install callback is required for install` {
		t.Fatalf("unexpected install validation error: %v", err)
	}
	if err := Execute(ctx, planner.Plan{Op: planner.OpUninstall}, root, Callbacks{}); err == nil || err.Error() != `ResolveInstalledModule callback is required for uninstall` {
		t.Fatalf("unexpected uninstall validation error: %v", err)
	}
	if err := Execute(ctx, planner.Plan{Op: planner.OpUninstall}, root, Callbacks{ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil }}); err == nil || err.Error() != `Uninstall callback is required for uninstall` {
		t.Fatalf("unexpected uninstall validation error: %v", err)
	}
	if err := Execute(ctx, planner.Plan{Op: planner.OpUpgrade}, root, Callbacks{ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil }}); err == nil || err.Error() != `Upgrade callback is required for upgrade` {
		t.Fatalf("unexpected upgrade validation error: %v", err)
	}
	if err := Execute(context.TODO(), planner.Plan{Op: planner.OpType("unknown")}, root, Callbacks{}); err == nil {
		t.Fatal("expected unknown op error")
	}

	planWithApps := planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}
	cb := Callbacks{ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return root, nil }, Install: func(module *meta.IrModule) error { return nil }}
	if err := Execute(ctx, planWithApps, root, cb); err == nil || err.Error() != `AppTargets callback is required` {
		t.Fatalf("unexpected missing AppTargets error: %v", err)
	}
	cb.AppTargets = func(appName string) (string, ModulesAppTargets, error) {
		rootDir := t.TempDir()
		return filepath.Join(rootDir, "dist", appName), ModulesAppTargets{
			ProtoDir:   filepath.Join(rootDir, "modules", "api", "proto", appName),
			WebDir:     filepath.Join(rootDir, "modules", "api", "web", appName),
			ServiceDir: filepath.Join(rootDir, "modules", "api", "service", appName),
		}, nil
	}
	if err := Execute(ctx, planWithApps, root, cb); err == nil || err.Error() != `GenerateApp callback is required` {
		t.Fatalf("unexpected missing GenerateApp error: %v", err)
	}
	cb.GenerateApp = func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
		return nil
	}
	if err := Execute(ctx, planWithApps, root, cb); err == nil || err.Error() != `BuildBackendApp callback is required` {
		t.Fatalf("unexpected missing BuildBackendApp error: %v", err)
	}
	cb.BuildBackendApp = func(ctx context.Context, appName string, distAppStagingDir string) error { return nil }
	cb.WriteDistManifest = func(ctx context.Context, distManifestStagingPath string) error { return nil }
	if err := Execute(ctx, planWithApps, root, cb); err == nil || err.Error() != `DistManifestTarget callback is required` {
		t.Fatalf("unexpected missing DistManifestTarget error: %v", err)
	}
}

func TestExecuteInstallGenerateModulesCanceledBeforeCallbacks(t *testing.T) {
	ctx, cancel := context.WithCancel(staging.WithTmpRoot(context.Background(), t.TempDir()))
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}

	err := Execute(ctx, planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}}, root, Callbacks{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return nil, errors.New("fetch should not be called for root module")
		},
		Install: func(module *meta.IrModule) error {
			cancel()
			return nil
		},
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			t.Fatal("AppTargets should not be called after canceled context")
			return "", ModulesAppTargets{}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			t.Fatal("GenerateApp should not be called after canceled context")
			return nil
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestExecuteUpgradeRunAppStageValidationErrors(t *testing.T) {
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}
	baseCallbacks := Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
	}

	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, baseCallbacks); err == nil || err.Error() != `AppTargets callback is required` {
		t.Fatalf("unexpected missing AppTargets error for upgrade runAppStage: %v", err)
	}

	baseCallbacks.AppTargets = func(appName string) (string, ModulesAppTargets, error) {
		rootDir := t.TempDir()
		return filepath.Join(rootDir, "dist", "apps", appName), ModulesAppTargets{
			ProtoDir:   filepath.Join(rootDir, "modules", "api", "proto", appName),
			WebDir:     filepath.Join(rootDir, "modules", "api", "web", appName),
			ServiceDir: filepath.Join(rootDir, "modules", "api", "service", appName),
		}, nil
	}
	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, baseCallbacks); err == nil || err.Error() != `GenerateApp callback is required` {
		t.Fatalf("unexpected missing GenerateApp error for upgrade runAppStage: %v", err)
	}

	baseCallbacks.GenerateApp = func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
		return nil
	}
	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, baseCallbacks); err == nil || err.Error() != `BuildBackendApp callback is required` {
		t.Fatalf("unexpected missing BuildBackendApp error for upgrade runAppStage: %v", err)
	}
}

func TestExecuteInstallAppStageSuccess(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{
		Op:                  planner.OpInstall,
		ModuleOrder:         []string{"base"},
		AffectedApps:        []string{"crm"},
		NeedsGlobalWebBuild: true,
	}

	installCalls := 0
	bundlesCalls := 0
	manifestCalls := 0
	webCalls := 0

	appTargets := func(appName string) (string, ModulesAppTargets, error) {
		return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
			ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
			WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
			ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
		}, nil
	}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return root, nil },
		Install: func(module *meta.IrModule) error {
			installCalls++
			return nil
		},
		AppTargets: appTargets,
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		BundlesTarget: func() (string, error) { return filepath.Join(distRoot, "bundles"), nil },
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			bundlesCalls++
			if affectedProtoStaging["crm"] == "" {
				return errors.New("missing affected proto staging")
			}
			return writeStageFile(distBundlesStagingDir, "index.js", "console.log('bundles')")
		},
		WebTarget: func() (string, error) { return filepath.Join(distRoot, "web"), nil },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			webCalls++
			return writeStageFile(distWebStagingDir, "index.html", "<html></html>")
		},
		DistManifestTarget: func() (string, error) { return filepath.Join(distRoot, distmanifest.DistManifestFileName), nil },
		WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
			manifestCalls++
			return os.WriteFile(distManifestStagingPath, []byte(`{"apps":["crm"]}`), 0o644)
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if installCalls != 1 || bundlesCalls != 1 || webCalls != 1 || manifestCalls != 1 {
		t.Fatalf("unexpected callback counts: install=%d bundles=%d web=%d manifest=%d", installCalls, bundlesCalls, webCalls, manifestCalls)
	}

	for _, path := range []string{
		filepath.Join(distRoot, "apps", "crm", "index.js"),
		filepath.Join(distRoot, "bundles", "index.js"),
		filepath.Join(distRoot, "web", "index.html"),
		filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"),
		filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"),
		filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"),
		filepath.Join(distRoot, distmanifest.DistManifestFileName),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected committed artifact %s: %v", path, err)
		}
	}
}

func TestExecuteInstallEmitsUnifiedProgressEvents(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	pl := planner.Plan{
		Op:                  planner.OpInstall,
		ModuleOrder:         []string{"base"},
		AffectedApps:        []string{"crm"},
		NeedsGlobalWebBuild: true,
	}

	var events []ProgressEvent
	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), pl, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return root, nil },
		Install:                        func(module *meta.IrModule) error { return nil },
		OnProgress: func(event ProgressEvent) {
			events = append(events, event)
		},
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		BundlesTarget: func() (string, error) { return filepath.Join(distRoot, "bundles"), nil },
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			return writeStageFile(distBundlesStagingDir, "index.js", "console.log('bundles')")
		},
		WebTarget: func() (string, error) { return filepath.Join(distRoot, "web"), nil },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			return writeStageFile(distWebStagingDir, "index.html", "<html></html>")
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	counts := map[ProgressStage]int{}
	for _, event := range events {
		counts[event.Stage]++
	}

	for _, stage := range []ProgressStage{
		ProgressStageModuleInstallStarted,
		ProgressStageModuleInstallCompleted,
		ProgressStageAppStageStarted,
		ProgressStageAppBuildStarted,
		ProgressStageAppGenerateStarted,
		ProgressStageBundlesBuildStarted,
		ProgressStageBundlesBuildCompleted,
		ProgressStageWebBuildStarted,
		ProgressStageWebBuildCompleted,
		ProgressStageAppStageCompleted,
	} {
		if counts[stage] == 0 {
			t.Fatalf("expected progress stage %s to be emitted, got events=%v", stage, counts)
		}
	}
}

func TestExecuteInstallAppStageSuccessWithRuntimeProtoTarget(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	runtimeProtoRoot := filepath.Join(rootDir, "runtime", "api")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{
		Op:           planner.OpInstall,
		ModuleOrder:  []string{"base"},
		AffectedApps: []string{"crm"},
	}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return root, nil },
		Install:                        func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:        filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:          filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir:      filepath.Join(modulesRoot, "api", "service", appName),
				RuntimeProtoDir: filepath.Join(runtimeProtoRoot, appName, "proto"),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertFileContent(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"), "syntax = \"proto3\";")
	assertFileContent(t, filepath.Join(runtimeProtoRoot, "crm", "proto", "index.proto"), "syntax = \"proto3\";")
}

func TestExecuteUpgradeBasePreservesSiblingRuntimeProtoDirs(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	runtimeProtoRoot := filepath.Join(rootDir, "runtime", "api")
	_ = os.MkdirAll(filepath.Join(distRoot, "bundles"), 0o755)
	_ = os.WriteFile(filepath.Join(distRoot, "bundles", "index.js"), []byte("old-bundles"), 0o644)
	_ = os.MkdirAll(filepath.Join(distRoot, "web"), 0o755)
	_ = os.WriteFile(filepath.Join(distRoot, "web", "index.html"), []byte("old-web"), 0o644)

	for _, app := range []string{"auth", "base", "task"} {
		protoLive := filepath.Join(runtimeProtoRoot, app, "proto")
		if err := os.MkdirAll(protoLive, 0o755); err != nil {
			t.Fatalf("mkdir runtime proto: %v", err)
		}
		if err := os.WriteFile(filepath.Join(protoLive, app+".proto"), []byte("old-"+app), 0o644); err != nil {
			t.Fatalf("write runtime proto: %v", err)
		}
	}

	root := &meta.IrModule{Name: "base", ApplicationStr: "base"}
	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{
		Op:                  planner.OpUpgrade,
		ModuleOrder:         []string{"base"},
		AffectedApps:        []string{"base"},
		NeedsGlobalWebBuild: true,
	}, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return "", ModulesAppTargets{
				ProtoDir:        filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:          filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir:      filepath.Join(modulesRoot, "api", "service", appName),
				RuntimeProtoDir: filepath.Join(runtimeProtoRoot, appName, "proto"),
			}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, appName+".proto", "new-"+appName); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export {}"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export {}")
		},
		BundlesTarget: func() (string, error) { return filepath.Join(distRoot, "bundles"), nil },
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			return writeStageFile(distBundlesStagingDir, "index.js", "new-bundles")
		},
		WebTarget: func() (string, error) { return filepath.Join(distRoot, "web"), nil },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			return writeStageFile(distWebStagingDir, "index.html", "new-web")
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertFileContent(t, filepath.Join(runtimeProtoRoot, "base", "proto", "base.proto"), "new-base")
	assertFileContent(t, filepath.Join(runtimeProtoRoot, "auth", "proto", "auth.proto"), "old-auth")
	assertFileContent(t, filepath.Join(runtimeProtoRoot, "task", "proto", "task.proto"), "old-task")
}

func TestExecuteInfoLogsSummarizeAppStageAndHideManifestCommit(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "core"}
	var logBuf bytes.Buffer

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{
		Op:           planner.OpUpgrade,
		ModuleOrder:  []string{"core"},
		AffectedApps: []string{"task", "", "base", "task"},
	}, root, Callbacks{
		Logger:                 slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		DistManifestTarget: func() (string, error) { return filepath.Join(distRoot, distmanifest.DistManifestFileName), nil },
		WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
			return os.WriteFile(distManifestStagingPath, []byte(`{"apps":["base","task"]}`), 0o644)
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, `"msg":"pipeline module stage started"`) {
		t.Fatalf("expected module stage start summary log, got %q", logs)
	}
	if !strings.Contains(logs, `"msg":"pipeline module stage completed"`) {
		t.Fatalf("expected module stage completed summary log, got %q", logs)
	}
	if !strings.Contains(logs, `"modules_count":1`) {
		t.Fatalf("expected summarized module count in info logs, got %q", logs)
	}
	if !strings.Contains(logs, `"modules":["core"]`) {
		t.Fatalf("expected summarized module names in info logs, got %q", logs)
	}
	if !strings.Contains(logs, `"msg":"pipeline app stage started"`) {
		t.Fatalf("expected app stage start summary log, got %q", logs)
	}
	if !strings.Contains(logs, `"msg":"pipeline app stage completed"`) {
		t.Fatalf("expected app stage completed summary log, got %q", logs)
	}
	if !strings.Contains(logs, `"apps_count":2`) {
		t.Fatalf("expected summarized app count in info logs, got %q", logs)
	}
	if !strings.Contains(logs, `"apps":["base","task"]`) {
		t.Fatalf("expected summarized app names in info logs, got %q", logs)
	}
	appStageSummaryFound := false
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		if !strings.Contains(line, `"msg":"pipeline app stage completed"`) {
			continue
		}
		appStageSummaryFound = true
		if !strings.Contains(line, `"duration_ms":`) {
			t.Fatalf("expected duration_ms on app stage summary record, got %q", line)
		}
	}
	if !appStageSummaryFound {
		t.Fatalf("expected app stage summary record in logs, got %q", logs)
	}
}
func TestExecuteInstallSkipsWebModuleGeneration(t *testing.T) {
	root := &meta.IrModule{Name: "webmod", ApplicationStr: "web"}
	installCalls := 0
	generateCalls := 0
	appTargetsCalls := 0

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{
		Op:          planner.OpInstall,
		ModuleOrder: []string{"webmod"},
	}, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return nil, errors.New("fetch should not be used for root module")
		},
		Install: func(module *meta.IrModule) error {
			installCalls++
			return nil
		},
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			appTargetsCalls++
			return "", ModulesAppTargets{}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			generateCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if installCalls != 1 {
		t.Fatalf("install calls = %d, want 1", installCalls)
	}
	if appTargetsCalls != 0 || generateCalls != 0 {
		t.Fatalf("expected web app module generation to be skipped, got appTargets=%d generate=%d", appTargetsCalls, generateCalls)
	}
}

func TestExecuteUpgradeRollsBackCommittedStagesOnManifestFailure(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{
		Op:           planner.OpUpgrade,
		ModuleOrder:  []string{"base"},
		AffectedApps: []string{"", "crm", "crm"},
	}

	upgradeCalls := 0
	generateCalls := 0

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade: func(module *meta.IrModule) error {
			upgradeCalls++
			return nil
		},
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return "", ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			generateCalls++
			if distAppStagingDir != "" {
				return errors.New("expected empty dist staging dir")
			}
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		DistManifestTarget: func() (string, error) { return filepath.Join(distRoot, distmanifest.DistManifestFileName), nil },
		WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
			return errors.New("manifest write failed")
		},
	})
	if err == nil || err.Error() != "manifest write failed" {
		t.Fatalf("expected manifest error, got %v", err)
	}
	if upgradeCalls != 1 || generateCalls != 1 {
		t.Fatalf("unexpected callback counts: upgrade=%d generate=%d", upgradeCalls, generateCalls)
	}

	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(distRoot, distmanifest.DistManifestFileName))
}

func TestExecuteUpgradeGlobalWebFailureRollsBackCommittedStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{
		Op:                  planner.OpUpgrade,
		ModuleOrder:         []string{"base"},
		AffectedApps:        []string{"crm"},
		NeedsGlobalWebBuild: true,
	}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		WebTarget: func() (string, error) { return filepath.Join(distRoot, "web"), nil },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			return errors.New("web build failed")
		},
	})
	if err == nil || err.Error() != "web build failed" {
		t.Fatalf("expected web build error, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(distRoot, "web", "index.html"))
}

func TestExecuteGlobalWebAndBundlesValidation(t *testing.T) {
	rootDir := t.TempDir()
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}, NeedsGlobalWebBuild: true}
	baseCallbacks := Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return root, nil },
		Install:                        func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(rootDir, "dist", "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(rootDir, "modules", "api", "proto", appName),
				WebDir:     filepath.Join(rootDir, "modules", "api", "web", appName),
				ServiceDir: filepath.Join(rootDir, "modules", "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "ok")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			return nil
		},
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			return nil
		},
	}
	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, baseCallbacks); err == nil || err.Error() != `BundlesTarget callback is required` {
		t.Fatalf("unexpected missing BundlesTarget error: %v", err)
	}
	baseCallbacks.BundlesTarget = func() (string, error) { return filepath.Join(rootDir, "dist", "bundles"), nil }
	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, baseCallbacks); err == nil || err.Error() != `WebTarget callback is required` {
		t.Fatalf("unexpected missing WebTarget error: %v", err)
	}
	baseCallbacks.WebTarget = func() (string, error) { return filepath.Join(rootDir, "dist", "web"), nil }
	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, baseCallbacks); err == nil || err.Error() != `GlobalWebBuild callback is required` {
		t.Fatalf("unexpected missing GlobalWebBuild error: %v", err)
	}
}

func TestExecuteUpgradeCallsUpgradeOnly(t *testing.T) {
	plan := planner.Plan{
		Op:                  planner.OpUpgrade,
		ModuleOrder:         []string{"core"},
		AffectedApps:        nil,
		NeedsGlobalWebBuild: false,
	}
	root := &meta.IrModule{Name: "core"}

	upgradeCalled := 0
	loadCalled := false

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
			loadCalled = true
			return nil, nil
		},
		Upgrade: func(module *meta.IrModule) error {
			upgradeCalled++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if loadCalled {
		t.Fatalf("expected Load not to be called for root upgrade")
	}
	if upgradeCalled != 1 {
		t.Fatalf("expected Upgrade to be called once, got %d", upgradeCalled)
	}
}

func TestExecuteLoadAndFetchErrors(t *testing.T) {
	root := &meta.IrModule{Name: "root"}

	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"dep"}}, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return nil, errors.New("fetch failed") },
		Install:                        func(module *meta.IrModule) error { return nil },
	}); err == nil || err.Error() != "resolve install module from origin dep: fetch failed" {
		t.Fatalf("unexpected install fetch error: %v", err)
	}

	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpUninstall, ModuleOrder: []string{"dep"}}, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return nil, errors.New("load failed") },
		Uninstall:              func(module *meta.IrModule) error { return nil },
	}); err == nil || err.Error() != "resolve installed module dep: load failed" {
		t.Fatalf("unexpected uninstall load error: %v", err)
	}

	if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"dep"}}, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return nil, errors.New("load failed") },
		Upgrade:                func(module *meta.IrModule) error { return nil },
	}); err == nil || err.Error() != "resolve installed module dep: load failed" {
		t.Fatalf("unexpected upgrade load error: %v", err)
	}
}

func TestExecuteCanceledContextStopsBeforeCallbacks(t *testing.T) {
	ctx, cancel := context.WithCancel(staging.WithTmpRoot(context.Background(), t.TempDir()))
	cancel()

	fetchCalled := false
	err := Execute(ctx, planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"dep"}}, nil, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			fetchCalled = true
			return &meta.IrModule{Name: name}, nil
		},
		Install: func(module *meta.IrModule) error { return nil },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if fetchCalled {
		t.Fatal("expected canceled context to stop before fetch")
	}
}

func TestExecuteInstallGenerateModulesValidationErrors(t *testing.T) {
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}}, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return nil, errors.New("fetch should not be called for root module")
		},
		Install: func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return "", ModulesAppTargets{ProtoDir: "", WebDir: "/tmp/web", ServiceDir: "/tmp/service"}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			return nil
		},
	})
	if err == nil || err.Error() != "AppTargets callback returned empty proto/web/service dir for app crm" {
		t.Fatalf("unexpected generateModules validation error: %v", err)
	}
}

func TestExecuteInstallGenerateModulesCallbackErrors(t *testing.T) {
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	tests := []struct {
		name string
		cb   Callbacks
		want string
	}{
		{
			name: "app targets error",
			cb: Callbacks{
				ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
					return nil, errors.New("fetch should not be called for root module")
				},
				Install: func(module *meta.IrModule) error { return nil },
				AppTargets: func(appName string) (string, ModulesAppTargets, error) {
					return "", ModulesAppTargets{}, errors.New("app targets failed")
				},
				GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
					return nil
				},
			},
			want: "app targets failed",
		},
		{
			name: "generate app error",
			cb: Callbacks{
				ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
					return nil, errors.New("fetch should not be called for root module")
				},
				Install: func(module *meta.IrModule) error { return nil },
				AppTargets: func(appName string) (string, ModulesAppTargets, error) {
					rootDir := t.TempDir()
					return "", ModulesAppTargets{
						ProtoDir:   filepath.Join(rootDir, "modules", "api", "proto", appName),
						WebDir:     filepath.Join(rootDir, "modules", "api", "web", appName),
						ServiceDir: filepath.Join(rootDir, "modules", "api", "service", appName),
					}, nil
				},
				GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
					return errors.New("generate modules failed")
				},
			},
			want: "generate modules failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}}, root, tc.cb)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("unexpected install generateModules error: %v", err)
			}
		})
	}
}

func TestExecuteNilModulesAreSkipped(t *testing.T) {
	tests := []struct {
		name string
		plan planner.Plan
		cb   Callbacks
		ops  *int
	}{
		{
			name: "install skips nil fetched module",
			plan: planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"dep"}},
			cb: Callbacks{
				ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) { return nil, nil },
				Install:                        func(module *meta.IrModule) error { return errors.New("install should not be called") },
			},
		},
		{
			name: "uninstall skips nil loaded module",
			plan: planner.Plan{Op: planner.OpUninstall, ModuleOrder: []string{"dep"}},
			cb: Callbacks{
				ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return nil, nil },
				Uninstall:              func(module *meta.IrModule) error { return errors.New("uninstall should not be called") },
			},
		},
		{
			name: "upgrade skips nil loaded module",
			plan: planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"dep"}},
			cb: Callbacks{
				ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return nil, nil },
				Upgrade:                func(module *meta.IrModule) error { return errors.New("upgrade should not be called") },
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), tc.plan, nil, tc.cb); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
		})
	}
}

func TestExecuteOperationCallbackErrors(t *testing.T) {
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	tests := []struct {
		name string
		plan planner.Plan
		cb   Callbacks
		want string
	}{
		{
			name: "install callback error",
			plan: planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}},
			cb: Callbacks{
				ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
					return nil, errors.New("fetch should not be called for root module")
				},
				Install: func(module *meta.IrModule) error { return errors.New("install failed") },
			},
			want: "install failed",
		},
		{
			name: "uninstall callback error",
			plan: planner.Plan{Op: planner.OpUninstall, ModuleOrder: []string{"base"}},
			cb: Callbacks{
				ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
					return nil, errors.New("load should not be called for root module")
				},
				Uninstall: func(module *meta.IrModule) error { return errors.New("uninstall failed") },
			},
			want: "uninstall failed",
		},
		{
			name: "upgrade callback error",
			plan: planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}},
			cb: Callbacks{
				ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
					return nil, errors.New("load should not be called for root module")
				},
				Upgrade: func(module *meta.IrModule) error { return errors.New("upgrade failed") },
			},
			want: "upgrade failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), tc.plan, root, tc.cb)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("unexpected callback error: %v", err)
			}
		})
	}
}

func TestExecuteUpgradeNoGlobalWebManifestSuccess(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	manifestCalls := 0
	webTargetCalls := 0
	webBuildCalls := 0

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"", "crm", "crm"}}, root, Callbacks{
		Logger:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		DistManifestTarget: func() (string, error) { return filepath.Join(distRoot, distmanifest.DistManifestFileName), nil },
		WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
			manifestCalls++
			return os.WriteFile(distManifestStagingPath, []byte(`{"apps":["crm"]}`), 0o644)
		},
		WebTarget: func() (string, error) {
			webTargetCalls++
			return filepath.Join(distRoot, "web"), nil
		},
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			webBuildCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if manifestCalls != 1 {
		t.Fatalf("manifest calls = %d, want 1", manifestCalls)
	}
	if webTargetCalls != 0 || webBuildCalls != 0 {
		t.Fatalf("expected no global web callbacks, got target=%d build=%d", webTargetCalls, webBuildCalls)
	}

	for _, path := range []string{
		filepath.Join(distRoot, "apps", "crm", "index.js"),
		filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"),
		filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"),
		filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"),
		filepath.Join(distRoot, distmanifest.DistManifestFileName),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected committed artifact %s: %v", path, err)
		}
	}
	assertPathNotExists(t, filepath.Join(distRoot, "web", "index.html"))
}

func TestExecuteBundlesTargetErrorRollsBackCommittedDist(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		BundlesTarget: func() (string, error) { return "", errors.New("bundles target failed") },
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			return nil
		},
	})
	if err == nil || err.Error() != "bundles target failed" {
		t.Fatalf("expected bundles target error, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
}

func TestExecuteDistManifestTargetErrorRollsBackCommittedStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		DistManifestTarget: func() (string, error) { return "", errors.New("manifest target failed") },
		WriteDistManifest:  func(ctx context.Context, distManifestStagingPath string) error { return nil },
	})
	if err == nil || err.Error() != "manifest target failed" {
		t.Fatalf("expected dist manifest target error, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(distRoot, distmanifest.DistManifestFileName))
}

func TestExecuteAffectedAppsValidationErrors(t *testing.T) {
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return "", ModulesAppTargets{ProtoDir: "/tmp/proto", WebDir: "", ServiceDir: "/tmp/service"}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			return nil
		},
	})
	if err == nil || err.Error() != "AppTargets callback returned empty proto/web/service dir for app crm" {
		t.Fatalf("unexpected affected app validation error: %v", err)
	}
}

func TestExecuteBundlesBuildFailureRollsBackCommittedDist(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		BundlesTarget: func() (string, error) { return filepath.Join(distRoot, "bundles"), nil },
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			return errors.New("bundles build failed")
		},
	})
	if err == nil || err.Error() != "bundles build failed" {
		t.Fatalf("expected bundles build error, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(distRoot, "bundles", "index.js"))
}

func TestExecuteWebTargetErrorRollsBackCommittedStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}, NeedsGlobalWebBuild: true}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		WebTarget:      func() (string, error) { return "", errors.New("web target failed") },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error { return nil },
	})
	if err == nil || err.Error() != "web target failed" {
		t.Fatalf("expected web target error, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(distRoot, "web", "index.html"))
}

func TestExecuteGenerateAppFailureForLaterAppAbortsAllStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm", "erp"}}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", appName)
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if appName == "erp" {
				return errors.New("generate erp failed")
			}
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", appName); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", appName); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", appName)
		},
	})
	if err == nil || err.Error() != "generate erp failed" {
		t.Fatalf("expected second app generate error, got %v", err)
	}

	for _, app := range []string{"crm", "erp"} {
		assertPathNotExists(t, filepath.Join(distRoot, "apps", app, "index.js"))
		assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", app, "index.proto"))
		assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", app, "index.ts"))
		assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", app, "index.ts"))
	}
}

func TestExecuteBuildBackendFailureForLaterAppAbortsAllStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm", "erp"}}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			if appName == "erp" {
				return errors.New("build erp failed")
			}
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
	})
	if err == nil || err.Error() != "build erp failed" {
		t.Fatalf("expected second app build error, got %v", err)
	}

	for _, app := range []string{"crm", "erp"} {
		assertPathNotExists(t, filepath.Join(distRoot, "apps", app, "index.js"))
		assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", app, "index.proto"))
		assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", app, "index.ts"))
		assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", app, "index.ts"))
	}
}

func TestExecuteInstallGeneratesModulesOncePerApplication(t *testing.T) {
	rootDir := t.TempDir()
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	dep := &meta.IrModule{Name: "dep", ApplicationStr: "crm"}
	appTargetsCalls := 0
	generateCalls := 0

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base", "dep"}}, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			if name == "dep" {
				return dep, nil
			}
			return nil, errors.New("unexpected fetch")
		},
		Install: func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			appTargetsCalls++
			return "", ModulesAppTargets{
				ProtoDir:   filepath.Join(rootDir, "modules", "api", "proto", appName),
				WebDir:     filepath.Join(rootDir, "modules", "api", "web", appName),
				ServiceDir: filepath.Join(rootDir, "modules", "api", "service", appName),
			}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			generateCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if appTargetsCalls != 1 || generateCalls != 1 {
		t.Fatalf("expected single module generation for shared app, got appTargets=%d generate=%d", appTargetsCalls, generateCalls)
	}
}

func TestExecuteInstallProgressReportsFailedWhenPostInstallGenerationFails(t *testing.T) {
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())
	root := &meta.IrModule{Name: "core", ApplicationStr: "base"}
	genErr := errors.New("generate app failed")
	rootDir := t.TempDir()

	progressEvents := make([]ModuleInstallProgress, 0, 2)
	err := Execute(ctx, planner.Plan{
		Op:          planner.OpInstall,
		ModuleOrder: []string{"core"},
	}, root, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return root, nil
		},
		Install: func(module *meta.IrModule) error {
			return nil
		},
		OnInstallProgress: func(progress ModuleInstallProgress) {
			progressEvents = append(progressEvents, progress)
		},
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return "", ModulesAppTargets{
				ProtoDir:   filepath.Join(rootDir, "proto", appName),
				WebDir:     filepath.Join(rootDir, "web", appName),
				ServiceDir: filepath.Join(rootDir, "service", appName),
			}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			return genErr
		},
	})
	if !errors.Is(err, genErr) {
		t.Fatalf("Execute() error = %v, want %v", err, genErr)
	}
	if len(progressEvents) != 2 {
		t.Fatalf("progress events = %d, want 2", len(progressEvents))
	}
	if progressEvents[0].Stage != ModuleInstallProgressStageStarted {
		t.Fatalf("first progress stage = %q, want started", progressEvents[0].Stage)
	}
	if progressEvents[1].Stage != ModuleInstallProgressStageFailed {
		t.Fatalf("second progress stage = %q, want failed", progressEvents[1].Stage)
	}
	if !errors.Is(progressEvents[1].Err, genErr) {
		t.Fatalf("failed progress err = %v, want %v", progressEvents[1].Err, genErr)
	}
}

func TestExecuteInstallModuleCommitFailuresRollbackCommittedStages(t *testing.T) {
	tests := []struct {
		name               string
		failingStage       string
		preexistingTargets []string
	}{
		{name: "proto commit failure aborts later stages", failingStage: "proto"},
		{name: "web commit failure rolls back proto", failingStage: "web", preexistingTargets: []string{"proto"}},
		{name: "service commit failure rolls back proto and web", failingStage: "service", preexistingTargets: []string{"proto", "web"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const opID = "pipeline-modules-commit-failure"
			rootDir := t.TempDir()
			modulesRoot := filepath.Join(rootDir, "modules")
			root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}

			stageDir := func(stage string) string {
				return filepath.Join(modulesRoot, "api", stage, "crm")
			}

			for _, stage := range tc.preexistingTargets {
				target := stageDir(stage)
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatalf("MkdirAll(%s): %v", target, err)
				}
				if err := os.WriteFile(filepath.Join(target, "existing.txt"), []byte(stage+"-old"), 0o644); err != nil {
					t.Fatalf("WriteFile(%s existing): %v", stage, err)
				}
			}

			failingTarget := stageDir(tc.failingStage)
			if err := os.MkdirAll(failingTarget, 0o755); err != nil {
				t.Fatalf("MkdirAll failing target: %v", err)
			}
			if err := os.WriteFile(filepath.Join(failingTarget, "existing.txt"), []byte(tc.failingStage+"-old"), 0o644); err != nil {
				t.Fatalf("WriteFile failing target: %v", err)
			}
			backupDir := failingTarget + ".old." + opID
			if err := os.MkdirAll(backupDir, 0o755); err != nil {
				t.Fatalf("MkdirAll backup dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(backupDir, "keep.txt"), []byte("conflict"), 0o644); err != nil {
				t.Fatalf("WriteFile backup conflict: %v", err)
			}

			err := Execute(staging.WithOpID(staging.WithTmpRoot(context.Background(), t.TempDir()), opID), planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}}, root, Callbacks{
				ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
					return nil, errors.New("fetch should not be called for root module")
				},
				Install: func(module *meta.IrModule) error { return nil },
				AppTargets: func(appName string) (string, ModulesAppTargets, error) {
					return "", ModulesAppTargets{
						ProtoDir:   stageDir("proto"),
						WebDir:     stageDir("web"),
						ServiceDir: stageDir("service"),
					}, nil
				},
				GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
					if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
						return err
					}
					if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
						return err
					}
					return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
				},
			})
			if err == nil {
				t.Fatal("expected module commit failure")
			}

			assertFileContent(t, filepath.Join(failingTarget, "existing.txt"), tc.failingStage+"-old")
			assertPathNotExists(t, filepath.Join(failingTarget, "generated.txt"))
			assertFileContent(t, filepath.Join(backupDir, "keep.txt"), "conflict")

			for _, stage := range tc.preexistingTargets {
				target := stageDir(stage)
				assertFileContent(t, filepath.Join(target, "existing.txt"), stage+"-old")
				assertPathNotExists(t, filepath.Join(target, "generated.txt"))
			}

			for _, stage := range []string{"proto", "web", "service"} {
				if stage == tc.failingStage {
					continue
				}
				shouldExist := false
				for _, existing := range tc.preexistingTargets {
					if existing == stage {
						shouldExist = true
						break
					}
				}
				target := stageDir(stage)
				if shouldExist {
					continue
				}
				assertPathNotExists(t, filepath.Join(target, "existing.txt"))
				assertPathNotExists(t, filepath.Join(target, "generated.txt"))
			}
		})
	}
}

func TestExecuteInstallLaterModuleFailureRollsBackEarlierModuleGen(t *testing.T) {
	const opID = "pipeline-later-install-failure"
	rootDir := t.TempDir()
	modulesRoot := filepath.Join(rootDir, "modules")
	distRoot := filepath.Join(rootDir, "dist")
	stageDir := func(stage, app string) string {
		return filepath.Join(modulesRoot, "api", stage, app)
	}
	for _, stage := range []string{"proto", "web", "service"} {
		target := stageDir(stage, "crm")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", target, err)
		}
		if err := os.WriteFile(filepath.Join(target, "existing.txt"), []byte(stage+"-old"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", target, err)
		}
	}

	installCalls := 0
	wantErr := errors.New("second module commit failed")
	err := Execute(staging.WithOpID(staging.WithTmpRoot(context.Background(), t.TempDir()), opID), planner.Plan{
		Op:           planner.OpInstall,
		ModuleOrder:  []string{"base", "extra"},
		AffectedApps: []string{"crm"},
	}, &meta.IrModule{Name: "base", ApplicationStr: "crm"}, Callbacks{
		ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
			if name == "extra" {
				return &meta.IrModule{Name: "extra"}, nil
			}
			return nil, errors.New("unexpected resolve: " + name)
		},
		Install: func(module *meta.IrModule) error {
			installCalls++
			if module.Name == "extra" {
				return wantErr
			}
			return nil
		},
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   stageDir("proto", appName),
				WebDir:     stageDir("web", appName),
				ServiceDir: stageDir("service", appName),
			}, nil
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "should-not-land")
		},
		DistManifestTarget: func() (string, error) {
			return filepath.Join(distRoot, distmanifest.DistManifestFileName), nil
		},
		WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
			return os.WriteFile(distManifestStagingPath, []byte(`{"apps":["crm"]}`), 0o644)
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if installCalls != 2 {
		t.Fatalf("installCalls = %d, want 2", installCalls)
	}
	for _, stage := range []string{"proto", "web", "service"} {
		target := stageDir(stage, "crm")
		assertFileContent(t, filepath.Join(target, "existing.txt"), stage+"-old")
		assertPathNotExists(t, filepath.Join(target, "generated.txt"))
	}
	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(distRoot, distmanifest.DistManifestFileName))
}

func TestExecuteInstallModulePrepareFailuresAbortPreparedStages(t *testing.T) {
	tests := []struct {
		name             string
		failingStage     string
		preparedStages   []string
		blockerPathParts []string
	}{
		{name: "proto prepare failure returns immediately", failingStage: "proto", blockerPathParts: []string{"modules", "api", "proto"}},
		{name: "web prepare failure aborts proto", failingStage: "web", preparedStages: []string{"proto"}, blockerPathParts: []string{"modules", "api", "web"}},
		{name: "service prepare failure aborts proto and web", failingStage: "service", preparedStages: []string{"proto", "web"}, blockerPathParts: []string{"modules", "api", "service"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := t.TempDir()
			modulesRoot := filepath.Join(rootDir, "modules")
			root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}

			blockerPath := filepath.Join(append([]string{rootDir}, tc.blockerPathParts...)...)
			writeBlockerFile(t, blockerPath)

			err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), planner.Plan{Op: planner.OpInstall, ModuleOrder: []string{"base"}}, root, Callbacks{
				ResolveInstallModuleFromOrigin: func(ctx context.Context, name string) (*meta.IrModule, error) {
					return nil, errors.New("fetch should not be called for root module")
				},
				Install: func(module *meta.IrModule) error { return nil },
				AppTargets: func(appName string) (string, ModulesAppTargets, error) {
					return "", ModulesAppTargets{
						ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
						WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
						ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
					}, nil
				},
				GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
					return errors.New("generate should not be called when prepare fails")
				},
			})
			if err == nil {
				t.Fatal("expected prepare failure")
			}

			for _, stage := range tc.preparedStages {
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", stage, "crm", "generated.txt"))
			}
			assertPathNotExists(t, filepath.Join(rootDir, ".choysum", "tmp", "staging"))
			assertPathNotExists(t, filepath.Join(modulesRoot, "api", tc.failingStage, "crm", "generated.txt"))
		})
	}
}

func TestExecuteUpgradePrepareFailuresAbortPreparedStages(t *testing.T) {
	tests := []struct {
		name             string
		failingStage     string
		preparedStages   []string
		blockerPathParts []string
		plan             planner.Plan
		callbacks        func(rootDir string, distRoot string, modulesRoot string) Callbacks
	}{
		{
			name:             "dist prepare failure returns before modules staging",
			failingStage:     "dist",
			blockerPathParts: []string{"dist", "apps"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return errors.New("build should not be called when prepare fails")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						return errors.New("generate should not be called when prepare fails")
					},
				}
			},
		},
		{
			name:             "proto prepare failure aborts dist staging",
			failingStage:     "proto",
			preparedStages:   []string{"dist"},
			blockerPathParts: []string{"modules", "api", "proto"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						return errors.New("generate should not be called when prepare fails")
					},
				}
			},
		},
		{
			name:             "web prepare failure aborts dist and proto staging",
			failingStage:     "web",
			preparedStages:   []string{"dist", "proto"},
			blockerPathParts: []string{"modules", "api", "web"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						return errors.New("generate should not be called when prepare fails")
					},
				}
			},
		},
		{
			name:             "service prepare failure aborts dist proto and web staging",
			failingStage:     "service",
			preparedStages:   []string{"dist", "proto", "web"},
			blockerPathParts: []string{"modules", "api", "service"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						return errors.New("generate should not be called when prepare fails")
					},
				}
			},
		},
		{
			name:             "bundles prepare failure rolls back committed dist",
			failingStage:     "bundles",
			preparedStages:   []string{"dist", "proto", "web", "service"},
			blockerPathParts: []string{".choysum", "tmp", "staging"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
							return err
						}
						if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
							return err
						}
						return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
					},
					BundlesTarget: func() (string, error) { return filepath.Join(distRoot, "bundles"), nil },
					BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
						return errors.New("bundles build should not run when prepare fails")
					},
				}
			},
		},
		{
			name:             "global web prepare failure rolls back committed stages",
			failingStage:     "global-web",
			preparedStages:   []string{"dist", "proto", "web", "service"},
			blockerPathParts: []string{".choysum", "tmp", "staging"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}, NeedsGlobalWebBuild: true},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
							return err
						}
						if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
							return err
						}
						return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
					},
					WebTarget: func() (string, error) { return filepath.Join(distRoot, "web"), nil },
					GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
						return errors.New("web build should not run when prepare fails")
					},
				}
			},
		},
		{
			name:             "manifest prepare failure rolls back committed stages",
			failingStage:     "manifest",
			preparedStages:   []string{"dist", "proto", "web", "service"},
			blockerPathParts: []string{".choysum", "tmp", "staging"},
			plan:             planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}},
			callbacks: func(rootDir string, distRoot string, modulesRoot string) Callbacks {
				manifestPath := filepath.Join(distRoot, "manifest-parent", distmanifest.DistManifestFileName)
				return Callbacks{
					ResolveInstalledModule: func(name string) (*meta.IrModule, error) {
						return &meta.IrModule{Name: "base", ApplicationStr: "crm"}, nil
					},
					Upgrade: func(module *meta.IrModule) error { return nil },
					AppTargets: func(appName string) (string, ModulesAppTargets, error) {
						return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
							ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
							WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
							ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
						}, nil
					},
					BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
						return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
					},
					GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
						if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
							return err
						}
						if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
							return err
						}
						return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
					},
					DistManifestTarget: func() (string, error) { return manifestPath, nil },
					WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
						return errors.New("manifest write should not run when prepare fails")
					},
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := t.TempDir()
			distRoot := filepath.Join(rootDir, "dist")
			modulesRoot := filepath.Join(rootDir, "modules")
			writeBlockerFile(t, filepath.Join(append([]string{rootDir}, tc.blockerPathParts...)...))

			root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
			err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), tc.plan, root, tc.callbacks(rootDir, distRoot, modulesRoot))
			if err == nil {
				t.Fatal("expected prepare failure")
			}

			for _, stage := range tc.preparedStages {
				switch stage {
				case "dist":
					assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "generated.txt"))
				case "proto", "web", "service":
					assertPathNotExists(t, filepath.Join(modulesRoot, "api", stage, "crm", "generated.txt"))
				}
			}

			switch tc.failingStage {
			case "bundles":
				assertPathNotExists(t, filepath.Join(distRoot, "bundles", "generated.txt"))
				assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "generated.txt"))
			case "global-web":
				assertPathNotExists(t, filepath.Join(distRoot, "web", "generated.txt"))
				assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "generated.txt"))
			case "manifest":
				assertPathNotExists(t, filepath.Join(distRoot, "manifest-parent", distmanifest.DistManifestFileName))
				assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "generated.txt"))
				assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "generated.txt"))
			}
		})
	}
}
func TestExecuteUpgradeCommitFailuresRollbackCommittedStages(t *testing.T) {
	tests := []struct {
		name               string
		failingStage       string
		preexistingTargets []string
	}{
		{name: "dist commit failure aborts staged modules", failingStage: "dist", preexistingTargets: []string{"dist"}},
		{name: "proto commit failure rolls back dist", failingStage: "proto", preexistingTargets: []string{"dist", "proto"}},
		{name: "web commit failure rolls back dist and proto", failingStage: "web", preexistingTargets: []string{"dist", "proto", "web"}},
		{name: "service commit failure rolls back earlier app commits", failingStage: "service", preexistingTargets: []string{"dist", "proto", "web", "service"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const opID = "pipeline-upgrade-commit-failure"
			rootDir := t.TempDir()
			distRoot := filepath.Join(rootDir, "dist")
			modulesRoot := filepath.Join(rootDir, "modules")
			root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}

			targetDir := func(stage string) string {
				switch stage {
				case "dist":
					return filepath.Join(distRoot, "apps", "crm")
				case "proto":
					return filepath.Join(modulesRoot, "api", "proto", "crm")
				case "web":
					return filepath.Join(modulesRoot, "api", "web", "crm")
				case "service":
					return filepath.Join(modulesRoot, "api", "service", "crm")
				default:
					t.Fatalf("unexpected stage %s", stage)
					return ""
				}
			}

			for _, stage := range tc.preexistingTargets {
				target := targetDir(stage)
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatalf("MkdirAll(%s): %v", target, err)
				}
				if err := os.WriteFile(filepath.Join(target, "existing.txt"), []byte(stage+"-old"), 0o644); err != nil {
					t.Fatalf("WriteFile(%s existing): %v", stage, err)
				}
			}

			failingTarget := targetDir(tc.failingStage)
			backupDir := failingTarget + ".old." + opID
			if err := os.MkdirAll(backupDir, 0o755); err != nil {
				t.Fatalf("MkdirAll backup dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(backupDir, "keep.txt"), []byte("conflict"), 0o644); err != nil {
				t.Fatalf("WriteFile backup conflict: %v", err)
			}

			err := Execute(staging.WithOpID(staging.WithTmpRoot(context.Background(), t.TempDir()), opID), planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}, root, Callbacks{
				ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
				Upgrade:                func(module *meta.IrModule) error { return nil },
				AppTargets: func(appName string) (string, ModulesAppTargets, error) {
					return targetDir("dist"), ModulesAppTargets{
						ProtoDir:   targetDir("proto"),
						WebDir:     targetDir("web"),
						ServiceDir: targetDir("service"),
					}, nil
				},
				BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
					return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
				},
				GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
					if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
						return err
					}
					if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
						return err
					}
					return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
				},
			})
			if err == nil {
				t.Fatal("expected commit failure")
			}

			assertFileContent(t, filepath.Join(failingTarget, "existing.txt"), tc.failingStage+"-old")
			assertPathNotExists(t, filepath.Join(failingTarget, "generated.txt"))
			assertFileContent(t, filepath.Join(backupDir, "keep.txt"), "conflict")

			for _, stage := range tc.preexistingTargets {
				target := targetDir(stage)
				assertFileContent(t, filepath.Join(target, "existing.txt"), stage+"-old")
				assertPathNotExists(t, filepath.Join(target, "generated.txt"))
			}

			for _, stage := range []string{"dist", "proto", "web", "service"} {
				present := false
				for _, existing := range tc.preexistingTargets {
					if existing == stage {
						present = true
						break
					}
				}
				if present {
					continue
				}
				target := targetDir(stage)
				assertPathNotExists(t, filepath.Join(target, "existing.txt"))
				assertPathNotExists(t, filepath.Join(target, "generated.txt"))
			}
		})
	}
}

func TestExecuteUpgradeGlobalWebCommitFailureRollsBackCommittedStages(t *testing.T) {
	const opID = "pipeline-upgrade-global-web-commit"
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}

	targets := map[string]string{
		"dist":       filepath.Join(distRoot, "apps", "crm"),
		"proto":      filepath.Join(modulesRoot, "api", "proto", "crm"),
		"web":        filepath.Join(modulesRoot, "api", "web", "crm"),
		"service":    filepath.Join(modulesRoot, "api", "service", "crm"),
		"global-web": filepath.Join(distRoot, "web"),
	}

	for stage, target := range targets {
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", stage, err)
		}
		if err := os.WriteFile(filepath.Join(target, "existing.txt"), []byte(stage+"-old"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s existing): %v", stage, err)
		}
	}

	webBackupDir := targets["global-web"] + ".old." + opID
	if err := os.MkdirAll(webBackupDir, 0o755); err != nil {
		t.Fatalf("MkdirAll web backup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webBackupDir, "keep.txt"), []byte("conflict"), 0o644); err != nil {
		t.Fatalf("WriteFile web backup conflict: %v", err)
	}

	err := Execute(staging.WithOpID(staging.WithTmpRoot(context.Background(), t.TempDir()), opID), planner.Plan{
		Op:                  planner.OpUpgrade,
		ModuleOrder:         []string{"base"},
		AffectedApps:        []string{"crm"},
		NeedsGlobalWebBuild: true,
	}, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return targets["dist"], ModulesAppTargets{
				ProtoDir:   targets["proto"],
				WebDir:     targets["web"],
				ServiceDir: targets["service"],
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "generated.txt", "dist-new")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "generated.txt", "proto-new"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "generated.txt", "web-new"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "generated.txt", "service-new")
		},
		WebTarget: func() (string, error) { return targets["global-web"], nil },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			return writeStageFile(distWebStagingDir, "generated.txt", "global-web-new")
		},
	})
	if err == nil {
		t.Fatal("expected global web commit failure")
	}

	for stage, target := range targets {
		assertFileContent(t, filepath.Join(target, "existing.txt"), stage+"-old")
		assertPathNotExists(t, filepath.Join(target, "generated.txt"))
	}
	assertFileContent(t, filepath.Join(webBackupDir, "keep.txt"), "conflict")
	assertPathNotExists(t, filepath.Join(distRoot, distmanifest.DistManifestFileName))
}

func TestExecuteGlobalWebManifestWriteFailureRollsBackCommittedStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}, NeedsGlobalWebBuild: true}

	err := Execute(staging.WithTmpRoot(context.Background(), t.TempDir()), plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		WebTarget: func() (string, error) { return filepath.Join(distRoot, "web"), nil },
		GlobalWebBuild: func(ctx context.Context, distWebStagingDir string) error {
			return writeStageFile(distWebStagingDir, "index.html", "<html></html>")
		},
		DistManifestTarget: func() (string, error) { return filepath.Join(distRoot, distmanifest.DistManifestFileName), nil },
		WriteDistManifest: func(ctx context.Context, distManifestStagingPath string) error {
			return errors.New("manifest write failed after web")
		},
	})
	if err == nil || err.Error() != "manifest write failed after web" {
		t.Fatalf("expected manifest write error after web build, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(distRoot, "web", "index.html"))
	assertPathNotExists(t, filepath.Join(distRoot, distmanifest.DistManifestFileName))
}

func TestExecuteCanceledAfterPhase1AbortsPreparedStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}
	ctx, cancel := context.WithCancel(staging.WithTmpRoot(context.Background(), t.TempDir()))

	err := Execute(ctx, plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			defer cancel()
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled after phase1, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
}

func TestExecuteCanceledAfterBundlesCommitRollsBackCommittedStages(t *testing.T) {
	rootDir := t.TempDir()
	distRoot := filepath.Join(rootDir, "dist")
	modulesRoot := filepath.Join(rootDir, "modules")
	root := &meta.IrModule{Name: "base", ApplicationStr: "crm"}
	plan := planner.Plan{Op: planner.OpUpgrade, ModuleOrder: []string{"base"}, AffectedApps: []string{"crm"}}
	ctx, cancel := context.WithCancel(staging.WithTmpRoot(context.Background(), t.TempDir()))

	err := Execute(ctx, plan, root, Callbacks{
		ResolveInstalledModule: func(name string) (*meta.IrModule, error) { return root, nil },
		Upgrade:                func(module *meta.IrModule) error { return nil },
		AppTargets: func(appName string) (string, ModulesAppTargets, error) {
			return filepath.Join(distRoot, "apps", appName), ModulesAppTargets{
				ProtoDir:   filepath.Join(modulesRoot, "api", "proto", appName),
				WebDir:     filepath.Join(modulesRoot, "api", "web", appName),
				ServiceDir: filepath.Join(modulesRoot, "api", "service", appName),
			}, nil
		},
		BuildBackendApp: func(ctx context.Context, appName string, distAppStagingDir string) error {
			return writeStageFile(distAppStagingDir, "index.js", "console.log('backend')")
		},
		GenerateApp: func(ctx context.Context, appName string, modulesStaging ModulesAppTargets, distAppStagingDir string) error {
			if err := writeStageFile(modulesStaging.ProtoDir, "index.proto", "syntax = \"proto3\";"); err != nil {
				return err
			}
			if err := writeStageFile(modulesStaging.WebDir, "index.ts", "export const web = true"); err != nil {
				return err
			}
			return writeStageFile(modulesStaging.ServiceDir, "index.ts", "export const service = true")
		},
		BundlesTarget: func() (string, error) { return filepath.Join(distRoot, "bundles"), nil },
		BuildBackendBundles: func(ctx context.Context, distBundlesStagingDir string, affectedProtoStaging map[string]string) error {
			defer cancel()
			return writeStageFile(distBundlesStagingDir, "index.js", "console.log('bundles')")
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled after bundles commit, got %v", err)
	}

	assertPathNotExists(t, filepath.Join(distRoot, "apps", "crm", "index.js"))
	assertPathNotExists(t, filepath.Join(distRoot, "bundles", "index.js"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "proto", "crm", "index.proto"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "web", "crm", "index.ts"))
	assertPathNotExists(t, filepath.Join(modulesRoot, "api", "service", "crm", "index.ts"))
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected path to be absent %s, got err=%v", path, err)
	} else if !os.IsNotExist(err) {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) && pathErr.Err == os.ErrNotExist {
			return
		}
		if errors.As(err, &pathErr) && pathErr.Err.Error() == "not a directory" {
			return
		}
		t.Fatalf("expected path to be absent %s, got err=%v", path, err)
	}
}

func writeStageFile(dir string, name string, content string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("file %s content = %q, want %q", path, string(got), want)
	}
}

func writeBlockerFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func TestSummarizeInfoNames(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		count, names := summarizeInfoNames(nil)
		if count != 0 || names != nil {
			t.Fatalf("summarizeInfoNames(nil) = (%d, %#v), want (0, nil)", count, names)
		}
	})

	t.Run("single", func(t *testing.T) {
		count, names := summarizeInfoNames([]string{"base"})
		if count != 1 || len(names) != 1 || names[0] != "base" {
			t.Fatalf("summarizeInfoNames([base]) = (%d, %#v), want (1, [base])", count, names)
		}
	})

	t.Run("dedup", func(t *testing.T) {
		count, names := summarizeInfoNames([]string{"base", "base", "core"})
		if count != 2 || len(names) != 2 || names[0] != "base" || names[1] != "core" {
			t.Fatalf("summarizeInfoNames([base, base, core]) = (%d, %#v), want (2, [base core])", count, names)
		}
	})

	t.Run("empty strings skipped", func(t *testing.T) {
		count, names := summarizeInfoNames([]string{"", "base", "", "core"})
		if count != 2 || len(names) != 2 || names[0] != "base" || names[1] != "core" {
			t.Fatalf("summarizeInfoNames with empties = (%d, %#v), want (2, [base core])", count, names)
		}
	})

	t.Run("all empty returns zero", func(t *testing.T) {
		count, names := summarizeInfoNames([]string{"", "", ""})
		if count != 0 || names != nil {
			t.Fatalf("summarizeInfoNames(all empty) = (%d, %#v), want (0, nil)", count, names)
		}
	})

	t.Run("over limit returns count only", func(t *testing.T) {
		values := make([]string, infoSummaryNameListLimit+1)
		for i := range values {
			values[i] = "m" + string(rune('a'+i%26))
			// Ensure uniqueness by appending index.
			values[i] = values[i] + string(rune('0'+i/26))
		}
		count, names := summarizeInfoNames(values)
		if count != len(values) || names != nil {
			t.Fatalf("summarizeInfoNames(over limit) = (%d, %#v), want (%d, nil)", count, names, len(values))
		}
	})
}
