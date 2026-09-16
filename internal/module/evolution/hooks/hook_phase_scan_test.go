// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestModuleSourceDeclaresHookPhase(t *testing.T) {
	root := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "post_init.ts"), `
export class Hooks {
  @HookPostInit()
  static async ensure(): Promise<void> {}
}
`)
	mustWriteHookScanFile(t, filepath.Join(root, "service", "hook", "post_init.test.ts"), `
  @HookPreInit()
  static async onlyInTest(): Promise<void> {}
`)
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: root}, PhasePostInit) {
		t.Fatal("expected @HookPostInit to be detected")
	}
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: root}, PhasePreInit) {
		t.Fatal("test-only @HookPreInit must not count")
	}

	commented := filepath.Join(root, "commented")
	if err := os.MkdirAll(commented, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(commented, "hook.ts"), `
export class Hooks {
  // @HookPostInit()
  static async ensure(): Promise<void> {}
}
`)
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: commented}, PhasePostInit) {
		t.Fatal("commented @HookPostInit must not count")
	}

	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(empty, "index.ts"), "export const x = 1\n")
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: empty}, PhasePostInit) {
		t.Fatal("empty tree must not declare post_init")
	}

	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: ""}, PhasePreInit) {
		t.Fatal("empty Path must fail open")
	}
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: filepath.Join(root, "missing")}, PhasePreInit) {
		t.Fatal("missing Path must fail open")
	}
	if moduleSourceDeclaresHookPhase(nil, PhasePreInit) {
		t.Fatal("nil module must be false")
	}

	typed := filepath.Join(root, "typed")
	if err := os.MkdirAll(typed, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(typed, "hooks.ts"), `
  @HookPreUpgrade<Options>()
  static async pre(): Promise<void> {}
`)
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: typed}, PhasePreUpgrade) {
		t.Fatal("expected typed @HookPreUpgrade to match")
	}

	nestedGeneric := filepath.Join(root, "nested_generic")
	if err := os.MkdirAll(nestedGeneric, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(nestedGeneric, "hooks.ts"), `
  @HookPreUpgrade<Map<string, number>>()
  static async pre(): Promise<void> {}
`)
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: nestedGeneric}, PhasePreUpgrade) {
		t.Fatal("expected nested generic @HookPreUpgrade to match")
	}

	parenGeneric := filepath.Join(root, "paren_generic")
	if err := os.MkdirAll(parenGeneric, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(parenGeneric, "hooks.ts"), `
  @HookPreInit<() => void>()
  static async pre(): Promise<void> {}
`)
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: parenGeneric}, PhasePreInit) {
		t.Fatal("expected parenthesized generic @HookPreInit to match")
	}

	// Module root basename may itself be an ignore name (e.g. .../demo).
	demoRoot := filepath.Join(root, "demo")
	mustWriteHookScanFile(t, filepath.Join(demoRoot, "service", "hook.ts"), `
  @HookPostInit()
  static async ensure(): Promise<void> {}
`)
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: demoRoot}, PhasePostInit) {
		t.Fatal("module rooted at demo/ must still be scanned")
	}

	// Two files declaring the same phase hit the per-phase "already" continue.
	dupPhase := filepath.Join(root, "dup_phase")
	mustWriteHookScanFile(t, filepath.Join(dupPhase, "a.ts"), "@HookPostUpgrade()\n")
	mustWriteHookScanFile(t, filepath.Join(dupPhase, "b.ts"), "@HookPostUpgrade()\n")
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: dupPhase}, PhasePostUpgrade) {
		t.Fatal("duplicate phase declarations must still match")
	}

	skippedDirs := []string{"node_modules", "dist", "demo", "__tests__", "coverage"}
	for _, dir := range skippedDirs {
		blocked := filepath.Join(root, "skip_"+dir)
		if err := os.MkdirAll(filepath.Join(blocked, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWriteHookScanFile(t, filepath.Join(blocked, dir, "hook.ts"), "@HookPostUninstall()\n")
		mustWriteHookScanFile(t, filepath.Join(blocked, "other.ts"), "export {}\n")
		if moduleSourceDeclaresHookPhase(&meta.Module{Path: blocked}, PhasePostUninstall) {
			t.Fatalf("%s contents must be ignored", dir)
		}
	}

	realMod := filepath.Join(root, "real_mod")
	mustWriteHookScanFile(t, filepath.Join(realMod, "service", "hook.ts"), `
export class Hooks {
  @HookPostInit()
  static async ensure(): Promise<void> {}
}
`)
	linkMod := filepath.Join(root, "link_mod")
	if err := os.Symlink(realMod, linkMod); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: linkMod}, PhasePostInit) {
		t.Fatal("symlinked module root must still detect @HookPostInit")
	}

	nested := filepath.Join(root, "nested_link")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(nested, "other.ts"), "export {}\n")
	if err := os.Symlink(filepath.Join(realMod, "service"), filepath.Join(nested, "service")); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: nested}, PhasePostInit) {
		t.Fatal("nested source symlink must fail open")
	}

	// Symlinked dependency dirs must not force fail-open (e.g. pnpm node_modules).
	depLink := filepath.Join(root, "with_nm_link")
	if err := os.MkdirAll(depLink, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(depLink, "index.ts"), "export {}\n")
	nmTarget := filepath.Join(root, "nm_target")
	if err := os.MkdirAll(nmTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteHookScanFile(t, filepath.Join(nmTarget, "hook.ts"), "@HookPreInit()\n")
	if err := os.Symlink(nmTarget, filepath.Join(depLink, "node_modules")); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: depLink}, PhasePreInit) {
		t.Fatal("symlinked node_modules must be ignored, not fail-open")
	}
}

