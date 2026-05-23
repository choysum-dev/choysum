// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
)

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
