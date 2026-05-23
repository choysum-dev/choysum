// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package staging

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/xid"
	xfmt "golang.org/x/exp/errors/fmt"
)

type opIDKey struct{}
type tmpRootKey struct{}

// NewOpID generates an operation id suitable for grouping staging operations.
func NewOpID() string {
	return xid.New().String()
}

// WithOpID associates an opid with ctx.
// When present, WithStagingDir will reuse this opid for its staging directory name.
func WithOpID(ctx context.Context, opid string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	opid = filepath.Clean(opid)
	if opid == "." || opid == string(filepath.Separator) {
		opid = ""
	}
	if opid == "" {
		return ctx
	}
	return context.WithValue(ctx, opIDKey{}, opid)
}

// OpIDFromContext returns the opid associated with ctx.
func OpIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(opIDKey{})
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// WithTmpRoot associates a tmp root with ctx so staging helpers can place
// intermediate artifacts under a caller-provided root.
func WithTmpRoot(ctx context.Context, tmpRoot string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	tmpRoot = strings.TrimSpace(tmpRoot)
	if tmpRoot == "" {
		return ctx
	}
	if absTmpRoot, err := filepath.Abs(tmpRoot); err == nil {
		tmpRoot = absTmpRoot
	}
	tmpRoot = filepath.Clean(tmpRoot)
	if tmpRoot == "." || tmpRoot == string(filepath.Separator) {
		return ctx
	}
	return context.WithValue(ctx, tmpRootKey{}, tmpRoot)
}

func tmpRootFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(tmpRootKey{})
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

// DirStaging represents a prepared staging directory for a targetDir.
// Call Commit() to atomically replace the target directory, or Abort() to clean up.
type DirStaging struct {
	TargetDir   string
	StagingDir  string
	BackupDir   string
	OpDir       string
	StagingRoot string
	OpID        string
	HadTarget   bool
	committed   bool
}

// PrepareDir creates a staging directory for targetDir but does NOT swap it into place.
// The staging directory layout matches WithStagingDir:
//
//	<tmpRoot>/staging/<opid>/<target-hash>
func PrepareDir(ctx context.Context, targetDir string) (*DirStaging, error) {
	if targetDir == "" {
		return nil, xfmt.Errorf("targetDir is empty")
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
	stagingRoot, err := WorkspaceStagingRoot(filepath.Dir(targetDir), tmpRoot)
	if err != nil {
		return nil, xfmt.Errorf("resolve staging root: %w", err)
	}
	opid, ok := OpIDFromContext(ctx)
	if !ok {
		opid = NewOpID()
	}
	opDir := filepath.Join(stagingRoot, opid)
	stagingDir := filepath.Join(opDir, stagingEntryName("dir", targetDir))
	backupDir := targetDir + ".old." + opid

	_ = os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, xfmt.Errorf("create staging dir: %w", err)
	}

	return &DirStaging{
		TargetDir:   targetDir,
		StagingDir:  stagingDir,
		BackupDir:   backupDir,
		OpDir:       opDir,
		StagingRoot: stagingRoot,
		OpID:        opid,
	}, nil
}

func (s *DirStaging) Abort() error {
	if s == nil {
		return nil
	}
	if s.committed {
		return nil
	}
	if s.StagingDir != "" {
		_ = os.RemoveAll(s.StagingDir)
	}
	if s.OpDir != "" {
		_ = os.Remove(s.OpDir) // only removes if empty
	}
	if s.StagingRoot != "" {
		_ = os.Remove(s.StagingRoot) // only removes if empty
	}
	return nil
}

func (s *DirStaging) Commit() error {
	if s == nil {
		return nil
	}
	if s.committed {
		return nil
	}
	if err := s.CommitKeepBackup(); err != nil {
		return err
	}
	return s.Finalize()
}

// CommitKeepBackup atomically replaces the target directory with the staging directory,
// but keeps the backup (if any) so callers can rollback if a later step fails.
//
// Use Finalize() after all related commits succeed, or Rollback() on failure.
func (s *DirStaging) CommitKeepBackup() error {
	if s == nil {
		return nil
	}
	if s.committed {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.TargetDir), 0o755); err != nil {
		return xfmt.Errorf("mkdir target parent: %w", err)
	}

	if _, err := os.Stat(s.TargetDir); err == nil {
		s.HadTarget = true
	} else if os.IsNotExist(err) {
		s.HadTarget = false
	} else {
		return xfmt.Errorf("stat target dir: %w", err)
	}

	if err := atomicReplaceDir(s.StagingDir, s.TargetDir, s.BackupDir); err != nil {
		return err
	}
	s.committed = true
	return nil
}

