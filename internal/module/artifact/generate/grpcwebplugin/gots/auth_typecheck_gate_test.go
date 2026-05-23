// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gots

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tctypecheck "github.com/choysum-dev/choysum/internal/testing/typecheck"
)

func TestAuthTypecheckGate(t *testing.T) {
	if os.Getenv("CHOYSUM_ENABLE_AUTH_TYPECHECK_GATE") != "1" {
		t.Skip("auth typecheck gate disabled; set CHOYSUM_ENABLE_AUTH_TYPECHECK_GATE=1 to enable")
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	addonsPath, err := resolveAddonsPath(repoRoot)
	if err != nil {
		t.Fatalf("resolve addons path: %v", err)
	}

	opts := tctypecheck.RunOptions{
		AddonsPath: addonsPath,
		RepoRoot:   repoRoot,
		Target:     "auth",
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}
	if err := tctypecheck.Run(context.Background(), opts); err != nil {
		t.Fatalf("auth typecheck gate failed: %v", err)
	}
}

func findRepoRoot() (string, error) {
	if v := os.Getenv("CHOYSUM_REPO_ROOT"); strings.TrimSpace(v) != "" {
		repoRoot := v
		if !filepath.IsAbs(repoRoot) {
			abs, err := filepath.Abs(repoRoot)
			if err != nil {
				return "", err
			}
			repoRoot = abs
		}
		repoRoot = normalizePath(repoRoot)
		if err := validateRepoRoot(repoRoot); err != nil {
			return "", fmt.Errorf("validate CHOYSUM_REPO_ROOT %q: %w", repoRoot, err)
		}
		return repoRoot, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return normalizePath(cur), nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			return "", os.ErrNotExist
		}
		cur = next
	}
}

func validateRepoRoot(repoRoot string) error {
	if err := validateDirectoryPath("repo root", repoRoot); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		return fmt.Errorf("missing go.mod under repo root %q: %w", repoRoot, err)
	}
	return nil
}

func resolveAddonsPath(repoRoot string) (string, error) {
	if v := os.Getenv("CHOYSUM_ADDONS_PATH"); strings.TrimSpace(v) != "" {
		addonsPath := v
		if !filepath.IsAbs(addonsPath) {
			addonsPath = filepath.Join(repoRoot, addonsPath)
		}
		addonsPath = normalizePath(addonsPath)
		if err := validateDirectoryPath("addons path", addonsPath); err != nil {
			return "", err
		}
		return addonsPath, nil
	}

	addonsPath := filepath.Join(repoRoot, "addons")
	addonsPath = normalizePath(addonsPath)
	if err := validateDirectoryPath("addons path", addonsPath); err != nil {
		return "", err
	}
	return addonsPath, nil
}

func validateDirectoryPath(label, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s %q: %w", label, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", label, path)
	}
	return nil
}

func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return cleaned
	}
	return filepath.Clean(resolved)
}

func canonicalPathForTest(path string) string {
	cleaned := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return cleaned
	}
	return filepath.Clean(resolved)
}

func TestFindRepoRoot_FromEnvOverride(t *testing.T) {
	t.Run("relative repo root override", func(t *testing.T) {
		base := t.TempDir()
		repoRoot := filepath.Join(base, "repo")
		if err := os.MkdirAll(repoRoot, 0o755); err != nil {
			t.Fatalf("mkdir repoRoot: %v", err)
		}
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}

		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(oldWD) })
		if err := os.Chdir(base); err != nil {
			t.Fatalf("chdir base: %v", err)
		}

		t.Setenv("CHOYSUM_REPO_ROOT", "repo")
		got, err := findRepoRoot()
		if err != nil {
			t.Fatalf("findRepoRoot failed: %v", err)
		}
		if canonicalPathForTest(got) != canonicalPathForTest(repoRoot) {
			t.Fatalf("unexpected repo root: got=%q want=%q", got, repoRoot)
		}
	})

	t.Run("invalid repo root override", func(t *testing.T) {
		t.Setenv("CHOYSUM_REPO_ROOT", t.TempDir())
		_, err := findRepoRoot()
		if err == nil {
			t.Fatalf("expected error for invalid repo root override")
		}
		if !strings.Contains(err.Error(), "missing go.mod") {
			t.Fatalf("expected missing go.mod context, got: %v", err)
		}
	})
}

func TestValidateRepoRoot(t *testing.T) {
	t.Run("ok with go.mod", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := validateRepoRoot(dir); err != nil {
			t.Fatalf("validateRepoRoot failed: %v", err)
		}
	})

	t.Run("error without go.mod", func(t *testing.T) {
		dir := t.TempDir()
		if err := validateRepoRoot(dir); err == nil {
			t.Fatalf("expected error for missing go.mod")
		}
	})

	t.Run("error when repo root is a file", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "repo.txt")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		err := validateRepoRoot(file)
		if err == nil {
			t.Fatalf("expected error when repo root is a file")
		}
		if !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("expected not-a-directory context, got: %v", err)
		}
	})
}

func TestResolveAddonsPath(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	defaultAddons := filepath.Join(repoRoot, "addons")
	if err := os.MkdirAll(defaultAddons, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}

	t.Run("default addons path", func(t *testing.T) {
		t.Setenv("CHOYSUM_ADDONS_PATH", "")
		got, err := resolveAddonsPath(repoRoot)
		if err != nil {
			t.Fatalf("resolveAddonsPath failed: %v", err)
		}
		if canonicalPathForTest(got) != canonicalPathForTest(defaultAddons) {
			t.Fatalf("unexpected addons path: got=%q want=%q", got, defaultAddons)
		}
	})

	t.Run("absolute addons override", func(t *testing.T) {
		override := t.TempDir()
		t.Setenv("CHOYSUM_ADDONS_PATH", override)
		got, err := resolveAddonsPath(repoRoot)
		if err != nil {
			t.Fatalf("resolveAddonsPath failed: %v", err)
		}
		if canonicalPathForTest(got) != canonicalPathForTest(override) {
			t.Fatalf("unexpected addons path: got=%q want=%q", got, override)
		}
	})

	t.Run("relative addons override resolved from repoRoot", func(t *testing.T) {
		rel := filepath.Join("custom", "addons")
		want := filepath.Join(repoRoot, rel)
		if err := os.MkdirAll(want, 0o755); err != nil {
			t.Fatalf("mkdir custom addons: %v", err)
		}
		t.Setenv("CHOYSUM_ADDONS_PATH", rel)
		got, err := resolveAddonsPath(repoRoot)
		if err != nil {
			t.Fatalf("resolveAddonsPath failed: %v", err)
		}
		if canonicalPathForTest(got) != canonicalPathForTest(want) {
			t.Fatalf("unexpected addons path: got=%q want=%q", got, want)
		}
	})

	t.Run("override path missing", func(t *testing.T) {
		t.Setenv("CHOYSUM_ADDONS_PATH", filepath.Join(repoRoot, "not-exist"))
		if _, err := resolveAddonsPath(repoRoot); err == nil {
			t.Fatalf("expected error for missing addons path")
		}
	})

	t.Run("override path is file", func(t *testing.T) {
		file := filepath.Join(repoRoot, "addons-file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		t.Setenv("CHOYSUM_ADDONS_PATH", file)
		_, err := resolveAddonsPath(repoRoot)
		if err == nil {
			t.Fatalf("expected error for addons file path")
		}
		if !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("expected not-a-directory context, got: %v", err)
		}
	})
}
