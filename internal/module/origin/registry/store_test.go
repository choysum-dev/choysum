// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"os"
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
	cfg.Registries["legacy"] = Entry{IndexURL: "https://github.com/acme/registry"}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := store.Resolve("broken"); err == nil || !strings.Contains(err.Error(), "empty indexURL") {
		t.Fatalf("expected empty indexURL error, got %v", err)
	}
	if _, err := store.Resolve("legacy"); err == nil || !strings.Contains(err.Error(), "must point to an index.json resource") {
		t.Fatalf("expected index.json validation error, got %v", err)
	}
}

func TestIsValidRegistryIndexURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		raw   string
		valid bool
	}{
		{name: "valid url", raw: "https://index.example.com/v1/index.json", valid: true},
		{name: "valid url with spaces", raw: "  https://index.example.com/v1/index.json  ", valid: true},
		{name: "invalid parse", raw: "://bad-url", valid: false},
		{name: "invalid scheme", raw: "ftp://index.example.com/v1/index.json", valid: false},
		{name: "missing host", raw: "https:///v1/index.json", valid: false},
		{name: "invalid path", raw: "https://index.example.com/v1/catalog.json", valid: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidRegistryIndexURL(tt.raw); got != tt.valid {
				t.Fatalf("isValidRegistryIndexURL(%q) = %v, want %v", tt.raw, got, tt.valid)
			}
		})
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

func TestStoreLoadMigratesLegacyURLFieldToIndexURL(t *testing.T) {
	home := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewStore(WithHomeDir(home), WithDefaultChoysumPath(defaultChoysumPath))

	path, err := store.filePath()
	if err != nil {
		t.Fatalf("filePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	legacyConfig := "version: 1\n" +
		"registries:\n" +
		"  official:\n" +
		"    url: https://legacy-official.example.com/v1/index.json\n" +
		"    authRef: token://official\n" +
		"  corp:\n" +
		"    url: https://legacy-corp.example.com/v1/index.json\n" +
		"    authRef: token://corp\n"
	if err := os.WriteFile(path, []byte(legacyConfig), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	entry, ok := cfg.Registries[DefaultRegistryAlias]
	if !ok {
		t.Fatalf("expected default alias %q to exist", DefaultRegistryAlias)
	}
	if entry.IndexURL != "https://legacy-official.example.com/v1/index.json" {
		t.Fatalf("official indexURL = %q, want %q", entry.IndexURL, "https://legacy-official.example.com/v1/index.json")
	}
	if entry.AuthRef != "token://official" {
		t.Fatalf("official authRef = %q, want %q", entry.AuthRef, "token://official")
	}
	if entry.URL != "" {
		t.Fatalf("official legacy url field should be cleared after migration, got %q", entry.URL)
	}

	corpEntry, err := store.Resolve("corp")
	if err != nil {
		t.Fatalf("Resolve(corp) error = %v", err)
	}
	if corpEntry.IndexURL != "https://legacy-corp.example.com/v1/index.json" {
		t.Fatalf("corp indexURL = %q, want %q", corpEntry.IndexURL, "https://legacy-corp.example.com/v1/index.json")
	}
	if corpEntry.AuthRef != "token://corp" {
		t.Fatalf("corp authRef = %q, want %q", corpEntry.AuthRef, "token://corp")
	}
}

func TestStoreLoadBackfillsOfficialIndexURLWhenMissing(t *testing.T) {
	home := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewStore(WithHomeDir(home), WithDefaultChoysumPath(defaultChoysumPath))

	path, err := store.filePath()
	if err != nil {
		t.Fatalf("filePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	missingIndexConfig := "version: 1\n" +
		"registries:\n" +
		"  official:\n" +
		"    authRef: token://official\n"
	if err := os.WriteFile(path, []byte(missingIndexConfig), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	entry, ok := cfg.Registries[DefaultRegistryAlias]
	if !ok {
		t.Fatalf("expected default alias %q to exist", DefaultRegistryAlias)
	}
	if entry.IndexURL != DefaultRegistryIndexURL {
		t.Fatalf("official indexURL = %q, want %q", entry.IndexURL, DefaultRegistryIndexURL)
	}
	if entry.AuthRef != "token://official" {
		t.Fatalf("official authRef = %q, want %q", entry.AuthRef, "token://official")
	}
}

func TestStoreLoadMigratesLegacyOfficialGitHubRegistryURLToDefaultIndex(t *testing.T) {
	home := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewStore(WithHomeDir(home), WithDefaultChoysumPath(defaultChoysumPath))

	path, err := store.filePath()
	if err != nil {
		t.Fatalf("filePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	legacyConfig := "version: 1\n" +
		"registries:\n" +
		"  official:\n" +
		"    url: https://github.com/project-choysum/registry/\n" +
		"    authRef: token://official\n"
	if err := os.WriteFile(path, []byte(legacyConfig), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	entry, ok := cfg.Registries[DefaultRegistryAlias]
	if !ok {
		t.Fatalf("expected default alias %q to exist", DefaultRegistryAlias)
	}
	if entry.IndexURL != DefaultRegistryIndexURL {
		t.Fatalf("official indexURL = %q, want %q", entry.IndexURL, DefaultRegistryIndexURL)
	}
	if entry.AuthRef != "token://official" {
		t.Fatalf("official authRef = %q, want %q", entry.AuthRef, "token://official")
	}
	if entry.URL != "" {
		t.Fatalf("official legacy url field should be cleared after migration, got %q", entry.URL)
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
