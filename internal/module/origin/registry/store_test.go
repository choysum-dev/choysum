// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreLoadResolveAndSave(t *testing.T) {
	home := t.TempDir()
	store := NewStore(WithHomeDir(home), WithDefaultChoysumPath(t.TempDir()))

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load(default) error = %v", err)
	}
	if cfg.Registries[DefaultRegistryAlias].IndexURL != DefaultRegistryIndexURL {
		t.Fatalf("unexpected default registry index url: %#v", cfg.Registries)
	}

	cfg.Registries["corp"] = Entry{IndexURL: "https://index.acme.dev/v1/index.json", AuthRef: "token:corp"}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	entry, err := store.Resolve("corp")
	if err != nil {
		t.Fatalf("Resolve(corp) error = %v", err)
	}
	if entry.IndexURL != "https://index.acme.dev/v1/index.json" || entry.AuthRef != "token:corp" {
		t.Fatalf("unexpected resolved registry entry: %#v", entry)
	}

	if _, err := store.Resolve("missing"); err == nil {
		t.Fatal("expected Resolve(missing) error")
	}

	if _, err := filepath.Abs(home); err != nil {
		t.Fatalf("abs home failed: %v", err)
	}
}

func TestStoreLoadRequiresDefaultChoysumPath(t *testing.T) {
	store := NewStore(WithHomeDir(t.TempDir()))
	if _, err := store.Load(); err == nil {
		t.Fatal("expected Load() to fail when defaultChoysumPath is missing")
	}
}

func TestStoreResolveValidationPaths(t *testing.T) {
	store := NewStore(WithHomeDir(t.TempDir()), WithDefaultChoysumPath(t.TempDir()))

	if _, err := store.Resolve("   "); err == nil || !strings.Contains(err.Error(), "registry alias is empty") {
		t.Fatalf("expected empty alias error, got %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Registries["broken"] = Entry{IndexURL: "   "}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := store.Resolve("broken"); err == nil || !strings.Contains(err.Error(), "empty indexURL") {
		t.Fatalf("expected empty indexURL error, got %v", err)
	}
}

func TestStoreFilePathDefaultsAndCloneNormalization(t *testing.T) {
	root := t.TempDir()
	store := NewStore(WithDefaultChoysumPath(root))

	path, err := store.filePath()
	if err != nil {
		t.Fatalf("filePath() error = %v", err)
	}
	if want := filepath.Join(root, "registries.yaml"); path != want {
		t.Fatalf("filePath() = %q, want %q", path, want)
	}

	if err := store.Save(&Config{}); err != nil {
		t.Fatalf("Save(empty config) error = %v", err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("Version = %d, want 1", cfg.Version)
	}
	if got := cfg.Registries[DefaultRegistryAlias].IndexURL; got != DefaultRegistryIndexURL {
		t.Fatalf("default indexURL = %q, want %q", got, DefaultRegistryIndexURL)
	}
}

func TestStorePathValidationAndOptionReset(t *testing.T) {
	rootStore := NewStore(WithDefaultChoysumPath(string(filepath.Separator)))
	if _, err := rootStore.Load(); err == nil || !strings.Contains(err.Error(), "non-root directory") {
		t.Fatalf("expected non-root path error, got %v", err)
	}

	resetStore := NewStore(WithDefaultChoysumPath(t.TempDir()), WithDefaultChoysumPath("   "))
	if _, err := resetStore.Load(); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath is required") {
		t.Fatalf("expected required defaultChoysumPath error after reset, got %v", err)
	}
}
