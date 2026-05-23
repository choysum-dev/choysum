// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tmpdir

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceKeyShortAndDeterministic(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	key1 := workspaceKey(workspaceRoot)
	key2 := workspaceKey(workspaceRoot)
	if key1 != key2 {
		t.Fatalf("workspaceKey should be deterministic, got %q and %q", key1, key2)
	}
	if len(key1) != 10 {
		t.Fatalf("workspaceKey length = %d, want 10", len(key1))
	}
	if !regexp.MustCompile(`^[0-9a-f]{10}$`).MatchString(key1) {
		t.Fatalf("workspaceKey should be 10-char lowercase hex, got %q", key1)
	}
}

func TestResolveWorkspaceTmpDirRequiresTmpRoot(t *testing.T) {
	if _, err := ResolveWorkspaceTmpDir(t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "tmpRoot is required") {
		t.Fatalf("ResolveWorkspaceTmpDir() error = %v, want tmpRoot required", err)
	}
}

func TestResolveWorkspaceTmpDirUsesCWDWhenEmpty(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	workspaceRoot := t.TempDir()
	if err := os.Chdir(workspaceRoot); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() after chdir error = %v", err)
	}

	tmpRoot := filepath.Join(t.TempDir(), "tmp")
	got, err := ResolveWorkspaceTmpDir("", tmpRoot)
	if err != nil {
		t.Fatalf("ResolveWorkspaceTmpDir(empty) error = %v", err)
	}
	want, err := ResolveWorkspaceTmpDir(cwd, tmpRoot)
	if err != nil {
		t.Fatalf("ResolveWorkspaceTmpDir(cwd) error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("ResolveWorkspaceTmpDir(empty) = %q, want %q", got, want)
	}
}

func TestResolveWorkspaceTmpDirUsesInjectedTmpRoot(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	workspaceHash := workspaceKey(absWorkspaceRoot)

	injectedTmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
	absInjectedTmpRoot, err := filepath.Abs(injectedTmpRoot)
	if err != nil {
		t.Fatalf("filepath.Abs(injectedTmpRoot) error = %v", err)
	}

	want := filepath.Join(absInjectedTmpRoot, "workspaces", workspaceHash)
	got, err := ResolveWorkspaceTmpDir(workspaceRoot, injectedTmpRoot)
	if err != nil {
		t.Fatalf("ResolveWorkspaceTmpDir(injected) error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("ResolveWorkspaceTmpDir(injected) = %q, want %q", got, want)
	}
}

func TestResolveTestingTmpDirRequiresKind(t *testing.T) {
	if _, err := ResolveTestingTmpDir(t.TempDir(), t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "kind is required") {
		t.Fatalf("ResolveTestingTmpDir() error = %v, want kind required", err)
	}
}

func TestResolveTestingTmpDirRequiresTmpRoot(t *testing.T) {
	if _, err := ResolveTestingTmpDir(t.TempDir(), "", "coverage"); err == nil || !strings.Contains(err.Error(), "tmpRoot is required") {
		t.Fatalf("ResolveTestingTmpDir() error = %v, want tmpRoot required", err)
	}
}

func TestResolveTestingTmpDirUsesInjectedTmpRoot(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	workspaceHash := workspaceKey(absWorkspaceRoot)

	injectedTmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
	absInjectedTmpRoot, err := filepath.Abs(injectedTmpRoot)
	if err != nil {
		t.Fatalf("filepath.Abs(injectedTmpRoot) error = %v", err)
	}

	want := filepath.Join(absInjectedTmpRoot, "testing", workspaceHash, "coverage")
	got, err := ResolveTestingTmpDir(workspaceRoot, injectedTmpRoot, "coverage")
	if err != nil {
		t.Fatalf("ResolveTestingTmpDir(injected) error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("ResolveTestingTmpDir(injected) = %q, want %q", got, want)
	}
}

func TestResolveTestingTmpDirWithRunIDUsesWorkspaceFirstLayout(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	workspaceHash := workspaceKey(absWorkspaceRoot)

	injectedTmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
	absInjectedTmpRoot, err := filepath.Abs(injectedTmpRoot)
	if err != nil {
		t.Fatalf("filepath.Abs(injectedTmpRoot) error = %v", err)
	}

	runID := "r123456"
	want := filepath.Join(absInjectedTmpRoot, "testing", workspaceHash, runID, "coverage")
	got, err := ResolveTestingTmpDirWithRunID(workspaceRoot, injectedTmpRoot, "coverage", runID)
	if err != nil {
		t.Fatalf("ResolveTestingTmpDirWithRunID() error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("ResolveTestingTmpDirWithRunID() = %q, want %q", got, want)
	}
}

func TestResolveTestingTmpDirWithRunIDRejectsPathLikeRunID(t *testing.T) {
	if _, err := ResolveTestingTmpDirWithRunID(t.TempDir(), t.TempDir(), "coverage", "r1/r2"); err == nil || !strings.Contains(err.Error(), "single path segment") {
		t.Fatalf("ResolveTestingTmpDirWithRunID() error = %v, want single path segment", err)
	}
}

func TestResolveTestingTmpDirFromContextUsesRunID(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	workspaceHash := workspaceKey(absWorkspaceRoot)

	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
	absTmpRoot, err := filepath.Abs(tmpRoot)
	if err != nil {
		t.Fatalf("filepath.Abs(tmpRoot) error = %v", err)
	}

	ctx := ContextWithTestingRunID(context.Background(), "rabc")
	got, err := ResolveTestingTmpDirFromContext(ctx, workspaceRoot, tmpRoot, "backend")
	if err != nil {
		t.Fatalf("ResolveTestingTmpDirFromContext() error = %v", err)
	}
	want := filepath.Join(absTmpRoot, "testing", workspaceHash, "rabc", "backend")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("ResolveTestingTmpDirFromContext() = %q, want %q", got, want)
	}
}

func TestNewTestingRunIDAndContextHelpers(t *testing.T) {
	runID := NewTestingRunID()
	if strings.TrimSpace(runID) == "" {
		t.Fatal("expected non-empty run id")
	}
	if strings.Contains(runID, "/") || strings.Contains(runID, "\\") {
		t.Fatalf("expected single-segment run id, got %q", runID)
	}
	parts := strings.SplitN(runID, "-r", 2)
	if len(parts) != 2 {
		t.Fatalf("expected run id format '<yyMMdd-HHmmss>-r<hex>', got %q", runID)
	}
	if _, err := time.Parse("060102-150405", parts[0]); err != nil {
		t.Fatalf("expected readable time prefix, got %q (err=%v)", parts[0], err)
	}
	if !regexp.MustCompile(`^[0-9a-f]+$`).MatchString(parts[1]) {
		t.Fatalf("expected hex random suffix, got %q", parts[1])
	}

	ctx := ContextWithTestingRunID(context.Background(), runID)
	if got := TestingRunIDFromContext(ctx); got != runID {
		t.Fatalf("TestingRunIDFromContext() = %q, want %q", got, runID)
	}
	if got := TestingRunIDFromContext(context.Background()); got != "" {
		t.Fatalf("TestingRunIDFromContext(background) = %q, want empty", got)
	}
}
