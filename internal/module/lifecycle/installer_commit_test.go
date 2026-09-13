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
	"time"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	moduleresult "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	internaltask "github.com/choysum-dev/choysum/internal/task"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type commitStubBuilder struct{}

func (commitStubBuilder) Build() (*moduleresult.BuildResult, error) {
	return &moduleresult.BuildResult{}, nil
}

func TestCommitInstallSoftDeleteRestoreAndSave(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	i18nDir := filepath.Join(modulePath, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgid ""
msgstr ""

msgctxt "a@t"
msgid "Hello"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := &meta.Module{
		Name:           "demo",
		Version:        "1.0.0",
		Status:         meta.ToInstall,
		Path:           modulePath,
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	mod.DeletedAt = gorm.DeletedAt{Time: time.Now().UTC(), Valid: true}
	if err := runtimeScope.Session().Unscoped().Create(mod).Error; err != nil {
		t.Fatalf("create soft-deleted module: %v", err)
	}
	// Installer typically holds a fresh module descriptor; DeletedAt lives only in DB.
	mod.DeletedAt = gorm.DeletedAt{}

	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
		builder:       nil,
	}
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall: %v", err)
	}

	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "demo").Take(&got).Error; err != nil {
		t.Fatalf("load module: %v", err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected soft-delete restored")
	}
	if got.Status != meta.Installed {
		t.Fatalf("status = %q, want installed", got.Status)
	}

	var term i18nmodels.TranslationTerm
	if err := runtimeScope.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&term).Error; err != nil {
		t.Fatalf("expected imported term: %v", err)
	}
}

func TestRunInstallCommitTX_WithAndWithoutManager(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "commit_tx_demo", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}

	withMgr := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	var buildResult *moduleresult.BuildResult
	if err := withMgr.runInstallCommitTX(runtimeScope, runtimeScope.Context(), &buildResult, false); err != nil {
		t.Fatalf("with manager: %v", err)
	}
	if withMgr.moduleManager.pauseLeaseRenewDepth.Load() != 0 {
		t.Fatal("pause should clear")
	}
	mod2 := &meta.Module{
		Name: "commit_tx_demo_ctx", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod2.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod2).Error; err != nil {
		t.Fatal(err)
	}
	withMgr.module = mod2
	buildResult = nil
	if err := withMgr.runInstallCommitTX(runtimeScope, nil, &buildResult, false); err != nil {
		t.Fatalf("nil ctx: %v", err)
	}
	if err := withMgr.runInstallCommitTX(nil, context.Background(), &buildResult, false); err == nil || !strings.Contains(err.Error(), "scope is nil") {
		t.Fatalf("nil txRoot: %v", err)
	}
	if err := withMgr.runInstallCommitTX(runtimeScope, context.Background(), nil, false); err == nil || !strings.Contains(err.Error(), "build result slot is nil") {
		t.Fatalf("nil build result slot: %v", err)
	}
}

func TestModuleInstallerInstall_RunsCommitPath(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "install_path_demo", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if err := installer.install(); err != nil {
		t.Fatalf("install: %v", err)
	}
	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "install_path_demo").Take(&got).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Status != meta.Installed {
		t.Fatalf("status=%v", got.Status)
	}

	if err := (*moduleInstaller)(nil).installAfterPrepare(nil, false); err == nil || !strings.Contains(err.Error(), "scope is nil") {
		t.Fatalf("nil installer: %v", err)
	}
	if err := (&moduleInstaller{}).installAfterPrepare(nil, false); err == nil || !strings.Contains(err.Error(), "scope is nil") {
		t.Fatalf("nil runtimeScope: %v", err)
	}
	closed := newLifecycleCommitTestScope(t)
	sqlDB, err := closed.Session().DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	failInst := &moduleInstaller{
		module:       &meta.Module{Name: "x", Version: "1", Path: t.TempDir(), ApplicationStr: "auth"},
		runtimeScope: closed, moduleManager: &ModuleManager{runtimeScope: closed, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx: newOpContext(),
	}
	if err := failInst.installAfterPrepare(nil, false); err == nil {
		t.Fatal("expected closed-db error")
	}
}

