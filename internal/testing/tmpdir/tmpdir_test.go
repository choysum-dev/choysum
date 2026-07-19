// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tmpdir

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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

func TestCLITestTmpRootDefaultsUnderOSTemp(t *testing.T) {
	t.Setenv(EnvCLITestTMP, "")
	got, err := CLITestTmpRoot()
	if err != nil {
		t.Fatalf("CLITestTmpRoot() error = %v", err)
	}
	want := filepath.Join(os.TempDir(), defaultCLITestTmpDirName)
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("CLITestTmpRoot() = %q, want %q", got, want)
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Fatalf("expected CLI test tmp root dir, stat err=%v", err)
	}
}

func TestCLITestTmpRootHonorsEnvOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-cli-test-tmp")
	t.Setenv(EnvCLITestTMP, override)
	got, err := CLITestTmpRoot()
	if err != nil {
		t.Fatalf("CLITestTmpRoot() error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(override) {
		t.Fatalf("CLITestTmpRoot() = %q, want %q", got, override)
	}
}

func TestResolveCLITestingRunHomeUsesRunID(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	tmpRoot := filepath.Join(t.TempDir(), "cli-tmp")
	ctx := ContextWithTestingRunID(context.Background(), "runhome1")
	got, err := ResolveCLITestingRunHome(ctx, workspaceRoot, tmpRoot)
	if err != nil {
		t.Fatalf("ResolveCLITestingRunHome() error = %v", err)
	}
	want, err := ResolveTestingTmpDirFromContext(ctx, workspaceRoot, tmpRoot, CLITestingRunHomeKind)
	if err != nil {
		t.Fatalf("ResolveTestingTmpDirFromContext(home) error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("ResolveCLITestingRunHome() = %q, want %q", got, want)
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Fatalf("expected run home dir, stat err=%v", err)
	}
	assertPkgLinksToCache(t, got, tmpRoot)
}

func TestBindCLITestRuntimePathsSetsContextOverrides(t *testing.T) {
	cliTmp := filepath.Join(t.TempDir(), "bound-cli-tmp")
	t.Setenv(EnvCLITestTMP, cliTmp)
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	ctx, testTmp, runHome, err := BindCLITestRuntimePaths(context.Background(), workspaceRoot)
	if err != nil {
		t.Fatalf("BindCLITestRuntimePaths() error = %v", err)
	}
	if filepath.Clean(testTmp) != filepath.Clean(cliTmp) {
		t.Fatalf("testTmp = %q, want %q", testTmp, cliTmp)
	}
	if got := CLITestTmpRootFromContext(ctx); filepath.Clean(got) != filepath.Clean(cliTmp) {
		t.Fatalf("CLITestTmpRootFromContext() = %q, want %q", got, cliTmp)
	}
	if got := EffectiveCLITestRunHome(ctx); filepath.Clean(got) != filepath.Clean(runHome) {
		t.Fatalf("EffectiveCLITestRunHome() = %q, want %q", got, runHome)
	}
	if TestingRunIDFromContext(ctx) == "" {
		t.Fatal("expected testing run-id on context")
	}
	if filepath.Base(runHome) != CLITestingRunHomeKind {
		t.Fatalf("runHome base = %q, want %q", filepath.Base(runHome), CLITestingRunHomeKind)
	}
	if !strings.HasPrefix(filepath.Clean(runHome), filepath.Clean(cliTmp)+string(filepath.Separator)) {
		t.Fatalf("runHome = %q, want under %q", runHome, cliTmp)
	}
	assertPkgLinksToCache(t, runHome, cliTmp)
}

func TestCLITestingPkgCachePersistsAcrossRunHomes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	workspaceRoot := filepath.Join(t.TempDir(), "repo")
	tmpRoot := filepath.Join(t.TempDir(), "cli-tmp")

	home1, err := ResolveCLITestingRunHome(ContextWithTestingRunID(context.Background(), "run-a"), workspaceRoot, tmpRoot)
	if err != nil {
		t.Fatalf("ResolveCLITestingRunHome(run-a) error = %v", err)
	}
	home2, err := ResolveCLITestingRunHome(ContextWithTestingRunID(context.Background(), "run-b"), workspaceRoot, tmpRoot)
	if err != nil {
		t.Fatalf("ResolveCLITestingRunHome(run-b) error = %v", err)
	}
	if filepath.Clean(home1) == filepath.Clean(home2) {
		t.Fatalf("expected distinct run homes, both %q", home1)
	}

	pkgCache, err := ResolveCLITestingPkgCache(tmpRoot)
	if err != nil {
		t.Fatalf("ResolveCLITestingPkgCache() error = %v", err)
	}
	marker := filepath.Join(pkgCache, "esm", "warm-marker.txt")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatalf("mkdir esm cache: %v", err)
	}
	if err := os.WriteFile(marker, []byte("warm"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	viaHome1 := filepath.Join(home1, "pkg", "esm", "warm-marker.txt")
	viaHome2 := filepath.Join(home2, "pkg", "esm", "warm-marker.txt")
	raw1, err := os.ReadFile(viaHome1)
	if err != nil {
		t.Fatalf("read via home1: %v", err)
	}
	raw2, err := os.ReadFile(viaHome2)
	if err != nil {
		t.Fatalf("read via home2: %v", err)
	}
	if string(raw1) != "warm" || string(raw2) != "warm" {
		t.Fatalf("expected shared cache content, got home1=%q home2=%q", raw1, raw2)
	}
}

func TestEnsureCLITestingPkgLinkMigratesExistingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	home := filepath.Join(t.TempDir(), "home")
	pkgCache := filepath.Join(t.TempDir(), "cache", "pkg")
	legacy := filepath.Join(home, "pkg", "esm", "legacy.txt")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("mkdir legacy pkg: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	if err := EnsureCLITestingPkgLink(home, pkgCache); err != nil {
		t.Fatalf("EnsureCLITestingPkgLink() error = %v", err)
	}
	st, err := os.Lstat(filepath.Join(home, "pkg"))
	if err != nil {
		t.Fatalf("lstat home/pkg: %v", err)
	}
	if st.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected home/pkg to be a symlink after migration")
	}
	migrated := filepath.Join(pkgCache, "esm", "legacy.txt")
	raw, err := os.ReadFile(migrated)
	if err != nil {
		t.Fatalf("read migrated file: %v", err)
	}
	if string(raw) != "keep" {
		t.Fatalf("migrated content = %q, want keep", raw)
	}
}

func assertPkgLinksToCache(t *testing.T, choysumHome, tmpRoot string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	pkgCache, err := ResolveCLITestingPkgCache(tmpRoot)
	if err != nil {
		t.Fatalf("ResolveCLITestingPkgCache() error = %v", err)
	}
	linkPath := filepath.Join(choysumHome, "pkg")
	st, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat %s: %v", linkPath, err)
	}
	if st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", linkPath)
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink %s: %v", linkPath, err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}
	if filepath.Clean(target) != filepath.Clean(pkgCache) {
		t.Fatalf("pkg link target = %q, want %q", target, pkgCache)
	}
}
