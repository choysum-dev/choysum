// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageGeneratedPbIntoSpecsDir(t *testing.T) {
	runDir := t.TempDir()
	specsDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	pbFile := filepath.Join(runDir, ".choysum", "generated", "web", "auth", "pb", "auth_pb.ts")
	if err := os.MkdirAll(filepath.Dir(pbFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pbFile, []byte("export const User = {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cleanup, err := stageGeneratedPbIntoSpecsDir(specsDir, runtimePath)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	defer cleanup()

	link := filepath.Join(specsDir, ".generated", "auth_pb.ts")
	st, err := os.Stat(link)
	if err != nil {
		t.Fatalf("staged file missing: %v", err)
	}
	if !st.Mode().IsRegular() {
		t.Fatalf("expected regular file, mode=%v", st.Mode())
	}
	raw, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "export const User = {}\n" {
		t.Fatalf("staged content = %q", raw)
	}
}