func TestCommitInstall_NoHooksInsideTX(t *testing.T) {
	// Commit TX must not run pre_init: nil js executor used to fail inside commitInstall.
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "demo_commit_no_hook", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	installer := &moduleInstaller{
		module:       mod,
		runtimeScope: runtimeScope,
		ctx:          newOpContext(),
	}
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall without js executor must succeed (no hooks in TX): %v", err)
	}
	if mod.Status != meta.Installed {
		t.Fatalf("status=%q want installed", mod.Status)
	}
}

func TestInstall_PreInitRunsOutsideCommitTX(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "demo_pre_init_outside", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}

	// Commit alone leaves module installed without needing hooks / js executor.
	installer := &moduleInstaller{
		module:       mod,
		runtimeScope: runtimeScope,
		ctx:          newOpContext(),
	}
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall: %v", err)
	}

	// pre_init without executor fails outside TX helpers.
	noExec := &moduleInstaller{
		module:       mod,
		runtimeScope: runtimeScope,
		ctx:          newOpContext(),
	}
	if err := noExec.runInstallPreInit(nil); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("expected pre_init outside TX to require executor, got %v", err)
	}

	// installAfterPrepare: commit succeeds, then pre_init observes Installed and fails.
	mod2 := &meta.Module{
		Name: "demo_pre_init_after_commit", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod2.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod2).Error; err != nil {
		t.Fatal(err)
	}
	prev := hooksNewRunner
	t.Cleanup(func() { hooksNewRunner = prev })
	hooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		var row meta.Module
		if err := runtimeScope.Session().Where("name = ?", "demo_pre_init_after_commit").Take(&row).Error; err != nil {
			t.Fatal(err)
		}
		if row.Status != meta.Installed {
			t.Fatalf("pre_init must observe the committed install, got status %q", row.Status)
		}
		return nil, errors.New("pre_init boom")
	}
	failing := &moduleInstaller{
		module:        mod2,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if err := failing.installAfterPrepare(nil, false); err == nil || !strings.Contains(err.Error(), "pre_init after commit") {
		t.Fatalf("expected pre_init failure after commit, got %v", err)
	}
	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "demo_pre_init_after_commit").Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != meta.ToInstall {
		t.Fatalf("after commit+failed pre_init status=%q want to install (retryable)", got.Status)
	}

	// Retry with hooks skipped must complete and mark Installed again.
	hooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return nil, nil
	}
	mod2.Status = meta.ToInstall
	retry := &moduleInstaller{
		module:        mod2,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if err := retry.installAfterPrepare(nil, false); err != nil {
		t.Fatalf("retry after pre_init failure must succeed: %v", err)
	}
	if mod2.Status != meta.Installed {
		t.Fatalf("retry status=%q want installed", mod2.Status)
	}
}

func TestInstallAfterPrepare_FinalizeFailureRevertsStatus(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "demo_finalize_fail", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	prev := hooksNewRunner
	t.Cleanup(func() { hooksNewRunner = prev })
	n := 0
	hooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		n++
		if n == 1 {
			return nil, nil // skip pre_init
		}
		return nil, errors.New("finalize runner boom")
	}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	err := installer.installAfterPrepare(nil, false)
	if err == nil || !strings.Contains(err.Error(), "finalizing install after commit") {
		t.Fatalf("got %v", err)
	}
	var got meta.Module
	if dbErr := runtimeScope.Session().Where("name = ?", "demo_finalize_fail").Take(&got).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if got.Status != meta.ToInstall {
		t.Fatalf("status=%q want to install", got.Status)
	}
	hooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return nil, nil
	}
	if err := installer.installAfterPrepare(nil, false); err != nil {
		t.Fatalf("retry after finalize failure must succeed: %v", err)
	}
	if mod.Status != meta.Installed {
		t.Fatalf("retry status=%q want installed", mod.Status)
	}
}

