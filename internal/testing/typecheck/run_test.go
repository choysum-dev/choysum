// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	t.Run("requires modules path", func(t *testing.T) {
		err := Run(context.Background(), RunOptions{})
		if err == nil || !strings.Contains(err.Error(), "modules_path is required") {
			t.Fatalf("expected modules path error, got %v", err)
		}
	})

	t.Run("prints no tests found when resolver finds none", func(t *testing.T) {
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "empty"))
		var stdout strings.Builder

		err := Run(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			Target:      "all",
			Stdout:      &stdout,
		})
		if err != nil {
			t.Fatalf("expected no-tests-found success, got %v", err)
		}
		if stdout.String() != "no tests found\n" {
			t.Fatalf("unexpected stdout: %q", stdout.String())
		}
	})

	t.Run("prints no tests found when target has no ts inputs", func(t *testing.T) {
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		var stdout strings.Builder

		err := Run(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			Target:      "auth",
			Stdout:      &stdout,
		})
		if err != nil {
			t.Fatalf("expected no-tests-found success, got %v", err)
		}
		if stdout.String() != "no tests found\n" {
			t.Fatalf("unexpected stdout: %q", stdout.String())
		}
	})

	t.Run("defaults target to all and processes apps in sorted order", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "zeta", "web"))
		writeFile(t, filepath.Join(modulesPath, "zeta", "web", "index.ts"), "export const z = 1\n")
		makeDir(t, filepath.Join(modulesPath, "alpha", "service"))
		writeFile(t, filepath.Join(modulesPath, "alpha", "service", "index.ts"), "export const a = 1\n")
		makeDir(t, filepath.Join(modulesPath, ".choysum", "service"))
		makeDir(t, filepath.Join(modulesPath, "tmp", "web"))

		var stderr strings.Builder
		err := Run(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			Stderr:      &stderr,
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}

		got := stderr.String()
		alphaStart := strings.Index(got, "# typecheck alpha")
		zetaStart := strings.Index(got, "# typecheck zeta")
		if alphaStart == -1 || zetaStart == -1 || alphaStart > zetaStart {
			t.Fatalf("expected sorted typecheck order, got %q", got)
		}
	})

	t.Run("uses current working directory as repo root when omitted", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := filepath.Join(repoRoot, "modules")
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd returned error: %v", err)
		}
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatalf("Chdir(%q): %v", repoRoot, err)
		}
		defer func() {
			_ = os.Chdir(originalWD)
		}()

		err = Run(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			Target:      "auth",
			Stderr:      &strings.Builder{},
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	t.Run("propagates target resolution errors", func(t *testing.T) {
		modulesPath := t.TempDir()
		err := Run(context.Background(), RunOptions{ModulesPath: modulesPath, Target: "missing"})
		if err == nil || !strings.Contains(err.Error(), "unknown app") {
			t.Fatalf("expected unknown app error, got %v", err)
		}
	})

	t.Run("propagates typecheck diagnostics as failure", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "bad.ts"), "const x: number = 'nope'\n")

		var stderr strings.Builder
		err := Run(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			Target:      "auth",
			Stderr:      &stderr,
		})
		if err == nil || !strings.Contains(err.Error(), "typecheck failed for auth") {
			t.Fatalf("expected typecheck failure, got %v", err)
		}
		if !strings.Contains(stderr.String(), "TS") {
			t.Fatalf("expected TS diagnostics on stderr, got %q", stderr.String())
		}
	})

	t.Run("propagates context cancellation from app typecheck", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := Run(ctx, RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			Target:      "auth",
		})
		if err == nil || err != context.Canceled {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})
}
