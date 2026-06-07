// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	xfmt "golang.org/x/exp/errors/fmt"
)

const ModulesLockVersion = 1

type WorkspaceModulesLock struct {
	Version   int                `json:"version"`
	UpdatedAt string             `json:"updatedAt"`
	Modules   map[string]Binding `json:"modules"`
}

func normalizeBinding(binding Binding) Binding {
	binding.ModuleName = strings.TrimSpace(binding.ModuleName)
	binding.OriginType = strings.TrimSpace(binding.OriginType)
	binding.OriginRef = strings.TrimSpace(binding.OriginRef)
	binding.ResolvedVersion = strings.TrimSpace(binding.ResolvedVersion)
	binding.Integrity = strings.TrimSpace(binding.Integrity)
	binding.LocalPath = strings.TrimSpace(binding.LocalPath)
	binding.UpdatedAt = strings.TrimSpace(binding.UpdatedAt)
	return binding
}

func equalBindingContent(a, b Binding) bool {
	a = normalizeBinding(a)
	b = normalizeBinding(b)
	a.UpdatedAt = ""
	b.UpdatedAt = ""
	return a == b
}

func newWorkspaceModulesLock() *WorkspaceModulesLock {
	return &WorkspaceModulesLock{
		Version: ModulesLockVersion,
		Modules: map[string]Binding{},
	}
}

type LockStoreOption func(*LockStore)

func WithLockStoreDefaultChoysumPath(defaultChoysumPath string) LockStoreOption {
	return func(s *LockStore) {
		if s == nil {
			return
		}
		s.defaultChoysumPath = strings.TrimSpace(defaultChoysumPath)
	}
}

type LockStore struct {
	defaultChoysumPath string
}

func NewLockStore(opts ...LockStoreOption) *LockStore {
	store := &LockStore{}
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store
}

func (s *LockStore) Read(workspaceRoot string) (*WorkspaceModulesLock, error) {
	path, err := modulesLockFilePath(workspaceRoot, s.defaultChoysumPath)
	if err != nil {
		return nil, xfmt.Errorf("resolve modules lock file path: %w", err)
	}
	return readWorkspaceModulesLock(path)
}

func (s *LockStore) LookupBinding(workspaceRoot, moduleName string) (Binding, bool, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return Binding{}, false, xfmt.Errorf("module name is empty")
	}
	lock, err := s.Read(workspaceRoot)
	if err != nil {
		return Binding{}, false, err
	}
	binding, ok := lock.Modules[moduleName]
	return binding, ok, nil
}

func (s *LockStore) UpsertBinding(workspaceRoot string, binding Binding) error {
	moduleName := strings.TrimSpace(binding.ModuleName)
	if moduleName == "" {
		return xfmt.Errorf("module binding moduleName is empty")
	}
	release, err := AcquireModulesLockLease(workspaceRoot, "modules-lock-upsert", s.defaultChoysumPath)
	if err != nil {
		return err
	}
	defer release()

	path, err := modulesLockFilePath(workspaceRoot, s.defaultChoysumPath)
	if err != nil {
		return xfmt.Errorf("resolve modules lock file path: %w", err)
	}
	lock, err := readWorkspaceModulesLock(path)
	if err != nil {
		return err
	}
	binding = normalizeBinding(binding)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	binding.ModuleName = moduleName
	if existing, ok := lock.Modules[moduleName]; ok && equalBindingContent(existing, binding) {
		return nil
	}
	binding.UpdatedAt = now
	lock.Modules[moduleName] = binding
	lock.UpdatedAt = now
	if err := writeWorkspaceModulesLock(path, lock); err != nil {
		return err
	}
	return nil
}

func (s *LockStore) DeleteBinding(workspaceRoot, moduleName string) error {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return xfmt.Errorf("module name is empty")
	}
	release, err := AcquireModulesLockLease(workspaceRoot, "modules-lock-delete", s.defaultChoysumPath)
	if err != nil {
		return err
	}
	defer release()

	path, err := modulesLockFilePath(workspaceRoot, s.defaultChoysumPath)
	if err != nil {
		return xfmt.Errorf("resolve modules lock file path: %w", err)
	}
	lock, err := readWorkspaceModulesLock(path)
	if err != nil {
		return err
	}
	if _, exists := lock.Modules[moduleName]; !exists {
		return nil
	}
	delete(lock.Modules, moduleName)
	lock.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeWorkspaceModulesLock(path, lock); err != nil {
		return err
	}
	return nil
}

func readWorkspaceModulesLock(path string) (*WorkspaceModulesLock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newWorkspaceModulesLock(), nil
		}
		return nil, xfmt.Errorf("read modules lock file failed: %w", err)
	}
	lock := &WorkspaceModulesLock{}
	if err := json.Unmarshal(data, lock); err != nil {
		return nil, xfmt.Errorf("decode modules lock file failed: %w", err)
	}
	if lock.Version == 0 {
		lock.Version = ModulesLockVersion
	}
	if lock.Modules == nil {
		lock.Modules = map[string]Binding{}
	}
	return lock, nil
}

func writeWorkspaceModulesLock(path string, lock *WorkspaceModulesLock) error {
	if lock == nil {
		lock = newWorkspaceModulesLock()
	}
	if lock.Version == 0 {
		lock.Version = ModulesLockVersion
	}
	if lock.Modules == nil {
		lock.Modules = map[string]Binding{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return xfmt.Errorf("create modules lock dir failed: %w", err)
	}
	payload, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return xfmt.Errorf("encode modules lock file failed: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "modules.lock.*.tmp")
	if err != nil {
		return xfmt.Errorf("create modules lock temp file failed: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(payload); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return xfmt.Errorf("write modules lock temp file failed: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return xfmt.Errorf("close modules lock temp file failed: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return xfmt.Errorf("replace modules lock file failed: %w", err)
	}
	return nil
}