func TestMarkPostCommitHooksIncomplete(t *testing.T) {
	if err := (*moduleInstaller)(nil).markPostCommitHooksIncomplete(); err != nil {
		t.Fatal(err)
	}
	if err := (&moduleInstaller{}).markPostCommitHooksIncomplete(); err != nil {
		t.Fatal(err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	emptyName := &moduleInstaller{
		module:       &meta.Module{Name: "  "},
		runtimeScope: runtimeScope,
	}
	if err := emptyName.markPostCommitHooksIncomplete(); err != nil {
		t.Fatal(err)
	}

	mod := &meta.Module{
		Name: "demo_mark_incomplete", Version: "1.0.0", Status: meta.Installed,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope}
	if err := installer.markPostCommitHooksIncomplete(); err != nil {
		t.Fatal(err)
	}
	if mod.Status != meta.ToInstall {
		t.Fatalf("memory status=%q", mod.Status)
	}
	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "demo_mark_incomplete").Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != meta.ToInstall {
		t.Fatalf("db status=%q", got.Status)
	}
	// Already ToInstall → affected=0.
	if err := installer.markPostCommitHooksIncomplete(); err == nil || !strings.Contains(err.Error(), "was not") {
		t.Fatalf("expected unchanged-status error, got %v", err)
	}

	prev := updatePostCommitIncompleteStatus
	t.Cleanup(func() { updatePostCommitIncompleteStatus = prev })
	updatePostCommitIncompleteStatus = func(*scope.Session, *meta.Module) (int64, error) {
		return 0, errors.New("db boom")
	}
	mod.Status = meta.Installed
	_ = runtimeScope.Session().Model(mod).Update("status", meta.Installed)
	if err := installer.markPostCommitHooksIncomplete(); err == nil || !strings.Contains(err.Error(), "db boom") {
		t.Fatalf("expected db error, got %v", err)
	}
	if mod.Status != meta.Installed {
		t.Fatalf("memory status must stay installed when update fails, got %q", mod.Status)
	}

	// wrapPostCommitHookError joins mark failure.
	failing := &moduleInstaller{
		module:       mod,
		runtimeScope: runtimeScope,
		ctx:          newOpContext(),
	}
	err := failing.wrapPostCommitHookError("error running pre_init after commit (module persisted, not finalized)", errors.New("hook boom"))
	if err == nil || !strings.Contains(err.Error(), "also failed reverting status") || !strings.Contains(err.Error(), "hook boom") {
		t.Fatalf("got %v", err)
	}
}

func TestInstallPreInitHookError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name: "demo_pre_init_err", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	installer := &moduleInstaller{
		module:       mod,
		runtimeScope: runtimeScope,
		ctx:          newOpContext(),
	}
	if err := installer.runInstallPreInit(nil); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("expected pre_init hook error, got %v", err)
	}
	if err := (*moduleInstaller)(nil).runInstallPreInit(nil); err != nil {
		t.Fatalf("nil installer: %v", err)
	}
	if err := (&moduleInstaller{module: mod}).runInstallPreInit(nil); err == nil || !strings.Contains(err.Error(), "scope is nil") {
		t.Fatalf("nil scope: %v", err)
	}
	if err := (&moduleInstaller{runtimeScope: runtimeScope}).runInstallPreInit(nil); err == nil || !strings.Contains(err.Error(), "module is nil") {
		t.Fatalf("nil module: %v", err)
	}
}

func TestFinalizeInstallNoopHooks(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	installer := &moduleInstaller{
		module:        &meta.Module{Name: "demo", Path: t.TempDir()},
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if err := installer.finalizeInstall(nil); err != nil {
		t.Fatalf("finalizeInstall: %v", err)
	}
	noMgr := &moduleInstaller{
		module:       &meta.Module{Name: "demo", Path: t.TempDir()},
		runtimeScope: runtimeScope,
		ctx:          newOpContext(),
	}
	if err := noMgr.finalizeInstall(nil); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("finalizeInstall without manager: %v", err)
	}
	if err := (&moduleInstaller{runtimeScope: runtimeScope}).finalizeInstall(nil); err != nil {
		t.Fatalf("finalizeInstall nil module: %v", err)
	}
	if err := (*moduleInstaller)(nil).finalizeInstall(nil); err != nil {
		t.Fatalf("finalizeInstall nil installer: %v", err)
	}
}

