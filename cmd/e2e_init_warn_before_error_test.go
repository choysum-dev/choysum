// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
)

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
