// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
)

func TestCLIRunRejectsLegacyBootstrapFlags(t *testing.T) {
	output, code := runCLI(t, "run", "--non-interactive")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --non-interactive") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}