func TestRunInstallHookPhaseBranches(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, nil, nil, hooks.PhasePostInit, nil, "post_init"); err != nil {
		t.Fatalf("nil module: %v", err)
	}
	exec := &moduleManagerNoopScriptExecutor{}
	mod := &meta.Module{Name: "demo"}
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, exec, mod, hooks.PhasePostInit, nil, "post_init"); err != nil {
		t.Fatalf("nil buildResult: %v", err)
	}
	empty := &moduleresult.BuildResult{}
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, exec, mod, hooks.PhasePostInit, empty, "post_init"); err != nil {
		t.Fatalf("empty buildResult: %v", err)
	}
	withScript := &moduleresult.BuildResult{
		EsbuildResult: &api.BuildResult{
			OutputFiles: []api.OutputFile{{Path: "index.js", Contents: []byte("export {}")}},
		},
	}
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, exec, mod, hooks.PhasePostInit, withScript, "post_init"); err != nil {
		t.Fatalf("with script: %v", err)
	}
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, nil, mod, hooks.PhasePostInit, nil, "post_init"); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("nil executor: %v", err)
	}

	origRunner, origScript := hooksNewRunner, hooksScriptFromBuildResult
	t.Cleanup(func() {
		hooksNewRunner = origRunner
		hooksScriptFromBuildResult = origScript
	})
	hooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return nil, errors.New("runner boom")
	}
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, exec, mod, hooks.PhasePostInit, nil, "post_init"); err == nil || !strings.Contains(err.Error(), "runner boom") {
		t.Fatalf("NewRunner err: %v", err)
	}
	hooksNewRunner = origRunner
	hooksScriptFromBuildResult = func(*moduleresult.BuildResult) (*jsengine.JsScript, error) {
		return nil, errors.New("script boom")
	}
	if err := runInstallHookPhase(runtimeScope, nil, plan.OpInstall, exec, mod, hooks.PhasePostInit, empty, "post_init"); err == nil || !strings.Contains(err.Error(), "script boom") {
		t.Fatalf("ScriptFromBuildResult err: %v", err)
	}
}

func TestInstallerJSExecutorAndServiceEntryPoint(t *testing.T) {
	if installerJSExecutor(nil) != nil {
		t.Fatal("nil installer")
	}
	if installerJSExecutor(&moduleInstaller{}) != nil {
		t.Fatal("nil manager")
	}
	exec := &moduleManagerNoopScriptExecutor{}
	got := installerJSExecutor(&moduleInstaller{moduleManager: &ModuleManager{jsExecutor: exec}})
	if got != exec {
		t.Fatalf("jsExecutor=%v", got)
	}
	if installerServiceEntryPoint(nil) != "" {
		t.Fatal("nil installer entry")
	}
	if installerServiceEntryPoint(&moduleInstaller{}) != "" {
		t.Fatal("nil module entry")
	}
	mod := &meta.Module{ServiceEntryPoint: "service/main.ts"}
	if installerServiceEntryPoint(&moduleInstaller{module: mod}) != "service/main.ts" {
		t.Fatalf("entry=%q", installerServiceEntryPoint(&moduleInstaller{module: mod}))
	}
	if installerReuseExecutorScripts(nil) {
		t.Fatal("nil exec reuse")
	}
	if !installerReuseExecutorScripts(exec) {
		t.Fatal("exec reuse")
	}
}

func TestRunInstallCommitTX_PersistLaterNoDuplicateModule(t *testing.T) {
	// BuildWithoutPersist leaves buildResult.Module on the outer installer module.
	// forCommitScope copies that module for the Required TX; Persist must write the
	// copy (via bindCommitBuildModule), otherwise the final Save inserts a second row.
	runtimeScope := newLifecycleCommitTestScope(t)
	outer := &meta.Module{
		Name: "persist_tx_local", Version: "1.0.0", Status: meta.ToInstall,
		Path: t.TempDir(), ApplicationStr: "auth",
	}
	installer := &moduleInstaller{
		module:        outer,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
		// forCommitScope replaces this with a real SplitBuilder; any SplitBuilder is enough
		// so prepare-time type assert would succeed if install() were used.
		builder: &commitStubSplitBuilder{},
	}
	buildResult := &moduleresult.BuildResult{Module: outer}
	if err := installer.runInstallCommitTX(runtimeScope, runtimeScope.Context(), &buildResult, true); err != nil {
		t.Fatalf("runInstallCommitTX: %v", err)
	}
	var count int64
	if err := runtimeScope.Session().Model(&meta.Module{}).Where("name = ?", outer.Name).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("meta_module rows for %s = %d, want 1", outer.Name, count)
	}
	if !outer.Id.Valid || strings.TrimSpace(outer.Id.String) == "" {
		t.Fatal("expected outer module Id copied back after commit")
	}
	if outer.Status != meta.Installed {
		t.Fatalf("status=%v", outer.Status)
	}
	if buildResult == nil || buildResult.Module != outer {
		t.Fatal("published BuildResult.Module must repoint at caller's module")
	}
}

