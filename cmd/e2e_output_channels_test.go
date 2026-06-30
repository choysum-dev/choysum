package

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
cmd

import (
	"strings"
	"testing"
)

func TestCLIErrorBlockUsesStderrOnly(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "run", "--config", " ")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	assertLastErrorBlock(t, stderr)
}
func TestCLIE2EWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "test", "e2e", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
func TestCLIInitErrorBlockIsLastAndStderrOnly(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "init", "--admin-password-stdin")
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
func TestCLIInstallWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "install", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
func TestCLIRunInfoWritesToStderrOnly(t *testing.T) {
	configPath, _, _ := writeTempInitializedRunConfigWithDB(t, false)

	stdout, stderr, _ := runCLIUntilLineSeparated(t, func(line string) bool {
		return strings.Contains(line, "http server listening")
	}, "run", "--config", configPath)

	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "http server listening") {
		t.Fatalf("expected stderr to contain listening log, got %q", stderr)
	}
	if strings.Contains(stderr, "ERROR: ") || strings.Contains(stderr, "REASON: ") {
		t.Fatalf("did not expect error block in stderr, got %q", stderr)
	}
	if strings.Contains(stderr, "server starting; NEXT: open") {
		t.Fatalf("did not expect CLI startup hint in stderr, got %q", stderr)
	}
}
func TestCLITestWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "test", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
func TestCLITypecheckWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "test", "typecheck", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
