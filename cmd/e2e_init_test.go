package

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInitCommandRemoved(t *testing.T) {
	output, code := runCLI(t, "init")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", output)
	}
}
func TestCLIInitCommandRemovedStderrOnly(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "init", "--non-interactive")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", stderr)
	}
}
func TestCLIWarnBeforeErrorBlock(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real: %v", err)
	}
	linkDir := filepath.Join(base, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	dsn := filepath.Join(linkDir, "missing.db")
	configPath := writeTempConfigWithDSN(t, "sqlite", dsn, "")

	stdout, stderr, code := runCLISeparated(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR:") {
		t.Fatalf("expected error block, got %q", stderr)
	}

	lines := lastNonEmptyLines(stderr)
	warnIndex := -1
	errorIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "WARN: sqlite parent directory is a symlink") {
			warnIndex = i
		}
		if strings.HasPrefix(line, "ERROR:") {
			errorIndex = i
		}
	}
	if warnIndex == -1 {
		t.Fatalf("expected symlink warning in stderr")
	}
	if errorIndex == -1 {
		t.Fatalf("expected error line in stderr")
	}
	if warnIndex >= errorIndex {
		t.Fatalf("expected warning before error block")
	}
}