func TestBindCommitBuildModule(t *testing.T) {
	bindCommitBuildModule(nil, &meta.Module{})
	bindCommitBuildModule(&moduleresult.BuildResult{}, nil)
	mod := &meta.Module{Name: "x"}
	br := &moduleresult.BuildResult{Module: &meta.Module{Name: "other"}}
	bindCommitBuildModule(br, mod)
	if br.Module != mod {
		t.Fatal("expected Module rebound")
	}
}

func TestCommitInstallNilInstaller(t *testing.T) {
	if _, err := (*moduleInstaller)(nil).commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "install commit installer is nil") {
		t.Fatalf("nil installer: %v", err)
	}
	if _, err := (&moduleInstaller{}).commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "install commit installer is nil") {
		t.Fatalf("nil module: %v", err)
	}
}

func TestCommitInstallPersistLaterBranches(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:   "demo",
		Path:   t.TempDir(),
		Status: meta.ToInstall,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}

	split := &commitStubSplitBuilder{}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
		builder:       split,
	}
	if _, err := installer.commitInstall(&moduleresult.BuildResult{}, true); err != nil {
		t.Fatalf("persistLater success: %v", err)
	}
	if split.persistCalls != 1 {
		t.Fatalf("persistCalls = %d", split.persistCalls)
	}

	installer.builder = commitStubBuilder{}
	if _, err := installer.commitInstall(&moduleresult.BuildResult{}, true); err == nil || !strings.Contains(err.Error(), "does not support Persist") {
		t.Fatalf("expected Persist unsupported error, got %v", err)
	}
}

func TestCommitInstall_ReturnsBuiltResult(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:           "commit_build_result",
		Version:        "1.0.0",
		Status:         meta.ToInstall,
		Path:           t.TempDir(),
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
		builder:       commitStubBuilder{},
	}
	got, err := installer.commitInstall(nil, false)
	if err != nil {
		t.Fatalf("commitInstall: %v", err)
	}
	if got == nil {
		t.Fatal("expected BuildResult from builder.Build")
	}
}

func TestCommitInstallNewMigratorError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:           "demo_mig_err",
		Version:        "1.0.0",
		Status:         meta.ToInstall,
		Path:           t.TempDir(),
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(runtimeScope.Session().DB, mod.Id.String, []*meta.Model{
		{Name: "A", Path: "/a.ts", ModelTable: "a", ModuleId: mod.Id, Extends: "/b.ts"},
		{Name: "B", Path: "/b.ts", ModelTable: "b", ModuleId: mod.Id, Extends: "/a.ts"},
	}); err != nil {
		t.Fatalf("create circular declaration: %v", err)
	}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if _, err := installer.commitInstall(nil, false); err == nil || !strings.Contains(err.Error(), "error preparing schema migrator") {
		t.Fatalf("expected NewMigrator error, got %v", err)
	}
}

func TestCommitUpgradeNewMigratorError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:           "demo_upgrade_mig_err",
		Version:        "1.0.0",
		Status:         meta.Installed,
		Path:           t.TempDir(),
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{
		Name:           mod.Name,
		Version:        "1.1.0",
		Status:         meta.ToUpgrade,
		Path:           mod.Path,
		ApplicationStr: "auth",
	}
	target.Id = mod.Id
	if _, err := modmeta.ReplaceModuleDeclarations(runtimeScope.Session().DB, target.Id.String, []*meta.Model{
		{Name: "A", Path: "/ua.ts", ModelTable: "ua", ModuleId: target.Id, Extends: "/ub.ts"},
		{Name: "B", Path: "/ub.ts", ModelTable: "ub", ModuleId: target.Id, Extends: "/ua.ts"},
	}); err != nil {
		t.Fatalf("create circular declaration: %v", err)
	}
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
	if _, err := upgrader.commitUpgrade(installer, "1.0.0", nil, false); err == nil || !strings.Contains(err.Error(), "error preparing schema migrator") {
		t.Fatalf("expected NewMigrator error, got %v", err)
	}
}

