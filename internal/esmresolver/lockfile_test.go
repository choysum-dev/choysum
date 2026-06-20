// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadLockfile_NotExist(t *testing.T) {
	lock, err := ReadLockfile("/nonexistent/path/esm.lock")
	if err != nil {
		t.Fatalf("ReadLockfile returned error for nonexistent file: %v", err)
	}
	if lock != nil {
		t.Fatal("expected nil lockfile for nonexistent file")
	}
}

func TestReadLockfile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "esm.lock")

	lock := &EsmLockfile{
		Version: 1,
		Packages: map[string]LockEntry{
			"kysely":         {Version: "0.27.6", Resolved: "https://esm.sh/kysely@0.27.6"},
			"@scope/pkg/sub": {Version: "1.2.3", Resolved: "https://esm.sh/@scope/pkg@1.2.3"},
		},
	}
	if err := WriteLockfile(path, lock); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	loaded, err := ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil lockfile")
	}
	if loaded.Version != 1 {
		t.Fatalf("version = %d, want 1", loaded.Version)
	}
	if len(loaded.Packages) != 2 {
		t.Fatalf("packages len = %d, want 2", len(loaded.Packages))
	}
}

func TestReadLockfile_BadVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "esm.lock")
	if err := os.WriteFile(path, []byte(`{"version":99,"packages":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadLockfile(path)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestReadLockfile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "esm.lock")
	if err := os.WriteFile(path, []byte(`not json`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadLockfile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestWriteLockfile_Atomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "esm.lock")

	lock := &EsmLockfile{Packages: map[string]LockEntry{"a": {Version: "1.0"}}}
	if err := WriteLockfile(path, lock); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	// Verify tmp file was cleaned up.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("expected tmp file to not exist after successful write")
	}

	// Verify content is valid JSON.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed EsmLockfile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("written file is not valid JSON: %v", err)
	}
	if parsed.Version != 1 {
		t.Fatalf("version = %d, want 1", parsed.Version)
	}
}

func TestWriteLockfile_NilLock(t *testing.T) {
	if err := WriteLockfile("/tmp/should-not-create.lock", nil); err != nil {
		t.Fatalf("WriteLockfile(nil) should be a no-op, got: %v", err)
	}
}

func TestLookupLockedSpec(t *testing.T) {
	lock := &EsmLockfile{
		Version: 1,
		Packages: map[string]LockEntry{
			"kysely":             {Version: "0.27.6"},
			"@bufbuild/protobuf": {Version: "2.12.0"},
		},
	}

	tests := []struct {
		name string
		spec string
		want string
	}{
		{"found simple", "kysely", "kysely@0.27.6"},
		{"found scoped", "@bufbuild/protobuf", "@bufbuild/protobuf@2.12.0"},
		{"not found", "unknown-pkg", "unknown-pkg"},
		{"empty lock", "kysely", "kysely"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l *EsmLockfile
			if tt.name != "empty lock" {
				l = lock
			}
			got := LookupLockedSpec(l, tt.spec)
			if got != tt.want {
				t.Fatalf("LookupLockedSpec(%q) = %q, want %q", tt.spec, got, tt.want)
			}
		})
	}
}

func TestLockSpecifier(t *testing.T) {
	tests := []struct {
		spec    string
		version string
		want    string
	}{
		{"kysely", "0.27.6", "kysely@0.27.6"},
		{"@scope/pkg", "1.2.3", "@scope/pkg@1.2.3"},
		{"kysely@1.0.0", "2.0.0", "kysely@2.0.0"},         // strips existing version
		{"@scope/pkg@1.0.0", "2.0.0", "@scope/pkg@2.0.0"}, // strips existing
		{"", "1.0.0", ""},
		{"pkg", "", ""},
		{"  pkg  ", "  1.0  ", "pkg@1.0"},
	}

	for _, tt := range tests {
		got := lockSpecifier(tt.spec, tt.version)
		if got != tt.want {
			t.Fatalf("lockSpecifier(%q, %q) = %q, want %q", tt.spec, tt.version, got, tt.want)
		}
	}
}

func TestResolver_LockedSpecifier_Integration(t *testing.T) {
	dir := t.TempDir()
	lockfilePath := filepath.Join(dir, "esm.lock")

	lock := &EsmLockfile{
		Version: 1,
		Packages: map[string]LockEntry{
			"kysely": {Version: "0.27.6"},
		},
	}
	if err := WriteLockfile(lockfilePath, lock); err != nil {
		t.Fatal(err)
	}

	r := New(WithLockfile(lockfilePath))
	got := r.lockedSpecifier("kysely")
	if want := "kysely@0.27.6"; got != want {
		t.Fatalf("lockedSpecifier = %q, want %q", got, want)
	}

	// Unknown package passes through unchanged.
	got = r.lockedSpecifier("unknown")
	if want := "unknown"; got != want {
		t.Fatalf("lockedSpecifier = %q, want %q", got, want)
	}
}

// ---- WriteLockfile edge case tests ----

func TestWriteLockfile_InvalidPath(t *testing.T) {
	err := WriteLockfile("/dev/null/should-fail/esm.lock", &EsmLockfile{
		Version:  1,
		Packages: map[string]LockEntry{"a": {Version: "1.0"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// ---- ReadLockfile edge case tests ----

func TestReadLockfile_ZeroVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "esm.lock")
	if err := os.WriteFile(path, []byte(`{"version":0,"packages":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadLockfile(path)
	if err == nil {
		t.Fatal("expected error for unsupported version 0")
	}
}

func TestReadLockfile_NilPackages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "esm.lock")
	if err := os.WriteFile(path, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	lock, err := ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile should handle nil packages: %v", err)
	}
	if lock.Packages == nil {
		t.Fatal("packages should be non-nil after read")
	}
	if len(lock.Packages) != 0 {
		t.Fatalf("packages len = %d, want 0", len(lock.Packages))
	}
}

// ---- LookupLockedSpec edge case tests ----

func TestLookupLockedSpec_EmptySpec(t *testing.T) {
	lock := &EsmLockfile{Version: 1, Packages: map[string]LockEntry{"pkg": {Version: "1.0"}}}
	got := LookupLockedSpec(lock, "")
	if got != "" {
		t.Fatalf("LookupLockedSpec with empty spec = %q, want empty", got)
	}
}
