// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package staging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithOpIDAndOpIDFromContext_EdgeCases(t *testing.T) {
	if got := WithOpID(context.TODO(), ""); got == nil {
		t.Fatalf("expected WithOpID to return a background context for empty opid")
	}
	if got := WithOpID(context.Background(), "."); got == nil {
		t.Fatalf("expected sanitized dot opid to still return a context")
	}
	if _, ok := OpIDFromContext(context.Background()); ok {
		t.Fatalf("expected background context to have no opid")
	}
	if _, ok := OpIDFromContext(context.TODO()); ok {
		t.Fatalf("expected TODO context to have no opid")
	}
}

func TestPrepareDirAndWithStagingDir_InputValidation(t *testing.T) {
	if _, err := PrepareDir(context.Background(), ""); err == nil {
		t.Fatalf("expected empty targetDir error")
	}
	if _, err := PrepareDir(context.Background(), filepath.Join(t.TempDir(), "dist", "app")); err == nil || !strings.Contains(err.Error(), "tmpRoot is required") {
		t.Fatalf("expected tmpRoot required error, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PrepareDir(ctx, filepath.Join(t.TempDir(), "dist", "app")); err != context.Canceled {
		t.Fatalf("PrepareDir canceled error = %v, want %v", err, context.Canceled)
	}
	if err := WithStagingDir(context.Background(), "", func(string) error { return nil }); err == nil {
		t.Fatalf("expected empty targetDir error")
	}
	if err := WithStagingDir(context.Background(), filepath.Join(t.TempDir(), "dist"), nil); err == nil {
		t.Fatalf("expected nil callback error")
	}
	if err := WithStagingDir(context.Background(), filepath.Join(t.TempDir(), "dist"), func(string) error { return nil }); err == nil || !strings.Contains(err.Error(), "tmpRoot is required") {
		t.Fatalf("expected tmpRoot required error, got %v", err)
	}
	if err := WithStagingDir(ctx, filepath.Join(t.TempDir(), "dist"), func(string) error { return nil }); err != context.Canceled {
		t.Fatalf("WithStagingDir canceled error = %v, want %v", err, context.Canceled)
	}
}

func TestDirStaging_NilAndNoOpBranches(t *testing.T) {
	if err := (*DirStaging)(nil).Abort(); err != nil {
		t.Fatalf("nil Abort error = %v", err)
	}
	if err := (*DirStaging)(nil).Commit(); err != nil {
		t.Fatalf("nil Commit error = %v", err)
	}
	if err := (*DirStaging)(nil).CommitKeepBackup(); err != nil {
		t.Fatalf("nil CommitKeepBackup error = %v", err)
	}
	if err := (*DirStaging)(nil).Finalize(); err != nil {
		t.Fatalf("nil Finalize error = %v", err)
	}
	if err := (*DirStaging)(nil).Rollback(); err != nil {
		t.Fatalf("nil Rollback error = %v", err)
	}

	s := &DirStaging{committed: true}
	if err := s.Abort(); err != nil {
		t.Fatalf("Abort on committed staging: %v", err)
	}
	if err := s.Commit(); err != nil {
		t.Fatalf("Commit on committed staging: %v", err)
	}
	if err := s.CommitKeepBackup(); err != nil {
		t.Fatalf("CommitKeepBackup on committed staging: %v", err)
	}
	if err := s.Rollback(); err != nil {
		t.Fatalf("Rollback on committed staging without target data: %v", err)
	}

	if err := (&DirStaging{}).Rollback(); err != nil {
		t.Fatalf("Rollback before commit: %v", err)
	}
}

func TestWriteFileAtomicAndAtomicReplaceDir_Errors(t *testing.T) {
	if err := WriteFileAtomic("", []byte("x"), 0o644); err == nil {
		t.Fatalf("expected empty path error")
	}
	if err := atomicReplaceDir("", "/tmp/target", "/tmp/backup"); err == nil {
		t.Fatalf("expected empty path validation error")
	}

	root := t.TempDir()
	target := filepath.Join(root, "dist")
	backup := filepath.Join(root, "dist.old")
	stagingDir := filepath.Join(root, "missing-staging")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := atomicReplaceDir(stagingDir, target, backup); err == nil {
		t.Fatalf("expected promote staging dir error")
	}
	if _, err := os.Stat(filepath.Join(target, "old.txt")); err != nil {
		t.Fatalf("expected target restored after failed promote: %v", err)
	}
}

func TestWithStagingDir_SuccessReplaces(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "dist", "app", "assets")
	ctx := WithTmpRoot(context.Background(), filepath.Join(base, "custom-tmp"))
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}

	err := WithStagingDir(ctx, target, func(stagingDir string) error {
		if err := os.MkdirAll(stagingDir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(stagingDir, "new.txt"), []byte("new"), 0o644)
	})
	if err != nil {
		t.Fatalf("WithStagingDir: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "new.txt")); err != nil {
		t.Fatalf("expected new.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "old.txt")); err == nil {
		t.Fatalf("expected old.txt removed")
	}
}

func TestWithStagingDir_FailureDoesNotReplace(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "dist", "app", "assets")
	ctx := WithTmpRoot(context.Background(), filepath.Join(base, "custom-tmp"))
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}

	err := WithStagingDir(ctx, target, func(stagingDir string) error {
		if err := os.WriteFile(filepath.Join(stagingDir, "new.txt"), []byte("new"), 0o644); err != nil {
			return err
		}
		return os.ErrPermission
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	if _, err := os.Stat(filepath.Join(target, "old.txt")); err != nil {
		t.Fatalf("expected old.txt still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "new.txt")); err == nil {
		t.Fatalf("expected new.txt not present")
	}
}

func TestWithStagingDir_UsesOpIDFromContext(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "dist", "app", "assets")
	tmpRoot := filepath.Join(base, "custom-tmp")

	opid := "op-test-123"
	ctx := WithTmpRoot(WithOpID(context.Background(), opid), tmpRoot)

	var gotStagingDir string
	err := WithStagingDir(ctx, target, func(stagingDir string) error {
		gotStagingDir = stagingDir
		return os.WriteFile(filepath.Join(stagingDir, "ok.txt"), []byte("ok"), 0o644)
	})
	if err != nil {
		t.Fatalf("WithStagingDir: %v", err)
	}
	if gotStagingDir == "" {
		t.Fatalf("expected staging dir")
	}
	stagingRoot, err := WorkspaceStagingRoot(filepath.Dir(target), tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot: %v", err)
	}
	wantPrefix := filepath.Join(stagingRoot, opid)
	if filepath.Clean(filepath.Dir(gotStagingDir)) != filepath.Clean(wantPrefix) {
		t.Fatalf("unexpected staging dir parent: got %q, want %q", filepath.Dir(gotStagingDir), wantPrefix)
	}
	if !strings.HasPrefix(filepath.Base(gotStagingDir), filepath.Base(target)+"-") {
		t.Fatalf("unexpected staging dir leaf: %q", filepath.Base(gotStagingDir))
	}
}

func TestPrepareDir_UsesTmpRootFromContext(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "dist", "app", "assets")
	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")

	ctx := WithTmpRoot(WithOpID(context.Background(), "op-tmp-root"), tmpRoot)
	s, err := PrepareDir(ctx, target)
	if err != nil {
		t.Fatalf("PrepareDir: %v", err)
	}
	defer s.Abort()

	stagingRoot, err := WorkspaceStagingRoot(filepath.Dir(target), tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot: %v", err)
	}
	wantParent := filepath.Join(stagingRoot, "op-tmp-root")
	if filepath.Clean(filepath.Dir(s.StagingDir)) != filepath.Clean(wantParent) {
		t.Fatalf("unexpected staging dir parent: got %q, want %q", filepath.Dir(s.StagingDir), wantParent)
	}
}

