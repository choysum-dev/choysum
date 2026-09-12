// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	moduleresult "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func lifecycleCommitModule(t *testing.T, name string) (*meta.Module, string) {
	t.Helper()
	modulePath := t.TempDir()
	i18nDir := filepath.Join(modulePath, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`msgid ""
msgstr ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{
		Name:           name,
		Version:        "1.0.0",
		Status:         meta.ToInstall,
		Path:           modulePath,
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	return mod, modulePath
}

func TestCommitInstall_applyInitdataWithDemo(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod, _ := lifecycleCommitModule(t, "demo_with_demo")
	opCtx := newOpContext()
	opCtx.withDemo = true
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           opCtx,
	}
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall with withDemo: %v", err)
	}
}

func TestCommitInstall_applyInitdataNilCtx(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod, _ := lifecycleCommitModule(t, "demo_nil_ctx")
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           nil,
	}
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall with nil ctx: %v", err)
	}
}

func TestCommitUpgrade_applyInitdataWithDemo(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	dep := &meta.Module{
		Name:    "demo_upgrade_dep",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    t.TempDir(),
	}
	dep.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(dep).Error; err != nil {
		t.Fatalf("create dep: %v", err)
	}
	mod := &meta.Module{
		Name:    "demo_upgrade_demo",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{
		Name: "demo_upgrade_demo", Version: "2.0.0", Status: meta.Installed, Path: modulePath,
		Dependencies: []*meta.Module{dep},
	}
	target.Id = mod.Id

	opCtx := newOpContext()
	opCtx.withDemo = true
	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           opCtx,
	}
	installer := &moduleInstaller{
		module:        target,
		runtimeScope:  runtimeScope,
		moduleManager: upgrader.moduleManager,
		ctx:           opCtx,
	}
	if _, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false); err != nil {
		t.Fatalf("commitUpgrade with withDemo: %v", err)
	}
}

func TestCommitUpgrade_DependenciesReplaceError(t *testing.T) {
	orig := replaceModuleDependenciesFn
	t.Cleanup(func() { replaceModuleDependenciesFn = orig })
	replaceModuleDependenciesFn = func(*scope.Session, *meta.Module) error {
		return errors.New("dep replace boom")
	}

	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:    "demo_upgrade_dep_err",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    t.TempDir(),
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	dep := &meta.Module{Name: "dep_only", Version: "1.0.0", Status: meta.Installed, Path: t.TempDir()}
	dep.Id = sql.NullString{String: xid.New().String(), Valid: true}
	target := &meta.Module{
		Name: "demo_upgrade_dep_err", Version: "2.0.0", Status: meta.Installed, Path: mod.Path,
		Dependencies: []*meta.Module{dep},
	}
	target.Id = mod.Id

	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	installer := &moduleInstaller{
		module:        target,
		runtimeScope:  runtimeScope,
		moduleManager: upgrader.moduleManager,
		ctx:           upgrader.ctx,
	}
	_, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false)
	if err == nil || !strings.Contains(err.Error(), "error saving module dependencies") {
		t.Fatalf("expected dependencies error, got %v", err)
	}
}

func TestCommitUpgrade_applyInitdataNilCtx(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	mod := &meta.Module{
		Name:    "demo_upgrade_nil_ctx",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{Name: "demo_upgrade_nil_ctx", Version: "2.0.0", Status: meta.Installed, Path: modulePath}
	target.Id = mod.Id

	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           nil,
	}
	installer := &moduleInstaller{
		module:        target,
		runtimeScope:  runtimeScope,
		moduleManager: upgrader.moduleManager,
		ctx:           nil,
	}
	if _, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false); err != nil {
		t.Fatalf("commitUpgrade with nil ctx: %v", err)
	}
}

func TestCommitInstall_applyInitdataError(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, errors.New("forced initdata failure")
	})

	runtimeScope := newLifecycleCommitTestScope(t)
	mod, _ := lifecycleCommitModule(t, "demo_initdata_err")
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	_, err := installer.commitInstall(nil, false)
	if err == nil || !strings.Contains(err.Error(), "error applying data for module") {
		t.Fatalf("commitInstall error = %v, want apply-data failure", err)
	}
}

func TestCommitUpgrade_applyInitdataError(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, errors.New("forced initdata failure")
	})

	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	mod := &meta.Module{
		Name:    "demo_upgrade_initdata_err",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    modulePath,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{Name: "demo_upgrade_initdata_err", Version: "2.0.0", Status: meta.Installed, Path: modulePath}
	target.Id = mod.Id

	upgrader := &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	installer := &moduleInstaller{
		module:        target,
		runtimeScope:  runtimeScope,
		moduleManager: upgrader.moduleManager,
		ctx:           upgrader.ctx,
	}
	_, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false)
	if err == nil || !strings.Contains(err.Error(), "error applying data for module") {
		t.Fatalf("commitUpgrade error = %v, want apply-data failure", err)
	}
}

type failSchemaMigrator struct{ err error }

func (f failSchemaMigrator) Migrate() error { return f.err }
func (f failSchemaMigrator) PlanOnly() (schema.SchemaPlan, error) {
	return schema.SchemaPlan{}, f.err
}

func TestCommitInstall_ErrorBranchesForPatchCoverage(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mgr := &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}}

	t.Run("restoreSoftDeleted", func(t *testing.T) {
		mod, _ := lifecycleCommitModule(t, "cov_restore_err")
		installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
		sqlDB, err := runtimeScope.Session().DB.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error checking existing module") {
			t.Fatalf("restore/check error: %v", err)
		}
	})

	runtimeScope = newLifecycleCommitTestScope(t)
	mgr = &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}}

	t.Run("persist", func(t *testing.T) {
		mod, _ := lifecycleCommitModule(t, "cov_persist_err")
		installer := &moduleInstaller{
			module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext(),
			builder: &commitStubSplitBuilder{persistErr: errors.New("persist boom")},
		}
		if _, err := installer.commitInstall(&moduleresult.BuildResult{}, true); err == nil || !strings.Contains(err.Error(), "error persisting module") {
			t.Fatalf("persist error: %v", err)
		}
	})

	t.Run("build", func(t *testing.T) {
		mod, _ := lifecycleCommitModule(t, "cov_build_err")
		installer := &moduleInstaller{
			module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext(),
			builder: &commitStubSplitBuilder{buildErr: errors.New("build boom")},
		}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error building module") {
			t.Fatalf("build error: %v", err)
		}
	})

	t.Run("migrate", func(t *testing.T) {
		orig := newInstallSchemaMigrator
		t.Cleanup(func() { newInstallSchemaMigrator = orig })
		newInstallSchemaMigrator = func(scope.Scope, *meta.Module, ...schema.MigratorOption) (schema.Migrator, error) {
			return failSchemaMigrator{err: errors.New("migrate boom")}, nil
		}
		mod, _ := lifecycleCommitModule(t, "cov_migrate_err")
		installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error migrating module") {
			t.Fatalf("migrate error: %v", err)
		}
	})

	t.Run("dependencies", func(t *testing.T) {
		orig := replaceModuleDependenciesFn
		t.Cleanup(func() { replaceModuleDependenciesFn = orig })
		replaceModuleDependenciesFn = func(*scope.Session, *meta.Module) error {
			return errors.New("dep replace boom")
		}
		mod, _ := lifecycleCommitModule(t, "cov_deps_err")
		dep := &meta.Module{Name: "cov_dep", Version: "1.0.0", Status: meta.Installed, Path: t.TempDir()}
		dep.Id = sql.NullString{String: xid.New().String(), Valid: true}
		mod.Dependencies = []*meta.Module{dep}
		installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error saving module dependencies") {
			t.Fatalf("deps error: %v", err)
		}
	})

	t.Run("terminology", func(t *testing.T) {
		t.Cleanup(func() { importpkg.SetRun(runner.Run) })
		importpkg.SetRun(func(_ context.Context, _ scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
			if spec.Profile == importpkg.ProfileTerminology {
				return importpkg.Report{}, errors.New("forced terminology failure")
			}
			return importpkg.Report{Profile: spec.Profile}, nil
		})
		mod, _ := lifecycleCommitModule(t, "cov_term_err")
		installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "import terminology") {
			t.Fatalf("terminology error: %v", err)
		}
	})

	t.Run("metaSchedule", func(t *testing.T) {
		orig := installerScheduleDBFn
		t.Cleanup(func() { installerScheduleDBFn = orig })
		installerScheduleDBFn = func(scope.Scope) (*gorm.DB, error) {
			return nil, errors.New("sched boom")
		}
		mod, _ := lifecycleCommitModule(t, "meta")
		mod.Name = "meta"
		installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error disabling legacy module index schedule") {
			t.Fatalf("meta schedule error: %v", err)
		}
	})

	t.Run("documentSchedule", func(t *testing.T) {
		orig := installerScheduleDBFn
		t.Cleanup(func() { installerScheduleDBFn = orig })
		installerScheduleDBFn = func(scope.Scope) (*gorm.DB, error) {
			return nil, errors.New("sched boom")
		}
		mod, _ := lifecycleCommitModule(t, "document")
		mod.Name = "document"
		installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope, moduleManager: mgr, ctx: newOpContext()}
		if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error ensuring document attachment gc schedule") {
			t.Fatalf("document schedule error: %v", err)
		}
	})
}

func TestRunInstallCommitTX_DoesNotPublishOnCommitError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod, _ := lifecycleCommitModule(t, "cov_tx_publish")
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	orig := newInstallSchemaMigrator
	t.Cleanup(func() { newInstallSchemaMigrator = orig })
	newInstallSchemaMigrator = func(scope.Scope, *meta.Module, ...schema.MigratorOption) (schema.Migrator, error) {
		return failSchemaMigrator{err: errors.New("migrate boom")}, nil
	}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	sentinel := &moduleresult.BuildResult{}
	buildResult := sentinel
	if err := installer.runInstallCommitTX(runtimeScope, runtimeScope.Context(), &buildResult, false); err == nil {
		t.Fatal("expected commit error")
	}
	if buildResult != sentinel {
		t.Fatal("failed install commit must not publish build result")
	}
}