func TestCommitInstallSaveModuleError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:    "demo_save_err",
		Version: "1.0.0",
		Status:  meta.ToInstall,
		Path:    t.TempDir(),
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	origDeps := replaceModuleDependenciesFn
	t.Cleanup(func() { replaceModuleDependenciesFn = origDeps })
	replaceModuleDependenciesFn = func(*scope.Session, *meta.Module) error { return nil }
	if err := runtimeScope.Session().Exec(`
CREATE TRIGGER block_module_save
BEFORE UPDATE ON meta_module
BEGIN
  SELECT RAISE(ABORT, 'module save blocked');
END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	_, err := installer.commitInstall(nil, false)
	if err == nil || !strings.Contains(err.Error(), "error saving module:") || strings.Contains(err.Error(), "dependencies") {
		t.Fatalf("expected save module error, got %v", err)
	}
}

func TestCommitUpgradeSaveModuleError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{
		Name:    "demo_upgrade_save_err",
		Version: "1.0.0",
		Status:  meta.Installed,
		Path:    t.TempDir(),
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	target := &meta.Module{
		Name:    mod.Name,
		Version: "2.0.0",
		Status:  meta.Installed,
		Path:    mod.Path,
	}
	target.Id = mod.Id
	origDeps := replaceModuleDependenciesFn
	t.Cleanup(func() { replaceModuleDependenciesFn = origDeps })
	replaceModuleDependenciesFn = func(*scope.Session, *meta.Module) error { return nil }
	if err := runtimeScope.Session().Exec(`
CREATE TRIGGER block_module_upgrade_save
BEFORE UPDATE ON meta_module
BEGIN
  SELECT RAISE(ABORT, 'module upgrade save blocked');
END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
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
	if err == nil || !strings.Contains(err.Error(), "error saving module:") || strings.Contains(err.Error(), "dependencies") {
		t.Fatalf("expected upgrade save module error, got %v", err)
	}
}

func TestCommitInstallMetaAndDocumentSchedules(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&internaltask.Schedule{}, &meta.Module{}); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	now := time.Now().UTC()
	if err := db.Create(&internaltask.Schedule{
		Id: "sch_legacy", Active: true, Name: "meta.module_index.daily_sync",
		TargetApp: "meta", FullMethod: "meta.MetaModuleIndex/Sync",
		SchedulerUserId: "admin", TriggeredByUserId: "admin",
		CronExpr: "0 0 * * *", Timezone: "UTC", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	modulesPath := t.TempDir()
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)

	metaMod := &meta.Module{Name: "meta", Path: filepath.Join(modulesPath, "meta"), Status: meta.ToInstall}
	metaMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	installer := &moduleInstaller{
		module:        metaMod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("meta commitInstall: %v", err)
	}
	if err := internaltask.WhereScheduleNameEq(db, "meta.module_index.daily_sync").Take(&internaltask.Schedule{}).Error; err == nil {
		t.Fatal("expected legacy schedule deleted")
	}

	docMod := &meta.Module{Name: "document", Path: filepath.Join(modulesPath, "document"), Status: meta.ToInstall}
	docMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	installer.module = docMod
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("document commitInstall: %v", err)
	}
	var gc internaltask.Schedule
	if err := internaltask.WhereScheduleNameEq(db, "document.attachment.gc").Take(&gc).Error; err != nil {
		t.Fatalf("expected GC schedule: %v", err)
	}
	if !gc.Active || gc.CronExpr != "*/5 * * * *" {
		t.Fatalf("gc schedule = %+v", gc)
	}
	if internaltask.DecodeTranslatedScheduleName(gc.Name) != "document.attachment.gc" {
		t.Fatalf("expected translated schedule name, got %q", gc.Name)
	}

	// Update existing GC schedule path.
	if _, err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("document commitInstall update: %v", err)
	}
}

func TestEnsureDocumentAttachmentGCScheduleDirect(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&internaltask.Schedule{}); err != nil {
		t.Fatal(err)
	}
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	if err := ensureDocumentAttachmentGCSchedule(runtimeScope); err != nil {
		t.Fatal(err)
	}
	if err := ensureDocumentAttachmentGCSchedule(runtimeScope); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := internaltask.WhereScheduleNameEq(db.Model(&internaltask.Schedule{}), "document.attachment.gc").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
	_ = mustJSON(map[string]any{"ok": true})
}

func TestRestoreModuleIfSoftDeletedStandalone(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.Module{Name: "solo", Status: meta.Uninstalled, Path: t.TempDir()}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	mod.DeletedAt = gorm.DeletedAt{Time: time.Now().UTC(), Valid: true}
	if err := runtimeScope.Session().Unscoped().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope}
	if err := installer.restoreModuleIfSoftDeleted(); err != nil {
		t.Fatal(err)
	}
	var got meta.Module
	if err := runtimeScope.Session().Where("name = ?", "solo").Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected restored")
	}
}
