// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
)

func newTestUnitFEIllegalCmdFromScope(scopeGetter func() scope.Scope) *cobra.Command {
	return newTestUnitFEIllegalCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
}

func TestNewTestUnitFEIllegalCmd_ArgsAndScan(t *testing.T) {
	cmd := newTestUnitFEIllegalCmdFromScope(func() scope.Scope { return nil })
	if cmd.Use != "unit-fe-illegal [app]" {
		t.Fatalf("use = %q", cmd.Use)
	}
	if err := cmd.Args(cmd, nil); err == nil || !strings.Contains(err.Error(), "requires exactly 1 app argument") {
		t.Fatalf("args err = %v", err)
	}
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("--all no args: %v", err)
	}
	if err := cmd.Args(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
		t.Fatalf("conflict err = %v", err)
	}

	if err := cmd.RunE(cmd, []string{}); err == nil || !strings.Contains(err.Error(), "invalid runtime options") {
		t.Fatalf("nil scope = %v", err)
	}

	cmd = newTestUnitFEIllegalCmdFromScope(func() scope.Scope { return &commandTestScope{} })
	_ = cmd.Flags().Set("all", "false")
	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "invalid runtime options") {
		t.Fatalf("empty opts = %v", err)
	}

	if err := cmd.Args(cmd, []string{"auth"}); err != nil {
		t.Fatalf("single app args: %v", err)
	}

	repo := t.TempDir()
	modulesPath := filepath.Join(repo, "modules")
	web := filepath.Join(modulesPath, "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "bad.test.ts"), []byte("import { mount } from '@vue/test-utils'\nmount(x)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(modulesPath, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, ".hidden"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := newCommandTestConfig(modulesPath)
	scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }

	var stderr bytes.Buffer
	cmd = newTestUnitFEIllegalCmdFromScope(scopeGetter)
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, []string{"demo"}); err != nil {
		t.Fatalf("scan demo: %v", err)
	}
	if !strings.Contains(stderr.String(), "illegal FE unit mark") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stderr.Reset()
	cmd = newTestUnitFEIllegalCmdFromScope(scopeGetter)
	cmd.SetErr(&stderr)
	if err := cmd.Flags().Set("github-annotations", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"demo"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "::warning file=") {
		t.Fatalf("annotations = %q", stderr.String())
	}

	cmd = newTestUnitFEIllegalCmdFromScope(scopeGetter)
	if err := cmd.Flags().Set("fail", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"demo"}); err == nil || !strings.Contains(err.Error(), "illegal mark") {
		t.Fatalf("fail mode = %v", err)
	}

	stderr.Reset()
	cmd = newTestUnitFEIllegalCmdFromScope(scopeGetter)
	cmd.SetErr(&stderr)
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("--all: %v", err)
	}
	if !strings.Contains(stderr.String(), "illegal FE unit mark") {
		t.Fatalf("--all stderr = %q", stderr.String())
	}

	cleanRepo := t.TempDir()
	cleanModules := filepath.Join(cleanRepo, "modules")
	cleanWeb := filepath.Join(cleanModules, "pure", "web")
	if err := os.MkdirAll(cleanWeb, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cleanWeb, "ok.test.ts"), []byte("import { it } from 'vitest'\nit('ok', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	cmd = newTestUnitFEIllegalCmdFromScope(func() scope.Scope {
		return &commandTestScope{cfg: newCommandTestConfig(cleanModules)}
	})
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, []string{"pure"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "no illegal FE marks") {
		t.Fatalf("clean stderr = %q", stderr.String())
	}

	blocked := filepath.Join(web, "blocked.test.ts")
	if err := os.WriteFile(blocked, []byte("mount(x)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(blocked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(blocked, 0o644) })
		cmd = newTestUnitFEIllegalCmdFromScope(scopeGetter)
		if err := cmd.RunE(cmd, []string{"demo"}); err == nil || !strings.Contains(err.Error(), "frontend illegal scan") {
			t.Fatalf("unreadable scan = %v", err)
		}
	}
}

func TestNewTestUnitFEIllegalCmd_CwdFallbackAndReadError(t *testing.T) {
	repo := t.TempDir()
	modulesPath := filepath.Join(repo, "modules")
	web := filepath.Join(modulesPath, "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "ok.test.ts"), []byte("import { it } from 'vitest'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	cfg := newCommandTestConfig(filepath.Join(t.TempDir(), "not-modules"))
	cmd := newTestUnitFEIllegalCmdFromScope(func() scope.Scope {
		return &commandTestScope{cfg: cfg}
	})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("cwd fallback --all: %v stderr=%s", err, stderr.String())
	}

	broken := newCommandTestConfig(filepath.Join(t.TempDir(), "missing-modules-root"))
	cmd = newTestUnitFEIllegalCmdFromScope(func() scope.Scope {
		return &commandTestScope{cfg: broken}
	})
	_ = cmd.Flags().Set("all", "true")
	// Point cwd somewhere without modules so fallback fails and ReadDir errors.
	empty := t.TempDir()
	t.Chdir(empty)
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "read modules") {
		t.Fatalf("expected read modules error, got %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	discRepo := t.TempDir()
	discModules := filepath.Join(discRepo, "modules")
	locked := filepath.Join(discModules, "bad", "web", "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "a.test.ts"), []byte("test('x', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	cmd = newTestUnitFEIllegalCmdFromScope(func() scope.Scope {
		return &commandTestScope{cfg: newCommandTestConfig(discModules)}
	})
	_ = cmd.Flags().Set("all", "true")
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "frontend discover") {
		t.Fatalf("discover --all err = %v", err)
	}
}
