// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

	modulegenerator "github.com/choysum-dev/choysum/internal/module/artifact/generate"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilterRegistry_SortsAndFiltersByVersionPhaseAndOrder(t *testing.T) {
	entries := []RegistryEntry{
		{Version: "0.2.0", Phase: PhasePre, Order: 2, Name: "b"},
		{Version: "0.1.0", Phase: PhasePre, Order: 1, Name: "a"},
		{Version: "0.2.0", Phase: PhasePre, Order: 1, Name: "a"},
		{Version: "0.3.0", Phase: PhasePost, Order: 1, Name: "c"},
	}

	filtered := filterRegistry(entries, "0.1.0", "0.2.0", PhasePre)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(filtered))
	}
	if filtered[0].Version != "0.2.0" || filtered[0].Order != 1 || filtered[0].Name != "a" {
		t.Fatalf("unexpected first entry: %#v", filtered[0])
	}
	if filtered[1].Version != "0.2.0" || filtered[1].Order != 2 || filtered[1].Name != "b" {
		t.Fatalf("unexpected second entry: %#v", filtered[1])
	}
}

func TestNormalizeVersion_AddsPrefix(t *testing.T) {
	if got := normalizeVersion("0.1.0"); got != "v0.1.0" {
		t.Fatalf("expected v0.1.0, got %q", got)
	}
	if got := normalizeVersion("v0.1.0"); got != "v0.1.0" {
		t.Fatalf("expected v0.1.0, got %q", got)
	}
}

func TestRunnerHelpers(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	mod := &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.0.0", ServiceEntryPoint: "service/index.ts"}

	runner := NewRunner(testRuntimeScope, nil, mod)
	if runner == nil || runner.store == nil {
		t.Fatalf("expected runner with history store, got %#v", runner)
	}
	if NewRunner(nil, nil, nil) != nil {
		t.Fatal("expected nil runner when env/module missing")
	}

	runner.module.Name = ""
	if wrapper := runner.buildMigrationWrapperScript(); wrapper != nil {
		t.Fatalf("expected nil wrapper without module name, got %#v", wrapper)
	}
	runner.module.Name = "base"
	if wrapper := runner.buildMigrationWrapperScript(); wrapper == nil || !strings.Contains(wrapper.Content, `__choysum_migration_list__`) || !strings.Contains(wrapper.Content, `MIGRATION_FAILED`) {
		t.Fatalf("unexpected migration wrapper: %#v", wrapper)
	}

	runner.module.ServiceEntryPoint = ""
	if script, err := runner.buildModuleEntryScript(context.Background()); err != nil || script != nil {
		t.Fatalf("expected nil entry script for empty entrypoint, got %#v err=%v", script, err)
	}
	if resolved, err := runner.resolveScripts(context.Background()); err != nil || resolved != nil {
		t.Fatalf("expected nil resolveScripts with no runtime scripts, got %#v err=%v", resolved, err)
	}
}

func TestBuildModuleEntryScript_PrefersContextSessionForBuilderRuntimeState(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	testRuntimeScope.cfg.DefaultChoysumPath = filepath.Join(t.TempDir(), ".choysum")
	if err := os.MkdirAll(testRuntimeScope.cfg.ModulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testRuntimeScope.cfg.ModulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":".","paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	entryPoint := filepath.Join(testRuntimeScope.cfg.ModulesPath, "base", "service", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entryPoint), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPoint, []byte("export const migration = {};\n"), 0o644); err != nil {
		t.Fatalf("write entry point: %v", err)
	}

	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration_runtime_builder.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	if err := runtimeDB.AutoMigrate(&meta.IrModule{}); err != nil {
		t.Fatalf("migrate runtime modules: %v", err)
	}
	_, _, serviceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "crm", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(crm) error = %v", err)
	}
	serviceIndex := filepath.Join(serviceDir, "index.ts")
	if err := os.MkdirAll(filepath.Dir(serviceIndex), 0o755); err != nil {
		t.Fatalf("mkdir service index dir: %v", err)
	}
	if err := os.WriteFile(serviceIndex, []byte("globalThis.__migrationInjected = (globalThis.__migrationInjected || 0) + 1;\nexport {};\n"), 0o644); err != nil {
		t.Fatalf("write service index: %v", err)
	}
	if err := runtimeDB.Create(&meta.IrModule{Name: "crm_mod", ApplicationStr: "crm", Status: meta.Installed, ServiceEntryPoint: "./service/index.ts"}).Error; err != nil {
		t.Fatalf("seed runtime installed module: %v", err)
	}

	runner := NewRunner(testRuntimeScope, nil, &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"})
	baseScript, err := runner.buildModuleEntryScript(context.Background())
	if err != nil {
		t.Fatalf("buildModuleEntryScript(base) error = %v", err)
	}
	runtimeScope := &scriptsTestScope{ctx: context.Background(), cfg: testRuntimeScope.cfg, logger: testRuntimeScope.logger, session: &scope.Session{DB: runtimeDB}}
	runtimeCtx := scope.ContextWithScope(context.Background(), runtimeScope)
	runtimeScript, err := runner.buildModuleEntryScript(runtimeCtx)
	if err != nil {
		t.Fatalf("buildModuleEntryScript(runtime) error = %v", err)
	}
	if baseScript == nil || runtimeScript == nil {
		t.Fatalf("expected both scripts to be built, base=%#v runtime=%#v", baseScript, runtimeScript)
	}
	if strings.Contains(baseScript.Content, "__migrationInjected") {
		t.Fatalf("expected base build to skip runtime-only import, got: %s", baseScript.Content)
	}
	if !strings.Contains(runtimeScript.Content, "__migrationInjected") {
		t.Fatalf("expected runtime build to include runtime-only import, got: %s", runtimeScript.Content)
	}
}

