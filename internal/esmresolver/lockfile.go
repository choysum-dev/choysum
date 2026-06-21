// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const lockfileVersion = 1

// EsmLockfile represents the esm.lock file that pins resolved ESM package
// versions for reproducible builds.
type EsmLockfile struct {
	Version  int                  `json:"version"`
	Packages map[string]LockEntry `json:"packages"`
}

// LockEntry records the resolved version and source URL for a package specifier.
type LockEntry struct {
	Version  string `json:"version"`
	Resolved string `json:"resolved"`
}

// ReadLockfile reads and parses an esm.lock file at the given path.
// Returns nil if the file does not exist.
func ReadLockfile(lockfilePath string) (*EsmLockfile, error) {
	data, err := os.ReadFile(lockfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read esm.lock: %w", err)
	}
	var lock EsmLockfile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parse esm.lock: %w", err)
	}
	if lock.Version != lockfileVersion {
		return nil, fmt.Errorf("unsupported esm.lock version %d (expected %d)", lock.Version, lockfileVersion)
	}
	if lock.Packages == nil {
		lock.Packages = make(map[string]LockEntry)
	}
	return &lock, nil
}

// WriteLockfile writes the lockfile to the given path atomically.
func WriteLockfile(lockfilePath string, lock *EsmLockfile) error {
	if lock == nil {
		return nil
	}
	lock.Version = lockfileVersion
	if lock.Packages == nil {
		lock.Packages = make(map[string]LockEntry)
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal esm.lock: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(lockfilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create esm.lock directory: %w", err)
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(lockfilePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create esm.lock tmp: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpClosed := false
	cleanup := true
	defer func() {
		if !tmpClosed {
			_ = tmpFile.Close()
		}
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(0644); err != nil {
		return fmt.Errorf("chmod esm.lock tmp: %w", err)
	}
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write esm.lock tmp: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		tmpClosed = true
		return fmt.Errorf("close esm.lock tmp: %w", err)
	}
	tmpClosed = true

	if err := renameFileWithBackup(tmpPath, lockfilePath); err != nil {
		return fmt.Errorf("rename esm.lock tmp: %w", err)
	}

	cleanup = false
	return nil
}

func renameFileWithBackup(tmpPath, destPath string) error {
	if err := os.Rename(tmpPath, destPath); err == nil {
		return nil
	} else {
		backupPath, hasBackup, backupErr := moveExistingFileToBackup(destPath)
		if backupErr != nil {
			return fmt.Errorf("%w (backup failed: %v)", err, backupErr)
		}
		if retryErr := os.Rename(tmpPath, destPath); retryErr != nil {
			if hasBackup {
				if restoreErr := os.Rename(backupPath, destPath); restoreErr != nil {
					return fmt.Errorf("%w (restore backup failed: %v)", retryErr, restoreErr)
				}
			}
			return retryErr
		}
		if hasBackup {
			if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("cleanup backup file: %w", err)
			}
		}
		return nil
	}
}

func moveExistingFileToBackup(destPath string) (string, bool, error) {
	info, err := os.Stat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("destination path is a directory")
	}

	backupFile, err := os.CreateTemp(filepath.Dir(destPath), filepath.Base(destPath)+".bak-*")
	if err != nil {
		return "", false, err
	}
	backupPath := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		_ = os.Remove(backupPath)
		return "", false, err
	}
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if err := os.Rename(destPath, backupPath); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	return backupPath, true, nil
}

// LookupLockedSpec returns the locked version specifier for a package import
// path. For example, given "kysely", returns "kysely@0.27.6". For scoped
// packages like "@scope/pkg/sub", the lock key is the full import specifier.
// Returns the original specifier if no lock entry exists.
func LookupLockedSpec(lock *EsmLockfile, specifier string) string {
	if lock == nil || len(lock.Packages) == 0 {
		return specifier
	}
	entry, ok := lock.Packages[specifier]
	if !ok || entry.Version == "" {
		return specifier
	}
	locked := lockSpecifier(specifier, entry.Version)
	if locked == "" {
		return specifier
	}
	return locked
}

// lockSpecifier constructs a versioned specifier like "kysely@0.27.6" from
// a bare specifier and resolved version.
func lockSpecifier(specifier, version string) string {
	specifier = strings.TrimSpace(specifier)
	version = strings.TrimSpace(version)
	if specifier == "" || version == "" {
		return ""
	}
	// If the specifier already contains a version (kysely@1.0.0), strip it.
	if idx := strings.LastIndex(specifier, "@"); idx > 0 {
		specifier = specifier[:idx]
	}
	return specifier + "@" + version
}
