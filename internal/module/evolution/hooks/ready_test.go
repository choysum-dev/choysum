// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestRunReadyForInstalledModulesSkipsWhenNoReadyHooks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "index.ts"), "export const x = 1\n")
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), nil, RunReadyOptions{
		Modules: []meta.Module{{Name: "noop", Path: root, ApplicationStr: "noop"}},
	})
	if err != nil {
		t.Fatalf("expected nil with nil executor when no ready hooks, got %v", err)
	}
}

func TestRunReadyForInstalledModulesFiltersByApplication(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), `
export class ReadyHooks {
  @HookReady()
  static async onReady(): Promise<void> {}
}
`)
	other := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(other, "service", "hook", "ready.ts"), `
export class ReadyHooks {
  @HookReady()
  static async onReady(): Promise<void> {}
}
`)
	// Nil executor: modules that declare ready would call NewRunner then RunPhase and fail
	// if selected. Filter to empty application set → no candidates → nil.
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), nil, RunReadyOptions{
		ApplicationNames: []string{"only-missing"},
		Modules: []meta.Module{
			{Name: "doc", Path: root, ApplicationStr: "document"},
			{Name: "other", Path: other, ApplicationStr: "meta"},
		},
	})
	if err != nil {
		t.Fatalf("expected no candidates, got %v", err)
	}
}

func TestRunReadyForInstalledModulesNilRuntimeScope(t *testing.T) {
	t.Parallel()
	err := RunReadyForInstalledModules(context.Background(), nil, &readyHookExec{}, RunReadyOptions{
		Modules: []meta.Module{{Name: "x", Path: t.TempDir()}},
	})
	if err != nil {
		t.Fatalf("nil scope should no-op, got %v", err)
	}
}

func TestRunReadyForInstalledModulesSkipsEmptyModuleName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), "@HookReady()\n")
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), nil, RunReadyOptions{
		Modules: []meta.Module{{Name: "   ", Path: root, ApplicationStr: "document"}},
	})
	if err != nil {
		t.Fatalf("empty name should skip module, got %v", err)
	}
}

func TestRunReadyForInstalledModulesNilExecutorWithCandidates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), `
export class ReadyHooks {
  @HookReady()
  static async onReady(): Promise<void> {}
}
`)
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), nil, RunReadyOptions{
		Modules: []meta.Module{{Name: "doc", Path: root, ApplicationStr: "document"}},
	})
	if err == nil || !strings.Contains(err.Error(), "js executor is nil") {
		t.Fatalf("expected nil executor error, got %v", err)
	}
}

type readyHookExec struct {
	calls int
}

func (e *readyHookExec) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	e.calls++
	return &jsengine.JsResponse{Id: req.Id, Result: nil}, nil
}
func (e *readyHookExec) GetJsScripts() []*jsengine.JsScript         { return nil }
func (e *readyHookExec) SetJsScripts(scripts []*jsengine.JsScript)  {}
func (e *readyHookExec) Reload(scripts ...*jsengine.JsScript) error { return nil }

func TestRunReadyForInstalledModulesInvokeOnlySuccess(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), `
export class ReadyHooks {
  @HookReady()
  static async onReady(): Promise<void> {}
}
`)
	exec := &readyHookExec{}
	err := RunReadyForInstalledModules(context.Background(), newHooksTestScope(t), exec, RunReadyOptions{
		Modules:    []meta.Module{{Name: "doc", Path: root, ApplicationStr: "document"}},
		InvokeOnly: true,
	})
	if err != nil {
		t.Fatalf("RunReadyForInstalledModules: %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("expected one hook invoke, got %d", exec.calls)
	}
}

func TestRunReadyForInstalledModulesCyclicDependencyWarns(t *testing.T) {
	t.Parallel()
	rootA := t.TempDir()
	rootB := t.TempDir()
	for _, root := range []string{rootA, rootB} {
		mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), "@HookReady()\n")
	}
	depends := func(names ...string) []byte {
		raw, _ := json.Marshal(names)
		return raw
	}
	var logBuf bytes.Buffer
	scope := newHooksTestScope(t)
	scope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	err := RunReadyForInstalledModules(context.Background(), scope, &readyHookExec{}, RunReadyOptions{
		Modules: []meta.Module{
			{Name: "a", Path: rootA, ApplicationStr: "document", DependsStr: depends("b")},
			{Name: "b", Path: rootB, ApplicationStr: "document", DependsStr: depends("a")},
		},
		InvokeOnly: true,
	})
	if err != nil {
		t.Fatalf("cyclic order should still run, got %v", err)
	}
	if !strings.Contains(logBuf.String(), "dependency cycle") {
		t.Fatalf("expected cycle warning, got %q", logBuf.String())
	}
}

func TestRunReadyForInstalledModulesRunPhaseError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "ready.ts"), "@HookReady()\n")
	testScope := newHooksTestScope(t)
	engine := &hooksSelectiveEngine{execute: func(_ context.Context, _ *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
		return nil, errors.New("HOOK_UNSUPPORTED: ready failed")
	}}
	executor := newHooksTestExecutorWithEngine(t, testScope, engine)
	err := RunReadyForInstalledModules(context.Background(), testScope, executor, RunReadyOptions{
		Modules:    []meta.Module{{Name: "doc", Path: root, ApplicationStr: "document"}},
		InvokeOnly: true,
	})
	if err == nil || !strings.Contains(err.Error(), "ready hook for module doc") {
		t.Fatalf("expected wrapped RunPhase error, got %v", err)
	}
}
