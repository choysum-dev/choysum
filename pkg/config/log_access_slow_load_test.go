// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogAccessSlowMsLoadFromYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	omitPath := filepath.Join(dir, "omit.yaml")
	if err := os.WriteFile(omitPath, []byte("log:\n  level: info\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	omit, err := NewConfig(omitPath)
	if err != nil {
		t.Fatalf("omit load: %v", err)
	}
	if omit.Log == nil || omit.Log.AccessSlowMs != 1000 {
		t.Fatalf("omit AccessSlowMs = %#v, want 1000", omit.Log)
	}

	zeroPath := filepath.Join(dir, "zero.yaml")
	if err := os.WriteFile(zeroPath, []byte("log:\n  level: info\n  accessSlowMs: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zero, err := NewConfig(zeroPath)
	if err != nil {
		t.Fatalf("zero load: %v", err)
	}
	if zero.Log == nil || zero.Log.AccessSlowMs != 0 {
		t.Fatalf("explicit 0 AccessSlowMs = %#v, want 0", zero.Log)
	}

	customPath := filepath.Join(dir, "custom.yaml")
	if err := os.WriteFile(customPath, []byte("log:\n  accessSlowMs: 2500\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	custom, err := NewConfig(customPath)
	if err != nil {
		t.Fatalf("custom load: %v", err)
	}
	if custom.Log == nil || custom.Log.AccessSlowMs != 2500 {
		t.Fatalf("custom AccessSlowMs = %#v, want 2500", custom.Log)
	}
}
