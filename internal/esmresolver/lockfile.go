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
	Version  int                   `json:"version"`
	Packages map[string]LockEntry  `json:"packages"`
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
	tmpFile := lockfilePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("write esm.lock tmp: %w", err)
	}
	if err := os.Rename(tmpFile, lockfilePath); err != nil {
		return fmt.Errorf("rename esm.lock tmp: %w", err)
	}
	return nil
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
