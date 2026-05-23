// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
)

func TestCLIErrorBlockIsLastOutput(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "run", "--config", " ")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	assertLastErrorBlock(t, stderr)
}