func TestRunnerValidationAndParsingHelpers(t *testing.T) {
	if err := (*Runner)(nil).RunPhase(context.Background(), RunOptions{Phase: PhasePre}); err != nil {
		t.Fatalf("expected nil runner RunPhase to be no-op, got %v", err)
	}

	testRuntimeScope := newScriptsTestScope(t)
	runner := &Runner{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "base", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"}, store: NewHistoryStore(testRuntimeScope)}
	runner.jsExecutor = jsexecutortest.NewUninitializedExecutor()
	if err := runner.RunPhase(context.Background(), RunOptions{}); err != nil {
		t.Fatalf("expected empty phase RunPhase to be no-op, got %v", err)
	}
	runner.jsExecutor = nil
	if err := runner.Validate(context.Background(), "", ""); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("expected Validate to require js executor, got %v", err)
	}
	if err := runner.RunPhase(context.Background(), RunOptions{Phase: PhasePre}); err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("expected RunPhase to require js executor, got %v", err)
	}

	entries, err := decodeRegistry([]map[string]any{{"version": "1.0.0", "phase": "pre", "order": 1, "name": "init"}})
	if err != nil || len(entries) != 1 || entries[0].Name != "init" {
		t.Fatalf("unexpected decodeRegistry result: %#v err=%v", entries, err)
	}
	if _, err := decodeRegistry(make(chan int)); err == nil {
		t.Fatal("expected decodeRegistry to fail for unsupported raw value")
	}

	entry := RegistryEntry{Version: "1.0.0", Phase: PhasePre, Order: 2, Name: "seed"}
	checksum := checksumForEntry(entry)
	if len(checksum) != 64 || checksum != checksumForEntry(entry) {
		t.Fatalf("unexpected checksum: %q", checksum)
	}

	if script, err := ScriptFromBuildResult(nil); err != nil || script != nil {
		t.Fatalf("expected nil script for nil build result, got %#v err=%v", script, err)
	}
	script, err := ScriptFromBuildResult(&module.BuildResult{EsbuildResult: &api.BuildResult{OutputFiles: []api.OutputFile{{Path: "chunk.css", Contents: []byte("body{}")}, {Path: "other.js", Contents: []byte("console.log('other')")}, {Path: "index.js", Contents: []byte("console.log('index')")}}}})
	if err != nil || script == nil || script.FileName != "index.js" {
		t.Fatalf("unexpected selected script: %#v err=%v", script, err)
	}
}

