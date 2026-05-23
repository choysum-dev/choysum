// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWithCompilerFsExposesFileHelpers(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("choysum"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	symlinkPath := filepath.Join(tempDir, "sample-link.txt")
	if err := os.Symlink(filePath, symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	engine := newTestQuickjsEngine(t, WithCompilerFs())
	quotedFile := strconv.Quote(filePath)
	quotedLink := strconv.Quote(symlinkPath)

	if !evalBool(t, engine, "compilerFs.fileExists("+quotedFile+")") {
		t.Fatal("expected fileExists to return true for existing file")
	}
	if evalBool(t, engine, "compilerFs.fileExists("+strconv.Quote(filepath.Join(tempDir, "missing.txt"))+")") {
		t.Fatal("expected fileExists to return false for missing file")
	}
	if got := evalString(t, engine, "compilerFs.readFile("+quotedFile+")"); got != "choysum" {
		t.Fatalf("readFile = %q, want choysum", got)
	}
	if got := evalString(t, engine, "compilerFs.realpath("+quotedLink+")"); got != filePath {
		want, err := filepath.EvalSymlinks(filePath)
		if err != nil {
			t.Fatalf("EvalSymlinks(filePath): %v", err)
		}
		got = filepath.Clean(got)
		want = filepath.Clean(want)
		if got != want {
			t.Fatalf("realpath = %q, want %q", got, want)
		}
	}

	errText := evalString(t, engine, "(() => { try { compilerFs.readFile('missing-file'); return ''; } catch (e) { return String(e); } })()")
	if !strings.Contains(errText, "missing-file") {
		t.Fatalf("expected readFile error to mention missing file, got %q", errText)
	}
}