func TestPrepareDir_CommitReplacesAndSharesOpID(t *testing.T) {
	base := t.TempDir()
	ctx := WithTmpRoot(WithOpID(context.Background(), "op-shared"), filepath.Join(base, "custom-tmp"))

	targetA := filepath.Join(base, "dist", "a")
	targetB := filepath.Join(base, "modules", "api", "a")
	if err := os.MkdirAll(targetA, 0o755); err != nil {
		t.Fatalf("mkdir targetA: %v", err)
	}
	if err := os.MkdirAll(targetB, 0o755); err != nil {
		t.Fatalf("mkdir targetB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetA, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("seed targetA: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetB, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("seed targetB: %v", err)
	}

	sa, err := PrepareDir(ctx, targetA)
	if err != nil {
		t.Fatalf("PrepareDir A: %v", err)
	}
	defer sa.Abort()

	sb, err := PrepareDir(ctx, targetB)
	if err != nil {
		t.Fatalf("PrepareDir B: %v", err)
	}
	defer sb.Abort()

	if !strings.HasPrefix(filepath.Base(sb.StagingDir), filepath.Base(targetB)+"-") {
		t.Fatalf("unexpected staging dir B leaf: %q", filepath.Base(sb.StagingDir))
	}
	if filepath.Base(filepath.Dir(sa.StagingDir)) != "op-shared" {
		t.Fatalf("unexpected staging dir A opid: %q", sa.StagingDir)
	}
	if filepath.Base(filepath.Dir(sb.StagingDir)) != "op-shared" {
		t.Fatalf("unexpected staging dir B opid: %q", sb.StagingDir)
	}
	if !strings.HasPrefix(filepath.Base(sa.StagingDir), filepath.Base(targetA)+"-") {
		t.Fatalf("unexpected staging dir A leaf: %q", filepath.Base(sa.StagingDir))
	}
	if sa.StagingDir == sb.StagingDir {
		t.Fatalf("expected unique staging dirs for different targets, got %q", sa.StagingDir)
	}

	if err := os.WriteFile(filepath.Join(sa.StagingDir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write staging A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sb.StagingDir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write staging B: %v", err)
	}

	if err := sa.Commit(); err != nil {
		t.Fatalf("Commit A: %v", err)
	}
	if err := sb.Commit(); err != nil {
		t.Fatalf("Commit B: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetA, "new.txt")); err != nil {
		t.Fatalf("expected targetA new.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetA, "old.txt")); err == nil {
		t.Fatalf("expected targetA old removed")
	}
	if _, err := os.Stat(filepath.Join(targetB, "new.txt")); err != nil {
		t.Fatalf("expected targetB new.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetB, "old.txt")); err == nil {
		t.Fatalf("expected targetB old removed")
	}
}

func TestPrepareDir_CommitKeepBackupRollbackRestoresTarget(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	target := filepath.Join(base, "dist", "app")
	ctx := WithTmpRoot(WithOpID(context.Background(), "rollback-op"), filepath.Join(base, "custom-tmp"))
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	s, err := PrepareDir(ctx, target)
	if err != nil {
		t.Fatalf("PrepareDir: %v", err)
	}
	defer s.Abort()

	if err := os.WriteFile(filepath.Join(s.StagingDir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write staging: %v", err)
	}
	if err := s.CommitKeepBackup(); err != nil {
		t.Fatalf("CommitKeepBackup: %v", err)
	}
	if err := s.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "old.txt")); err != nil {
		t.Fatalf("expected old target restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "new.txt")); err == nil {
		t.Fatalf("expected promoted file removed after rollback")
	}
	if _, err := os.Stat(target + ".failed." + s.OpID); err == nil {
		t.Fatalf("expected failed target artifact removed")
	}
}

func TestPrepareDir_RollbackRemovesNewTargetWhenOriginalMissing(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	target := filepath.Join(base, "dist", "app")
	ctx := WithTmpRoot(WithOpID(context.Background(), "rollback-new"), filepath.Join(base, "custom-tmp"))
	s, err := PrepareDir(ctx, target)
	if err != nil {
		t.Fatalf("PrepareDir: %v", err)
	}
	defer s.Abort()

	if err := os.WriteFile(filepath.Join(s.StagingDir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write staging: %v", err)
	}
	if err := s.CommitKeepBackup(); err != nil {
		t.Fatalf("CommitKeepBackup: %v", err)
	}
	if err := s.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected target removed after rollback, got err=%v", err)
	}
}

func TestCleanupCrashArtifacts_RemovesStaleArtifacts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
	currentOp := "keep-me"
	stagingRoot, err := WorkspaceStagingRoot(root, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot: %v", err)
	}
	staleStaging := filepath.Join(stagingRoot, "stale-op", "assets")
	currentStaging := filepath.Join(stagingRoot, currentOp, "assets")
	leftoverOld := filepath.Join(root, "dist.old.stale-op")
	leftoverFailed := filepath.Join(root, "dist.failed.stale-op")

	for _, path := range []string{staleStaging, currentStaging, leftoverOld, leftoverFailed} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write keep marker: %v", err)
	}

	ctx := WithTmpRoot(context.Background(), tmpRoot)
	if err := CleanupCrashArtifacts(ctx, []string{"", root}, currentOp); err != nil {
		t.Fatalf("CleanupCrashArtifacts() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(stagingRoot, "stale-op")); err == nil {
		t.Fatalf("expected stale staging op removed")
	}
	if _, err := os.Stat(filepath.Join(stagingRoot, currentOp)); err != nil {
		t.Fatalf("expected current op staging kept: %v", err)
	}
	if _, err := os.Stat(leftoverOld); err == nil {
		t.Fatalf("expected .old artifact removed")
	}
	if _, err := os.Stat(leftoverFailed); err == nil {
		t.Fatalf("expected .failed artifact removed")
	}
	if _, err := os.Stat(filepath.Join(root, "keep.txt")); err != nil {
		t.Fatalf("expected unrelated files preserved: %v", err)
	}
}

func TestCleanupCrashArtifacts_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := CleanupCrashArtifacts(ctx, []string{t.TempDir()}, "current"); err != context.Canceled {
		t.Fatalf("CleanupCrashArtifacts() error = %v, want %v", err, context.Canceled)
	}
}

func TestCleanupCrashArtifacts_UsesTmpRootFromContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
	currentOp := "keep-me"
	stagingRoot, err := WorkspaceStagingRoot(root, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot: %v", err)
	}
	staleStaging := filepath.Join(stagingRoot, "stale-op", "assets")
	currentStaging := filepath.Join(stagingRoot, currentOp, "assets")

	for _, path := range []string{staleStaging, currentStaging} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	ctx := WithTmpRoot(context.Background(), tmpRoot)
	if err := CleanupCrashArtifacts(ctx, []string{root}, currentOp); err != nil {
		t.Fatalf("CleanupCrashArtifacts() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(stagingRoot, "stale-op")); err == nil {
		t.Fatalf("expected stale staging op removed")
	}
	if _, err := os.Stat(filepath.Join(stagingRoot, currentOp)); err != nil {
		t.Fatalf("expected current op staging kept: %v", err)
	}
}

func TestWorkspaceStagingRoot_IsSharedAcrossChoysumTargets(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")

	distTarget := filepath.Join(repoRoot, ".choysum", "dist", "apps", "auth")
	generatedTarget := filepath.Join(repoRoot, ".choysum", "generated", "proto", "auth")

	distStagingRoot, err := WorkspaceStagingRoot(distTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot(dist) error = %v", err)
	}
	generatedStagingRoot, err := WorkspaceStagingRoot(generatedTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot(generated) error = %v", err)
	}

	if filepath.Clean(distStagingRoot) != filepath.Clean(generatedStagingRoot) {
		t.Fatalf("expected dist and generated staging roots to match, got dist=%q generated=%q", distStagingRoot, generatedStagingRoot)
	}
	wantStagingRoot := filepath.Join(tmpRoot, "staging")
	if filepath.Clean(distStagingRoot) != filepath.Clean(wantStagingRoot) {
		t.Fatalf("expected staging root %q, got %q", wantStagingRoot, distStagingRoot)
	}
	if strings.Contains(filepath.ToSlash(distStagingRoot), "/workspaces/") {
		t.Fatalf("expected staging root without workspace-key segment, got %q", distStagingRoot)
	}
	if filepath.Base(distStagingRoot) != "staging" {
		t.Fatalf("expected staging root leaf to be 'staging', got %q", filepath.Base(distStagingRoot))
	}
}

func TestWorkspaceStagingRoot_UsesRepoRootForClassicDistAndModulesPaths(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")

	distTarget := filepath.Join(repoRoot, "dist", "apps", "auth")
	modulesTarget := filepath.Join(repoRoot, "modules", "auth")

	distStagingRoot, err := WorkspaceStagingRoot(distTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot(dist) error = %v", err)
	}
	modulesStagingRoot, err := WorkspaceStagingRoot(modulesTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot(modules) error = %v", err)
	}

	if filepath.Clean(distStagingRoot) != filepath.Clean(modulesStagingRoot) {
		t.Fatalf("expected dist and modules staging roots to match, got dist=%q modules=%q", distStagingRoot, modulesStagingRoot)
	}
}

func TestWorkspaceTmpRoot_IsSharedAcrossChoysumTargets(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")

	distTarget := filepath.Join(repoRoot, ".choysum", "dist", "apps", "auth")
	generatedTarget := filepath.Join(repoRoot, ".choysum", "generated", "proto", "auth")

	distTmpRoot, err := WorkspaceTmpRoot(distTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceTmpRoot(dist) error = %v", err)
	}
	generatedTmpRoot, err := WorkspaceTmpRoot(generatedTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceTmpRoot(generated) error = %v", err)
	}

	if filepath.Clean(distTmpRoot) != filepath.Clean(generatedTmpRoot) {
		t.Fatalf("expected dist and generated tmp roots to match, got dist=%q generated=%q", distTmpRoot, generatedTmpRoot)
	}
	if filepath.Clean(distTmpRoot) != filepath.Clean(tmpRoot) {
		t.Fatalf("expected shared tmp root %q, got %q", tmpRoot, distTmpRoot)
	}
	if strings.Contains(filepath.ToSlash(distTmpRoot), "/workspaces/") {
		t.Fatalf("expected workspace tmp root without workspace-key segment, got %q", distTmpRoot)
	}

	distStagingRoot, err := WorkspaceStagingRoot(distTarget, tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot(dist) error = %v", err)
	}
	if filepath.Clean(distStagingRoot) != filepath.Clean(filepath.Join(distTmpRoot, "staging")) {
		t.Fatalf("expected staging root to be workspace tmp root + staging, got staging=%q tmp=%q", distStagingRoot, distTmpRoot)
	}
}

func TestWriteFileAtomic_ReplacesExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.js")

	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("expected new content, got %q", string(got))
	}
}

func TestWriteFileAtomic_CreatesParents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "index.js")

	if err := WriteFileAtomic(path, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "ok" {
		t.Fatalf("expected ok, got %q", string(got))
	}
}
