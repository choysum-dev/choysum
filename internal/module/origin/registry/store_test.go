// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"path/filepath"
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
