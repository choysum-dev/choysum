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

	"github.com/choysum-dev/choysum/internal/module/artifact/pipeline"
	moduleresult "github.com/choysum-dev/choysum/internal/module/artifact/result"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type codegenStubSplitBuilder struct {
	entrySeen    string
	buildErr     error
	persistErr   error
	persistCalls int
}

func (b *codegenStubSplitBuilder) Build() (*moduleresult.BuildResult, error) {
	return &moduleresult.BuildResult{}, nil
}

func (b *codegenStubSplitBuilder) BuildWithoutPersist() (*moduleresult.BuildResult, error) {
	if b.buildErr != nil {
		return nil, b.buildErr
	}
	return &moduleresult.BuildResult{}, nil
}

func (b *codegenStubSplitBuilder) Persist(result *moduleresult.BuildResult) error {
	b.persistCalls++
	return b.persistErr
}

func TestEnsureInjectedAppModelsForCodegenEarlyReturns(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := (*ModuleManager)(nil).ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{}); err != nil {
		t.Fatalf("nil manager: %v", err)
	}
	if err := manager.ensureInjectedAppModelsForCodegen(context.Background(), nil); err != nil {
		t.Fatalf("nil mod: %v", err)
	}
	if err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{ApplicationStr: "core"}); err != nil {
		t.Fatalf("core app: %v", err)
	}
	if err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{ApplicationStr: "  "}); err != nil {
		t.Fatalf("blank app: %v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.ensureInjectedAppModelsForCodegen(canceled, &meta.Module{ApplicationStr: "auth"}); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestEnsureInjectedAppModelsForCodegenCountPaths(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)

	if err := db.Create(&meta.Model{
		Name:        "TranslationTerm",
		Path:        "/virtual/auth/TranslationTerm",
		Application: "auth",
		Abstract:    false,
	}).Error; err != nil {
		t.Fatalf("seed TranslationTerm: %v", err)
	}
	if err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{
		Name: "auth", ApplicationStr: "auth",
	}); err != nil {
		t.Fatalf("count>0 early return: %v", err)
	}

	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{
		Name: "auth", ApplicationStr: "auth",
	})
	if err == nil || !strings.Contains(err.Error(), "count TranslationTerm") {
		t.Fatalf("count error = %v", err)
	}
}

func TestEnsureInjectedAppModelsForCodegenBuilderBranches(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	modulesPath := t.TempDir()
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)

	prev := newCodegenModuleBuilderFn
	t.Cleanup(func() { newCodegenModuleBuilderFn = prev })

	t.Run("not split builder", func(t *testing.T) {
		newCodegenModuleBuilderFn = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, string) any {
			return struct{}{}
		}
		err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{
			Name: "auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts",
		})
		if err == nil || !strings.Contains(err.Error(), "does not support BuildWithoutPersist") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("build error", func(t *testing.T) {
		stub := &codegenStubSplitBuilder{buildErr: errors.New("build boom")}
		newCodegenModuleBuilderFn = func(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, entry string) any {
			stub.entrySeen = entry
			return stub
		}
		err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{
			Name: "auth", ApplicationStr: "auth", ServiceEntryPoint: "service/index.ts",
		})
		if err == nil || !strings.Contains(err.Error(), "rebuild module auth") {
			t.Fatalf("err = %v", err)
		}
		want := filepath.Join(modulesPath, "auth", "service/index.ts")
		if stub.entrySeen != want {
			t.Fatalf("entry = %q, want %q", stub.entrySeen, want)
		}
	})

	t.Run("persist error", func(t *testing.T) {
		stub := &codegenStubSplitBuilder{persistErr: errors.New("persist boom")}
		newCodegenModuleBuilderFn = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, string) any {
			return stub
		}
		err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{
			Name: "web", ApplicationStr: "web", ServiceEntryPoint: "",
		})
		if err == nil || !strings.Contains(err.Error(), "persist TranslationTerm inject") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		stub := &codegenStubSplitBuilder{}
		newCodegenModuleBuilderFn = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, string) any {
			return stub
		}
		if err := manager.ensureInjectedAppModelsForCodegen(context.Background(), &meta.Module{
			Name: "web", ApplicationStr: "web", ServiceEntryPoint: "/abs/entry.ts",
		}); err != nil {
			t.Fatalf("success path: %v", err)
		}
		if stub.persistCalls != 1 {
			t.Fatalf("persistCalls = %d, want 1", stub.persistCalls)
		}
	})
}

func TestGenerateAppToDirsPropagatesEnsureInjectedError(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&meta.Module{
		Name: "auth", ApplicationStr: "auth", Status: meta.Installed,
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	prev := newCodegenModuleBuilderFn
	newCodegenModuleBuilderFn = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, string) any {
		return &codegenStubSplitBuilder{buildErr: errors.New("inject boom")}
	}
	t.Cleanup(func() { newCodegenModuleBuilderFn = prev })

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	err := manager.generateAppToDirs(context.Background(), "auth", pipeline.ModulesAppTargets{}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "rebuild module auth") {
		t.Fatalf("err = %v, want ensureInjected rebuild error", err)
	}
}

func TestBuildBackendAppToDir_EnsureOnlyStubAndInjectError(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	modPath := filepath.Join(modulesPath, "web")
	if err := os.MkdirAll(modPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	baseModel := filepath.Join(modulesPath, "core", "service", "orm", "model", "translation_term_base_model.ts")
	if err := os.MkdirAll(filepath.Dir(baseModel), 0o755); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}
	if err := os.WriteFile(baseModel, []byte("export default class TranslationTermBaseModel {}\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := db.Create(&meta.Module{
		Name: "web", ApplicationStr: "web", Status: meta.Installed,
		ServiceEntryPoint: "", Path: modPath,
	}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	distAppDir := t.TempDir()

	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	err := manager.buildBackendAppToDir(context.Background(), "web", distAppDir)
	if err == nil || !strings.Contains(err.Error(), "inject app models for bundles") {
		t.Fatalf("expected inject error, got %v", err)
	}

	entryRaw, readErr := os.ReadFile(filepath.Join(distAppDir, "__choysum_app_entry.ts"))
	if readErr != nil {
		t.Fatalf("read entry: %v", readErr)
	}
	if !strings.Contains(string(entryRaw), "export {};") {
		t.Fatalf("expected stub export for Ensure-only web, got %q", entryRaw)
	}
}
