// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package staging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
)

func TestPrepareFile_InputValidationAndNoTargetCommit(t *testing.T) {
	t.Run("validates empty path and canceled context", func(t *testing.T) {
		if _, err := PrepareFile(context.Background(), ""); err == nil {
			t.Fatalf("expected empty targetPath error")
		}
		if _, err := PrepareFile(context.Background(), filepath.Join(t.TempDir(), distmanifest.DistManifestFileName)); err == nil || !strings.Contains(err.Error(), "tmpRoot is required") {
			t.Fatalf("expected tmpRoot required error, got %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := PrepareFile(ctx, filepath.Join(t.TempDir(), distmanifest.DistManifestFileName)); err != context.Canceled {
			t.Fatalf("PrepareFile canceled error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("commits and finalizes when target did not exist", func(t *testing.T) {
		root := t.TempDir()
		tmpRoot := filepath.Join(root, "custom-tmp")
		ctx := WithTmpRoot(WithOpID(context.Background(), "op-no-target"), tmpRoot)
		target := filepath.Join(root, distmanifest.DistManifestFileName)

		s, err := PrepareFile(ctx, target)
		if err != nil {
			t.Fatalf("PrepareFile: %v", err)
		}
		defer s.Abort()

		stagingRoot, err := WorkspaceStagingRoot(filepath.Dir(target), tmpRoot)
		if err != nil {
			t.Fatalf("WorkspaceStagingRoot: %v", err)
		}
		wantParent := filepath.Join(stagingRoot, "op-no-target")
		if filepath.Clean(filepath.Dir(s.StagingPath)) != filepath.Clean(wantParent) {
			t.Fatalf("unexpected staging file parent: got %q, want %q", filepath.Dir(s.StagingPath), wantParent)
		}
		if !strings.HasPrefix(filepath.Base(s.StagingPath), distmanifest.DistManifestFileName+"-") {
			t.Fatalf("unexpected staging file name: %q", filepath.Base(s.StagingPath))
		}

		if err := os.WriteFile(s.StagingPath, []byte("new"), 0o644); err != nil {
			t.Fatalf("write staging: %v", err)
		}
		if err := s.CommitKeepBackup(); err != nil {
			t.Fatalf("CommitKeepBackup: %v", err)
		}
		if s.HadTarget {
			t.Fatalf("expected HadTarget false for brand-new target")
		}
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("expected promoted target: %v", err)
		}
		if err := s.Finalize(); err != nil {
			t.Fatalf("Finalize: %v", err)
		}
	})
}

func TestFileStaging_NilAndAbortBranches(t *testing.T) {
	if err := (*FileStaging)(nil).Abort(); err != nil {
		t.Fatalf("nil Abort error = %v", err)
	}
	if err := (*FileStaging)(nil).CommitKeepBackup(); err != nil {
		t.Fatalf("nil CommitKeepBackup error = %v", err)
	}
	if err := (*FileStaging)(nil).Finalize(); err != nil {
		t.Fatalf("nil Finalize error = %v", err)
	}
	if err := (*FileStaging)(nil).Rollback(); err != nil {
		t.Fatalf("nil Rollback error = %v", err)
	}

	t.Run("abort removes uncommitted staging file", func(t *testing.T) {
		root := t.TempDir()
		ctx := WithTmpRoot(WithOpID(context.Background(), "op-abort"), filepath.Join(root, "custom-tmp"))
		target := filepath.Join(root, distmanifest.DistManifestFileName)

		s, err := PrepareFile(ctx, target)
		if err != nil {
			t.Fatalf("PrepareFile: %v", err)
		}
		if err := os.WriteFile(s.StagingPath, []byte("staged"), 0o644); err != nil {
			t.Fatalf("write staging: %v", err)
		}
		if err := s.Abort(); err != nil {
			t.Fatalf("Abort: %v", err)
		}
		if _, err := os.Stat(s.StagingPath); !os.IsNotExist(err) {
			t.Fatalf("expected staging file removed, got err=%v", err)
		}
	})

	t.Run("rollback before commit is a no-op", func(t *testing.T) {
		s := &FileStaging{}
		if err := s.Rollback(); err != nil {
			t.Fatalf("Rollback before commit: %v", err)
		}
	})
}

func TestPrepareFile_CommitKeepBackup_Rollback(t *testing.T) {
	root := t.TempDir()
	ctx := WithTmpRoot(WithOpID(context.Background(), "op-test"), filepath.Join(root, "custom-tmp"))
	target := filepath.Join(root, distmanifest.DistManifestFileName)

	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	s, err := PrepareFile(ctx, target)
	if err != nil {
		t.Fatalf("PrepareFile: %v", err)
	}
	defer s.Abort()

	if err := os.WriteFile(s.StagingPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("write staging: %v", err)
	}

	if err := s.CommitKeepBackup(); err != nil {
		t.Fatalf("CommitKeepBackup: %v", err)
	}

	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target after commit: %v", err)
	}
	if string(b) != "new" {
		t.Fatalf("unexpected target content after commit: %q", string(b))
	}

	if err := s.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	b, err = os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target after rollback: %v", err)
	}
	if string(b) != "old" {
		t.Fatalf("unexpected target content after rollback: %q", string(b))
	}
}

func TestPrepareFile_CommitKeepBackup_Finalize_RemovesBackup(t *testing.T) {
	root := t.TempDir()
	ctx := WithTmpRoot(WithOpID(context.Background(), "op-test"), filepath.Join(root, "custom-tmp"))
	target := filepath.Join(root, distmanifest.DistManifestFileName)

	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	s, err := PrepareFile(ctx, target)
	if err != nil {
		t.Fatalf("PrepareFile: %v", err)
	}
	defer s.Abort()

	if err := os.WriteFile(s.StagingPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("write staging: %v", err)
	}

	if err := s.CommitKeepBackup(); err != nil {
		t.Fatalf("CommitKeepBackup: %v", err)
	}

	if _, err := os.Stat(s.BackupPath); err != nil {
		t.Fatalf("expected backup to exist after commit: %v", err)
	}

	if err := s.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if _, err := os.Stat(s.BackupPath); err == nil {
		t.Fatalf("expected backup to be removed")
	}
}

func TestPrepareFile_UsesTmpRootFromContext(t *testing.T) {
	ctx := WithTmpRoot(WithOpID(context.Background(), "op-file-tmp"), filepath.Join(t.TempDir(), "custom-tmp"))
	target := filepath.Join(t.TempDir(), distmanifest.DistManifestFileName)

	s, err := PrepareFile(ctx, target)
	if err != nil {
		t.Fatalf("PrepareFile: %v", err)
	}
	defer s.Abort()

	tmpRoot, _ := tmpRootFromContext(ctx)
	stagingRoot, err := WorkspaceStagingRoot(filepath.Dir(target), tmpRoot)
	if err != nil {
		t.Fatalf("WorkspaceStagingRoot: %v", err)
	}
	wantParent := filepath.Join(stagingRoot, "op-file-tmp")
	if filepath.Clean(filepath.Dir(s.StagingPath)) != filepath.Clean(wantParent) {
		t.Fatalf("unexpected staging file parent: got %q, want %q", filepath.Dir(s.StagingPath), wantParent)
	}
}
