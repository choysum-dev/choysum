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
	t.Run("requires addons path", func(t *testing.T) {
		err := Run(context.Background(), RunOptions{})
		if err == nil || !strings.Contains(err.Error(), "addons_path is required") {
			t.Fatalf("expected addons path error, got %v", err)
		}
	})

	t.Run("returns no apps to check when resolver finds none", func(t *testing.T) {
		addonsPath := t.TempDir()
		makeDir(t, filepath.Join(addonsPath, "empty"))

		err := Run(context.Background(), RunOptions{
			AddonsPath: addonsPath,
			Target:     "all",
		})
		if err == nil || !strings.Contains(err.Error(), "no apps to check") {
			t.Fatalf("expected no apps to check error, got %v", err)
		}
	})

	t.Run("defaults target to all and processes apps in sorted order", func(t *testing.T) {
		repoRoot := t.TempDir()
		addonsPath := t.TempDir()
		makeDir(t, filepath.Join(addonsPath, "zeta", "web"))
		makeDir(t, filepath.Join(addonsPath, "alpha", "service"))
		makeDir(t, filepath.Join(addonsPath, ".choysum", "service"))
		makeDir(t, filepath.Join(addonsPath, "tmp", "web"))
		makeDir(t, filepath.Join(repoRoot, "node_modules", "vite"))
		writeFile(t, filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts"), "declare interface ImportMetaEnv {}\n")

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")

		var stderr strings.Builder
		err := Run(context.Background(), RunOptions{
			AddonsPath: addonsPath,
			NpmPath:    npmPath,
			RepoRoot:   repoRoot,
			Stderr:     &stderr,
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
		addonsPath := filepath.Join(repoRoot, "addons")
		makeDir(t, filepath.Join(addonsPath, "auth", "service"))
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")

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
			AddonsPath: addonsPath,
			NpmPath:    npmPath,
			Target:     "auth",
			Stderr:     &strings.Builder{},
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	t.Run("propagates target resolution errors", func(t *testing.T) {
		addonsPath := t.TempDir()
		err := Run(context.Background(), RunOptions{AddonsPath: addonsPath, Target: "missing"})
		if err == nil || !strings.Contains(err.Error(), "unknown app") {
			t.Fatalf("expected unknown app error, got %v", err)
		}
	})

	t.Run("propagates typecheck app errors", func(t *testing.T) {
		repoRoot := t.TempDir()
		addonsPath := t.TempDir()
		makeDir(t, filepath.Join(addonsPath, "auth", "service"))
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "printf 'compile failed'; exit 7\n")

		var stderr strings.Builder
		err := Run(context.Background(), RunOptions{
			AddonsPath: addonsPath,
			NpmPath:    npmPath,
			RepoRoot:   repoRoot,
			Target:     "auth",
			Stderr:     &stderr,
		})
		if err == nil || !strings.Contains(err.Error(), "typecheck failed for auth") {
			t.Fatalf("expected typecheck failure, got %v", err)
		}
		if !strings.Contains(stderr.String(), "compile failed\n") {
			t.Fatalf("expected command output to be forwarded to stderr, got %q", stderr.String())
		}
	})

	t.Run("propagates context cancellation from app typecheck", func(t *testing.T) {
		repoRoot := t.TempDir()
		addonsPath := t.TempDir()
		makeDir(t, filepath.Join(addonsPath, "auth", "service"))
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := Run(ctx, RunOptions{
			AddonsPath: addonsPath,
			NpmPath:    npmPath,
			RepoRoot:   repoRoot,
			Target:     "auth",
		})
		if err == nil || err != context.Canceled {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})
}

func makeFakeTypecheckTooling(t *testing.T, repoRoot string, scriptBody string) (string, string, string) {
	t.Helper()

	binDir := filepath.Join(t.TempDir(), "bin")
	makeDir(t, binDir)

	npmPath := filepath.Join(binDir, "npm")
	npxPath := filepath.Join(binDir, "npx")
	writeFile(t, npmPath, "#!/bin/sh\nexit 0\n")
	writeFile(t, npxPath, "#!/bin/sh\nset -eu\nif [ -n \"${CHOYSUM_CAPTURE_TSCONFIG_PATH:-}\" ]; then\n  printf '%s\\n' \"$4\" > \"$CHOYSUM_CAPTURE_TSCONFIG_PATH\"\nfi\nif [ -n \"${CHOYSUM_COPY_TSCONFIG:-}\" ]; then\n  cp \"$4\" \"$CHOYSUM_COPY_TSCONFIG\"\nfi\n"+scriptBody)

	makeDir(t, filepath.Join(repoRoot, "node_modules", ".bin"))
	makeDir(t, filepath.Join(repoRoot, "node_modules", "vue-tsc"))
	writeFile(t, filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json"), "{}\n")

	copyPath := filepath.Join(t.TempDir(), "captured.tsconfig.json")
	argsPath := filepath.Join(t.TempDir(), "npx.args.txt")
	return npmPath, copyPath, argsPath
}
