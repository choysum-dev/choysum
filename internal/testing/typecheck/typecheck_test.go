// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"io"
	"path/filepath"
	"testing"
)

func TestTypecheckApp_NoNode(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")

	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}, "auth")
	if err != nil {
		t.Fatalf("TypecheckApp without Node: %v", err)
	}
}

func TestShouldSuggestTypeFetchFromOutput(t *testing.T) {
	if !shouldSuggestTypeFetchFromOutput("error TS2307: Cannot find module") {
		t.Fatal("expected TS2307 to suggest type-fetch")
	}
	if shouldSuggestTypeFetchFromOutput("error TS2322: Type mismatch") {
		t.Fatal("did not expect type-fetch for unrelated diagnostic")
	}
}
