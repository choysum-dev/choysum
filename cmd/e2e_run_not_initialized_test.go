// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
)

func TestCLIRunRejectsLegacyAdminUsernameFlag(t *testing.T) {
	output, code := runCLI(t, "run", "--admin-username", "admin")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --admin-username") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}

func TestCLIRunRejectsLegacyAdminPasswordFileFlag(t *testing.T) {
	output, code := runCLI(t, "run", "--admin-password-file", "pw.txt")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --admin-password-file") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}

func TestCLIRunRejectsLegacyAdminPasswordStdinFlag(t *testing.T) {
	output, code := runCLI(t, "run", "--admin-password-stdin")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --admin-password-stdin") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}