func TestModuleSourceDeclaresHookPhase_CoverageEdges(t *testing.T) {
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: t.TempDir()}, Phase("unknown")) {
		t.Fatal("unknown phase must fail open")
	}

	fileRoot := filepath.Join(t.TempDir(), "not_a_dir.ts")
	mustWriteHookScanFile(t, fileRoot, "export {}\n")
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: fileRoot}, PhasePreInit) {
		t.Fatal("file Path must fail open")
	}

	t.Run("stat error after eval", func(t *testing.T) {
		dir := t.TempDir()
		old := hookPhaseScanStat
		hookPhaseScanStat = func(string) (os.FileInfo, error) {
			return nil, os.ErrPermission
		}
		t.Cleanup(func() { hookPhaseScanStat = old })
		if !moduleSourceDeclaresHookPhase(&meta.Module{Path: dir}, PhasePreInit) {
			t.Fatal("Stat error must fail open")
		}
	})

	nonSrc := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(nonSrc, "readme.md"), "# no hooks\n")
	mustWriteHookScanFile(t, filepath.Join(nonSrc, "data.json"), "{}\n")
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: nonSrc}, PhasePostInit) {
		t.Fatal("non-source files must not declare hooks")
	}

	emptyFile := t.TempDir()
	mustWriteHookScanFile(t, filepath.Join(emptyFile, "empty.ts"), "")
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: emptyFile}, PhasePostInit) {
		t.Fatal("empty source must not declare hooks")
	}

	noNL := t.TempDir()
	pathNoNL := filepath.Join(noNL, "hook.ts")
	if err := os.MkdirAll(filepath.Dir(pathNoNL), 0o755); err != nil {
		t.Fatal(err)
	}
	// No trailing newline exercises stripLineComments' final segment.
	if err := os.WriteFile(pathNoNL, []byte("  @HookPostInit()"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !moduleSourceDeclaresHookPhase(&meta.Module{Path: noNL}, PhasePostInit) {
		t.Fatal("decorator without trailing newline must match")
	}

	if got := stripLineComments(nil); len(got) != 0 {
		t.Fatalf("nil input: got %q", got)
	}
	if got := stripLineComments([]byte{}); len(got) != 0 {
		t.Fatalf("empty input: got %q", got)
	}

	t.Run("unreadable source file", func(t *testing.T) {
		blocked := t.TempDir()
		blockedFile := filepath.Join(blocked, "service", "secret.ts")
		mustWriteHookScanFile(t, blockedFile, "export {}\n")
		if err := os.Chmod(blockedFile, 0); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(blockedFile, 0o644) })
		if _, err := os.ReadFile(blockedFile); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		if !moduleSourceDeclaresHookPhase(&meta.Module{Path: blocked}, PhasePostInit) {
			t.Fatal("unreadable source should fail open")
		}
	})

	t.Run("unreadable directory", func(t *testing.T) {
		blockedDir := t.TempDir()
		secretDir := filepath.Join(blockedDir, "secret")
		mustWriteHookScanFile(t, filepath.Join(secretDir, "x.ts"), "export {}\n")
		if err := os.Chmod(secretDir, 0); err != nil {
			t.Fatalf("chmod secret dir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(secretDir, 0o755) })
		if _, err := os.ReadDir(secretDir); err == nil {
			t.Skip("filesystem permits readdir despite mode 0")
		}
		if !moduleSourceDeclaresHookPhase(&meta.Module{Path: blockedDir}, PhasePostInit) {
			t.Fatal("unreadable directory should fail open")
		}
	})

	// Hit remaining ignored-dir name and source extensions.
	extRoot := t.TempDir()
	for _, name := range []string{"a.tsx", "b.js", "c.jsx", "d.mts", "e.cts"} {
		mustWriteHookScanFile(t, filepath.Join(extRoot, name), "export {}\n")
	}
	mustWriteHookScanFile(t, filepath.Join(extRoot, ".git", "hooks.ts"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "x.spec.ts"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "y.spec.tsx"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "z.spec.js"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "w.test.js"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "v.test.tsx"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "u.test.jsx"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "t.spec.jsx"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "s.test.mts"), "@HookPreInit()\n")
	mustWriteHookScanFile(t, filepath.Join(extRoot, "r.spec.cts"), "@HookPreInit()\n")
	if moduleSourceDeclaresHookPhase(&meta.Module{Path: extRoot}, PhasePreInit) {
		t.Fatal("ignored dirs/tests and empty sources must not declare pre_init")
	}
}

func TestRunPhase_NoHooks_NoExecutorLoad(t *testing.T) {
	testRuntimeScope := newHooksTestScope(t)
	writeHooksRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
	modRoot := filepath.Join(testRuntimeScope.cfg.ModulesPath, "base")
	mustWriteHookScanFile(t, filepath.Join(modRoot, "service", "index.ts"), "export const hook = {}\n")

	engine := &hooksSelectiveEngine{execute: func(_ context.Context, req *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
		t.Fatalf("unexpected executor execute for empty pre_init: %s", req.Service)
		return nil, nil
	}}
	baseExecutor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
	countingExecutor := &hooksReloadCountingExecutor{inner: baseExecutor}

	runner := &Runner{
		runtimeScope: testRuntimeScope,
		jsExecutor:   countingExecutor,
		module: &meta.Module{
			Name: "base", ApplicationStr: "core", Version: "1.0.0", ServiceEntryPoint: "service/index.ts",
			Path: modRoot,
		},
	}
	if err := runner.RunPhase(context.Background(), PhasePreInit, RunOptions{
		Scripts: []*jsengine.JsScript{{FileName: "provided.js", Content: "console.log(1)"}},
	}); err != nil {
		t.Fatalf("RunPhase() error = %v", err)
	}
	if countingExecutor.reloadCalls != 0 {
		t.Fatalf("expected no executor reload for empty pre_init, got %d", countingExecutor.reloadCalls)
	}
}

func TestRunPhase_DeclaredHook_StillLoads(t *testing.T) {
	testRuntimeScope := newHooksTestScope(t)
	writeHooksRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
	modRoot := filepath.Join(testRuntimeScope.cfg.ModulesPath, "document")
	mustWriteHookScanFile(t, filepath.Join(modRoot, "service", "hook", "post_init.ts"), `
export class DocumentAttachmentHooks {
  @HookPostInit()
  static async ensureAttachmentGcSchedule(): Promise<void> {}
}
`)

	engine := &hooksSelectiveEngine{execute: func(_ context.Context, req *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
		return &jsengine.JsResponse{Id: req.Id, Result: nil}, nil
	}}
	baseExecutor := newHooksTestExecutorWithEngine(t, testRuntimeScope, engine)
	countingExecutor := &hooksReloadCountingExecutor{inner: baseExecutor}

	runner := &Runner{
		runtimeScope: testRuntimeScope,
		jsExecutor:   countingExecutor,
		module: &meta.Module{
			Name: "document", ApplicationStr: "document", Version: "1.0.0", ServiceEntryPoint: "service/index.ts",
			Path: modRoot,
		},
	}
	if err := runner.RunPhase(context.Background(), PhasePostInit, RunOptions{
		Scripts: []*jsengine.JsScript{{FileName: "provided.js", Content: "console.log(1)"}},
	}); err != nil {
		t.Fatalf("RunPhase() error = %v", err)
	}
	if countingExecutor.reloadCalls == 0 {
		t.Fatal("expected executor reload when @HookPostInit is declared")
	}
}

func mustWriteHookScanFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
