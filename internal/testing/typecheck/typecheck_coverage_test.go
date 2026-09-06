// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	gonative "github.com/choysum-dev/choysum/internal/typecheck"
)

func TestRun_NilContext(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "empty"))
	if err := Run(nil, RunOptions{ModulesPath: modulesPath, Stdout: io.Discard}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveApps_EmptyTarget(t *testing.T) {
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	apps, err := ResolveApps(modulesPath, "  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0] != "auth" {
		t.Fatalf("apps = %v", apps)
	}
}

func TestTypecheckApp_NilContext(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")
	if err := TypecheckApp(nil, RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    t.TempDir(),
		Stderr:      io.Discard,
	}, "auth"); err != nil {
		t.Fatal(err)
	}
}

func TestTypecheckApp_RelativePaths(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repoRoot := t.TempDir()
	modulesPath := filepath.Join(repoRoot, "modules")
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: "modules",
		RepoRoot:    ".",
		TmpPath:     "rel-tmp",
		Stderr:      io.Discard,
	}, "auth"); err != nil {
		t.Fatal(err)
	}
}

func TestTypecheckApp_EnsureTypeAssetsFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	t.Setenv("CHOYSUM_TEST_TMP", "")
	orig := preferTypesWriteDir
	t.Cleanup(func() { preferTypesWriteDir = orig })
	preferTypesWriteDir = func() string { return "" }

	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")
	writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3.5.0/index.d.ts"]
    }
  }
}
`)
	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    t.TempDir(),
		TmpPath:     t.TempDir(),
	}, "auth")
	if err == nil || !strings.Contains(err.Error(), "cannot resolve type-fetch write dir") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_KeepDirMkdirFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")
	tmpPath := t.TempDir()

	ctx := testingpathing.ContextWithTestingRunID(context.Background(), testingpathing.NewTestingRunID())
	wantRoot, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, repoRoot, tmpPath, "typecheck")
	if err != nil {
		t.Fatal(err)
	}
	makeDir(t, wantRoot)
	writeFile(t, filepath.Join(wantRoot, sanitizeAppToken("auth")), "blocks mkdir\n")

	err = TypecheckApp(ctx, RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		TmpPath:     tmpPath,
		Keep:        true,
	}, "auth")
	if err == nil || !strings.Contains(err.Error(), "ensure keep dir") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_CheckReturnsError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")
	writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), "{ not json")

	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		TmpPath:     t.TempDir(),
	}, "auth")
	if err == nil || !strings.Contains(err.Error(), "typecheck failed for auth") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_DiagnosticsWriteFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "bad.ts"), "const x: number = 'nope'\n")
	tmpPath := t.TempDir()

	ctx := testingpathing.ContextWithTestingRunID(context.Background(), testingpathing.NewTestingRunID())
	wantRoot, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, repoRoot, tmpPath, "typecheck")
	if err != nil {
		t.Fatal(err)
	}
	keepDir := filepath.Join(wantRoot, sanitizeAppToken("auth"))
	makeDir(t, keepDir)
	if err := os.Chmod(keepDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(keepDir, 0o755) })

	var stderr strings.Builder
	err = TypecheckApp(ctx, RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		TmpPath:     tmpPath,
		Keep:        true,
		Stderr:      &stderr,
	}, "auth")
	if err == nil {
		t.Fatal("expected typecheck failure")
	}
	diagPath := filepath.Join(keepDir, "diagnostics.txt")
	if !strings.Contains(stderr.String(), "write "+diagPath) {
		t.Fatalf("expected diagnostics write warning, stderr=%q", stderr.String())
	}
}

func TestTypecheckApp_CheckInternalError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := TypecheckApp(ctx, RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		TmpPath:     t.TempDir(),
	}, "auth")
	if err == nil || err != context.Canceled {
		t.Fatalf("err = %v", err)
	}
}

func TestHasTypecheckInputs_WalkCallbackError(t *testing.T) {
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "demo", "service"))
	writeFile(t, filepath.Join(modulesPath, "demo", "service", "a.ts"), "export {}\n")

	origWalk := walkTypecheckInputsDir
	t.Cleanup(func() { walkTypecheckInputsDir = origWalk })
	walkTypecheckInputsDir = func(root string, fn fs.WalkDirFunc) error {
		return fn(filepath.Join(root, "service", "a.ts"), nil, errors.New("entry boom"))
	}
	has, err := hasTypecheckInputs(modulesPath, "demo")
	if err == nil || !strings.Contains(err.Error(), "entry boom") || has {
		t.Fatalf("has=%v err=%v", has, err)
	}
}

func TestTypecheckApp_ScanInputsError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	origWalk := walkTypecheckInputsDir
	t.Cleanup(func() { walkTypecheckInputsDir = origWalk })
	walkTypecheckInputsDir = func(string, fs.WalkDirFunc) error {
		return errors.New("scan boom")
	}
	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    t.TempDir(),
		TmpPath:     t.TempDir(),
	}, "auth")
	if err == nil || !strings.Contains(err.Error(), "scan inputs for auth") {
		t.Fatalf("err = %v", err)
	}
}

func TestHasTypecheckInputs_NonDirAndWalkError(t *testing.T) {
	modulesPath := t.TempDir()
	writeFile(t, filepath.Join(modulesPath, "auth"), "not a dir\n")
	has, err := hasTypecheckInputs(modulesPath, "auth")
	if err != nil || has {
		t.Fatalf("has=%v err=%v", has, err)
	}

	makeDir(t, filepath.Join(modulesPath, "demo", "service"))
	writeFile(t, filepath.Join(modulesPath, "demo", "service", "a.ts"), "export {}\n")
	origWalk := walkTypecheckInputsDir
	t.Cleanup(func() { walkTypecheckInputsDir = origWalk })
	walkTypecheckInputsDir = func(string, fs.WalkDirFunc) error {
		return errors.New("walk boom")
	}
	has, err = hasTypecheckInputs(modulesPath, "demo")
	if err == nil || !strings.Contains(err.Error(), "walk boom") || has {
		t.Fatalf("has=%v err=%v", has, err)
	}
}

func TestRun_EmptyRepoRootFromGetwd(t *testing.T) {
	orig := osGetwd
	t.Cleanup(func() { osGetwd = orig })
	osGetwd = func() (string, error) { return "", nil }
	err := Run(context.Background(), RunOptions{ModulesPath: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "cannot determine repo root") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_EmptyRepoRootAndTmp(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	origGetwd := osGetwd
	t.Cleanup(func() { osGetwd = origGetwd })
	osGetwd = func() (string, error) { return "", nil }
	err := TypecheckApp(context.Background(), RunOptions{ModulesPath: modulesPath}, "auth")
	if err == nil || !strings.Contains(err.Error(), "cannot determine repo root") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_KeepResolveTmpError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	orig := resolveTestingTmpDir
	t.Cleanup(func() { resolveTestingTmpDir = orig })
	resolveTestingTmpDir = func(context.Context, string, string, string) (string, error) {
		return "", errors.New("tmp resolve boom")
	}
	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    t.TempDir(),
		TmpPath:     t.TempDir(),
		Keep:        true,
		Stderr:      io.Discard,
	}, "auth")
	if err == nil || !strings.Contains(err.Error(), "resolve tmp dir") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_NativeCheckError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	orig := nativeCheck
	t.Cleanup(func() { nativeCheck = orig })
	nativeCheck = func(context.Context, gonative.Options) (gonative.Result, error) {
		return gonative.Result{}, errors.New("native boom")
	}
	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    t.TempDir(),
		TmpPath:     t.TempDir(),
		Stderr:      io.Discard,
	}, "auth")
	if err == nil || !strings.Contains(err.Error(), "native boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestTypecheckApp_NativeCheckNoRootFiles(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const x = 1\n")

	orig := nativeCheck
	t.Cleanup(func() { nativeCheck = orig })
	nativeCheck = func(context.Context, gonative.Options) (gonative.Result, error) {
		return gonative.Result{}, gonative.ErrNoRootFiles
	}
	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    t.TempDir(),
		TmpPath:     t.TempDir(),
		Stderr:      io.Discard,
	}, "auth")
	if !errors.Is(err, errNoTypecheckInputs) {
		t.Fatalf("err = %v", err)
	}
}