// Finalize removes any backup created by CommitKeepBackup and cleans up empty staging dirs.
// It is safe to call multiple times.
func (s *DirStaging) Finalize() error {
	if s == nil {
		return nil
	}
	if !s.committed {
		return nil
	}
	if s.BackupDir != "" {
		_ = os.RemoveAll(s.BackupDir)
	}
	if s.OpDir != "" {
		_ = os.Remove(s.OpDir) // only removes if empty
	}
	if s.StagingRoot != "" {
		_ = os.Remove(s.StagingRoot) // only removes if empty
	}
	return nil
}

// Rollback restores the target directory from the backup kept by CommitKeepBackup.
// If the target did not exist prior to the commit, rollback removes the target.
// It is best-effort and safe to call multiple times.
func (s *DirStaging) Rollback() error {
	if s == nil {
		return nil
	}
	if !s.committed {
		return nil
	}

	// If there was no prior target, simply remove the new target.
	if !s.HadTarget {
		_ = os.RemoveAll(s.TargetDir)
		return nil
	}

	if _, err := os.Stat(s.BackupDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return xfmt.Errorf("stat backup dir: %w", err)
	}

	failedNew := s.TargetDir + ".failed." + s.OpID
	_ = os.RemoveAll(failedNew)

	if _, err := os.Stat(s.TargetDir); err == nil {
		if err := os.Rename(s.TargetDir, failedNew); err != nil {
			return xfmt.Errorf("rollback move new target: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return xfmt.Errorf("stat target dir for rollback: %w", err)
	}

	if err := os.Rename(s.BackupDir, s.TargetDir); err != nil {
		// Best-effort restore the new target back.
		if _, st := os.Stat(failedNew); st == nil {
			_ = os.Rename(failedNew, s.TargetDir)
		}
		return xfmt.Errorf("rollback restore backup: %w", err)
	}

	_ = os.RemoveAll(failedNew)
	if s.OpDir != "" {
		_ = os.Remove(s.OpDir) // only removes if empty
	}
	if s.StagingRoot != "" {
		_ = os.Remove(s.StagingRoot) // only removes if empty
	}
	return nil
}

// WriteFileAtomic writes data to path using a temporary file in the same
// directory and then renames it into place.
//
// On POSIX filesystems, rename is atomic and replaces the target if it exists.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return xfmt.Errorf("path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return xfmt.Errorf("mkdir parent: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp.*")
	if err != nil {
		return xfmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return xfmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return xfmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return xfmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		// Windows won't overwrite an existing file. Best-effort fallback.
		if rmErr := os.Remove(path); rmErr == nil {
			if err2 := os.Rename(tmpName, path); err2 == nil {
				return nil
			}
		}
		return xfmt.Errorf("rename into place: %w", err)
	}

	return nil
}

// WithStagingDir runs fn in a centralized staging directory and then atomically
// replaces targetDir with the staged directory via rename.
//
// Layout:
//
//	<workspace>/.choysum/tmp/staging/<opid>/<target-hash>  (stagingDir)
//
// The swap is best-effort atomic and uses rename to promote stagingDir.
func WithStagingDir(ctx context.Context, targetDir string, fn func(stagingDir string) error) error {
	if targetDir == "" {
		return xfmt.Errorf("targetDir is empty")
	}
	if fn == nil {
		return xfmt.Errorf("fn is nil")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s, err := PrepareDir(ctx, targetDir)
	if err != nil {
		return err
	}
	defer func() { _ = s.Abort() }()

	if err := fn(s.StagingDir); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := s.Commit(); err != nil {
		return err
	}
	return nil
}

func atomicReplaceDir(stagingDir, targetDir, backupDir string) error {
	if stagingDir == "" || targetDir == "" || backupDir == "" {
		return xfmt.Errorf("atomicReplaceDir: empty path")
	}

	if _, err := os.Stat(targetDir); err == nil {
		if err := os.Rename(targetDir, backupDir); err != nil {
			return xfmt.Errorf("backup target dir: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return xfmt.Errorf("stat target dir: %w", err)
	}

	if err := os.Rename(stagingDir, targetDir); err != nil {
		// Best-effort rollback.
		if _, st := os.Stat(backupDir); st == nil {
			_ = os.Rename(backupDir, targetDir)
		}
		return xfmt.Errorf("promote staging dir: %w", err)
	}

	return nil
}
