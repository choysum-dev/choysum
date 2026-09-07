// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
)

func vueHostFixtureDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "vue_host")
}

func vueHostRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func newVueHostCompiler(t *testing.T) jsexecutor.ScriptExecutor {
	t.Helper()
	cfg := &config.Config{
		Server: &config.ServerConfig{
			JsEngineFactory:   "quickjs",
			JsExecutorFactory: "default",
		},
	}
	runtimeScope := &spikeBuildScope{ctx: t.Context(), cfg: cfg}
	executor, err := jsexecutor.NewCompilerExecutor(runtimeScope)
	if err != nil {
		t.Fatalf("NewCompilerExecutor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = executor.Stop() })
	return executor
}

func runVueHostEntry(t *testing.T, entryName string) map[string]any {
	t.Helper()
	repoRoot := vueHostRepoRoot(t)
	fixtureDir := vueHostFixtureDir(t)
	entryPath := filepath.Join(fixtureDir, entryName)
	outJS := filepath.Join(t.TempDir(), "out", entryName+".bundle.js")

	executor := newVueHostCompiler(t)
	bundle, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:      repoRoot,
		EntryPath:     entryPath,
		Outfile:       outJS,
		Sourcemap:     false,
		WorkingDir:    fixtureDir,
		JsExecutor:    executor,
		WithVuePlugin: true,
	})
	if err != nil {
		t.Fatalf("BuildFrontendVueHostBundle: %v", err)
	}

	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if err := PrepareVueHostEngine(engine); err != nil {
		t.Fatalf("PrepareVueHostEngine: %v", err)
	}
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: bundle.JSPath, Content: bundle.JS},
	}); err != nil {
		t.Fatalf("Load bundle: %v", err)
	}

	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatalf("engine type %T", engine)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		qjs.Ctx.ProcessJobs()
		_ = qjs.Ctx.LoopOnce()

		val := qjs.Ctx.Eval(`(typeof globalThis.__hostResult === "undefined") ? "null" : JSON.stringify(globalThis.__hostResult)`)
		if val.IsException() {
			err := qjs.Ctx.Exception()
			val.Free()
			t.Fatalf("read __hostResult: %v", err)
		}
		raw := val.String()
		val.Free()
		if raw != "" && raw != "null" {
			var result map[string]any
			if err := json.Unmarshal([]byte(raw), &result); err != nil {
				t.Fatalf("parse __hostResult: %v raw=%s", err, raw)
			}
			if ready, _ := result["ready"].(bool); ready {
				return result
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for __hostResult.ready; raw=%s", raw)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestMinimalDOM_createAppMount(t *testing.T) {
	result := runVueHostEntry(t, "entry_mount.ts")
	if has, _ := result["hasBtn"].(bool); !has {
		t.Fatalf("expected bump button: %#v", result)
	}
	if marker, _ := result["markerText"].(string); strings.TrimSpace(marker) != "42" {
		t.Fatalf("marker = %#v", result["markerText"])
	}
	if after, _ := result["btnTextAfter"].(string); strings.TrimSpace(after) != "1" {
		t.Fatalf("expected click to bump count to 1; got %#v", result)
	}
}

func TestChoysumMount_stubsAndFind(t *testing.T) {
	result := runVueHostEntry(t, "entry_stubs.ts")
	if label, _ := result["label"].(string); strings.TrimSpace(label) != "parent" {
		t.Fatalf("label = %#v", result)
	}
	if has, _ := result["hasStub"].(bool); !has {
		t.Fatalf("expected stub child: %#v", result)
	}
	if hasReal, _ := result["hasRealChild"].(bool); hasReal {
		t.Fatalf("real child should be stubbed away: %#v", result)
	}
}

func TestChoysumMount_flushPromises(t *testing.T) {
	result := runVueHostEntry(t, "entry_flush.ts")
	if before, _ := result["before"].(string); strings.TrimSpace(before) != "pending" {
		t.Fatalf("before = %#v", result)
	}
	if after, _ := result["after"].(string); strings.TrimSpace(after) != "done" {
		t.Fatalf("after = %#v", result)
	}
}

func TestFrozenVTUSubsetAPIs(t *testing.T) {
	if len(FrozenVTUSubsetAPIs) < 6 {
		t.Fatalf("FrozenVTUSubsetAPIs = %v", FrozenVTUSubsetAPIs)
	}
	if VueHostPackageVersion() == "" {
		t.Fatal("empty VueHostPackageVersion")
	}
	if ChoysumMountScript() == "" || !strings.Contains(ChoysumMountScript(), "export function mount") {
		t.Fatal("ChoysumMountScript missing mount export")
	}
}
