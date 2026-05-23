// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
)

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