func TestRunnerRunPhaseExecutesMigrationsAndRestoresScripts(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	bundlePath := writeScriptsRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
	moduleRef := &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.2.0", ServiceEntryPoint: "service/index.ts"}

	var services []string
	var migrationArgs [][]interface{}
	loadedByService := map[string][]string{}
	engine := &scriptsSelectiveEngine{execute: func(req *jsengine.JsRequest, loaded []*jsengine.JsScript) (*jsengine.JsResponse, error) {
		services = append(services, req.Service)
		fileNames := make([]string, 0, len(loaded))
		for _, script := range loaded {
			fileNames = append(fileNames, script.FileName)
		}
		loadedByService[req.Service] = append([]string(nil), fileNames...)

		switch req.Service {
		case "__choysum_migration_list__":
			return &jsengine.JsResponse{Id: req.Id, Result: []map[string]any{{"version": "1.0.0", "phase": "pre", "order": 1, "name": "ignored-old"}, {"version": "1.2.0", "phase": "post", "order": 1, "name": "ignored-phase"}, {"version": "1.2.0", "phase": "pre", "order": 2, "name": "seed"}, {"version": "1.1.0", "phase": "pre", "order": 1, "name": "init"}}}, nil
		case "__choysum_migration__":
			migrationArgs = append(migrationArgs, append([]interface{}(nil), req.Args...))
			return &jsengine.JsResponse{Id: req.Id, Result: map[string]any{"ok": true}}, nil
		default:
			return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
		}
	}}
	executor := newScriptsTestExecutorWithEngine(t, testRuntimeScope, engine)
	prevScripts := []*jsengine.JsScript{{FileName: "prev.js", Content: "prev"}}
	executor.SetJsScripts(prevScripts)
	runner := NewRunner(testRuntimeScope, executor, moduleRef)

	err := runner.RunPhase(context.Background(), RunOptions{FromVersion: "1.0.0", ToVersion: "1.2.0", Phase: PhasePre})
	if err != nil {
		t.Fatalf("RunPhase() error = %v", err)
	}

	if !reflect.DeepEqual(executor.GetJsScripts(), prevScripts) {
		t.Fatalf("expected executor scripts restored, got %#v", executor.GetJsScripts())
	}
	if got := loadedByService["__choysum_migration_list__"]; !reflect.DeepEqual(got, []string{bundlePath, "migration_wrapper.js"}) {
		t.Fatalf("unexpected scripts used for registry load: %#v", got)
	}
	if !reflect.DeepEqual(services, []string{"__choysum_migration_list__", "__choysum_migration__", "__choysum_migration__"}) {
		t.Fatalf("unexpected service sequence: %#v", services)
	}
	if !reflect.DeepEqual(migrationArgs, [][]interface{}{{"1.1.0", "pre", "init"}, {"1.2.0", "pre", "seed"}}) {
		t.Fatalf("unexpected migration args: %#v", migrationArgs)
	}

	var history []metadata.IrModuleMigrationHistory
	if err := testRuntimeScope.session.WithContext(context.Background()).Order("version asc, script asc").Find(&history).Error; err != nil {
		t.Fatalf("load migration history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history rows, got %d", len(history))
	}
	if history[0].Status != "success" || history[0].Version != "1.1.0" || history[0].Script != "init" {
		t.Fatalf("unexpected first history row: %#v", history[0])
	}
	if history[1].Status != "success" || history[1].Version != "1.2.0" || history[1].Script != "seed" {
		t.Fatalf("unexpected second history row: %#v", history[1])
	}
}

func TestRunnerValidateAndRunPhaseFailurePaths(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	writeScriptsRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
	moduleRef := &meta.IrModule{Name: "base", ApplicationStr: "core", Version: "1.2.0", ServiceEntryPoint: "service/index.ts"}

	t.Run("Validate restores previous scripts on registry error", func(t *testing.T) {
		engine := &scriptsSelectiveEngine{execute: func(req *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			if req.Service == "__choysum_migration_list__" {
				return nil, errors.New("registry boom")
			}
			return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
		}}
		executor := newScriptsTestExecutorWithEngine(t, testRuntimeScope, engine)
		prevScripts := []*jsengine.JsScript{{FileName: "prev.js", Content: "prev"}}
		executor.SetJsScripts(prevScripts)
		runner := NewRunner(testRuntimeScope, executor, moduleRef)

		err := runner.Validate(context.Background(), "1.0.0", "1.2.0")
		if err == nil || !strings.Contains(err.Error(), "registry boom") {
			t.Fatalf("expected registry error, got %v", err)
		}
		if !reflect.DeepEqual(executor.GetJsScripts(), prevScripts) {
			t.Fatalf("expected executor scripts restored after Validate, got %#v", executor.GetJsScripts())
		}
	})

	t.Run("RunPhase marks failed history on migration error", func(t *testing.T) {
		engine := &scriptsSelectiveEngine{execute: func(req *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
			switch req.Service {
			case "__choysum_migration_list__":
				return &jsengine.JsResponse{Id: req.Id, Result: []map[string]any{{"version": "1.1.0", "phase": "pre", "order": 1, "name": "broken"}}}, nil
			case "__choysum_migration__":
				return nil, errors.New("migration boom")
			default:
				return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
			}
		}}
		executor := newScriptsTestExecutorWithEngine(t, testRuntimeScope, engine)
		runner := NewRunner(testRuntimeScope, executor, moduleRef)

		err := runner.RunPhase(context.Background(), RunOptions{FromVersion: "1.0.0", ToVersion: "1.2.0", Phase: PhasePre})
		if err == nil || !strings.Contains(err.Error(), "migration boom") {
			t.Fatalf("expected migration failure, got %v", err)
		}

		var row metadata.IrModuleMigrationHistory
		if err := testRuntimeScope.session.WithContext(context.Background()).Where("script = ?", "broken").Take(&row).Error; err != nil {
			t.Fatalf("load failed history row: %v", err)
		}
		if row.Status != "failed" || !strings.Contains(row.Error, "migration boom") {
			t.Fatalf("unexpected failed history row: %#v", row)
		}
	})
}
