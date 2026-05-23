// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package staging

import (
	"context"
	"os"
	"path/filepath"

	xfmt "golang.org/x/exp/errors/fmt"
)

// FileStaging represents a prepared staging file for a target file path.
// Call CommitKeepBackup() to atomically replace the target file, then Finalize() or Rollback().
type FileStaging struct {
	TargetPath  string
	StagingPath string
	BackupPath  string
	OpDir       string
	StagingRoot string
	OpID        string
	HadTarget   bool
	committed   bool
}

// PrepareFile creates a staging file path for targetPath but does NOT swap it into place.
// The staging path layout matches the centralized PrepareDir layout:
//
//	<tmpRoot>/staging/<opid>/<target-hash>
func PrepareFile(ctx context.Context, targetPath string) (*FileStaging, error) {
	if targetPath == "" {
		return nil, xfmt.Errorf("targetPath is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	tmpRoot, _ := tmpRootFromContext(ctx)
	stagingRoot, err := WorkspaceStagingRoot(filepath.Dir(targetPath), tmpRoot)
	if err != nil {
		return nil, xfmt.Errorf("resolve staging root: %w", err)
	}
	opid, ok := OpIDFromContext(ctx)
	if !ok {
		opid = NewOpID()
	}
	opDir := filepath.Join(stagingRoot, opid)
	stagingPath := filepath.Join(opDir, stagingEntryName("file", targetPath))
	backupPath := targetPath + ".old." + opid

	if err := os.MkdirAll(opDir, 0o755); err != nil {
		return nil, xfmt.Errorf("create file staging dir: %w", err)
	}
	// Ensure any old staging file doesn't leak across retries.
	_ = os.Remove(stagingPath)

	return &FileStaging{
		TargetPath:  targetPath,
		StagingPath: stagingPath,
		BackupPath:  backupPath,
		OpDir:       opDir,
		StagingRoot: stagingRoot,
		OpID:        opid,
	}, nil
}

func (s *FileStaging) Abort() error {
	if s == nil {
		return nil
	}
	if s.committed {
		return nil
	}
	if s.StagingPath != "" {
		_ = os.Remove(s.StagingPath)
	}
	if s.OpDir != "" {
		_ = os.Remove(s.OpDir) // only removes if empty
	}
	if s.StagingRoot != "" {
		_ = os.Remove(s.StagingRoot) // only removes if empty
	}
	return nil
}

// CommitKeepBackup atomically replaces the target file with the staging file,
// keeping a backup so callers can rollback if a later step fails.
func (s *FileStaging) CommitKeepBackup() error {
	if s == nil {
		return nil
	}
	if s.committed {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.TargetPath), 0o755); err != nil {
		return xfmt.Errorf("mkdir target parent: %w", err)
	}

	if _, err := os.Stat(s.TargetPath); err == nil {
		s.HadTarget = true
	} else if os.IsNotExist(err) {
		s.HadTarget = false
	} else {
		return xfmt.Errorf("stat target file: %w", err)
	}

	if s.HadTarget {
		_ = os.Remove(s.BackupPath)
		if err := os.Rename(s.TargetPath, s.BackupPath); err != nil {
			return xfmt.Errorf("backup target file: %w", err)
		}
	}

	if err := os.Rename(s.StagingPath, s.TargetPath); err != nil {
		// Best-effort rollback.
		if s.HadTarget {
			_ = os.Rename(s.BackupPath, s.TargetPath)
		}
		return xfmt.Errorf("promote staging file: %w", err)
	}

	s.committed = true
	return nil
}

func (s *FileStaging) Finalize() error {
	if s == nil {
		return nil
	}
	if !s.committed {
		return nil
	}
	if s.BackupPath != "" {
		_ = os.Remove(s.BackupPath)
	}
	if s.OpDir != "" {
		_ = os.Remove(s.OpDir) // only removes if empty
	}
	if s.StagingRoot != "" {
		_ = os.Remove(s.StagingRoot) // only removes if empty
	}
	return nil
}

func (s *FileStaging) Rollback() error {
	if s == nil {
		return nil
	}
	if !s.committed {
		return nil
	}

	if !s.HadTarget {
		_ = os.Remove(s.TargetPath)
		return nil
	}

	if _, err := os.Stat(s.BackupPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return xfmt.Errorf("stat backup file: %w", err)
	}

	failedNew := s.TargetPath + ".failed." + s.OpID
	_ = os.Remove(failedNew)

	if _, err := os.Stat(s.TargetPath); err == nil {
		_ = os.Rename(s.TargetPath, failedNew)
	}

	if err := os.Rename(s.BackupPath, s.TargetPath); err != nil {
		// Best-effort restore new target back.
		_ = os.Rename(failedNew, s.TargetPath)
		return xfmt.Errorf("rollback restore backup file: %w", err)
	}

	_ = os.Remove(failedNew)
	if s.OpDir != "" {
		_ = os.Remove(s.OpDir) // only removes if empty
	}
	if s.StagingRoot != "" {
		_ = os.Remove(s.StagingRoot) // only removes if empty
	}
	return nil
}
